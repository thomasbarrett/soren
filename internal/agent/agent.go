package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	openai "github.com/openai/openai-go"

	"github.com/thomasbarrett/soren/internal/history"
	"github.com/thomasbarrett/soren/internal/tools"
	"github.com/thomasbarrett/soren/internal/transcript"
)

// Config holds the configuration for an agent
type Config struct {
	Transcript   *transcript.Transcript
	ToolProvider tools.Provider
	ModelClient  openai.Client
	ConfigPath   string // Path to config.yaml for dynamic prompt building
	Model        string
	Name         string
	Tools        ToolSet
}

// Agent holds the configuration and tools for an AI agent.
// Use NewSession to start a conversation.
type Agent struct {
	Config       Config
	tools        []openai.ChatCompletionToolParam
	systemPrompt string
}

// NewAgent creates a new agent with the given config and builds its system prompt and tools.
func NewAgent(ctx context.Context, config Config) (*Agent, error) {
	agent := &Agent{
		Config: config,
	}

	if err := agent.buildSystemPrompt(ctx); err != nil {
		return nil, fmt.Errorf("failed to build system prompt: %w", err)
	}

	return agent, nil
}

// NewSession creates a new conversation session backed by this agent.
func (a *Agent) NewSession() *Session {
	h := history.NewHistory(a.Config.Transcript)
	h.Add(openai.SystemMessage(a.systemPrompt))
	return &Session{agent: a, History: h}
}

// Session holds the conversation history for a single exchange with an agent.
type Session struct {
	agent   *Agent
	History *history.History
}

// StreamEvent is implemented by all events emitted by Session.Stream.
type StreamEvent interface {
	streamEvent()
}

// EventThinking is emitted each time the LLM is called.
type EventThinking struct{}

// EventToolUse is emitted when the LLM requests a tool call.
type EventToolUse struct {
	ID        string
	Name      string
	Arguments string
}

// EventToolResult is emitted when a tool call completes.
type EventToolResult struct {
	CallID  string
	Content string
}

// EventOutput is emitted with the agent's final text response.
type EventOutput struct{ Content string }

// EventThinkingDelta is emitted for each reasoning/thinking token streamed from the model.
type EventThinkingDelta struct{ Token string }

// EventOutputDelta is emitted for each response text token streamed from the model.
type EventOutputDelta struct{ Token string }

// EventError is emitted if the agent encounters an unrecoverable error.
type EventError struct{ Err error }

func (EventThinking) streamEvent()      {}
func (EventToolUse) streamEvent()       {}
func (EventToolResult) streamEvent()    {}
func (EventOutput) streamEvent()        {}
func (EventThinkingDelta) streamEvent() {}
func (EventOutputDelta) streamEvent()   {}
func (EventError) streamEvent()         {}

// Stream sends a user message and emits a StreamEvent for each significant step.
// The returned channel is closed after EventOutput or EventError is sent.
func (s *Session) Stream(ctx context.Context, message string) <-chan StreamEvent {
	ch := make(chan StreamEvent)

	go func() {
		defer close(ch)

		if err := s.History.Add(openai.UserMessage(message)); err != nil {
			ch <- EventError{Err: fmt.Errorf("failed to record user message: %w", err)}
			return
		}

		for {
			ch <- EventThinking{}

			stream := s.agent.Config.ModelClient.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
				Model:    s.agent.Config.Model,
				Messages: s.History.Messages(),
				Tools:    s.agent.tools,
				ToolChoice: openai.ChatCompletionToolChoiceOptionUnionParam{
					OfAuto: openai.String(string(openai.ChatCompletionToolChoiceOptionAutoAuto)),
				},
			})
			acc := openai.ChatCompletionAccumulator{}
			for stream.Next() {
				chunk := stream.Current()
				acc.AddChunk(chunk)

				// Extract vLLM extended fields (reasoning_content, content) from the raw chunk.
				var raw struct {
					Choices []struct {
						Delta struct {
							ReasoningContent string `json:"reasoning_content"`
							Content          string `json:"content"`
						} `json:"delta"`
					} `json:"choices"`
				}
				if err := json.Unmarshal([]byte(chunk.RawJSON()), &raw); err == nil && len(raw.Choices) > 0 {
					if token := raw.Choices[0].Delta.ReasoningContent; token != "" {
						ch <- EventThinkingDelta{Token: token}
					}
					if token := raw.Choices[0].Delta.Content; token != "" {
						ch <- EventOutputDelta{Token: token}
					}
				}
			}
			if err := stream.Err(); err != nil {
				ch <- EventError{Err: err}
				return
			}

			if err := s.History.Add(acc.Choices[0].Message.ToParam()); err != nil {
				ch <- EventError{Err: fmt.Errorf("failed to record assistant message: %w", err)}
				return
			}

			if len(acc.Choices[0].Message.ToolCalls) > 0 {
				for _, call := range acc.Choices[0].Message.ToolCalls {
					ch <- EventToolUse{
						ID:        call.ID,
						Name:      call.Function.Name,
						Arguments: call.Function.Arguments,
					}

					content, _ := s.agent.callTool(ctx, call.ID, call.Function.Name, call.Function.Arguments)
					if err := s.History.Add(openai.ToolMessage(content, call.ID)); err != nil {
						ch <- EventError{Err: fmt.Errorf("failed to record tool result: %w", err)}
						return
					}
					ch <- EventToolResult{CallID: call.ID, Content: content}
				}
			} else {
				ch <- EventOutput{Content: acc.Choices[0].Message.Content}
				return
			}
		}
	}()

	return ch
}

// callTool executes a tool call
func (a *Agent) callTool(ctx context.Context, callID, name, argumentsJSON string) (string, error) {
	arguments := make(map[string]any)
	if err := json.Unmarshal([]byte(argumentsJSON), &arguments); err != nil {
		return "", err
	}

	res, err := a.Config.ToolProvider.CallTool(ctx, &mcp.CallToolParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		return "", err
	}

	var contentStr string
	if res.IsError {
		for _, c := range res.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				contentStr += fmt.Sprintf("<error>%s</error>\n", tc.Text)
			} else {
				contentStr += "<error>unknown error content</error>\n"
			}
		}
	} else {
		resultBytes, err := json.Marshal(res.StructuredContent)
		if err != nil {
			return "", err
		}
		contentStr = string(resultBytes)
	}

	return contentStr, nil
}

// Send sends a user message and returns the agent's final output.
func (s *Session) Send(ctx context.Context, message string) (string, error) {
	for event := range s.Stream(ctx, message) {
		switch e := event.(type) {
		case EventOutput:
			return e.Content, nil
		case EventError:
			return "", e.Err
		}
	}

	return "", fmt.Errorf("stream closed without response")
}
