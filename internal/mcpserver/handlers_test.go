package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/iteratr/internal/nats"
	"github.com/mark3labs/iteratr/internal/session"
	"github.com/mark3labs/mcp-go/mcp"
)

// setupTestServer creates a server with a test store
func setupTestServer(t *testing.T) (*Server, func()) {
	ctx := context.Background()

	// Create embedded NATS
	ns, _, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("failed to start NATS: %v", err)
	}

	// Connect to NATS
	nc, err := nats.ConnectInProcess(ns)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}

	// Create JetStream
	js, err := nats.CreateJetStream(nc)
	if err != nil {
		t.Fatalf("failed to create JetStream: %v", err)
	}

	// Setup stream
	stream, err := nats.SetupStream(ctx, js)
	if err != nil {
		t.Fatalf("failed to setup stream: %v", err)
	}

	// Create store
	store := session.NewStore(js, stream)

	// Create server
	sessionName := "test-session"
	srv := New(store, sessionName)

	cleanup := func() {
		nc.Close()
		ns.Shutdown()
	}

	return srv, cleanup
}

// extractText extracts text from CallToolResult.Content[0]
func extractText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		return textContent.Text
	}
	return ""
}

func TestHandleTaskAdd_Success(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create request with tasks array
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content":  "Test task 1",
						"status":   "remaining",
						"priority": float64(2),
					},
					map[string]any{
						"content": "Test task 2",
					},
				},
			},
		},
	}

	// Call handler
	result, err := srv.handleTaskAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleTaskAdd returned error: %v", err)
	}

	// Check result
	if len(result.Content) == 0 {
		t.Fatal("result has no content")
	}

	text := extractText(result)
	if !strings.Contains(text, "Added 2 task(s)") {
		t.Errorf("unexpected result: %s", text)
	}
	if !strings.Contains(text, "TAS-1") {
		t.Errorf("missing TAS-1 in result: %s", text)
	}
	if !strings.Contains(text, "TAS-2") {
		t.Errorf("missing TAS-2 in result: %s", text)
	}
}

func TestHandleTaskAdd_MissingTasksParam(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "task-add",
			Arguments: map[string]any{},
		},
	}

	result, err := srv.handleTaskAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleTaskAdd returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error: missing 'tasks' parameter") {
		t.Errorf("unexpected error message: %s", text)
	}
}

func TestHandleTaskAdd_EmptyContent(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "",
					},
				},
			},
		},
	}

	result, err := srv.handleTaskAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleTaskAdd returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") && !strings.Contains(text, "content") {
		t.Errorf("expected content validation error, got: %s", text)
	}
}

func TestHandleTaskAdd_DuplicateContent(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	ctx := context.Background()

	// Add first task
	req1 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Duplicate task",
					},
				},
			},
		},
	}
	_, err := srv.handleTaskAdd(ctx, req1)
	if err != nil {
		t.Fatalf("first handleTaskAdd failed: %v", err)
	}

	// Try to add duplicate
	req2 := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name: "task-add",
			Arguments: map[string]any{
				"tasks": []any{
					map[string]any{
						"content": "Duplicate task",
					},
				},
			},
		},
	}

	result, err := srv.handleTaskAdd(ctx, req2)
	if err != nil {
		t.Fatalf("handleTaskAdd returned error: %v", err)
	}

	text := extractText(result)
	if !strings.Contains(text, "error:") && !strings.Contains(text, "already exists") {
		t.Errorf("expected duplicate error, got: %s", text)
	}
}
