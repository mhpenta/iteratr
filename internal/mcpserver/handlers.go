package mcpserver

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// handleTaskAdd adds one or more tasks to the session.
func (s *Server) handleTaskAdd(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: implement
	return mcp.NewToolResultText("not implemented"), nil
}

// handleTaskUpdate updates a task's status, priority, or dependencies.
func (s *Server) handleTaskUpdate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: implement
	return mcp.NewToolResultText("not implemented"), nil
}

// handleTaskList returns all tasks grouped by status.
func (s *Server) handleTaskList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: implement
	return mcp.NewToolResultText("not implemented"), nil
}

// handleTaskNext returns the next highest priority unblocked task.
func (s *Server) handleTaskNext(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: implement
	return mcp.NewToolResultText("not implemented"), nil
}

// handleNoteAdd adds one or more notes to the session.
func (s *Server) handleNoteAdd(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: implement
	return mcp.NewToolResultText("not implemented"), nil
}

// handleNoteList returns all notes, optionally filtered by type.
func (s *Server) handleNoteList(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: implement
	return mcp.NewToolResultText("not implemented"), nil
}

// handleIterationSummary records a summary for the current iteration.
func (s *Server) handleIterationSummary(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: implement
	return mcp.NewToolResultText("not implemented"), nil
}

// handleSessionComplete marks the session as complete.
func (s *Server) handleSessionComplete(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// TODO: implement
	return mcp.NewToolResultText("not implemented"), nil
}
