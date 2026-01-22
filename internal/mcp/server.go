package mcp

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps an MCP SSE server with session tools.
type Server struct {
	mcpServer *server.MCPServer
	sseServer *server.SSEServer
	httpSrv   *http.Server
	store     *session.Store
	port      int
}

// New creates a new MCP server with session tools.
func New(store *session.Store, version string) *Server {
	mcpServer := server.NewMCPServer(
		"iteratr",
		version,
		server.WithToolCapabilities(true),
	)

	s := &Server{
		mcpServer: mcpServer,
		store:     store,
	}

	s.registerTools()

	return s
}

// Start starts the SSE server on a random available port.
// Returns the URL that can be used to connect to the server.
func (s *Server) Start() (string, error) {
	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to find available port: %w", err)
	}
	s.port = listener.Addr().(*net.TCPAddr).Port

	// Create SSE server
	s.sseServer = server.NewSSEServer(s.mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://127.0.0.1:%d", s.port)),
	)

	// Create HTTP server
	s.httpSrv = &http.Server{
		Handler: s.sseServer,
	}

	// Start serving
	go func() {
		if err := s.httpSrv.Serve(listener); err != nil && err != http.ErrServerClosed {
			// Log error but don't crash - server might be shutting down
		}
	}()

	url := fmt.Sprintf("http://127.0.0.1:%d/sse", s.port)
	return url, nil
}

