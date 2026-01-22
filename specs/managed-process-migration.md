# Managed Process Migration

Replace ACP-based OpenCode integration with managed process approach using `opencode run`.

## Overview

Migrate from ACP protocol to direct `opencode run --model X --format json` subprocess execution. Enables model selection via `--model` flag (blocked by ACP bug). Tools exposed via prompt instructions - agent calls `iteratr tool <command>` directly via Bash.

## User Story

**As a** developer using iteratr  
**I want** to specify which LLM model to use via `--model` flag  
**So that** I can choose between different providers/models without editing config files

## Requirements

### Functional

1. **Model Selection**
   - `--model` flag on `iteratr build` command
   - Format: `provider/model` (e.g., `anthropic/claude-sonnet-4-5`)
   - Passed directly to `opencode run --model`

2. **Tool CLI Commands**
   - `iteratr tool task-add --port PORT --name SESSION --content TEXT [--status STATUS]`
   - `iteratr tool task-status --port PORT --name SESSION --id ID --status STATUS`
   - `iteratr tool task-list --port PORT --name SESSION`
   - `iteratr tool note-add --port PORT --name SESSION --content TEXT --type TYPE`
   - `iteratr tool note-list --port PORT --name SESSION [--type TYPE]`
   - `iteratr tool inbox-list --port PORT --name SESSION`
   - `iteratr tool inbox-mark-read --port PORT --name SESSION --id ID`
   - `iteratr tool session-complete --port PORT --name SESSION`
   - Port number injected into prompt at runtime (known from NATS startup)

3. **Prompt-Based Tool Instructions**
   - Tools documented in prompt template
   - Agent calls tools via Bash: `iteratr tool task-add --port PORT --name SESSION ...`
   - No TypeScript tool file generation needed
   - Port number and session name injected into prompt at runtime

4. **Managed Process Runner**
   - Spawn `opencode run --model X --format json` per iteration
   - Pipe prompt via stdin, parse JSON events from stdout
   - Handle event types: `text`, `tool_use`, `tool_result`, `error`
   - Detect `session_complete` tool calls via output parsing

5. **Remove ACP/MCP**
   - Delete `internal/acp/` package
   - Delete `internal/mcp/` package
   - Remove HTTP server and random port complexity

### Non-Functional

1. Maintain backwards compatibility for sessions (NATS data unchanged)
2. Process overhead per iteration acceptable (ralph-tui proves viability)
3. Graceful shutdown via SIGTERM → SIGKILL

## Technical Implementation

### Architecture Change

**Before (ACP):**
```
iteratr ──ACP/stdio──> opencode acp
    │
    └──HTTP/SSE──> MCP server (tools)
```

**After (Managed Process):**
```
iteratr ──stdin/stdout──> opencode run --model X --format json
                              │
                              └──Bash──> iteratr tool <cmd>
```

### Package Structure Changes

```
internal/
  acp/           # DELETE
    client.go
    tools.go
  mcp/           # DELETE
    server.go
  agent/         # NEW
    runner.go    # OpenCode subprocess management
    parser.go    # JSON event parsing
cmd/
  iteratr/
    tool.go      # NEW - tool subcommand router
    tool_task.go # NEW - task subcommands
    tool_note.go # NEW - note subcommands  
    tool_inbox.go# NEW - inbox subcommands
    tool_session.go # NEW - session subcommands
```

### Tool CLI Implementation

Each tool command:
1. Takes `--port` flag for NATS connection
2. Connects to NATS via TCP at `127.0.0.1:PORT`
3. Calls session.Store methods
4. Outputs JSON result to stdout

```go
// cmd/iteratr/tool.go
var toolCmd = &cobra.Command{Use: "tool", Short: "Session tools for OpenCode"}

func init() {
    rootCmd.AddCommand(toolCmd)
    toolCmd.PersistentFlags().IntP("port", "p", 0, "NATS port (required)")
    toolCmd.PersistentFlags().StringP("name", "n", "", "Session name (required)")
    toolCmd.AddCommand(taskAddCmd, taskStatusCmd, taskListCmd)
    toolCmd.AddCommand(noteAddCmd, noteListCmd)
    toolCmd.AddCommand(inboxListCmd, inboxMarkReadCmd)
    toolCmd.AddCommand(sessionCompleteCmd)
}

// Shared helper - called by each subcommand
func connectNATS(cmd *cobra.Command) (*nats.Conn, error) {
    port, _ := cmd.Flags().GetInt("port")
    if port == 0 {
        return nil, errors.New("--port is required")
    }
    return nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", port))
}
```

