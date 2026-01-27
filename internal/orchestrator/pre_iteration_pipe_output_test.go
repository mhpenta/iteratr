package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/iteratr/internal/hooks"
)

// TestPreIterationRespectsPipeOutput verifies that pre_iteration hooks
// only pipe output to agent when pipe_output: true is set
func TestPreIterationRespectsPipeOutput(t *testing.T) {
	tests := []struct {
		name             string
		pipeOutput       bool
		expectedInOutput bool
	}{
		{
			name:             "pipe_output=true sends output to agent",
			pipeOutput:       true,
			expectedInOutput: true,
		},
		{
			name:             "pipe_output=false (default) does not send output to agent",
			pipeOutput:       false,
			expectedInOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test
			tmpDir := t.TempDir()

			// Create a pre-iteration hook script
			preHookPath := filepath.Join(tmpDir, "pre_hook.sh")
			hookScript := "#!/bin/sh\necho 'HOOK_OUTPUT'"
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
							PipeOutput: tt.pipeOutput,
						},
					},
				},
			}

			// Execute pre-iteration hooks using ExecuteAllPiped
			ctx := context.Background()
			hookVars := hooks.Variables{
				Session:   "test-session",
				Iteration: "1",
			}
			output, err := hooks.ExecuteAllPiped(ctx, hooksConfig.Hooks.PreIteration, tmpDir, hookVars)
			if err != nil {
				t.Fatalf("Pre-iteration hook execution failed: %v", err)
			}

			// Verify output based on pipe_output setting
			hasOutput := strings.Contains(output, "HOOK_OUTPUT")
			if tt.expectedInOutput && !hasOutput {
				t.Errorf("Expected output to contain 'HOOK_OUTPUT' when pipe_output=%v, but got: %q", tt.pipeOutput, output)
			}
			if !tt.expectedInOutput && hasOutput {
				t.Errorf("Expected output to NOT contain 'HOOK_OUTPUT' when pipe_output=%v, but got: %q", tt.pipeOutput, output)
			}
		})
	}
}

// TestPreIterationMultipleHooksPipeOutput verifies that only hooks with
// pipe_output=true contribute to the output when multiple hooks are configured
func TestPreIterationMultipleHooksPipeOutput(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Create two hook scripts
	hook1Path := filepath.Join(tmpDir, "hook1.sh")
	hook1Script := "#!/bin/sh\necho 'HOOK1_OUTPUT'"
	if err := os.WriteFile(hook1Path, []byte(hook1Script), 0755); err != nil {
		t.Fatalf("Failed to write hook1 script: %v", err)
	}

	hook2Path := filepath.Join(tmpDir, "hook2.sh")
	hook2Script := "#!/bin/sh\necho 'HOOK2_OUTPUT'"
	if err := os.WriteFile(hook2Path, []byte(hook2Script), 0755); err != nil {
		t.Fatalf("Failed to write hook2 script: %v", err)
	}

	// Create hooks config with one hook piped, one not piped
	hooksConfig := &hooks.Config{
		Version: 1,
		Hooks: hooks.HooksConfig{
			PreIteration: []*hooks.HookConfig{
				{
					Command:    hook1Path,
					Timeout:    5,
					PipeOutput: true, // This should be piped
				},
				{
					Command:    hook2Path,
					Timeout:    5,
					PipeOutput: false, // This should NOT be piped
				},
			},
		},
	}

	// Execute pre-iteration hooks using ExecuteAllPiped
	ctx := context.Background()
	hookVars := hooks.Variables{
		Session:   "test-session",
		Iteration: "1",
	}
	output, err := hooks.ExecuteAllPiped(ctx, hooksConfig.Hooks.PreIteration, tmpDir, hookVars)
	if err != nil {
		t.Fatalf("Pre-iteration hook execution failed: %v", err)
	}

	// Verify only HOOK1_OUTPUT is in the output (pipe_output=true)
	if !strings.Contains(output, "HOOK1_OUTPUT") {
		t.Errorf("Expected output to contain 'HOOK1_OUTPUT' (pipe_output=true), but got: %q", output)
	}

	// Verify HOOK2_OUTPUT is NOT in the output (pipe_output=false)
	if strings.Contains(output, "HOOK2_OUTPUT") {
		t.Errorf("Expected output to NOT contain 'HOOK2_OUTPUT' (pipe_output=false), but got: %q", output)
	}
}
