package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/thomasbarrett/soren/internal/history"
	"github.com/thomasbarrett/soren/internal/tools"
	"github.com/thomasbarrett/soren/internal/transcript"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	openai "github.com/openai/openai-go"
)

var (
	grey      = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render
	blue      = lipgloss.NewStyle().Foreground(lipgloss.Color("75")).Render
	userStyle = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "255", Dark: "255"}).
			Foreground(lipgloss.AdaptiveColor{Light: "0", Dark: "0"})
)

// Config holds the configuration for an agent
type Config struct {
	Transcript   *transcript.Transcript
	ToolProvider tools.Provider
	ModelClient  openai.Client
	ConfigPath   string // Path to config.yaml for dynamic prompt building
	Model        string
	Name         string
	MaxTurns     int
	Tools        ToolSet
}

type Viewable interface {
	View(width int) string
}

type inFlightResponse struct{}

func (m *inFlightResponse) View(w int) string {
	return grey("⏺") + " Thinking...\n\n"
}

// Agent wraps a config with conversation history
type Agent struct {
	Config            Config
	History           *history.History
	tools             []openai.ChatCompletionToolParam
	systemPrompt      string // Built on demand
	inFlight          Viewable
	inFlightToolCalls int
	busy              bool
}

// NewAgent creates a new agent with the given config and builds its system prompt
func NewAgent(ctx context.Context, config Config) (*Agent, error) {
	history := history.NewHistory(config.Transcript)

	agent := &Agent{
		Config:  config,
		History: history,
	}

	err := agent.buildSystemPrompt(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build system prompt: %w", err)
	}

	return agent, nil
}

func (a *Agent) Update(msg openai.ChatCompletionMessageParamUnion) tea.Cmd {
	a.History.Add(msg)
	a.busy = true

	if msg.OfUser != nil || msg.OfTool != nil {
		a.inFlight = &inFlightResponse{}

		if msg.OfTool != nil {
			a.inFlightToolCalls -= 1
			if a.inFlightToolCalls > 0 {
				return nil
			}
		}

		return func() tea.Msg {
			resp, err := a.Config.ModelClient.Chat.Completions.New(context.Background(), openai.ChatCompletionNewParams{
				Model: a.Config.Model,
				Tools: a.tools,
				ToolChoice: openai.ChatCompletionToolChoiceOptionUnionParam{
					OfAuto: openai.String(string(openai.ChatCompletionToolChoiceOptionAutoAuto)),
				},
				Messages: a.History.Messages(),
			})
			if err != nil {
				log.Fatal(err)
			}

			return resp.Choices[0].Message.ToParam()
		}
	} else if msg.OfAssistant != nil {
		a.inFlight = nil

		var cmds []tea.Cmd
		for _, call := range msg.OfAssistant.ToolCalls {
			arguments := make(map[string]any)
			if err := json.Unmarshal([]byte(call.Function.Arguments), &arguments); err != nil {
				log.Fatal(err)
			}

			params := &mcp.CallToolParams{
				Name:      call.Function.Name,
				Arguments: arguments,
			}

			cmds = append(cmds, func() tea.Msg {
				res, err := a.Config.ToolProvider.CallTool(context.Background(), params)
				if err != nil {
					log.Fatal(err)
				}

				bytes, err := json.Marshal(res.StructuredContent)
				if err != nil {
					log.Fatal(err)
				}

				return openai.ToolMessage(string(bytes), call.ID)
			})
		}

		if len(cmds) > 0 {
			a.inFlightToolCalls = len(cmds)
			return tea.Batch(cmds...)
		}
	}

	a.busy = false

	return nil
}

func (a *Agent) Busy() bool {
	return a.busy
}

// Run executes a single agent loop for the given user query and updates history
func (a *Agent) Run(ctx context.Context, query string) (string, error) {

	// Append user query to history
	a.History.Add(openai.UserMessage(query))

	for turn := 1; turn <= a.Config.MaxTurns; turn++ {
		resp, err := a.Config.ModelClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    a.Config.Model,
			Messages: a.History.Messages(),
			Tools:    a.tools,
			ToolChoice: openai.ChatCompletionToolChoiceOptionUnionParam{
				OfAuto: openai.String(string(openai.ChatCompletionToolChoiceOptionAutoAuto)),
			},
		})
		if err != nil {
			return "", err
		}

		a.History.Add(resp.Choices[0].Message.ToParam())

		if len(resp.Choices[0].Message.ToolCalls) > 0 {
			for _, call := range resp.Choices[0].Message.ToolCalls {

				arguments := make(map[string]any)
				if err := json.Unmarshal([]byte(call.Function.Arguments), &arguments); err != nil {
					return "", err
				}

				params := &mcp.CallToolParams{
					Name:      call.Function.Name,
					Arguments: arguments,
				}

				res, err := a.Config.ToolProvider.CallTool(ctx, params)
				if err != nil {
					return "", err
				}

				bytes, err := json.Marshal(res.StructuredContent)
				if err != nil {
					return "", err
				}

				a.History.Add(openai.ToolMessage(string(bytes), call.ID))
			}
		} else {
			return resp.Choices[0].Message.Content, nil
		}
	}

	return "", fmt.Errorf("max turns (%d) reached without final response", a.Config.MaxTurns)
}

func (a *Agent) View(width int) string {
	var b strings.Builder

	for _, msg := range a.History.Messages() {
		if msg.OfUser != nil {
			content := msg.OfUser.Content.OfString.String()
			b.WriteString(fmt.Sprintf("%s\n\n", userStyle.Render(fmt.Sprintf("\u276f %s ", content))))
		} else if msg.OfAssistant != nil {
			if len(msg.OfAssistant.ToolCalls) > 0 {
				for _, call := range msg.OfAssistant.ToolCalls {
					b.WriteString(fmt.Sprintf(blue("⏺")+" "+"%s(%s)\n", call.Function.Name, call.Function.Arguments))
					resp := a.History.FindResponse(call.ID)
					if resp != nil {
						content := resp.OfTool.Content.OfString.String()
						b.WriteString("  \u23bf  Done (" + content + ")\n\n")
					} else {
						b.WriteString("  \u23bf  " + grey("⏺") + " Waiting for tool response...\n\n")
					}
				}
			} else {
				content := msg.OfAssistant.Content.OfString.String()
				b.WriteString(fmt.Sprintf("%s\n\n", grey("⏺")+" "+strings.TrimSpace(withLinePrefix(content, "  "))))
			}
		}
	}

	if a.inFlight != nil {
		b.WriteString(a.inFlight.View(width))
	}

	return b.String()
}

func withLinePrefix(s, prefix string) string {
	lines := strings.Split(s, "\n")

	for i, line := range lines {
		lines[i] = prefix + line
	}

	return strings.Join(lines, "\n")
}
