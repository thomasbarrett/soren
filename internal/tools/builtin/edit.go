package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type EditArgs struct {
	FilePath   string `json:"file_path" jsonschema:"The absolute path to the file to modify"`
	OldString  string `json:"old_string" jsonschema:"The text to replace (must be unique in the file)."`
	NewString  string `json:"new_string" jsonschema:"The replacement text (must differ from old_string)."`
	ReplaceAll bool   `json:"replace_all,omitempty" jsonschema:"Replace all occurances (default: false)."`
}

func Edit(ctx context.Context, req *mcp.CallToolRequest, args EditArgs) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(args.FilePath)
	if err != nil {
		return nil, "", fmt.Errorf("file_path normalization failed")
	}
	if args.FilePath != absPath {
		return nil, "", fmt.Errorf("file_path must be absolute and normalized")
	}

	// Read file content
	data, err := os.ReadFile(args.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			cwd, _ := os.Getwd()
			return nil, "", fmt.Errorf("File does not exist. Note: your current working directory is %s", cwd)
		}
		return nil, "", fmt.Errorf("failed to read file: %w", err)
	}
	content := string(data)

	// Check for same string
	if args.OldString == args.NewString {
		return nil, "", fmt.Errorf("No changes to make: old_string and new_string are exactly the same")
	}

	// Count matches
	matchCount := strings.Count(content, args.OldString)
	if matchCount == 0 {
		return nil, "", fmt.Errorf("String to replace not found in file. String: %s;", args.OldString)
	}

	// Handle replace_all flag
	var newContent string
	if args.ReplaceAll {
		newContent = strings.ReplaceAll(content, args.OldString, args.NewString)
		if err := os.WriteFile(args.FilePath, []byte(newContent), 0o644); err != nil {
			return nil, "", fmt.Errorf("failed to write file: %w", err)
		}
		return nil, fmt.Sprintf("The file %s has been updated. All occurences of '%s' were successfully replaced with '%s'.", args.FilePath, args.OldString, args.NewString), nil
	}

	// Not replace_all, but multiple matches
	if matchCount > 1 {
		return nil, "", fmt.Errorf("Found %d matches of the string to replace, but replace_all is false.", matchCount)
	}

	// Single match, perform replacement
	newContent = strings.Replace(content, args.OldString, args.NewString, 1)
	if err := os.WriteFile(args.FilePath, []byte(newContent), 0o644); err != nil {
		return nil, "", fmt.Errorf("failed to write file: %w", err)
	}
	return nil, fmt.Sprintf("The file %s has been updated successfully.", args.FilePath), nil
}