// Stop gracefully shuts down the server.
func (s *Server) Stop(ctx context.Context) error {
	if s.httpSrv != nil {
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}

// Port returns the port the server is listening on.
func (s *Server) Port() int {
	return s.port
}

// registerTools adds all session management tools to the MCP server.
func (s *Server) registerTools() {
	// task_add
	s.mcpServer.AddTool(
		mcp.NewTool("task_add",
			mcp.WithDescription("Add a new task to the session task list"),
			mcp.WithString("session_name", mcp.Required(), mcp.Description("Session name from Context section")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Task description")),
			mcp.WithString("status", mcp.Description("Initial status: remaining (default), blocked"), mcp.Enum("remaining", "blocked")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionName := req.GetString("session_name", "")
			content := req.GetString("content", "")
			status := req.GetString("status", "remaining")

			if sessionName == "" {
				return mcp.NewToolResultError("session_name is required"), nil
			}
			if content == "" {
				return mcp.NewToolResultError("content is required"), nil
			}

			task, err := s.store.TaskAdd(ctx, sessionName, session.TaskAddParams{
				Content: content,
				Status:  status,
			})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to add task: %v", err)), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("Task added: [%s] %s", task.ID[:8], task.Content)), nil
		},
	)

	// task_status
	s.mcpServer.AddTool(
		mcp.NewTool("task_status",
			mcp.WithDescription("Update a task status by ID. Use IDs from task_list output."),
			mcp.WithString("session_name", mcp.Required(), mcp.Description("Session name from Context section")),
			mcp.WithString("id", mcp.Required(), mcp.Description("Task ID (full or 8+ char prefix)")),
			mcp.WithString("status", mcp.Required(), mcp.Description("New status"), mcp.Enum("in_progress", "completed", "blocked")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionName := req.GetString("session_name", "")
			id := req.GetString("id", "")
			status := req.GetString("status", "")

			if sessionName == "" {
				return mcp.NewToolResultError("session_name is required"), nil
			}
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}
			if status == "" {
				return mcp.NewToolResultError("status is required"), nil
			}

			err := s.store.TaskStatus(ctx, sessionName, session.TaskStatusParams{
				ID:     id,
				Status: status,
			})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to update task: %v", err)), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("Task %s marked as %s", id, status)), nil
		},
	)

	// task_list
	s.mcpServer.AddTool(
		mcp.NewTool("task_list",
			mcp.WithDescription("Get current task list grouped by status. Shows task IDs needed for task_status."),
			mcp.WithString("session_name", mcp.Required(), mcp.Description("Session name from Context section")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionName := req.GetString("session_name", "")

			if sessionName == "" {
				return mcp.NewToolResultError("session_name is required"), nil
			}

			result, err := s.store.TaskList(ctx, sessionName)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list tasks: %v", err)), nil
			}

			var output string
			formatTasks := func(status string, tasks []*session.Task) {
				if len(tasks) == 0 {
					return
				}
				output += fmt.Sprintf("%s:\n", status)
				for _, t := range tasks {
					output += fmt.Sprintf("  [%s] %s\n", t.ID[:8], t.Content)
				}
			}

			formatTasks("Remaining", result.Remaining)
			formatTasks("In Progress", result.InProgress)
			formatTasks("Completed", result.Completed)
			formatTasks("Blocked", result.Blocked)

			if output == "" {
				output = "No tasks"
			}

			return mcp.NewToolResultText(output), nil
		},
	)

	// note_add
	s.mcpServer.AddTool(
		mcp.NewTool("note_add",
			mcp.WithDescription("Add a note for future iterations (learnings, tips, blockers, decisions)"),
			mcp.WithString("session_name", mcp.Required(), mcp.Description("Session name from Context section")),
			mcp.WithString("content", mcp.Required(), mcp.Description("Note content")),
			mcp.WithString("type", mcp.Required(), mcp.Description("Note category"), mcp.Enum("learning", "stuck", "tip", "decision")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionName := req.GetString("session_name", "")
			content := req.GetString("content", "")
			noteType := req.GetString("type", "")

			if sessionName == "" {
				return mcp.NewToolResultError("session_name is required"), nil
			}
			if content == "" {
				return mcp.NewToolResultError("content is required"), nil
			}
			if noteType == "" {
				return mcp.NewToolResultError("type is required"), nil
			}

			note, err := s.store.NoteAdd(ctx, sessionName, session.NoteAddParams{
				Content: content,
				Type:    noteType,
			})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to add note: %v", err)), nil
			}

			preview := content
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			return mcp.NewToolResultText(fmt.Sprintf("Note added: [%s] %s", note.Type, preview)), nil
		},
	)

	// note_list
	s.mcpServer.AddTool(
		mcp.NewTool("note_list",
			mcp.WithDescription("List notes from this session"),
			mcp.WithString("session_name", mcp.Required(), mcp.Description("Session name from Context section")),
			mcp.WithString("type", mcp.Description("Filter by type: learning, stuck, tip, decision")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionName := req.GetString("session_name", "")
			noteType := req.GetString("type", "")

			if sessionName == "" {
				return mcp.NewToolResultError("session_name is required"), nil
			}

			notes, err := s.store.NoteList(ctx, sessionName, session.NoteListParams{
				Type: noteType,
			})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list notes: %v", err)), nil
			}

			if len(notes) == 0 {
				return mcp.NewToolResultText("No notes"), nil
			}

			var output string
			for _, note := range notes {
				output += fmt.Sprintf("[%s] (#%d) %s\n", note.Type, note.Iteration, note.Content)
			}

			return mcp.NewToolResultText(output), nil
		},
	)

	// inbox_list
	s.mcpServer.AddTool(
		mcp.NewTool("inbox_list",
			mcp.WithDescription("Get unread inbox messages. Check this at start of each iteration."),
			mcp.WithString("session_name", mcp.Required(), mcp.Description("Session name from Context section")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionName := req.GetString("session_name", "")

			if sessionName == "" {
				return mcp.NewToolResultError("session_name is required"), nil
			}

			messages, err := s.store.InboxList(ctx, sessionName)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to list inbox: %v", err)), nil
			}

			// Filter to unread only
			var unread []*session.Message
			for _, msg := range messages {
				if !msg.Read {
					unread = append(unread, msg)
				}
			}

			if len(unread) == 0 {
				return mcp.NewToolResultText("No unread messages"), nil
			}

			var output string
			for _, msg := range unread {
				output += fmt.Sprintf("[%s] %s\n", msg.ID[:8], msg.Content)
			}

			return mcp.NewToolResultText(output), nil
		},
	)

	// inbox_mark_read
	s.mcpServer.AddTool(
		mcp.NewTool("inbox_mark_read",
			mcp.WithDescription("Mark an inbox message as read after processing"),
			mcp.WithString("session_name", mcp.Required(), mcp.Description("Session name from Context section")),
			mcp.WithString("id", mcp.Required(), mcp.Description("Message ID from inbox_list")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionName := req.GetString("session_name", "")
			id := req.GetString("id", "")

			if sessionName == "" {
				return mcp.NewToolResultError("session_name is required"), nil
			}
			if id == "" {
				return mcp.NewToolResultError("id is required"), nil
			}

			err := s.store.InboxMarkRead(ctx, sessionName, session.InboxMarkReadParams{
				ID: id,
			})
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to mark message as read: %v", err)), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("Message %s marked as read", id)), nil
		},
	)

	// session_complete
	s.mcpServer.AddTool(
		mcp.NewTool("session_complete",
			mcp.WithDescription("Signal that ALL tasks are complete and terminate the iteratr session. Only call when every task is done."),
			mcp.WithString("session_name", mcp.Required(), mcp.Description("Session name from Context section")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionName := req.GetString("session_name", "")

			if sessionName == "" {
				return mcp.NewToolResultError("session_name is required"), nil
			}

			err := s.store.SessionComplete(ctx, sessionName)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to complete session: %v", err)), nil
			}

			return mcp.NewToolResultText(fmt.Sprintf("Session '%s' marked complete", sessionName)), nil
		},
	)
}
