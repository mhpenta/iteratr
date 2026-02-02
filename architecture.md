# Iteratr Architecture

## Overview

**iteratr** is a TUI-based orchestration tool that manages AI coding agents (via opencode) in iterative development loops. It enables autonomous or semi-autonomous software development by:

- Running AI agents in persistent sessions across multiple iterations
- Managing tasks, notes, and session state through event sourcing
- Providing real-time monitoring via a full-screen terminal UI
- Persisting all data in embedded NATS JetStream
- Allowing user intervention through TUI input or lifecycle hooks

## Architectural Layers

```
┌─────────────────────────────────────────────────────┐
│              CLI Layer (Cobra)                      │
│  Commands: build, setup, config, doctor, tool      │
└──────────────────┬──────────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────────┐
│          Orchestrator (Core Controller)             │
│  - Manages iteration loop lifecycle                 │
│  - Coordinates all components                       │
│  - Handles pause/resume/context cancellation        │
└─────┬──────────┬──────────┬──────────┬──────────┬───┘
      │          │          │          │          │
┌─────▼────┐ ┌──▼────┐ ┌───▼────┐ ┌───▼────┐ ┌───▼────┐
│   TUI    │ │ Agent │ │  NATS  │ │  MCP   │ │ Hooks  │
│(Bubbletea│ │Runner │ │JetStrm │ │Server  │ │ System │
│   v2)    │ │ (ACP) │ │(Store) │ │(Tools) │ │        │
└──────────┘ └───────┘ └────────┘ └────────┘ └────────┘
      │          │          │          │          │
┌─────▼──────────▼──────────▼──────────▼──────────▼───┐
│           Session Store (Event Sourcing)            │
│  - Tasks, Notes, Iterations, Control Events         │
│  - Event-driven state reconstruction                │
└─────────────────────────────────────────────────────┘
```

## Core Packages

| Package | Path | Responsibility |
|---------|------|----------------|
| orchestrator | `internal/orchestrator/` | Core iteration loop, component coordination |
| agent | `internal/agent/` | ACP protocol, subprocess management, file tracking |
| session | `internal/session/` | Event sourcing, state reconstruction |
| tui | `internal/tui/` | Bubbletea v2 TUI with Ultraviolet layouts |
| mcpserver | `internal/mcpserver/` | Embedded HTTP server for MCP tools |
| nats | `internal/nats/` | Embedded NATS JetStream server |
| config | `internal/config/` | Viper-based configuration management |
| hooks | `internal/hooks/` | Lifecycle hooks execution |
| template | `internal/template/` | Prompt template engine |
| logger | `internal/logger/` | Structured file + stderr logging |
| errors | `internal/errors/` | Error types, panic recovery, retry logic |

## Bootstrap Flow

Entry: `cmd/iteratr/build.go:runBuild()`

1. **Config Loading** - Viper loads from files + ENV vars + CLI flags
2. **Orchestrator Creation** - Creates orchestrator with config
3. **NATS Setup** - Connect to existing or start embedded server
4. **JetStream Setup** - Create stream "iteratr_events"
5. **MCP Server Start** - Start HTTP server on random port
6. **Session State Check** - Check if session complete, prompt for restart
7. **TUI Initialization** - Create Bubbletea app (if not headless)
8. **Hooks Loading** - Load `.iteratr.hooks.yml` if present
9. **Agent Runner Creation** - Create runner with callbacks

## Iteration Loop

Location: `internal/orchestrator/orchestrator.go`

```
1. Check context cancellation / iteration limit
2. Clear file tracker
3. Log iteration start (NATS event)
4. Send IterationStartMsg to TUI
5. Execute pre_iteration hooks
6. Build prompt with current state (template.BuildPrompt)
7. Run agent iteration with panic recovery
   - runner.RunIteration(ctx, prompt, hookOutput)
   - Agent callbacks update TUI in real-time
8. Log iteration complete (NATS event)
9. Execute post_iteration hooks
10. Run auto-commit if enabled and files modified
11. Check if session marked complete
12. Process queued user messages
13. Check if paused - block until resume
14. Increment iteration count
15. GOTO 1
```

## TUI Architecture

See `component-tree.md` for the full component tree.

### Stack

- **Bubbletea v2** - Core TUI framework (model-update-view)
- **Ultraviolet** - Rectangle-based layout management
- **Lipgloss v2** - Styling and content composition
- **Bubbles v2** - Pre-built components (viewport, textinput, spinner)
- **Glamour v2** - Markdown rendering

### Component Hierarchy

```
App (root)
├── StatusBar - Session info, git status, duration, pause state
├── Dashboard - Main content area with focus management
│   └── AgentOutput - Conversation display with ScrollList + textinput
├── Sidebar - Tasks/notes lists with logo
└── Modals (overlay, priority-based)
    ├── Dialog - Simple confirmation
    ├── LogViewer - Event history
    ├── TaskModal - Task detail viewer
    ├── NoteModal - Note detail viewer
    ├── TaskInputModal - Task creation
    ├── NoteInputModal - Note creation
    └── SubagentModal - Subagent session replay
```

### Layout Modes

