package orchestrator

import (
	"testing"
)

func TestDrainPendingOutputBeforePreIteration(t *testing.T) {
	t.Run("pending output is prepended to pre-iteration output", func(t *testing.T) {
		// Create orchestrator
		o := &Orchestrator{}

		// Add pending output (simulating session_start, post_iteration, etc.)
		o.appendPendingOutput("pending output 1")
		o.appendPendingOutput("pending output 2")

		// Verify pending buffer has content
		if !o.hasPendingOutput() {
			t.Fatal("Expected pending buffer to have content")
		}

		// Drain pending output (this is what happens at start of iteration)
		drained := o.drainPendingOutput()

		// Verify drained output contains all pending outputs in FIFO order
		expected := "pending output 1\npending output 2"
		if drained != expected {
			t.Errorf("Expected drained output to be %q, got %q", expected, drained)
		}

		// Verify pending buffer is now empty
		if o.hasPendingOutput() {
			t.Error("Expected pending buffer to be empty after draining")
		}

		// Simulate pre-iteration hook output
		preIterationOutput := "pre-iteration hook output"

		// Combine pending and pre-iteration output (as done in orchestrator)
		combinedOutput := drained
		if len(preIterationOutput) > 0 {
			combinedOutput = drained + "\n" + preIterationOutput
		}

		// Verify combined output has pending first, then pre-iteration
		expectedCombined := "pending output 1\npending output 2\npre-iteration hook output"
		if combinedOutput != expectedCombined {
			t.Errorf("Expected combined output to be %q, got %q", expectedCombined, combinedOutput)
		}
	})

	t.Run("empty pending buffer doesn't affect pre-iteration", func(t *testing.T) {
		o := &Orchestrator{}

		// Drain empty buffer
		drained := o.drainPendingOutput()

		// Verify drained output is empty
		if drained != "" {
			t.Errorf("Expected drained output to be empty, got %q", drained)
		}

		// Simulate pre-iteration hook output
		preIterationOutput := "pre-iteration hook output"

		// Combine (as done in orchestrator)
		combinedOutput := ""
		if len(drained) > 0 {
			if len(preIterationOutput) > 0 {
				combinedOutput = drained + "\n" + preIterationOutput
			} else {
				combinedOutput = drained
			}
		} else {
			combinedOutput = preIterationOutput
		}

		// Verify combined output is just pre-iteration output
		if combinedOutput != preIterationOutput {
			t.Errorf("Expected combined output to be %q, got %q", preIterationOutput, combinedOutput)
		}
	})

	t.Run("pending buffer is drained only once", func(t *testing.T) {
		o := &Orchestrator{}

		// Add pending output
		o.appendPendingOutput("pending output")

		// First drain
		drained1 := o.drainPendingOutput()
		if drained1 != "pending output" {
			t.Errorf("Expected first drain to be %q, got %q", "pending output", drained1)
		}

		// Second drain (should be empty)
		drained2 := o.drainPendingOutput()
		if drained2 != "" {
			t.Errorf("Expected second drain to be empty, got %q", drained2)
		}
	})

	t.Run("multiple pending sources maintain FIFO order", func(t *testing.T) {
		o := &Orchestrator{}

		// Simulate multiple sources appending to buffer in chronological order
		o.appendPendingOutput("session_start output")    // First
		o.appendPendingOutput("post_iteration output")   // Second
		o.appendPendingOutput("on_task_complete output") // Third

		// Drain and verify FIFO order
		drained := o.drainPendingOutput()
		expected := "session_start output\npost_iteration output\non_task_complete output"
		if drained != expected {
			t.Errorf("Expected FIFO order %q, got %q", expected, drained)
		}
	})

	t.Run("empty pending and no pre-iteration results in empty output", func(t *testing.T) {
		o := &Orchestrator{}

		// Drain empty buffer
		drained := o.drainPendingOutput()

		// No pre-iteration hooks
		preIterationOutput := ""

		// Combine
		combinedOutput := ""
		if len(drained) > 0 {
			if len(preIterationOutput) > 0 {
				combinedOutput = drained + "\n" + preIterationOutput
			} else {
				combinedOutput = drained
			}
		} else {
			combinedOutput = preIterationOutput
		}

		// Verify combined output is empty
		if combinedOutput != "" {
			t.Errorf("Expected combined output to be empty, got %q", combinedOutput)
		}
	})

	t.Run("pending output only with no pre-iteration hooks", func(t *testing.T) {
		o := &Orchestrator{}

		// Add pending output
		o.appendPendingOutput("pending output only")

		// Drain
		drained := o.drainPendingOutput()

		// No pre-iteration hooks
		preIterationOutput := ""

		// Combine
		combinedOutput := ""
		if len(drained) > 0 {
			if len(preIterationOutput) > 0 {
				combinedOutput = drained + "\n" + preIterationOutput
			} else {
				combinedOutput = drained
			}
		} else {
			combinedOutput = preIterationOutput
		}

		// Verify combined output is just pending output
		if combinedOutput != "pending output only" {
			t.Errorf("Expected combined output to be %q, got %q", "pending output only", combinedOutput)
		}
	})
}
