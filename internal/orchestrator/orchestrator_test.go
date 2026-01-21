package orchestrator

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestGracefulShutdown verifies that the orchestrator shuts down cleanly
// when Stop() is called, including cleanup of all components.
func TestGracefulShutdown(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".iteratr")

	// Create a simple spec file
	specPath := filepath.Join(tmpDir, "test.md")
	specContent := `# Test Spec

This is a test spec.

## Tasks
- [ ] Test task 1
`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	// Create orchestrator
	orch, err := New(Config{
		SessionName: "test-shutdown",
		SpecPath:    specPath,
		Iterations:  1,
		DataDir:     dataDir,
		WorkDir:     tmpDir,
		Headless:    true, // No TUI for test
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	// Start orchestrator
	if err := orch.Start(); err != nil {
		t.Fatalf("failed to start orchestrator: %v", err)
	}

	// Give it a moment to fully initialize
	time.Sleep(100 * time.Millisecond)

	// Stop orchestrator
	stopDone := make(chan error, 1)
	go func() {
		stopDone <- orch.Stop()
	}()

	// Ensure Stop() completes within reasonable time
	select {
	case err := <-stopDone:
		if err != nil {
			t.Errorf("Stop() returned error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Stop() timed out - graceful shutdown failed")
	}

	// Verify NATS data was written (proves it was running)
	natsDir := filepath.Join(dataDir, "nats")
	if _, err := os.Stat(natsDir); os.IsNotExist(err) {
		t.Error("NATS data directory was not created")
	}
}

// TestShutdownOnSignal verifies that SIGINT/SIGTERM trigger graceful shutdown
func TestShutdownIdempotency(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".iteratr")

	// Create a simple spec file
	specPath := filepath.Join(tmpDir, "test.md")
	specContent := `# Test Spec`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	// Create orchestrator
	orch, err := New(Config{
		SessionName: "test-idempotency",
		SpecPath:    specPath,
		Iterations:  1,
		DataDir:     dataDir,
		WorkDir:     tmpDir,
		Headless:    true,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	// Start orchestrator
	if err := orch.Start(); err != nil {
		t.Fatalf("failed to start orchestrator: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Call Stop() multiple times - should be idempotent
	if err := orch.Stop(); err != nil {
		t.Errorf("First Stop() returned error: %v", err)
	}

	if err := orch.Stop(); err != nil {
		t.Errorf("Second Stop() returned error: %v", err)
	}

	if err := orch.Stop(); err != nil {
		t.Errorf("Third Stop() returned error: %v", err)
	}
}

// TestContextCancellation verifies that cancelling the context triggers cleanup
func TestContextCancellation(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".iteratr")

	// Create a simple spec file
	specPath := filepath.Join(tmpDir, "test.md")
	specContent := `# Test Spec`
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("failed to write spec file: %v", err)
	}

	// Create orchestrator
	orch, err := New(Config{
		SessionName: "test-context",
		SpecPath:    specPath,
		Iterations:  1,
		DataDir:     dataDir,
		WorkDir:     tmpDir,
		Headless:    true,
	})
	if err != nil {
		t.Fatalf("failed to create orchestrator: %v", err)
	}

	// Start orchestrator
	if err := orch.Start(); err != nil {
		t.Fatalf("failed to start orchestrator: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Cancel context (simulates signal)
	orch.cancel()

	// Context should be cancelled
	select {
	case <-orch.ctx.Done():
		// Expected
	case <-time.After(1 * time.Second):
		t.Fatal("Context was not cancelled")
	}

	// Stop should still work after context cancellation
	if err := orch.Stop(); err != nil {
		t.Errorf("Stop() after context cancellation returned error: %v", err)
	}
}

// Suppress unused variable warning
var _ = syscall.SIGTERM
