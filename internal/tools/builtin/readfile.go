package builtin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const readFileDescription = `Read a file from disk. Required for inspecting code.

Usage:
- Use absolute or relative paths
- Supports UTF-8 encoding by default
- For binary files, an error will be returned
- You can call multiple ReadFile tools in parallel for efficiency
- Always prefer ReadFile over bash commands like 'cat', 'head', 'tail'`

type ReadFileInput struct {
	Path     string `json:"path"`
	Encoding string `json:"encoding,omitempty"`
}

type ReadFileOutput struct {
	Content string `json:"content,omitempty"`
	Error   string `json:"error,omitempty"`
}

func readFile(ctx context.Context, path, encoding string) (string, error) {
	absPath, err := validatePath(path)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Handle encoding (default to utf-8)
	if encoding == "" || strings.ToLower(encoding) == "utf-8" {
		if !utf8.Valid(data) {
			return "", errors.New("file contains invalid UTF-8")
		}
		return string(data), nil
	}

	// For now, only support utf-8
	return "", fmt.Errorf("unsupported encoding: %s", encoding)
}

func RegisterReadFile(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ReadFile",
		Description: readFileDescription,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args ReadFileInput) (*mcp.CallToolResult, ReadFileOutput, error) {
		content, err := readFile(ctx, args.Path, args.Encoding)
		if err != nil {
			return nil, ReadFileOutput{Error: err.Error()}, err
		}
		return nil, ReadFileOutput{Content: content}, nil
	})
}
