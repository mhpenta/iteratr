package mcpserver

import (
	"context"
	"fmt"

	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleTaskAdd adds one or more tasks to the session.
func (s *Server) handleTaskAdd(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments
	args := request.GetArguments()
	if args == nil {
		return mcp.NewToolResultText("error: no arguments provided"), nil
	}

	// Extract tasks array
	tasksRaw, ok := args["tasks"]
	if !ok {
		return mcp.NewToolResultText("error: missing 'tasks' parameter"), nil
	}

	// Type assert to []any (mcp-go returns arrays as []any)
	tasksArray, ok := tasksRaw.([]any)
	if !ok {
		return mcp.NewToolResultText("error: 'tasks' is not an array"), nil
	}

	if len(tasksArray) == 0 {
		return mcp.NewToolResultText("error: at least one task is required"), nil
	}

	// Parse each task into TaskAddParams
	taskParams := make([]session.TaskAddParams, 0, len(tasksArray))
	for i, taskRaw := range tasksArray {
		// Convert to map[string]any
		taskMap, ok := taskRaw.(map[string]any)
		if !ok {
			return mcp.NewToolResultText(fmt.Sprintf("error: task %d is not an object", i)), nil
		}

		// Extract content (required)
		content, ok := taskMap["content"].(string)
		if !ok || content == "" {
			return mcp.NewToolResultText(fmt.Sprintf("error: task %d missing or empty 'content' field", i)), nil
		}

		// Extract optional status
		status := ""
		if statusVal, ok := taskMap["status"].(string); ok {
			status = statusVal
		}

		// Extract optional priority (JSON numbers come as float64)
		priority := 0
		if priorityVal, ok := taskMap["priority"].(float64); ok {
			priority = int(priorityVal)
		}

		taskParams = append(taskParams, session.TaskAddParams{
			Content:  content,
			Status:   status,
			Priority: priority,
			// Iteration will be set by store based on current iteration
		})
	}

	// Call TaskBatchAdd
	tasks, err := s.store.TaskBatchAdd(ctx, s.sessName, taskParams)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("error: %v", err)), nil
	}

	// Return success message with task IDs
	result := fmt.Sprintf("Added %d task(s):", len(tasks))
	for _, task := range tasks {
		result += fmt.Sprintf("\n  %s: %s", task.ID, task.Content)
	}

	return mcp.NewToolResultText(result), nil
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
