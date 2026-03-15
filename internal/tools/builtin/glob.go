package builtin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxGlobResults = 100

type GlobArgs struct {
	Pattern string `json:"pattern" jsonschema:"The glob pattern to match files against"`
	Path    string `json:"path,omitempty" jsonschema:"The directory to search in. If not specified, the current working directory will be used."`
}

func Glob(ctx context.Context, req *mcp.CallToolRequest, args GlobArgs) (*mcp.CallToolResult, any, error) {
	if args.Pattern == "" {
		return nil, "", errors.New("pattern is required")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get working directory: %w", err)
	}

	path := cwd
	if args.Path != "" {
		path = args.Path
	}
	path = filepath.Clean(path)

	matches, err := doublestar.FilepathGlob(
		filepath.Join(path, args.Pattern),
		doublestar.WithFilesOnly(),
		doublestar.WithNoHidden(),
	)
	if err != nil {
		return nil, "", fmt.Errorf("glob pattern error: %w", err)
	}

	truncated := len(matches) > maxGlobResults
	if truncated {
		matches = matches[:maxGlobResults]
	}

	output := strings.Join(matches, "\n")
	if truncated {
		output += "\n(Results truncated. Consider using a more specific pattern or path.)"
	}

	return nil, output, nil
}
