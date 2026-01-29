package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/mark3labs/iteratr/internal/nats"
	"github.com/mark3labs/iteratr/internal/session"
)

const (
	headerSessionID = "X-Mcp-Session-Id"
	jsonRPCVersion  = "2.0"
	protocolVersion = "2024-11-05"
)

// TestServerIntegration tests the full MCP server lifecycle via HTTP
func TestServerIntegration(t *testing.T) {
	ctx := context.Background()

	// Create embedded NATS
	ns, _, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("failed to start NATS: %v", err)
	}
	defer ns.Shutdown()

	// Connect to NATS
	nc, err := nats.ConnectInProcess(ns)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

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
	sessionName := "integration-test-session"
	srv := New(store, sessionName)

	// Start server
	port, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("failed to stop server: %v", err)
		}
	}()

	// Give server a moment to fully initialize
	time.Sleep(100 * time.Millisecond)

	// Get server URL
	serverURL := srv.URL()
	expectedURL := fmt.Sprintf("http://localhost:%d/mcp", port)
	if serverURL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, serverURL)
	}

	// Initialize MCP session
	sessionID := initializeSession(t, serverURL)

	// Test 1: Add tasks via HTTP
	t.Run("AddTasks", func(t *testing.T) {
		tasks := []any{
			map[string]any{
				"content":  "Test task 1",
				"status":   "remaining",
				"priority": float64(2),
			},
			map[string]any{
				"content":  "Test task 2",
				"priority": float64(1),
			},
		}

		result := callTool(t, serverURL, sessionID, "task-add", map[string]any{
			"tasks": tasks,
		})

		if !contains(result, "Added 2 task(s)") {
			t.Errorf("expected success message, got: %s", result)
		}
		if !contains(result, "TAS-1") || !contains(result, "TAS-2") {
			t.Errorf("expected task IDs in result, got: %s", result)
		}
	})

	// Test 2: List tasks via HTTP
	t.Run("ListTasks", func(t *testing.T) {
		result := callTool(t, serverURL, sessionID, "task-list", nil)

		if !contains(result, "Remaining:") {
			t.Errorf("expected 'Remaining:' section, got: %s", result)
		}
		if !contains(result, "[TAS-1]") {
			t.Errorf("expected TAS-1 in list, got: %s", result)
		}
		if !contains(result, "[TAS-2]") {
			t.Errorf("expected TAS-2 in list, got: %s", result)
		}
	})

	// Test 3: Get next task via HTTP
	t.Run("GetNextTask", func(t *testing.T) {
		result := callTool(t, serverURL, sessionID, "task-next", nil)

		var taskData map[string]any
		if err := json.Unmarshal([]byte(result), &taskData); err != nil {
			t.Fatalf("expected JSON output, got: %s", result)
		}

		// Should be TAS-2 (higher priority)
		if taskData["id"] != "TAS-2" {
			t.Errorf("expected TAS-2 (highest priority), got: %v", taskData["id"])
		}
		if taskData["content"] != "Test task 2" {
			t.Errorf("expected 'Test task 2', got: %v", taskData["content"])
		}
	})

	// Test 4: Update task via HTTP
	t.Run("UpdateTask", func(t *testing.T) {
		result := callTool(t, serverURL, sessionID, "task-update", map[string]any{
			"id":     "TAS-1",
			"status": "in_progress",
		})

		if !contains(result, "Updated task TAS-1") {
			t.Errorf("expected success message, got: %s", result)
		}
		if !contains(result, "status=in_progress") {
			t.Errorf("expected status update, got: %s", result)
		}
	})

	// Test 5: Error handling via HTTP
	t.Run("ErrorHandling", func(t *testing.T) {
		// Try to update non-existent task
		result := callTool(t, serverURL, sessionID, "task-update", map[string]any{
			"id":     "TAS-999",
			"status": "completed",
		})

		if !contains(result, "error:") {
			t.Errorf("expected error message, got: %s", result)
		}
	})
}

