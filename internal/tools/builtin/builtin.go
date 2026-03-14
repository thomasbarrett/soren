package builtin

import (
	_ "embed"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register wires all tools into the MCP server

func Register(server *mcp.Server) {
	RegisterRead(server)
	RegisterWrite(server)
	RegisterEdit(server)
	RegisterBash(server)
	RegisterGlob(server)
	RegisterGrep(server)
}

//go:embed descriptions/read.md
var readDescription string

func RegisterRead(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "Read",
		Description: readDescription,
	}, Read)
}

//go:embed descriptions/write.md
var writeDescription string

func RegisterWrite(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "Write",
		Description: writeDescription,
	}, Write)
}

//go:embed descriptions/edit.md
var editDescription string

func RegisterEdit(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "Edit",
		Description: editDescription,
	}, Edit)
}

//go:embed descriptions/bash.md
var bashDescription string

func RegisterBash(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "Bash",
		Description: bashDescription,
	}, Bash)
}

//go:embed descriptions/glob.md
var globDescription string

func RegisterGlob(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "Glob",
		Description: globDescription,
	}, Glob)
}

//go:embed descriptions/grep.md
var grepDescription string

func RegisterGrep(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "Grep",
		Description: grepDescription,
	}, Grep)
}
