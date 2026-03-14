package builtin

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type BashArgs struct {
	Command     string `json:"command" jsonschema:"The command to execute."`
	Description string `json:"description,omitempty" jsonschema:"Clear, concise description of what this command does in active voice."`
	Timeout     uint64 `json:"timeout,omitempty" jsonschema:"Optional timeout in milliseconds."`
}

const defaultBashTimeout = 2 * time.Minute

func Bash(ctx context.Context, req *mcp.CallToolRequest, args BashArgs) (*mcp.CallToolResult, any, error) {
	timeout := defaultBashTimeout
	if args.Timeout != 0 {
		timeout = time.Duration(args.Timeout) * time.Millisecond
	}

	if strings.TrimSpace(args.Command) == "" {
		return nil, "", errors.New("command is required")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)

	output, err := cmd.CombinedOutput()

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, "", fmt.Errorf("command timeout exceeded (%s)", timeout)
		}
		return nil, "", fmt.Errorf("command failed: %w", err)
	}

	return nil, string(output), nil
}
