# Extended Hooks

## Overview

Expand hooks system beyond `pre_iteration` to cover full session lifecycle: session start/end, post-iteration, task completion, and error handling. Add `pipe_output` option to control whether hook output is sent to the agent.

## User Story

As a developer, I want hooks at different lifecycle points so I can:
- Run setup scripts once at session start (pull latest, verify deps)
- Run tests/lints after each iteration and have agent fix failures
- Send notifications when session completes
- Alert on errors and provide diagnostics to agent

## Requirements

- 5 new hook types: `session_start`, `session_end`, `post_iteration`, `on_task_complete`, `on_error`
- `pipe_output` field on all hooks (default: `false`)
- Template variables appropriate to each hook type
- Graceful error handling (hooks never fail the session)

## Breaking Changes

**pre_iteration default changes:** Previously, pre_iteration always piped output to the agent. Now it respects `pipe_output` (default: `false`). Existing configs must add `pipe_output: true` to maintain behavior.

## Lifecycle & Timing

```
Start() called
    |
    v
[NATS, Store, TUI initialized]
    |
    v
Run() called
    |
    v
+--[ session_start hooks ]--+  <-- runs once, output held for first iteration if pipe_output
    |
    v
+==========================+
|   ITERATION LOOP         |
|                          |
|  [ pre_iteration hooks ] |  <-- pending output + hook output piped to prompt
|           |              |
|           v              |
|     Agent executes       |
|           |              |
|           v              |
|  IterationComplete       |
|           |              |
|           v              |
|  [ post_iteration hooks ]|  <-- output held for NEXT iteration if pipe_output
|           |              |
|           v              |
|  Check session_complete  |
|           |              |
+===========|==============+
            |
            v (loop exits: complete, limit, or cancelled)
    |
    v
+--[ FINAL DELIVERY ]------+  <-- if pending piped output exists:
|   Send to agent          |      send to agent and wait for response
|   Wait for response      |      (gives agent chance to fix issues)
+--------------------------+
    |
    v
+--[ session_end hooks ]---+  <-- pipe_output ignored (no more iterations)
    |
    v
Stop() / cleanup
```

**on_task_complete**: Triggered when task status changes to `completed` via NATS event. Output accumulated in pending buffer if `pipe_output: true`.

**on_error**: Triggered on any iteration failure (agent error, network error, panic, etc.). Output sent to agent in immediate recovery attempt if `pipe_output: true`. Session continues to next iteration after recovery.

## Hook Timing Details

| Hook | When | pipe_output behavior |
|------|------|---------------------|
| `session_start` | Once, before first iteration | Output added to pending buffer |
| `pre_iteration` | Before each iteration | Output prepended to iteration prompt (with pending buffer) |
| `post_iteration` | After each iteration completes | Output added to pending buffer for next iteration |
| `session_end` | Once, after final delivery | Ignored (no more iterations) |
| `on_task_complete` | On task status -> completed | Output added to pending buffer |
| `on_error` | On any iteration failure | Output sent in immediate recovery prompt, then continue |

**Pending buffer**: Accumulates piped output from session_start, post_iteration, and on_task_complete in chronological (FIFO) order. Drained at start of each iteration (prepended to pre_iteration output).

**Final delivery**: After loop exits, if pending buffer has content, send to agent and wait for response before running session_end. Gives agent chance to address test failures, etc.

## Config Format

```yaml
version: 1

hooks:
  session_start:
    - command: "git pull --rebase"
      timeout: 30
    - command: "go build ./..."
      timeout: 60
      pipe_output: true  # agent sees build errors

  pre_iteration:
    - command: "golangci-lint run ./..."
      timeout: 30
      pipe_output: true

  post_iteration:
    - command: "go test ./... -short"
      timeout: 120
      pipe_output: true  # agent sees test failures next iteration
    - command: "curl -X POST $SLACK_WEBHOOK -d '{\"text\": \"Iteration {{iteration}} done\"}'"
      timeout: 5
      # pipe_output: false (default) - just notification

  session_end:
    - command: "git push origin HEAD"
      timeout: 30
    - command: "./scripts/notify-complete.sh {{session}}"
      timeout: 10

  on_task_complete:
    - command: "./scripts/validate-task.sh {{task_id}}"
      timeout: 30
      pipe_output: true

  on_error:
    - command: "git diff HEAD"
      timeout: 10
      pipe_output: true  # show agent what changed before error
```

## Template Variables

| Variable | Available in | Description |
|----------|--------------|-------------|
| `{{session}}` | All hooks | Session name |
| `{{iteration}}` | pre_iteration, post_iteration, on_error | Current iteration number |
| `{{task_id}}` | on_task_complete | Completed task ID |
| `{{task_content}}` | on_task_complete | Completed task content |
| `{{error}}` | on_error | Error message |

## Technical Implementation

### types.go Changes

```go
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
    Timeout    int    `yaml:"timeout"`
    PipeOutput bool   `yaml:"pipe_output"` // default false
}
```

### hooks.go Changes

- Add `TaskID`, `TaskContent`, `Error` to `Variables` struct
- Update `expandVariables()` to handle new placeholders

### Orchestrator Changes

1. **Pending output buffer**: Add `pendingHookOutput string` field + `pendingMu sync.Mutex` to hold output for next iteration (mutex needed for on_task_complete NATS callback)

