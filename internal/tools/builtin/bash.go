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
	Command     string  `json:"command" jsonschema:"The command to execute."`
	Description *string `json:"description,omitempty" jsonschema:"Clear, concise description of what this command does in active voice."`
	Timeout     *uint64 `json:"timeout,omitempty" jsonschema:"Optional timeout in milliseconds."`
}

const defaultBashTimeout = 2 * time.Minute

func Bash(ctx context.Context, req *mcp.CallToolRequest, args BashArgs) (*mcp.CallToolResult, any, error) {
	timeout := defaultBashTimeout
	if args.Timeout != nil {
		timeout = time.Duration(*args.Timeout) * time.Millisecond
	}

	if strings.TrimSpace(args.Command) == "" {
		return nil, "", errors.New("command is required")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)

	output, err := cmd.CombinedOutput()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, "", fmt.Errorf("command timeout exceeded (%s)", timeout)
		}

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, "", fmt.Errorf("command failed with exit code %d:\n%s", exitErr.ExitCode(), string(output))
		}

		return nil, "", err
	}

	return nil, string(output), nil
}
