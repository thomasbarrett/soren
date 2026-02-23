package builtin

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register wires all tools into the MCP server
func Register(server *mcp.Server) {
	RegisterReadFile(server)
	RegisterWrite(server)
	RegisterEdit(server)
	RegisterBash(server)
	RegisterGlob(server)
	RegisterGrep(server)
}
