package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const writeDescription = `Write content to a file on disk.

Usage:
- By default overwrites existing files (set overwrite=false to prevent)
- Creates parent directories automatically
- ALWAYS prefer editing existing files over creating new ones
- Use Write only when explicitly creating new files
- Never use bash commands like 'echo >' or 'cat <<EOF' - use this tool instead`

type WriteInput struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Overwrite *bool  `json:"overwrite,omitempty"`
}

type WriteOutput struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func writeFile(ctx context.Context, path, content string, overwrite bool) (string, error) {
	absPath, err := validatePath(path)
	if err != nil {
		return "", err
	}

	// Check if file exists and overwrite flag
	if _, err := os.Stat(absPath); err == nil && !overwrite {
		return "", fmt.Errorf("file already exists: %s (use overwrite=true to replace)", path)
	}

	// Create parent directories if needed
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("failed to create directories: %w", err)
	}

	// Write the file
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote file: %s", path), nil
}

func RegisterWrite(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "Write",
		Description: writeDescription,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args WriteInput) (*mcp.CallToolResult, WriteOutput, error) {
		overwrite := true
		if args.Overwrite != nil {
			overwrite = *args.Overwrite
		}

		result, err := writeFile(ctx, args.Path, args.Content, overwrite)
		if err != nil {
			return nil, WriteOutput{Error: err.Error()}, err
		}
		return nil, WriteOutput{Result: result}, nil
	})
}
