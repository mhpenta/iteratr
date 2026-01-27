package orchestrator

import (
	"sync"
	"testing"
)

func TestPendingOutputBuffer(t *testing.T) {
	t.Run("append and drain", func(t *testing.T) {
		o := &Orchestrator{}

		// Initially empty
		if o.hasPendingOutput() {
			t.Error("Expected no pending output initially")
		}

		// Append first output
		o.appendPendingOutput("First output")
		if !o.hasPendingOutput() {
			t.Error("Expected pending output after append")
		}

		// Append second output
		o.appendPendingOutput("Second output")

		// Drain should return both outputs in FIFO order
		output := o.drainPendingOutput()
		expected := "First output\nSecond output"
		if output != expected {
			t.Errorf("Expected %q, got %q", expected, output)
		}

		// After drain, buffer should be empty
		if o.hasPendingOutput() {
			t.Error("Expected no pending output after drain")
		}

		// Draining again should return empty string
		output = o.drainPendingOutput()
		if output != "" {
			t.Errorf("Expected empty string, got %q", output)
		}
	})

	t.Run("append empty string is no-op", func(t *testing.T) {
		o := &Orchestrator{}

		o.appendPendingOutput("")
		if o.hasPendingOutput() {
			t.Error("Expected no pending output after appending empty string")
		}

		o.appendPendingOutput("Real output")
		o.appendPendingOutput("")

		output := o.drainPendingOutput()
		if output != "Real output" {
			t.Errorf("Expected 'Real output', got %q", output)
		}
	})

	t.Run("thread safety", func(t *testing.T) {
		o := &Orchestrator{}

		// Simulate concurrent appends from NATS callbacks
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				o.appendPendingOutput("output")
			}(i)
		}

		wg.Wait()

		// All appends should have succeeded
		if !o.hasPendingOutput() {
			t.Error("Expected pending output after concurrent appends")
		}
	})

	t.Run("multiple append and drain cycles", func(t *testing.T) {
		o := &Orchestrator{}

		// Cycle 1
		o.appendPendingOutput("Output 1")
		output := o.drainPendingOutput()
		if output != "Output 1" {
			t.Errorf("Cycle 1: Expected 'Output 1', got %q", output)
		}

		// Cycle 2 - buffer should be fresh
		o.appendPendingOutput("Output 2")
		output = o.drainPendingOutput()
		if output != "Output 2" {
			t.Errorf("Cycle 2: Expected 'Output 2', got %q", output)
		}
	})
}
