package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/iteratr/internal/hooks"
)

// TestOnErrorHooksExecute verifies on_error hooks execute when error occurs
func TestOnErrorHooksExecute(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "error-hook-output.txt")

	// Create hooks config with on_error hook
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			OnError: []*hooks.HookConfig{
				{
					Command:    fmt.Sprintf("echo 'Error: {{error}}' > %s", outputFile),
					Timeout:    5,
					PipeOutput: false,
				},
			},
		},
	}

	ctx := context.Background()

	// Simulate error handling code from orchestrator
	testError := fmt.Errorf("simulated agent error")
	hookVars := hooks.Variables{
		Session:   "test-session",
		Iteration: "1",
		Error:     testError.Error(),
	}

	output, err := hooks.ExecuteAllPiped(ctx, hooksConfig.Hooks.OnError, tmpDir, hookVars)
	if err != nil {
		t.Fatalf("on_error hook execution failed: %v", err)
	}

	// Verify output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("on_error hook did not execute (output file not created)")
	}

	// Verify file contents
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	expectedSubstr := "simulated agent error"
	if !contains(string(content), expectedSubstr) {
		t.Errorf("Hook output missing error message.\nExpected substring: %q\nGot: %q", expectedSubstr, string(content))
	}

	// With pipe_output=false, should return empty string
	if output != "" {
		t.Errorf("Expected empty output with pipe_output=false, got: %q", output)
	}
}

// TestOnErrorHooksWithPipeOutput verifies piped output is returned
func TestOnErrorHooksWithPipeOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hooks config with pipe_output enabled
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			OnError: []*hooks.HookConfig{
				{
					Command:    "echo 'Diagnostic output for error: {{error}}'",
					Timeout:    5,
					PipeOutput: true,
				},
			},
		},
	}

	ctx := context.Background()
	testError := fmt.Errorf("network timeout")

	hookVars := hooks.Variables{
		Session:   "test-session",
		Iteration: "2",
		Error:     testError.Error(),
	}

	output, err := hooks.ExecuteAllPiped(ctx, hooksConfig.Hooks.OnError, tmpDir, hookVars)
	if err != nil {
		t.Fatalf("on_error hook execution failed: %v", err)
	}

	// With pipe_output=true, should return hook output
	if output == "" {
		t.Fatal("Expected non-empty output with pipe_output=true")
	}

	expectedSubstr := "network timeout"
	if !contains(output, expectedSubstr) {
		t.Errorf("Hook output missing error message.\nExpected substring: %q\nGot: %q", expectedSubstr, output)
	}
}

// TestOnErrorHooksVariableExpansion verifies all variables are expanded correctly
func TestOnErrorHooksVariableExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "variables.txt")

	// Create hooks config that uses all error hook variables
	cmd := fmt.Sprintf("echo 'Session: {{session}}, Iteration: {{iteration}}, Error: {{error}}' > %s", outputFile)
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			OnError: []*hooks.HookConfig{
				{
					Command: cmd,
					Timeout: 5,
				},
			},
		},
	}

	ctx := context.Background()

	hookVars := hooks.Variables{
		Session:   "my-session",
		Iteration: "5",
		Error:     "test error message",
	}

	_, err := hooks.ExecuteAllPiped(ctx, hooksConfig.Hooks.OnError, tmpDir, hookVars)
	if err != nil {
		t.Fatalf("Hook execution failed: %v", err)
	}

	// Read output file and verify all variables were expanded
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	output := string(content)
	expectedSubstrings := []string{
		"Session: my-session",
		"Iteration: 5",
		"Error: test error message",
	}

	for _, expected := range expectedSubstrings {
		if !contains(output, expected) {
			t.Errorf("Output missing expected substring: %q\nFull output: %q", expected, output)
		}
	}
}

// TestOnErrorHooksMultiple verifies multiple hooks execute in order
func TestOnErrorHooksMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "multi-output.txt")

	// Create multiple hooks that append to same file
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			OnError: []*hooks.HookConfig{
				{
					Command:    fmt.Sprintf("echo 'Hook 1' >> %s", outputFile),
					Timeout:    5,
					PipeOutput: false,
				},
				{
					Command:    fmt.Sprintf("echo 'Hook 2' >> %s", outputFile),
					Timeout:    5,
					PipeOutput: false,
				},
				{
					Command:    fmt.Sprintf("echo 'Hook 3' >> %s", outputFile),
					Timeout:    5,
					PipeOutput: false,
				},
			},
		},
	}

	ctx := context.Background()
	hookVars := hooks.Variables{
		Session:   "test-session",
		Iteration: "1",
		Error:     "test error",
	}

	_, err := hooks.ExecuteAllPiped(ctx, hooksConfig.Hooks.OnError, tmpDir, hookVars)
	if err != nil {
		t.Fatalf("Hook execution failed: %v", err)
	}

	// Verify all hooks executed
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}

	output := string(content)
	expectedHooks := []string{"Hook 1", "Hook 2", "Hook 3"}
	for _, expected := range expectedHooks {
		if !contains(output, expected) {
			t.Errorf("Output missing hook: %q\nFull output: %q", expected, output)
		}
	}
}

// TestOnErrorHooksContextCancellation verifies hooks respect context cancellation
func TestOnErrorHooksContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()

	// Create hook with long-running command
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			OnError: []*hooks.HookConfig{
				{
					Command: "sleep 10",
					Timeout: 30,
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	hookVars := hooks.Variables{
		Session:   "test-session",
		Iteration: "1",
		Error:     "test error",
	}

	_, err := hooks.ExecuteAllPiped(ctx, hooksConfig.Hooks.OnError, tmpDir, hookVars)
	if err == nil {
		t.Fatal("Expected error due to context cancellation")
	}

	// Should get context cancelled error
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// TestOnErrorNoHooksConfigured verifies no error when no hooks configured
func TestOnErrorNoHooksConfigured(t *testing.T) {
	tmpDir := t.TempDir()

	// No on_error hooks configured
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks:   hooks.HooksConfig{},
	}

	ctx := context.Background()
	hookVars := hooks.Variables{
		Session:   "test-session",
		Iteration: "1",
		Error:     "test error",
	}

	// Should not error when no hooks configured
	output, err := hooks.ExecuteAllPiped(ctx, hooksConfig.Hooks.OnError, tmpDir, hookVars)
	if err != nil {
		t.Fatalf("Unexpected error with no hooks configured: %v", err)
	}

	if output != "" {
		t.Errorf("Expected empty output with no hooks, got: %q", output)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
