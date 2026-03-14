package cli

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/thomasbarrett/soren/internal/agent"
	"github.com/thomasbarrett/soren/internal/config"
	"github.com/thomasbarrett/soren/internal/tools"
	"github.com/thomasbarrett/soren/internal/tools/builtin"
	"github.com/thomasbarrett/soren/internal/transcript"
	"github.com/thomasbarrett/soren/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/cobra"
)

const (
	DefaultBaseURL   = "http://10.0.0.76:8000/v1"
	DefaultModel     = "Qwen/Qwen3-14B-AWQ"
	DefaultAgentName = "default"
)

// NewRootCommand creates the root cobra command
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "soren",
		Short: "Soren - An AI agent with MCP tool support",
		Long:  `Soren is an AI agent that can use MCP (Model Context Protocol) tools to interact with various services and perform tasks.`,
		RunE:  run,
	}

	// Global flags
	rootCmd.PersistentFlags().String("model-url", DefaultBaseURL, "Base URL for the LLM API")
	rootCmd.PersistentFlags().String("model", DefaultModel, "Model name to use")
	rootCmd.PersistentFlags().String("agent", DefaultAgentName, "Agent name (loads from SOREN.md if 'default', or .soren/agents/<name>.md)")

	// Add subcommands
	rootCmd.AddCommand(NewMCPCommand())

	return rootCmd
}

// run runs the interactive agent (default command)
func run(cmd *cobra.Command, args []string) error {
	modelURL, _ := cmd.Flags().GetString("model-url")
	model, _ := cmd.Flags().GetString("model")
	agentName, _ := cmd.Flags().GetString("agent")

	ctx := context.Background()

	// Load configuration
	settings, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Initialize session
	sessionID := uuid.New()

	modelClient := openai.NewClient(option.WithBaseURL(modelURL))
	registry := tools.NewRegistry()

	t, err := transcript.NewTranscript(fmt.Sprintf(".soren/sessions/%s.jsonl", sessionID))
	if err != nil {
		return fmt.Errorf("failed to create transcript: %w", err)
	}
	defer t.Close()

	cfg := agent.Config{
		Transcript:   t,
		ToolProvider: registry,
		ModelClient:  modelClient,
		Model:        model,
		Name:         agentName,
		MaxTurns:     32,
	}

	// Create embedded server
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "soren",
			Version: "v0.1.0",
		},
		nil,
	)

	// Set up in-memory transport
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	// Start server in background
	go func() {
		if err := server.Run(ctx, serverTransport); err != nil {
			fmt.Printf("embedded server error: %v\n", err)
		}
	}()

	// Register builtin tools
	builtin.Register(server)

	// Register Agent tool for sub-agent delegation
	if err := agent.Register(server, cfg); err != nil {
		return fmt.Errorf("failed to register agent tool: %w", err)
	}

	if err := registry.Connect(ctx, "builtin", clientTransport); err != nil {
		return fmt.Errorf("failed to register builtin tools: %w", err)
	}

	for _, server := range settings.MCPServers {
		if err := registry.Connect(ctx, server.Name, &mcp.CommandTransport{
			Command: exec.Command(server.Command, server.Args...),
		}); err != nil {
			return err
		}
	}

	// Create agent and session
	a, err := agent.NewAgent(ctx, cfg)
	if err != nil {
		return err
	}

	session := a.NewSession()

	// Run UI
	p := tea.NewProgram(ui.New(session))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running program: %w", err)
	}

	return nil
}
