package orchestrator

import (
	"context"
	"testing"
)

// TestFinalDeliveryLogic verifies the logic of checking and draining pending output.
func TestFinalDeliveryLogic(t *testing.T) {
	t.Run("pending output is drained in final delivery", func(t *testing.T) {
		o := &Orchestrator{}

		// Simulate pending output from post_iteration
		o.appendPendingOutput("Test output from post_iteration")

		// Verify hasPendingOutput returns true
		if !o.hasPendingOutput() {
			t.Error("Expected pending output before final delivery")
		}

		// Drain pending output (simulating final delivery)
		output := o.drainPendingOutput()
		if output != "Test output from post_iteration" {
			t.Errorf("Expected 'Test output from post_iteration', got %q", output)
		}

		// Verify buffer is empty after drain
		if o.hasPendingOutput() {
			t.Error("Expected no pending output after final delivery")
		}
	})

	t.Run("no final delivery when buffer is empty", func(t *testing.T) {
		o := &Orchestrator{}

		// Initially empty
		if o.hasPendingOutput() {
			t.Error("Expected no pending output initially")
		}

		// Draining empty buffer returns empty string
		output := o.drainPendingOutput()
		if output != "" {
			t.Errorf("Expected empty string, got %q", output)
		}
	})

	t.Run("multiple sources combined in FIFO order", func(t *testing.T) {
		o := &Orchestrator{}

		// Simulate multiple sources appending to pending buffer
		o.appendPendingOutput("Output from session_start")
		o.appendPendingOutput("Output from post_iteration #1")
		o.appendPendingOutput("Output from on_task_complete")
		o.appendPendingOutput("Output from post_iteration #2")

		// Drain should return all outputs in FIFO order
		output := o.drainPendingOutput()
		expected := "Output from session_start\n" +
			"Output from post_iteration #1\n" +
			"Output from on_task_complete\n" +
			"Output from post_iteration #2"

		if output != expected {
			t.Errorf("Expected %q, got %q", expected, output)
		}

		// Buffer should be empty after drain
		if o.hasPendingOutput() {
			t.Error("Expected no pending output after drain")
		}
	})

	t.Run("context cancellation safety", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		o := &Orchestrator{
			ctx:    ctx,
			cancel: cancel,
		}

		// Append some output
		o.appendPendingOutput("Test output")

		// Verify we can still drain even after context cancellation
		// (the actual Run() method checks ctx.Err() and returns nil)
		output := o.drainPendingOutput()
		if output != "Test output" {
			t.Errorf("Expected 'Test output', got %q", output)
		}

		// Verify context is cancelled
		if ctx.Err() == nil {
			t.Error("Expected context to be cancelled")
		}
	})

	t.Run("empty string append is no-op", func(t *testing.T) {
		o := &Orchestrator{}

		// Append real output
		o.appendPendingOutput("Real output")

		// Append empty string (should not add newline)
		o.appendPendingOutput("")

		// Drain should only return real output
		output := o.drainPendingOutput()
		if output != "Real output" {
			t.Errorf("Expected 'Real output', got %q", output)
		}
	})
}
