package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type WriteArgs struct {
	Content  string `json:"content" jsonschema:"The content to write to the file."`
	FilePath string `json:"file_path" jsonschema:"The absolute path to the file to write (must be absolute, not relative)."`
}

func Write(ctx context.Context, req *mcp.CallToolRequest, args WriteArgs) (*mcp.CallToolResult, any, error) {
	absPath, err := validatePath(args.FilePath)
	if err != nil {
		return nil, "", err
	}

	// Check if file exists
	exists := false
	if _, err := os.Stat(absPath); err == nil {
		exists = true
	}

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return nil, "", fmt.Errorf("failed to create directories: %w", err)
	}

	// Write the file
	if err := os.WriteFile(absPath, []byte(args.Content), 0o644); err != nil {
		return nil, "", fmt.Errorf("failed to write file: %w", err)
	}

	if exists {
		return nil, fmt.Sprintf("The file %s has been updated successfully.", absPath), nil
	}

	return nil, fmt.Sprintf("File created successfully at: %s", absPath), nil
}
