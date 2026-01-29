package mcpserver

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// registerTools registers all MCP tools with the server.
// Tools are registered using mcp-go schema builders to provide native MCP protocol access.
func (s *Server) registerTools() error {
	// task-add: array of task objects
	s.mcpServer.AddTool(
		mcp.NewTool("task-add",
			mcp.WithDescription("Add one or more tasks to the session"),
			mcp.WithArray("tasks", mcp.Required(),
				mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{
							"type":        "string",
							"description": "Task description",
						},
						"status": map[string]any{
							"type":        "string",
							"description": "Task status (remaining, in_progress, completed, blocked, cancelled)",
						},
						"priority": map[string]any{
							"type":        "integer",
							"description": "Priority level (0=critical, 1=high, 2=medium, 3=low, 4=backlog)",
						},
					},
					"required": []string{"content"},
				})),
		),
		s.handleTaskAdd,
	)

	// task-update: id required, other fields optional
	s.mcpServer.AddTool(
		mcp.NewTool("task-update",
			mcp.WithDescription("Update task status, priority, or dependencies"),
			mcp.WithString("id", mcp.Required(), mcp.Description("Task ID or prefix")),
			mcp.WithString("status", mcp.Description("New status (remaining, in_progress, completed, blocked, cancelled)")),
			mcp.WithNumber("priority", mcp.Description("New priority (0-4)")),
			mcp.WithString("depends_on", mcp.Description("Task ID this task depends on")),
		),
		s.handleTaskUpdate,
	)

	// task-list: list all tasks grouped by status
	s.mcpServer.AddTool(
		mcp.NewTool("task-list",
			mcp.WithDescription("List all tasks grouped by status"),
		),
		s.handleTaskList,
	)

	// task-next: get next highest priority unblocked task
	s.mcpServer.AddTool(
		mcp.NewTool("task-next",
			mcp.WithDescription("Get the next highest priority unblocked task"),
		),
		s.handleTaskNext,
	)

	// note-add: array of note objects
	s.mcpServer.AddTool(
		mcp.NewTool("note-add",
			mcp.WithDescription("Add one or more notes to the session"),
			mcp.WithArray("notes", mcp.Required(),
				mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{
							"type":        "string",
							"description": "Note content",
						},
						"type": map[string]any{
							"type":        "string",
							"description": "Note type (learning, stuck, tip, decision)",
						},
					},
					"required": []string{"content", "type"},
				})),
		),
		s.handleNoteAdd,
	)

	// note-list: list notes, optional type filter
	s.mcpServer.AddTool(
		mcp.NewTool("note-list",
			mcp.WithDescription("List notes, optionally filtered by type"),
			mcp.WithString("type", mcp.Description("Filter by note type (learning, stuck, tip, decision)")),
		),
		s.handleNoteList,
	)

	// iteration-summary: record summary for current iteration
	s.mcpServer.AddTool(
		mcp.NewTool("iteration-summary",
			mcp.WithDescription("Record summary for the current iteration"),
			mcp.WithString("summary", mcp.Required(), mcp.Description("Summary of what was accomplished")),
		),
		s.handleIterationSummary,
	)

	// session-complete: mark session as complete
	s.mcpServer.AddTool(
		mcp.NewTool("session-complete",
			mcp.WithDescription("Mark the session as complete (all tasks must be in terminal state)"),
		),
		s.handleSessionComplete,
	)

	return nil
}
