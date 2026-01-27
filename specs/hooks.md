# Hooks

## Overview

Hooks allow users to run shell commands at key lifecycle points: session start/end, before/after iterations, on task completion, and on errors. Hook output can optionally be piped to the agent for automated fixes.

## User Story

As a developer, I want to run custom scripts at different lifecycle points so I can:
- Run setup/teardown tasks at session start/end
- Have the agent see and fix lint/test failures automatically
- Validate completed tasks
- Provide diagnostics on errors

## Requirements

- Config file: `.iteratr.hooks.yml` in working directory
- Six hook types: `session_start`, `pre_iteration`, `post_iteration`, `session_end`, `on_task_complete`, `on_error`
- `pipe_output` field controls whether output is sent to agent (default: `false`)
- Shell command execution with stdout/stderr capture
- Template variable expansion in commands (varies by hook type)
- 30 second default timeout
- Graceful error handling (continue session with error in output)
- Raw output (no framing headers)

## Breaking Changes (v2)

**pre_iteration default changed:** Previously, pre_iteration always piped output to the agent. Now it respects `pipe_output` (default: `false`). **Existing configs must add `pipe_output: true` to each pre_iteration hook** to maintain previous behavior.

## Config Format

```yaml
version: 1

hooks:
  session_start:
    - command: "go build ./..."
      timeout: 60
      pipe_output: true  # Send to agent in first iteration

  pre_iteration:
    - command: "golangci-lint run ./..."
      timeout: 30
      pipe_output: true  # REQUIRED for agent to see output
    - command: "git status --short"
      timeout: 5
      pipe_output: true

  post_iteration:
    - command: "go test ./... -short"
      timeout: 120
      pipe_output: true  # Agent sees failures next iteration

  session_end:
    - command: "git push origin HEAD"
      timeout: 30
      # pipe_output ignored (no more iterations)

  on_task_complete:
    - command: "./scripts/validate.sh {{task_id}}"
      timeout: 30
      pipe_output: true

  on_error:
    - command: "git diff HEAD"
      timeout: 10
      pipe_output: true  # Show agent changes before error
```

See `.iteratr.hooks.example.yml` for comprehensive examples.

## Hook Lifecycle & Timing

```
Start() → session_start hooks → ITERATION LOOP → FINAL DELIVERY → session_end → Stop()
          (piped to 1st iter)   ↓                 (drain pending)
                                 ↓
                          pre_iteration hooks (with pending buffer)
                                 ↓
                          Agent executes
                                 ↓
                          IterationComplete
                                 ↓
                          post_iteration hooks (piped to next iter)
                                 ↓
                          Check session_complete → loop or exit
```

**on_task_complete**: NATS subscription, triggers when task marked completed, output appended to pending buffer  
**on_error**: Triggers on iteration failure, sends immediate recovery prompt if piped, continues session  
**Pending buffer**: Accumulates piped output (session_start, post_iteration, on_task_complete) in FIFO order, drained at each iteration start  
**Final delivery**: After loop exits, sends pending buffer to agent (if any) before session_end

## Template Variables

| Variable | Available in | Description |
|----------|--------------|-------------|
| `{{session}}` | All hooks | Session name |
| `{{iteration}}` | pre_iteration, post_iteration, on_error | Current iteration number |
| `{{task_id}}` | on_task_complete | Completed task ID |
| `{{task_content}}` | on_task_complete | Completed task content text |
| `{{error}}` | on_error | Error message |

## Technical Implementation

### Package: `internal/hooks/`

**types.go** - Configuration structs:
```go
type Config struct {
    Version int         `yaml:"version"`
    Hooks   HooksConfig `yaml:"hooks"`
}

type HooksConfig struct {
    SessionStart   []*HookConfig `yaml:"session_start"`
    PreIteration   []*HookConfig `yaml:"pre_iteration"`
    PostIteration  []*HookConfig `yaml:"post_iteration"`
    SessionEnd     []*HookConfig `yaml:"session_end"`
    OnTaskComplete []*HookConfig `yaml:"on_task_complete"`
    OnError        []*HookConfig `yaml:"on_error"`
}

type HookConfig struct {
    Command    string `yaml:"command"`
    Timeout    int    `yaml:"timeout"`     // seconds, default 30
    PipeOutput bool   `yaml:"pipe_output"` // default false
}

type Variables struct {
    Session     string
    Iteration   int
    TaskID      string
    TaskContent string
    Error       string
}
```

**hooks.go** - Loading and execution:
- `LoadConfig(workDir string) (*Config, error)` - Load `.iteratr.hooks.yml`, return nil if not found
- `Execute(ctx, hook, workDir, vars)` - Run single command, capture output
- `ExecuteAll(ctx, hooks, workDir, vars)` - Run all hooks, concatenate output
- `ExecuteAllPiped(ctx, hooks, workDir, vars)` - Run all hooks, return only output from hooks with pipe_output=true

### ACP Changes (`internal/agent/acp.go`)

- `prompt()` accepts `[]string` texts instead of single string
- Multiple texts sent as separate content blocks in same request

### Runner Changes (`internal/agent/runner.go`)

- `RunIteration(ctx, prompt, hookOutput)` accepts optional hook output
- Hook output sent as first content block, main prompt as second

### Orchestrator Changes (`internal/orchestrator/orchestrator.go`)

1. Add `hooksConfig *hooks.Config` and `pendingHookOutput string` fields (with mutex for concurrency)
2. Load hooks config in `Start()` (optional, log if missing)
3. **session_start**: Execute at start of `Run()`, pipe output to pending buffer
4. **pre_iteration**: Execute before iteration, prepend pending buffer to hook output, drain buffer
5. **post_iteration**: Execute after `IterationComplete`, pipe output to pending buffer
6. **session_end**: Execute after final delivery, ignore pipe_output
7. **on_task_complete**: Subscribe to NATS task events before loop, pipe output to pending buffer (mutex-protected)
8. **on_error**: Execute on iteration failure, send immediate recovery prompt if piped, continue session
9. **Final delivery**: After loop exits, send pending buffer to agent if non-empty, wait for response

### Error Handling

- Config not found: Skip hooks, continue normally
- Config parse error: Log warning, continue without hooks
- Command failure/timeout: Include error in output, continue iteration
- Context cancelled: Propagate cancellation

### TUI Safety

- Never write hook stderr to `os.Stderr`
- Capture stderr, include in output or log via logger
- Use `cmd.Output()` or pipe-based capture

## Implementation Status

✅ **Completed** (v2 - Extended Hooks):
- All six hook types implemented
- `pipe_output` field with default false
- Pending buffer with FIFO ordering
- Final delivery mechanism
- NATS subscription for on_task_complete
- Error recovery with on_error hooks
- Template variable expansion for all contexts
- Comprehensive test coverage

## Migration Guide

**For existing .iteratr.hooks.yml files:**

Before (v1):
```yaml
hooks:
  pre_iteration:
    - command: "golangci-lint run ./..."
      timeout: 30
```

After (v2):
```yaml
hooks:
  pre_iteration:
    - command: "golangci-lint run ./..."
      timeout: 30
      pipe_output: true  # ADD THIS LINE
```

**Without this change, pre_iteration output will not be sent to the agent.**
