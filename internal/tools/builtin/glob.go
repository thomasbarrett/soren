package builtin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const globDescription = `List files matching a glob pattern.

Usage:
- Fast file pattern matching for any codebase size
- Supports glob patterns: '*.js', 'src/**/*.ts', '**/*.{go,py}'
- Returns files sorted by path
- Recursive search by default (set recursive=false for current directory only)
- Use for finding files by name/extension patterns
- Always prefer Glob over bash 'find' or 'ls' commands
- Combine with Grep for content-based searches after pattern matching`

type GlobInput struct {
	Pattern   string `json:"pattern"`
	Recursive *bool  `json:"recursive,omitempty"`
}

type GlobOutput struct {
	Files  []string `json:"files,omitempty"`
	Result string   `json:"result,omitempty"`
	Error  string   `json:"error,omitempty"`
}

func globFiles(ctx context.Context, pattern string, recursive bool) ([]string, error) {
	if pattern == "" {
		return nil, errors.New("pattern is required")
	}

	var matches []string
	var err error

	if recursive {
		// Use filepath.WalkDir for recursive matching
		err = filepath.WalkDir(".", func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil // Continue on errors
			}

			if d.IsDir() {
				return nil
			}

			// Check if the file matches the pattern
			matched, matchErr := filepath.Match(pattern, d.Name())
			if matchErr != nil {
				return nil // Continue on match errors
			}

			if matched {
				matches = append(matches, path)
			}

			return nil
		})
	} else {
		// Use filepath.Glob for non-recursive matching
		matches, err = filepath.Glob(pattern)
	}

	if err != nil {
		return nil, fmt.Errorf("glob pattern error: %w", err)
	}

	return matches, nil
}

func RegisterGlob(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "Glob",
		Description: globDescription,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args GlobInput) (*mcp.CallToolResult, GlobOutput, error) {
		recursive := true
		if args.Recursive != nil {
			recursive = *args.Recursive
		}

		files, err := globFiles(ctx, args.Pattern, recursive)
		if err != nil {
			return nil, GlobOutput{Error: err.Error()}, err
		}

		return nil, GlobOutput{
			Files:  files,
			Result: fmt.Sprintf("Found %d files matching pattern '%s'", len(files), args.Pattern),
		}, nil
	})
}
