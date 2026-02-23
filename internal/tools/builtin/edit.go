package builtin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const editDescription = `Make structured edits to a file (insert, replace, delete).

Usage:
- Supports multiple edits in a single call (applied in reverse line order)
- Line numbers are 1-indexed
- Edit types: 'insert' (before line), 'replace' (replace line), 'delete' (remove line)
- Always prefer Edit over bash commands like 'sed', 'awk'
- For simple string replacements, this is more reliable than text manipulation commands
- Read the file first to understand structure and line numbers`

type EditInput struct {
	Path  string `json:"path"`
	Edits []Edit `json:"edits"`
}

type Edit struct {
	Type    string `json:"type"`    // "insert", "replace", "delete"
	Line    int    `json:"line"`    // 1-indexed line number
	Content string `json:"content"` // Content to insert/replace (not used for delete)
}

type EditOutput struct {
	Result string `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func editFile(ctx context.Context, path string, edits []Edit) (string, error) {
	absPath, err := validatePath(path)
	if err != nil {
		return "", err
	}

	// Read the file
	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file not found: %s", path)
		}
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// Sort edits by line number in reverse order to avoid index shifting
	sortedEdits := make([]Edit, len(edits))
	copy(sortedEdits, edits)

	// Simple bubble sort by line number (descending)
	for i := 0; i < len(sortedEdits)-1; i++ {
		for j := 0; j < len(sortedEdits)-i-1; j++ {
			if sortedEdits[j].Line < sortedEdits[j+1].Line {
				sortedEdits[j], sortedEdits[j+1] = sortedEdits[j+1], sortedEdits[j]
			}
		}
	}

	// Apply edits in reverse order
	for _, edit := range sortedEdits {
		if edit.Line < 1 || edit.Line > len(lines)+1 {
			return "", fmt.Errorf("line %d out of bounds (file has %d lines)", edit.Line, len(lines))
		}

		switch edit.Type {
		case "insert":
			// Insert before the specified line (1-indexed)
			insertPos := edit.Line - 1
			if insertPos < 0 {
				insertPos = 0
			}
			if insertPos > len(lines) {
				insertPos = len(lines)
			}

			newLines := make([]string, 0, len(lines)+1)
			newLines = append(newLines, lines[:insertPos]...)
			newLines = append(newLines, edit.Content)
			newLines = append(newLines, lines[insertPos:]...)
			lines = newLines

		case "replace":
			// Replace the specified line (1-indexed)
			if edit.Line > len(lines) {
				return "", fmt.Errorf("cannot replace line %d: file only has %d lines", edit.Line, len(lines))
			}
			lines[edit.Line-1] = edit.Content

		case "delete":
			// Delete the specified line (1-indexed)
			if edit.Line > len(lines) {
				return "", fmt.Errorf("cannot delete line %d: file only has %d lines", edit.Line, len(lines))
			}
			lines = append(lines[:edit.Line-1], lines[edit.Line:]...)

		default:
			return "", fmt.Errorf("unknown edit type: %s", edit.Type)
		}
	}

	// Write back to file
	newContent := strings.Join(lines, "\n")
	if err := os.WriteFile(absPath, []byte(newContent), 0o644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully applied %d edits to %s", len(edits), path), nil
}

func RegisterEdit(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "Edit",
		Description: editDescription,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args EditInput) (*mcp.CallToolResult, EditOutput, error) {
		result, err := editFile(ctx, args.Path, args.Edits)
		if err != nil {
			return nil, EditOutput{Error: err.Error()}, err
		}
		return nil, EditOutput{Result: result}, nil
	})
}
