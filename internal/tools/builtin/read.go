package builtin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const defaultReadLimit = 2000

type ReadArgs struct {
	FilePath string `json:"file_path" jsonschema:"The absolute path to the file to read."`
	Limit    *int   `json:"limit,omitempty" jsonschema:"The number of lines to read. Only provide if the file is too large to read at once."`
	Offset   int    `json:"offset,omitempty" jsonschema:"The line number to start reading from. Only provide if the file is too large to read at once."`
}

func Read(ctx context.Context, req *mcp.CallToolRequest, args ReadArgs) (*mcp.CallToolResult, any, error) {
	// Only accept absolute, normalized paths
	if args.FilePath == "" || strings.Contains(args.FilePath, "\x00") {
		return nil, "", errors.New("file_path must be non-empty and not contain null bytes")
	}
	abs, err := filepath.Abs(args.FilePath)
	if err != nil || abs != args.FilePath {
		return nil, "", errors.New("file_path must be absolute and normalized")
	}
	if args.Offset < 0 {
		return nil, "", errors.New("offset must be >= 0")
	}
	absPath, err := validatePath(args.FilePath)
	if err != nil {
		return nil, "", err
	}

	file, err := os.Open(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", fmt.Errorf("file not found: %s", args.FilePath)
		}
		return nil, "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	limit := defaultReadLimit
	if args.Limit != nil {
		limit = *args.Limit
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	for i := 0; scanner.Scan(); i++ {
		if i < args.Offset {
			continue
		}
		lines = append(lines, scanner.Text())
		if len(lines) >= limit {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, "", fmt.Errorf("failed to scan file: %w", err)
	}

	return nil, strings.Join(lines, "\n"), nil
}