2. **session_start**: Execute after `Run()` starts, before iteration loop. If `pipe_output`, store in `pendingHookOutput`.

3. **pre_iteration**: Prepend `pendingHookOutput` to hook output, clear buffer after use.

4. **post_iteration**: Execute after `IterationComplete`. If `pipe_output`, append to `pendingHookOutput`.

5. **session_end**: Execute after final delivery. Ignore `pipe_output`.

6. **final delivery**: After loop exits, check if `pendingHookOutput` has content. If so, send to agent via `runner.SendMessages()` and wait for response. Then run session_end hooks.

7. **on_task_complete**: Subscribe to NATS `iteratr.events.{session}.task.completed` subject. Execute hooks on event. If `pipe_output`, append to `pendingHookOutput` (FIFO order).

8. **on_error**: Execute in error handling block on any iteration failure (agent error, network error, panic). If `pipe_output`, send immediate recovery prompt to agent, wait for response, then continue to next iteration (don't exit).

### Error Handling

- Hook failure: Log warning, include error in output (existing behavior)
- Hook timeout: Log warning, include partial output (existing behavior)
- Context cancelled: Propagate cancellation
- Never fail the session due to hook errors

### NATS Event Subscription (on_task_complete)

Subscribe to task completion events in `Run()` before iteration loop starts:
```go
sub, _ := nc.Subscribe("iteratr.events."+sessionName+".task.completed", func(msg *nats.Msg) {
    // Parse task ID/content from message
    // Execute on_task_complete hooks with Variables{TaskID, TaskContent}
    // If pipe_output, append to pendingHookOutput (mutex-protected)
})
defer sub.Unsubscribe()
```

Note: Task completions only occur during agent execution (agent calls task tools). No edge cases around session_start/session_end timing.

## Tasks

### Phase 1: Tracer Bullet (post_iteration + pipe_output)

Validates: config parsing, pending output buffer, iteration loop integration.

**Validation test:**
```yaml
hooks:
  post_iteration:
    - command: "echo 'Test output for agent'"
      pipe_output: true
    - command: "echo 'Side effect only'"
```
Expected: Agent sees "Test output for agent" at start of iteration 2, not "Side effect only".

- [ ] 1.1 Add `PostIteration` to `HooksConfig`, `PipeOutput` to `HookConfig`
- [ ] 1.2 Add `ExecuteAllPiped()` - runs all hooks, returns only piped output
- [ ] 1.3 Add `pendingHookOutput` field + helpers to orchestrator
- [ ] 1.4 Execute post_iteration after `IterationComplete`
- [ ] 1.5 Prepend pending output to next iteration's hook output

### Phase 2: session_start + session_end + final delivery

- [ ] 2.1 Add `SessionStart`, `SessionEnd` to `HooksConfig`
- [ ] 2.2 Execute session_start at start of `Run()`, before loop
- [ ] 2.3 session_start: if `pipe_output`, store in pending buffer
- [ ] 2.4 Add final delivery after loop exits (send pending buffer, wait for response)
- [ ] 2.5 Execute session_end after final delivery
- [ ] 2.6 session_end: ignore `pipe_output`

### Phase 3: on_task_complete

- [ ] 3.1 Add `OnTaskComplete` to `HooksConfig`
- [ ] 3.2 Add `TaskID`, `TaskContent` to `Variables`
- [ ] 3.3 Subscribe to NATS task completion events (before loop, with mutex for buffer)
- [ ] 3.4 Execute hooks on event, append piped output to pending buffer (FIFO)

### Phase 4: on_error

- [ ] 4.1 Add `OnError` to `HooksConfig`
- [ ] 4.2 Add `Error` to `Variables`
- [ ] 4.3 Execute on any iteration failure (agent, network, panic)
- [ ] 4.4 If `pipe_output`, send immediate recovery prompt, wait for response
- [ ] 4.5 Continue to next iteration (don't exit session)

### Phase 5: pre_iteration migration

- [ ] 5.1 Update pre_iteration to respect `pipe_output` (default false)
- [ ] 5.2 Update documentation and example config

## Out of Scope

- Conditional hooks (run only if condition met)
- Hook dependencies (run hook B only if hook A succeeds)
- Parallel hook execution
- Per-hook environment variables
- Hook-specific working directories

## Design Decisions

1. **on_error recovery**: Triggers on any iteration failure (agent, network, panic). Send immediate recovery prompt with hook output, wait for response, then continue to next iteration. Session doesn't exit on error.

2. **on_task_complete timing**: Execute hook immediately when task completes. Append piped output to pending buffer (FIFO). Buffer drains at next iteration start. Task completions only happen during agent execution, so no edge cases around session_end.

3. **session_end reliability**: Accept that hooks won't run on crash/SIGKILL. Document limitation; users add external monitoring if critical.

4. **Final delivery**: After loop exits, send any pending piped output to agent and wait for response before session_end. Gives agent chance to address test failures discovered in final post_iteration.

5. **Pending buffer ordering**: All sources (session_start, post_iteration, on_task_complete) append to buffer in chronological order (FIFO).

6. **pre_iteration breaking change**: Default changes to `pipe_output: false`. Document in release notes. Existing configs must add explicit `pipe_output: true`.
