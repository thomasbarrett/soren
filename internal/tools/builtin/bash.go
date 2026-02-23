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

const bashDescription = `Execute a shell command on the local system.

Usage:
- Default timeout: 30 seconds (configurable)
- Working directory persists between commands
- Quote file paths with spaces: "path with spaces/file.txt"
- Chain dependent commands with &&, independent commands with ;
- AVOID using bash for: file reading (use ReadFile), content search (use Grep), file search (use Glob), editing (use Edit), writing (use Write)
- USE for: git operations, package management, compilation, system commands
- For multiple independent commands, make parallel Bash calls
- For dependent commands, chain with && in single call`

type BashInput struct {
	Command string `json:"command"`
	Timeout *int   `json:"timeout,omitempty"`
}

type BashOutput struct {
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
	Error    string `json:"error,omitempty"`
}

func executeBash(ctx context.Context, command string, timeoutSec int) (string, string, int, error) {
	if strings.TrimSpace(command) == "" {
		return "", "", 1, errors.New("command is required")
	}

	// Set default timeout
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	exitCode := 0

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if ctx.Err() == context.DeadlineExceeded {
			return "", "", 1, fmt.Errorf("command timeout exceeded (%ds)", timeoutSec)
		} else if err != context.Canceled {
			return "", "", 1, fmt.Errorf("command failed: %w", err)
		}
	}

	return strings.TrimRight(stdout.String(), "\n"),
		strings.TrimRight(stderr.String(), "\n"),
		exitCode, nil
}

func RegisterBash(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "Bash",
		Description: bashDescription,
	}, func(ctx context.Context, req *mcp.CallToolRequest, args BashInput) (*mcp.CallToolResult, BashOutput, error) {
		timeout := 30
		if args.Timeout != nil {
			timeout = *args.Timeout
		}

		stdout, stderr, exitCode, err := executeBash(ctx, args.Command, timeout)
		if err != nil {
			return nil, BashOutput{
				Stdout:   stdout,
				Stderr:   stderr,
				ExitCode: exitCode,
				Error:    err.Error(),
			}, err
		}

		return nil, BashOutput{
			Stdout:   stdout,
			Stderr:   stderr,
			ExitCode: exitCode,
		}, nil
	})
}
