package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TaskInput defines the parameters for the task delegation tool
type TaskInput struct {
	Prompt    string `json:"prompt"`
	AgentName string `json:"agent_name,omitempty"` // optional; defaults to "default"
}

// TaskOutput is the result of task execution
type TaskOutput struct {
	Result string `json:"result"`
	Error  string `json:"error,omitempty"`
}

// Register wires the Agent delegation tool into the MCP server
func Register(server *mcp.Server, cfg Config) error {
	// Discover available agents
	availableAgents, err := LoadAgents()
	if err != nil {
		return err
	}

	// Build a map of agents for quick lookup
	agentMap := make(map[string]AgentMeta)
	for _, agent := range availableAgents {
		agentMap[agent.Name] = agent
	}

	// Build description with available agents
	var sb strings.Builder
	sb.WriteString(`Create a sub-agent. The sub-agent will be spawned with the specified agent name and will execute the task as a single-shot operation.`)

	if len(availableAgents) > 0 {
		sb.WriteString("\n\nAvailable agents:")
		for _, agent := range availableAgents {
			sb.WriteString("\n- " + agent.Name)
			if agent.Description != "" {
				sb.WriteString(": " + agent.Description)
			}
		}
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "Agent",
		Description: sb.String(),
	}, func(ctx context.Context, req *mcp.CallToolRequest, args TaskInput) (*mcp.CallToolResult, TaskOutput, error) {
		meta, found := agentMap[args.AgentName]
		if !found {
			return nil, TaskOutput{Error: fmt.Sprintf("agent '%s' not found", args.AgentName)}, nil
		}

		// Create sub-agent config with the specified name
		cfg2 := cfg
		cfg2.Name = args.AgentName
		cfg2.Tools = ToolSet{
			Parent:   &cfg.Tools,
			Allow:    meta.Tools,
			Disallow: meta.DisallowedTools,
		}

		// Create and run the sub-agent
		subAgent, err := NewAgent(ctx, cfg2)
		if err != nil {
			return nil, TaskOutput{Error: err.Error()}, err
		}

		result, err := subAgent.NewSession().Send(ctx, args.Prompt)
		if err != nil {
			return nil, TaskOutput{Error: err.Error()}, err
		}

		return nil, TaskOutput{Result: result}, nil
	})

	return nil
}
