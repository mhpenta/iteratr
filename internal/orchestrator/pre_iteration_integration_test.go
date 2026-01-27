package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/iteratr/internal/hooks"
)

// TestPreIterationWithPendingIntegration tests the full flow of pending output
// being drained and prepended to pre-iteration hook output
func TestPreIterationWithPendingIntegration(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create a pre-iteration hook script
	preHookPath := filepath.Join(tmpDir, "pre_hook.sh")
	hookScript := "#!/bin/sh\necho 'PRE_ITERATION_OUTPUT'"
	if err := os.WriteFile(preHookPath, []byte(hookScript), 0755); err != nil {
		t.Fatalf("Failed to write hook script: %v", err)
	}

	// Create hooks config
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			PreIteration: []*hooks.HookConfig{
				{
					Command:    preHookPath,
					Timeout:    5,
					PipeOutput: true,
				},
			},
		},
	}

	// Create orchestrator
	o := &Orchestrator{
		hooksConfig: hooksConfig,
	}

	// Simulate pending output from session_start (executed before iterations)
	o.appendPendingOutput("SESSION_START_OUTPUT")

	// Simulate pending output from post_iteration (from previous iteration)
	o.appendPendingOutput("POST_ITERATION_OUTPUT")

	// Now simulate what happens at the start of an iteration:
	// 1. Drain pending output
	pendingOutput := o.drainPendingOutput()

	// 2. Execute pre-iteration hooks
	ctx := context.Background()
	hookVars := hooks.Variables{
		Session:   "test-session",
		Iteration: "2",
	}
	preIterationOutput, err := hooks.ExecuteAll(ctx, hooksConfig.Hooks.PreIteration, tmpDir, hookVars)
	if err != nil {
		t.Fatalf("Pre-iteration hook execution failed: %v", err)
	}

	// 3. Combine pending and pre-iteration output
	var combinedOutput string
	if len(pendingOutput) > 0 {
		if len(preIterationOutput) > 0 {
			combinedOutput = pendingOutput + "\n" + preIterationOutput
		} else {
			combinedOutput = pendingOutput
		}
	} else {
		combinedOutput = preIterationOutput
	}

	// Verify the combined output contains all expected pieces in correct order
	if !strings.Contains(combinedOutput, "SESSION_START_OUTPUT") {
		t.Errorf("Combined output missing SESSION_START_OUTPUT.\nGot: %q", combinedOutput)
	}

	if !strings.Contains(combinedOutput, "POST_ITERATION_OUTPUT") {
		t.Errorf("Combined output missing POST_ITERATION_OUTPUT.\nGot: %q", combinedOutput)
	}

	if !strings.Contains(combinedOutput, "PRE_ITERATION_OUTPUT") {
		t.Errorf("Combined output missing PRE_ITERATION_OUTPUT.\nGot: %q", combinedOutput)
	}

	// Verify FIFO order: session_start -> post_iteration -> pre_iteration
	sessionIdx := strings.Index(combinedOutput, "SESSION_START_OUTPUT")
	postIdx := strings.Index(combinedOutput, "POST_ITERATION_OUTPUT")
	preIdx := strings.Index(combinedOutput, "PRE_ITERATION_OUTPUT")

	if sessionIdx == -1 || postIdx == -1 || preIdx == -1 {
		t.Fatalf("Not all outputs found in combined output: %q", combinedOutput)
	}

	if sessionIdx >= postIdx {
		t.Errorf("SESSION_START_OUTPUT should come before POST_ITERATION_OUTPUT.\nGot: %q", combinedOutput)
	}

	if postIdx >= preIdx {
		t.Errorf("POST_ITERATION_OUTPUT should come before PRE_ITERATION_OUTPUT.\nGot: %q", combinedOutput)
	}

	// Verify pending buffer was drained
	if o.hasPendingOutput() {
		t.Error("Pending buffer should be empty after draining")
	}
}
