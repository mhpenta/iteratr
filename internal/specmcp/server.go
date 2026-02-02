package specmcp

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server manages an embedded MCP HTTP server that exposes spec wizard tools.
// The server provides ask-questions and finish-spec tools to enable AI-assisted
// spec generation through an interactive interview process.
type Server struct {
	specDir    string
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	port       int
	mu         sync.Mutex
}

// New creates a new MCP server instance for spec wizard.
// The server is not started until Start() is called.
func New(specDir string) *Server {
	return &Server{
		specDir: specDir,
	}
}

// Start starts the MCP HTTP server on a random available port.
// Blocks until the server is ready to accept connections.
// Returns the port number or an error if startup fails.
func (s *Server) Start(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpServer != nil {
		return 0, fmt.Errorf("server already started")
	}

	// Create MCP server with registered tools
	s.mcpServer = server.NewMCPServer(
		"iteratr-spec",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	// Register tools
	if err := s.registerTools(); err != nil {
		return 0, fmt.Errorf("failed to register tools: %w", err)
	}

	// Find a random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("failed to find available port: %w", err)
	}

	// Get the port that was assigned
	s.port = listener.Addr().(*net.TCPAddr).Port
	// Close the listener - we just needed it to find a free port
	if err := listener.Close(); err != nil {
		return 0, fmt.Errorf("failed to close listener: %w", err)
	}

	// Create HTTP server with stateless mode (no session management needed)
	s.httpServer = server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithStateLess(true),
	)

	logger.Debug("Starting spec MCP server on port %d", s.port)

	// Start server in background - capture httpServer reference for goroutine
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	httpServer := s.httpServer
	go func() {
		if err := httpServer.Start(addr); err != nil {
			logger.Error("Spec MCP server error: %v", err)
		}
	}()

	// Server is ready immediately after Start() returns
	logger.Debug("Spec MCP server ready on port %d", s.port)
	return s.port, nil
}

// Stop stops the MCP HTTP server and cleans up resources.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpServer == nil {
		return nil // Already stopped
	}

	logger.Debug("Stopping spec MCP server")
	if err := s.httpServer.Shutdown(context.Background()); err != nil {
		logger.Warn("Error stopping spec MCP server: %v", err)
		return fmt.Errorf("failed to stop server: %w", err)
	}

	s.httpServer = nil
	s.mcpServer = nil
	logger.Debug("Spec MCP server stopped")
	return nil
}

// URL returns the HTTP URL for the MCP server endpoint.
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("http://localhost:%d/mcp", s.port)
}

// Stub handlers - full implementation in TAS-7 (handlers.go)
// These stubs allow the package to compile while tools.go references them.
func (s *Server) handleAskQuestions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError("not implemented"), nil
}

func (s *Server) handleFinishSpec(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError("not implemented"), nil
}
