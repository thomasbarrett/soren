package agent

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/adrg/frontmatter"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/shared"
)

// buildSystemPrompt generates a prompt for MCP tools using the agent's session,
// prefixed with content from .soren/agents/<agent-name>.md if it exists,
// or SOREN.md if the agent name is "default".
func (a *Agent) buildSystemPrompt(ctx context.Context) error {
	parts := []string{}

	var filePath string
	if a.Config.Name == "default" {
		filePath = "AGENTS.md"
	} else {
		filePath = fmt.Sprintf(".soren/agents/%s.md", a.Config.Name)
	}

	data, err := os.ReadFile(filePath)
	if err == nil {
		parts = append(parts, string(data))
	}

	toolsList, err := a.Config.ToolProvider.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		return err
	}

	tools := []openai.ChatCompletionToolParam{}
	for _, t := range toolsList.Tools {
		if t != nil && a.Config.Tools.isToolAllowed(t.Name) {
			tools = append(tools, openai.ChatCompletionToolParam{
				Function: shared.FunctionDefinitionParam{
					Name:        t.Name,
					Description: openai.String(t.Description),
					Parameters:  openai.FunctionParameters(t.InputSchema.(map[string]any)),
				},
			})
		}
	}

	a.tools = tools

	// Add environment section
	environmentSection := getEnvironmentSection()
	parts = append(parts, environmentSection)

	a.systemPrompt = strings.Join(parts, "\n\n")

	return nil
}

// getEnvironmentSection returns a formatted environment section for the system prompt
func getEnvironmentSection() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "(unknown)"
	}

	platform := runtime.GOOS
	shell := os.Getenv("SHELL")

	return fmt.Sprintf("# Environment\nYou have been invoked in the current environment:\n - Primary working directory: %s\n - Platform: %s\n - Shell: %s", cwd, platform, shell)
}

type ToolSet struct {
	Parent   *ToolSet
	Allow    []string
	Disallow []string
}

// isToolAllowed checks if a tool is allowed based on allowed/disallowed lists
func (t *ToolSet) isToolAllowed(name string) bool {
	if t == nil {
		return true
	}

	for _, disallowed := range t.Disallow {
		if name == disallowed {
			return false
		}
	}

	// If Allow is nil, all tools are allowed (subject to disallow list).
	// If Allow is empty, no tools are allowed.
	if t.Allow == nil {
		return t.Parent.isToolAllowed(name)
	}

	for _, allowed := range t.Allow {
		if name == allowed {
			return t.Parent.isToolAllowed(name)
		}
	}

	return false
}

// AgentMeta holds metadata about an available agent
type AgentMeta struct {
	Name            string   `yaml:"name"`
	Description     string   `yaml:"description"`
	Tools           []string `yaml:"tools"`
	DisallowedTools []string `yaml:"disallowedTools"`
}

// Validate checks if the agent metadata is valid
func (m *AgentMeta) Validate(expectedName string) error {
	if m.Name == "" {
		return fmt.Errorf("agent must have 'name' in frontmatter")
	}

	if m.Description == "" {
		return fmt.Errorf("agent %s must have 'description' in frontmatter", m.Name)
	}

	if expectedName != "" && m.Name != expectedName {
		return fmt.Errorf("agent name '%s' must match filename '%s'", m.Name, expectedName)
	}

	namePattern := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !namePattern.MatchString(m.Name) {
		return fmt.Errorf("agent name '%s' must be lowercase alphanumeric with hyphens only", m.Name)
	}

	return nil
}

// LoadAgents scans .soren/agents/ directory and returns list of available agents with metadata
func LoadAgents() ([]AgentMeta, error) {
	agents := []AgentMeta{
		{Name: "default", Description: "Default general-purpose agent"},
	}

	entries, err := os.ReadDir(".soren/agents")
	if err != nil {
		// Directory doesn't exist - just return default agent
		return agents, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			filePath := fmt.Sprintf(".soren/agents/%s", entry.Name())
			expectedName := strings.TrimSuffix(entry.Name(), ".md")

			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to read agent file %s: %w", filePath, err)
			}

			// Parse frontmatter
			var meta AgentMeta
			_, err = frontmatter.Parse(strings.NewReader(string(data)), &meta)
			if err != nil {
				return nil, fmt.Errorf("invalid frontmatter in %s: %w", filePath, err)
			}

			// Validate
			if err := meta.Validate(expectedName); err != nil {
				return nil, fmt.Errorf("%s: %w", filePath, err)
			}

			agents = append(agents, meta)
		}
	}

	return agents, nil
}
