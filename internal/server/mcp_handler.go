package server

import (
	"net/http"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

func newMCPHTTPHandler(s *mcpserver.MCPServer) http.Handler {
	return mcpserver.NewStreamableHTTPServer(s)
}