### NATS TCP Listener

Currently NATS uses `DontListen: true`. Need to enable TCP for tool CLI:

```go
// internal/nats/server.go
func StartEmbeddedNATS(dataDir string) (*server.Server, int, error) {
    port := findFreePort() // Find available port
    opts := &server.Options{
        JetStream:  true,
        StoreDir:   dataDir,
        Host:       "127.0.0.1",
        Port:       port,        // Enable TCP
        DontListen: false,       // Allow connections
    }
    // Return port so orchestrator can inject into prompt
    return ns, port, nil
}
```

### Prompt Template Tool Instructions

Tools documented in prompt - agent uses Bash to call them:

```markdown
## Tools

Use Bash to call iteratr tools. All commands require `--port {{port}} --name {{session}}`.

### Task Management
- `iteratr tool task-add --port {{port}} --name {{session}} --content "description" [--status remaining|in_progress|completed|blocked]`
- `iteratr tool task-status --port {{port}} --name {{session}} --id ID --status STATUS`
- `iteratr tool task-list --port {{port}} --name {{session}}`

### Notes
- `iteratr tool note-add --port {{port}} --name {{session}} --content "text" --type learning|stuck|tip|decision`
- `iteratr tool note-list --port {{port}} --name {{session}} [--type TYPE]`

### Inbox
- `iteratr tool inbox-list --port {{port}} --name {{session}}`
- `iteratr tool inbox-mark-read --port {{port}} --name {{session}} --id ID`

### Session Control
- `iteratr tool session-complete --port {{port}} --name {{session}}` - Call when ALL tasks done
```

### Managed Process Runner

```go
// internal/agent/runner.go
type Runner struct {
    model       string
    workDir     string
    sessionName string
    dataDir     string
    onText      func(text string)
    onToolUse   func(name string, input map[string]any)
    onToolResult func(id string, output string)
    onError     func(err error)
}

func (r *Runner) RunIteration(ctx context.Context, prompt string) error {
    args := []string{"run", "--format", "json"}
    if r.model != "" {
        args = append(args, "--model", r.model)
    }
    
    cmd := exec.CommandContext(ctx, "opencode", args...)
    cmd.Dir = r.workDir
    cmd.Env = os.Environ()
    
    stdin, _ := cmd.StdinPipe()
    stdout, _ := cmd.StdoutPipe()
    cmd.Stderr = os.Stderr
    
    if err := cmd.Start(); err != nil {
        return err
    }
    
    // Send prompt
    stdin.Write([]byte(prompt))
    stdin.Close()
    
    // Parse JSON events
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        r.parseEvent(scanner.Text())
    }
    
    return cmd.Wait()
}

func (r *Runner) parseEvent(line string) {
    var event struct {
        Type    string          `json:"type"`
        Content json.RawMessage `json:"content"`
    }
    if err := json.Unmarshal([]byte(line), &event); err != nil {
        return
    }
    
    switch event.Type {
    case "text":
        var text string
        json.Unmarshal(event.Content, &text)
        r.onText(text)
    case "tool_use":
        var tu struct {
            Name  string         `json:"name"`
            Input map[string]any `json:"input"`
        }
        json.Unmarshal(event.Content, &tu)
        r.onToolUse(tu.Name, tu.Input)
    case "tool_result":
        var tr struct {
            ID     string `json:"id"`
            Output string `json:"output"`
        }
        json.Unmarshal(event.Content, &tr)
        r.onToolResult(tr.ID, tr.Output)
    case "error":
        var errMsg string
        json.Unmarshal(event.Content, &errMsg)
        r.onError(errors.New(errMsg))
    }
}
```

### Orchestrator Changes

