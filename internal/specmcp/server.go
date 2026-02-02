package specmcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server manages an embedded MCP HTTP server for the spec wizard.
// This is a stub - full implementation in TAS-5.
type Server struct {
	mcpServer *server.MCPServer
}

// Stub handlers - will be implemented in TAS-7
func (s *Server) handleAskQuestions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return nil, nil
}

func (s *Server) handleFinishSpec(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return nil, nil
}
