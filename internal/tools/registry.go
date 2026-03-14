package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Provider is the subset of mcp.Session needed to list and call tools.
type Provider interface {
	ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error)
}

// Registry multiplexes tool calls across multiple MCP servers.
//
// Builtin tool names are kept as-is. External tools are namespaced as
// "mcp__<server>__<tool>" to avoid collisions; the prefix is stripped
// before forwarding to the server.
type Registry struct {
	client  *mcp.Client
	servers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{
		client: mcp.NewClient(&mcp.Implementation{
			Name:    "soren",
			Version: "v0.1.0",
		}, nil),
		servers: make(map[string]Provider),
	}
}

func (r *Registry) Connect(ctx context.Context, name string, transport mcp.Transport) error {
	if strings.Contains(name, "__") {
		return fmt.Errorf("server name %q must not contain \"__\"", name)
	}

	session, err := r.client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to server %s: %w", name, err)
	}

	r.servers[name] = session
	return nil
}

func (r *Registry) ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	var all []*mcp.Tool
	for name, provider := range r.servers {
		tools, err := listAllTools(ctx, provider)
		if err != nil {
			return nil, fmt.Errorf("failed to list tools from %s: %w", name, err)
		}
		for _, tool := range tools {
			t := *tool
			t.Name = qualifyToolName(name, tool.Name)
			all = append(all, &t)
		}
	}

	return &mcp.ListToolsResult{Tools: all}, nil
}

// listAllTools paginates through a provider's tool list until no cursor remains.
func listAllTools(ctx context.Context, provider Provider) ([]*mcp.Tool, error) {
	var all []*mcp.Tool
	var cursor string
	for {
		result, err := provider.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		all = append(all, result.Tools...)
		if result.NextCursor == "" {
			return all, nil
		}
		cursor = result.NextCursor
	}
}

func (r *Registry) CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	server, tool, ok := splitToolName(params.Name)
	if !ok {
		return nil, fmt.Errorf("invalid tool name: %s", params.Name)
	}

	provider, exists := r.servers[server]
	if !exists {
		return nil, fmt.Errorf("server %q not found", server)
	}

	return provider.CallTool(ctx, &mcp.CallToolParams{
		Name:      tool,
		Arguments: params.Arguments,
	})
}

// --- tool name encoding ---
//
// Builtin tools use bare names (e.g. "Bash"). External tools are encoded as
// "mcp__<server>__<tool>" so the server can be recovered from the name alone.

const builtinServer = "builtin"

func qualifyToolName(server, tool string) string {
	if server == builtinServer {
		return tool
	}
	return "mcp__" + server + "__" + tool
}

func splitToolName(qualified string) (server, tool string, ok bool) {
	if !strings.HasPrefix(qualified, "mcp__") {
		return builtinServer, qualified, true
	}
	rest := qualified[len("mcp__"):]
	i := strings.Index(rest, "__")
	if i < 0 {
		return "", "", false
	}
	return rest[:i], rest[i+2:], true
}