| Mode | Condition | Layout |
|------|-----------|--------|
| Desktop | width >= 100, height >= 25 | Status (1 row) \| Main (flex) \| Sidebar (45 cols) |
| Compact | below thresholds | Status (1 row) \| Main (full), sidebar toggles as overlay |

### Keyboard Routing Priority

1. Dialog visible - Dialog.Update()
2. Global keys (ctrl+c quit)
3. Prefix mode (ctrl+x + key)
4. Modal overlays (ESC closes)
5. Dashboard focus routing

## Event Sourcing

All session state is stored as append-only events in NATS JetStream.

### Event Structure

```go
type Event struct {
    ID        string          // NATS sequence ID
    Timestamp time.Time
    Session   string          // Session name
    Type      string          // task, note, iteration, control
    Action    string          // add, status, priority, etc.
    Meta      json.RawMessage // Action-specific metadata
    Data      string          // Primary content
}
```

### Subject Pattern

- `iteratr.{session}.task` - Task events
- `iteratr.{session}.note` - Note events
- `iteratr.{session}.iteration` - Iteration events
- `iteratr.{session}.control` - Session control events

### State Reconstruction

```go
// Load all events for session, apply in order
state := &State{Tasks: map[string]*Task{}}
for event := range events {
    state.Apply(event)  // Reducer pattern
}
```

## Agent Integration

### ACP Protocol

Communication with opencode via JSON-RPC 2.0 over stdio.

Location: `internal/agent/acp.go`

### Runner Lifecycle

```go
// 1. Start subprocess (once per session)
runner.Start(ctx)  // Spawns: opencode acp

// 2. Per-iteration (fresh ACP session each time)
runner.RunIteration(ctx, prompt, hookOutput)

// 3. Cleanup
runner.Stop()
```

### Callbacks

| Callback | Purpose |
|----------|---------|
| `OnText(string)` | Assistant text output |
| `OnToolCall(ToolCallEvent)` | Tool execution lifecycle |
| `OnThinking(string)` | Reasoning content |
| `OnFinish(FinishEvent)` | Iteration complete |
| `OnFileChange(FileChange)` | File modifications for auto-commit |

## MCP Tools Server

Embedded HTTP server exposes tools to the AI agent.

Location: `internal/mcpserver/`

### Available Tools

| Tool | Purpose |
|------|---------|
| `task-add` | Create task |
| `task-batch-add` | Create multiple tasks |
| `task-status` | Update task status |
| `task-priority` | Set task priority |
| `task-depends` | Add dependency |
| `task-list` | List tasks by status |
| `task-next` | Get next unblocked task |
| `note-add` | Record note |
| `note-list` | List notes |
| `iteration-summary` | Record iteration summary |
| `session-complete` | Mark session complete |

## Configuration

Viper-based with layered precedence: CLI flags > ENV vars > project config > global config > defaults

### Config Locations

- Global: `~/.config/iteratr/iteratr.yml`
- Project: `./iteratr.yml`

### Options

| Key | ENV Var | Default | Description |
|-----|---------|---------|-------------|
| `model` | `ITERATR_MODEL` | (required) | LLM model ID |
| `auto_commit` | `ITERATR_AUTO_COMMIT` | `true` | Auto-commit files |
| `data_dir` | `ITERATR_DATA_DIR` | `.iteratr` | Data directory |
| `log_level` | `ITERATR_LOG_LEVEL` | `info` | Log level |
| `log_file` | `ITERATR_LOG_FILE` | `""` | Log file path |
| `iterations` | `ITERATR_ITERATIONS` | `0` | Max iterations (0=infinite) |
| `headless` | `ITERATR_HEADLESS` | `false` | No TUI mode |
| `template` | `ITERATR_TEMPLATE` | `""` | Template path |

## Design Patterns

| Pattern | Usage |
|---------|-------|
| Event Sourcing | Session state as append-only events |
| Observer | NATS pub/sub for async communication |
| Command | Bubbletea messages encapsulate actions |
| MVC | Model/Update/View in Bubbletea TUI |
| Callback | Agent runner streams events |
| Repository | Session store abstracts data access |
| Factory | Component creation (NewRunner, NewApp) |

## Directory Structure

```
cmd/iteratr/           # CLI commands (entry point)
internal/
├── agent/            # Agent subprocess management
├── config/           # Viper config
├── errors/           # Error utilities
├── git/              # Git info
├── hooks/            # Lifecycle hooks
├── logger/           # Logging
├── mcpserver/        # MCP tools server
├── nats/             # Embedded NATS
├── orchestrator/     # Core controller
├── session/          # Event sourcing
├── template/         # Prompt templates
└── tui/              # Bubbletea UI
    ├── theme/        # Color themes
    └── wizard/       # Setup wizard
specs/                # Feature specifications
.iteratr/             # Runtime data (gitignored)
├── data/             # NATS JetStream storage
└── server.port       # Server port file
```

## Runtime Data

- `.iteratr/data/jetstream/` - NATS event stream storage
- `.iteratr/data/server.port` - NATS server port number
- `.iteratr.hooks.yml` - Lifecycle hooks configuration
- `iteratr.yml` - Project config
