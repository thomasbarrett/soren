package cli

import (
	"fmt"
	"strings"

	"github.com/thomasbarrett/soren/internal/config"

	"github.com/spf13/cobra"
)

// NewMCPCommand creates the MCP command group
func NewMCPCommand() *cobra.Command {
	mcpCmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage MCP servers",
		Long:  `Commands for managing Model Context Protocol (MCP) servers.`,
	}

	// MCP add command
	mcpAddCmd := &cobra.Command{
		Use:   "add [flags] <name> -- <command> [args...]",
		Short: "Add a new MCP server",
		Long: `Add a new MCP server configuration.

Examples:
  # Add Airtable server with environment variable
  soren mcp add --transport stdio --env AIRTABLE_API_KEY=YOUR_KEY airtable -- npx -y airtable-mcp-server

  # Add a local server  
  soren mcp add --transport stdio local-server -- ./my-mcp-server
`,
		Args: cobra.MinimumNArgs(1),
		RunE: runMCPAdd,
	}

	// Add flags to MCP add command
	mcpAddCmd.Flags().String("transport", "stdio", "Transport type (stdio)")
	mcpAddCmd.Flags().StringArray("env", []string{}, "Environment variables (format: KEY=VALUE)")

	// Stop parsing flags after first positional argument to preserve -- separator
	mcpAddCmd.Flags().SetInterspersed(false)

	mcpCmd.AddCommand(mcpAddCmd)
	return mcpCmd
}

// runMCPAdd handles the mcp add command
func runMCPAdd(cmd *cobra.Command, args []string) error {
	transport, _ := cmd.Flags().GetString("transport")
	envVars, _ := cmd.Flags().GetStringArray("env")

	// Find the -- separator
	separatorIndex := -1
	for i, arg := range args {
		if arg == "--" {
			separatorIndex = i
			break
		}
	}

	if separatorIndex == -1 {
		return fmt.Errorf("command separator '--' is required")
	}

	if separatorIndex == 0 {
		return fmt.Errorf("server name is required before '--'")
	}

	if separatorIndex >= len(args)-1 {
		return fmt.Errorf("server command is required after '--'")
	}

	serverName := args[0]
	serverCmd := args[separatorIndex+1:]

	return mcpServerAdd(serverName, transport, envVars, serverCmd)
}

// mcpServerAdd adds a new MCP server to the configuration
func mcpServerAdd(name, transport string, envVars, command []string) error {
	settings, err := config.LoadSettings()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	// Check if server already exists
	for _, server := range settings.MCPServers {
		if server.Name == name {
			return fmt.Errorf("MCP server '%s' already exists", name)
		}
	}

	// Create new server configuration
	newServer := config.MCPServer{
		Name:      name,
		Transport: transport,
		Command:   command[0],
		Args:      command[1:],
		Env:       envVars,
	}

	// Add to settings
	settings.MCPServers = append(settings.MCPServers, newServer)

	// Save settings
	if err := config.SaveSettings(settings); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	fmt.Printf("Successfully added MCP server '%s'\n", name)
	fmt.Printf("  Transport: %s\n", transport)
	fmt.Printf("  Command: %s %s\n", command[0], strings.Join(command[1:], " "))
	if len(envVars) > 0 {
		fmt.Printf("  Environment: %s\n", strings.Join(envVars, ", "))
	}

	return nil
}