```go
// internal/orchestrator/orchestrator.go

type Config struct {
    // ... existing fields
    Model string // NEW
}

func (o *Orchestrator) Start(ctx context.Context) error {
    runner := agent.NewRunner(agent.RunnerConfig{
        Model:       o.config.Model,
        WorkDir:     o.workDir,
        SessionName: o.session.Name,
        DataDir:     o.dataDir,
        OnText:      o.handleText,
        OnToolUse:   o.handleToolUse,
    })
    
    for i := 1; o.config.Iterations == 0 || i <= o.config.Iterations; i++ {
        prompt := o.buildPrompt(i)
        if err := runner.RunIteration(ctx, prompt); err != nil {
            return err
        }
        if o.session.IsComplete() {
            break
        }
    }
    return nil
}
```

### CLI Flag Addition

```go
// cmd/iteratr/build.go
var buildCmd = &cobra.Command{
    Use:   "build",
    Short: "Run agent iteration loop",
    RunE:  runBuild,
}

func init() {
    buildCmd.Flags().StringP("model", "m", "", "Model to use (e.g., anthropic/claude-sonnet-4-5)")
    // ... existing flags
}

func runBuild(cmd *cobra.Command, args []string) error {
    model, _ := cmd.Flags().GetString("model")
    // Pass to orchestrator config
}
```

## Tasks

### 1. Enable NATS TCP Listener
- [ ] Modify `internal/nats/server.go` to listen on localhost TCP port
- [ ] Update `StartEmbeddedNATS()` to return port number
- [ ] Add `findFreePort()` helper function

### 2. Create Tool CLI Infrastructure
- [ ] Create `cmd/iteratr/tool.go` with tool subcommand and `connectNATS()` helper
- [ ] Create `cmd/iteratr/tool_task.go` with task-add, task-status, task-list commands
- [ ] Create `cmd/iteratr/tool_note.go` with note-add, note-list commands
- [ ] Create `cmd/iteratr/tool_inbox.go` with inbox-list, inbox-mark-read commands
- [ ] Create `cmd/iteratr/tool_session.go` with session-complete command

### 3. Implement Managed Process Runner
- [ ] Create `internal/agent/runner.go` with Runner struct and RunIteration method
- [ ] Create `internal/agent/parser.go` with JSON event parsing logic

### 4. Update Orchestrator
- [ ] Add `Model` field to orchestrator Config struct
- [ ] Store NATS port from startup, inject into prompt template
- [ ] Remove MCP server creation and management
- [ ] Replace ACP client with agent Runner
- [ ] Update prompt template with `{{port}}` variable

### 5. Add CLI Model Flag
- [ ] Add `--model` / `-m` flag to build command
- [ ] Pass model through to orchestrator Config
- [ ] Add help text with example formats

### 6. Remove ACP/MCP Code
- [ ] Delete `internal/acp/client.go`
- [ ] Delete `internal/acp/tools.go`
- [ ] Delete `internal/mcp/server.go`
- [ ] Remove `github.com/coder/acp-go-sdk` from go.mod
- [ ] Remove `github.com/mark3labs/mcp-go` from go.mod (if unused elsewhere)
- [ ] Update imports across codebase

### 7. Testing
- [ ] Test tool CLI commands work standalone with running NATS
- [ ] Test agent runner parses OpenCode JSON events correctly
- [ ] Test model selection via `--model` flag
- [ ] Test complete iteration loop with prompt-based tools
- [ ] Test session_complete detection ends loop
- [ ] Test headless and TUI modes still work

## UI Mockup

N/A - No UI changes, only CLI additions.

## Out of Scope

- Multi-model support within single session
- Model validation against provider
- Caching/reusing OpenCode process across iterations
- Windows-specific process handling
- TypeScript tool file generation

## Open Questions

1. Should we support `ITERATR_MODEL` env var as alternative to `--model` flag?
   - **Defer**: Add later if requested

2. How to detect `session_complete` without native tool hooks?
   - **Answer**: Check session state in NATS after each iteration completes

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Tool CLI can't connect to NATS | High | Port injected at runtime, clear error messages |
| OpenCode JSON format changes | Medium | Version check, fallback parsing |
| Process overhead per iteration | Low | Acceptable per ralph-tui validation |
| Agent doesn't follow prompt tool instructions | Medium | Clear formatting, test with different models |

## References

- ralph-tui OpenCode plugin: https://github.com/subsy/ralph-tui/blob/main/src/plugins/agents/builtin/opencode.ts
- ACP bug: OpenCode uses `setSessionModel` but SDK expects `unstable_setSessionModel`
