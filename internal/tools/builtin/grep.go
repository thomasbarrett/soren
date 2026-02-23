package builtin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const grepDescription = `Search file contents for a pattern.

Usage:
- Supports full regex patterns and literal string search
- Case-insensitive search with ignore_case=true
- Returns matches in 'file:line:content' format
- Always prefer Grep over bash 'grep', 'rg', or 'ag' commands
- For case-insensitive: set ignore_case=true (faster than regex (?i) flag)
- Combine with Glob to search specific file types: Glob('*.py') then Grep
- Use parallel Grep calls when searching different file sets
- For complex multi-step searches, consider using multiple rounds`

type GrepInput struct {
	Pattern    string   `json:"pattern"`
	Files      []string `json:"files"`
	IgnoreCase *bool    `json:"ignore_case,omitempty"`
}

type GrepOutput struct {
	Matches []string `json:"matches,omitempty"`
	Result  string   `json:"result,omitempty"`
	Error   string   `json:"error,omitempty"`
}

func grepFiles(ctx context.Context, pattern string, files []string, ignoreCase bool) ([]string, error) {
	if pattern == "" {
		return nil, errors.New("pattern is required")
	}

	if len(files) == 0 {
		return nil, errors.New("files list is required")
	}

	// Compile regex pattern
	patternToCompile := pattern
	if ignoreCase {
		patternToCompile = "(?i)" + pattern
	}

	re, err := regexp.Compile(patternToCompile)
	if err != nil {
		// If regex compilation fails, treat as literal string
		if ignoreCase {
			pattern = strings.ToLower(pattern)
		}
	}

	var matches []string

	for _, filePath := range files {
		absPath, pathErr := validatePath(filePath)
		if pathErr != nil {
			continue // Skip invalid paths
		}

		data, readErr := os.ReadFile(absPath)
		if readErr != nil {
			continue // Skip unreadable files
		}

		content := string(data)
		lines := strings.Split(content, "\n")

		for lineNum, line := range lines {
			var matched bool

			if re != nil {
				matched = re.MatchString(line)
			} else {
				searchLine := line
				searchPattern := pattern
				if ignoreCase {
					searchLine = strings.ToLower(line)
					searchPattern = strings.ToLower(pattern)
				}
				matched = strings.Contains(searchLine, searchPattern)
			}

			if matched {
				matches = append(matches, fmt.Sprintf("%s:%d:%s", filePath, lineNum+1, line))
			}
		}
	}

	return matches, nil
}

func RegisterGrep(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "Grep",
		Description: grepDescription,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args GrepInput) (*mcp.CallToolResult, GrepOutput, error) {
		ignoreCase := false
		if args.IgnoreCase != nil {
			ignoreCase = *args.IgnoreCase
		}

		matches, err := grepFiles(ctx, args.Pattern, args.Files, ignoreCase)
		if err != nil {
			return nil, GrepOutput{Error: err.Error()}, err
		}

		return nil, GrepOutput{
			Matches: matches,
			Result:  fmt.Sprintf("Found %d matches for pattern '%s'", len(matches), args.Pattern),
		}, nil
	})
}