// TestServerStartStop tests the Start and Stop lifecycle
func TestServerStartStop(t *testing.T) {
	ctx := context.Background()

	// Create embedded NATS
	ns, _, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("failed to start NATS: %v", err)
	}
	defer ns.Shutdown()

	// Connect to NATS
	nc, err := nats.ConnectInProcess(ns)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

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
	srv := New(store, "test-session")

	// Start server
	port, err := srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	if port == 0 {
		t.Error("expected non-zero port")
	}

	// Verify URL is accessible
	serverURL := srv.URL()
	expectedURL := fmt.Sprintf("http://localhost:%d/mcp", port)
	if serverURL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, serverURL)
	}

	// Give server a moment to fully initialize
	time.Sleep(100 * time.Millisecond)

	// Verify server responds to HTTP requests
	resp, err := http.Get(serverURL)
	if err != nil {
		t.Fatalf("failed to connect to server: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Errorf("failed to close response body: %v", err)
	}

	// Stop server
	if err := srv.Stop(); err != nil {
		t.Fatalf("failed to stop server: %v", err)
	}

	// Verify server is stopped (connection should fail)
	time.Sleep(100 * time.Millisecond)
	_, err = http.Get(serverURL)
	if err == nil {
		t.Error("expected connection error after server stopped, but connection succeeded")
	}

	// Double-stop should be safe
	if err := srv.Stop(); err != nil {
		t.Errorf("double stop should be safe, got error: %v", err)
	}
}

// TestServerDoubleStart tests that starting a server twice fails
func TestServerDoubleStart(t *testing.T) {
	ctx := context.Background()

	// Create embedded NATS
	ns, _, err := nats.StartEmbeddedNATS(t.TempDir())
	if err != nil {
		t.Fatalf("failed to start NATS: %v", err)
	}
	defer ns.Shutdown()

	// Connect to NATS
	nc, err := nats.ConnectInProcess(ns)
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	defer nc.Close()

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
	srv := New(store, "test-session")

	// Start server
	_, err = srv.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	defer func() {
		if err := srv.Stop(); err != nil {
			t.Errorf("failed to stop server: %v", err)
		}
	}()

	// Try to start again
	_, err = srv.Start(ctx)
	if err == nil {
		t.Error("expected error when starting server twice, got nil")
	}
	if err != nil && !contains(err.Error(), "already started") {
		t.Errorf("expected 'already started' error, got: %v", err)
	}
}

// initializeSession initializes an MCP session and returns the session ID (or empty for stateless)
func initializeSession(t *testing.T, serverURL string) string {
	t.Helper()

	// Create initialize request
	initReq := map[string]any{
		"jsonrpc": jsonRPCVersion,
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": protocolVersion,
			"clientInfo": map[string]any{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	reqBody, err := json.Marshal(initReq)
	if err != nil {
		t.Fatalf("failed to marshal initialize request: %v", err)
	}

	// Make POST request
	resp, err := http.Post(serverURL, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed to make initialize request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("initialize request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Get session ID from header (may be empty for stateless servers)
	sessionID := resp.Header.Get(headerSessionID)

	t.Logf("Session ID: %q", sessionID)
	return sessionID
}

// callTool makes an HTTP request to the MCP server to call a tool
func callTool(t *testing.T, serverURL string, sessionID string, toolName string, args map[string]any) string {
	t.Helper()

	// Create JSON-RPC request for tools/call
	jsonrpcReq := map[string]any{
		"jsonrpc": jsonRPCVersion,
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": args,
		},
	}

	// Marshal to JSON
	reqBody, err := json.Marshal(jsonrpcReq)
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	// Make HTTP POST request with session ID header
	req, err := http.NewRequest(http.MethodPost, serverURL, bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(headerSessionID, sessionID)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to make HTTP request: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("failed to close response body: %v", err)
		}
	}()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	// Parse JSON-RPC response
	var jsonrpcResp map[string]any
	if err := json.Unmarshal(respBody, &jsonrpcResp); err != nil {
		t.Fatalf("failed to unmarshal response: %v (body: %s)", err, string(respBody))
	}

	// Extract result
	result, ok := jsonrpcResp["result"].(map[string]any)
	if !ok {
		// Check for error
		if errObj, ok := jsonrpcResp["error"]; ok {
			t.Fatalf("tool call failed: %v", errObj)
		}
		t.Fatalf("unexpected response format: %v", jsonrpcResp)
	}

	// Extract content array
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		return ""
	}

	// Extract text from first content item
	if contentItem, ok := content[0].(map[string]any); ok {
		if text, ok := contentItem["text"].(string); ok {
			return text
		}
	}

	t.Fatalf("unexpected content type: %T", content[0])
	return ""
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
