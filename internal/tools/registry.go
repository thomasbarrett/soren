package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Provider interface defines the methods needed to provide tools
// This mirrors the mcp.Session interface for ListTools and CallTool methods
type Provider interface {
	ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error)
	CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error)
}

// Registry manages multiple tool providers
type Registry struct {
	client *mcp.Client
	tools  map[string]Provider
}

// NewRegistry creates a new registry instance
func NewRegistry() *Registry {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "soren",
		Version: "v0.1.0",
	}, nil)

	return &Registry{
		client: client,
		tools:  make(map[string]Provider),
	}
}

func (r *Registry) Connect(ctx context.Context, name string, transport mcp.Transport) error {
	session, err := r.client.Connect(ctx, transport, nil)
	if err != nil {
		return fmt.Errorf("failed to connect to  server %s: %w", name, err)
	}

	if err := r.add(name, session); err != nil {
		return fmt.Errorf("failed to register server %s: %w", name, err)
	}

	return nil
}

// Add registers a provider by first listing its tools and mapping each tool name to the provider
func (r *Registry) add(name string, provider Provider) error {
	// List tools from the provider
	toolsList, err := provider.ListTools(context.Background(), &mcp.ListToolsParams{})
	if err != nil {
		return fmt.Errorf("failed to list tools from provider %s: %w", name, err)
	}

	// Map each tool name to this provider
	for _, tool := range toolsList.Tools {
		r.tools[tool.Name] = provider
	}

	return nil
}

// ListTools returns tools from all registered providers
// Implements the Provider interface
func (r *Registry) ListTools(ctx context.Context, params *mcp.ListToolsParams) (*mcp.ListToolsResult, error) {
	// Get unique providers
	seenProviders := make(map[Provider]bool)
	var uniqueProviders []Provider
	for _, provider := range r.tools {
		if !seenProviders[provider] {
			seenProviders[provider] = true
			uniqueProviders = append(uniqueProviders, provider)
		}
	}

	var allTools []*mcp.Tool
	for _, provider := range uniqueProviders {
		result, err := provider.ListTools(ctx, params)
		if err != nil {
			return nil, err
		}
		allTools = append(allTools, result.Tools...)
	}

	return &mcp.ListToolsResult{
		Tools: allTools,
	}, nil
}

// CallTool calls a tool by looking up the provider directly by tool name
func (r *Registry) CallTool(ctx context.Context, params *mcp.CallToolParams) (*mcp.CallToolResult, error) {
	provider, exists := r.tools[params.Name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", params.Name)
	}

	return provider.CallTool(ctx, params)
}
