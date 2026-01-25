# User Task Creation Modal

Ctrl+T opens a modal with text input and priority selector, allowing users to create tasks persisted via the existing NATS event mechanism.

## Overview

Add an interactive modal triggered by ctrl+t containing a textarea for task content and a priority selector (critical/high/medium/low/backlog). On submit, the task is published to NATS JetStream via `Store.TaskAdd()`, associating it with the current iteration. Modal closes on submit; state propagation handles sidebar updates.

## User Story

**As a** developer observing an agent session  
**I want** to quickly add tasks to the session  
**So that** I can inject work items for the agent to process or track manual tasks

## Requirements

### Functional

1. **Trigger**: Ctrl+T opens modal (blocked when another modal/dialog is visible)
2. **Priority Selector**: Cycle through critical/high/medium/low/backlog (left/right arrows when focused)
3. **Content Input**: Multi-line textarea with word wrap
4. **Submit Button**: Clickable button; also activated via Enter/Space when focused
5. **Submit Shortcut**: Ctrl+Enter submits from any focus zone
6. **Focus Cycling**: Tab/Shift+Tab cycles focus: priority selector -> textarea -> submit button
7. **Cancel**: ESC closes modal without saving, clears input state
8. **Validation**: Submit blocked if content is empty (button visually dimmed)

### Non-Functional

1. Modal blocks all other keyboard input while open
2. No flicker on open/close
3. Textarea supports at least 500 characters

## Technical Implementation

### Architecture

```
Ctrl+T keypress
└── App.handleGlobalKeys()
    └── Opens TaskInputModal (if no other modal visible)
        ├── Tab/Shift+Tab cycles focus: PrioritySelector -> TextArea -> SubmitButton
        ├── PrioritySelector (left/right cycles: critical -> high -> medium -> low -> backlog)
        ├── TextArea (Bubbles textarea component)
        └── Submit (ctrl+enter from any zone, Enter/Space on button, or click button)
            └── CreateTaskMsg -> App.Update()
                └── Store.TaskAdd(content, priority, iteration)
                    └── NATS event published
                        └── State rebuilt -> sidebar updates
```

### Reference: Existing Store.TaskAdd API

```go
// From internal/session/task.go:14-18
type TaskAddParams struct {
    Content   string `json:"content"`
    Status    string `json:"status,omitempty"` // Optional: remaining, in_progress, completed, blocked
    Iteration int    `json:"iteration"`
}

// From internal/session/task.go:35-97
func (s *Store) TaskAdd(ctx context.Context, session string, params TaskAddParams) (*Task, error)
```

Note: `TaskAddParams` has no `Priority` field. Priority defaults to 2 (medium) via the event reducer at `session.go:156-167`. To support priority selection:

**Option A**: Add `Priority int` field to `TaskAddParams` and include in event metadata
**Option B**: Call `Store.TaskPriority()` after `TaskAdd()` to set priority

Option A is cleaner - extend `TaskAddParams`:

```go
// Modified TaskAddParams
type TaskAddParams struct {
    Content   string `json:"content"`
    Status    string `json:"status,omitempty"`
    Priority  int    `json:"priority,omitempty"` // NEW: 0-4 (critical to backlog)
    Iteration int    `json:"iteration"`
}
```

And update `TaskAdd()` to include priority in metadata:

```go
meta, _ := json.Marshal(map[string]any{
    "status":    status,
    "priority":  params.Priority, // NEW
    "iteration": params.Iteration,
})
```

### Reference: NoteInputModal Pattern (to mirror)

```go
// From internal/tui/note_input_modal.go:14-20
type focusZone int

const (
    focusTypeSelector focusZone = iota
    focusTextarea
    focusSubmitButton
)

// From internal/tui/note_input_modal.go:22-37
type NoteInputModal struct {
    visible    bool
    textarea   textarea.Model
    noteType   string
    types      []string
    typeIndex  int
    focus      focusZone
    width      int
    height     int
    buttonArea uv.Rectangle
}
```

### New Component

```go
// internal/tui/task_input_modal.go

type focusZone int

const (
    focusPrioritySelector focusZone = iota
    focusTextarea
    focusSubmitButton
)

// Priority levels matching session.Task
var priorities = []struct {
    value int
    label string
    emoji string
}{
    {0, "critical", "🔴"},
    {1, "high", "🟠"},
    {2, "medium", "🟡"},
    {3, "low", "🟢"},
    {4, "backlog", "⚪"},
}

type TaskInputModal struct {
    visible       bool
    textarea      textarea.Model  // Bubbles v2 textarea
    priorityIndex int             // Current selected priority (0-4)
    focus         focusZone       // Which zone has keyboard focus
    width         int
    height        int
    buttonArea    uv.Rectangle    // Hit area for mouse click on submit button
}

func NewTaskInputModal() *TaskInputModal {
    ta := textarea.New()
    ta.SetWidth(50)
    ta.SetHeight(6)
    ta.Placeholder = "Describe the task..."
    ta.CharLimit = 500
    ta.ShowLineNumbers = false
    ta.Prompt = ""
    // Remove ctrl+t from LineNext to avoid conflict
    ta.KeyMap.LineNext = key.NewBinding(key.WithKeys("down"))
    
    // Style textarea to match modal theme (same as NoteInputModal)
    styles := textarea.DefaultDarkStyles()
    styles.Cursor.Color = lipgloss.Color(colorSecondary)
    styles.Cursor.Shape = tea.CursorBlock
    styles.Cursor.Blink = true
    ta.SetStyles(styles)
    
    return &TaskInputModal{
        textarea:      ta,
        priorityIndex: 2, // Default to medium
        focus:         focusTextarea,
        width:         60,
        height:        18, // Slightly taller than note modal to fit priority row
    }
}

func (m *TaskInputModal) IsVisible() bool {
    return m.visible
}

func (m *TaskInputModal) Show() tea.Cmd {
    m.visible = true
    m.focus = focusTextarea
    return m.textarea.Focus()
}

func (m *TaskInputModal) Close() {
    m.visible = false
    m.textarea.SetValue("")
    m.priorityIndex = 2 // Reset to medium
    m.focus = focusTextarea
    m.textarea.Blur()
}
```

### Messages

```go
// In internal/tui/app.go (alongside CreateNoteMsg)

type CreateTaskMsg struct {
    Content   string
    Priority  int
    Iteration int // Filled by App, not modal
}
```

### Key Bindings (inside modal)

| Key | Context | Action |
|-----|---------|--------|
| Tab | Any | Cycle focus: priority selector -> textarea -> submit button |
| Shift+Tab | Any | Cycle focus backward |
| Left / Right | Priority selector focused | Cycle priority level |
| Enter / Space | Submit button focused | Submit task |
| Ctrl+Enter | Any focus | Submit task (shortcut) |
| ESC | Any | Cancel and close |
| All other keys | Textarea focused | Forwarded to textarea |

### App Integration

Additions to `internal/tui/app.go`:

```go
// Add field to App struct (near noteInputModal)
type App struct {
    // ...
    noteInputModal *NoteInputModal
    taskInputModal *TaskInputModal  // NEW
    // ...
}

// Add to NewApp()
func NewApp(...) *App {
    return &App{
        // ...
        noteInputModal: NewNoteInputModal(),
        taskInputModal: NewTaskInputModal(),  // NEW
        // ...
    }
}

// Add to handleGlobalKeys() (after ctrl+n handling)
case "ctrl+t":
    // Guard: no modal/dialog visible (mirrors ctrl+n guard at line 490)
    if a.dialog.IsVisible() || a.taskModal.IsVisible() || a.noteModal.IsVisible() ||
       a.noteInputModal.IsVisible() || a.logsVisible {
        return nil
    }
    // Guard: must have active iteration
    if a.iteration == 0 {
        return nil
    }
    return a.taskInputModal.Show()

// Also update ctrl+n guard to include taskInputModal:
// Change line 490 from:
//   if a.dialog.IsVisible() || a.taskModal.IsVisible() || a.noteModal.IsVisible() || a.logsVisible {
// To:
//   if a.dialog.IsVisible() || a.taskModal.IsVisible() || a.noteModal.IsVisible() || 
//      a.taskInputModal.IsVisible() || a.logsVisible {

// Add priority check in handleKeyPress() (after noteInputModal)
if m.taskInputModal.IsVisible() {
    return m.taskInputModal.Update(msg)
}

// Add CreateTaskMsg handler in Update()
case CreateTaskMsg:
    msg.Iteration = m.iteration
    go func() {
        _, err := m.store.TaskAdd(context.Background(), m.session, session.TaskAddParams{
            Content:   msg.Content,
            Priority:  msg.Priority,
            Iteration: msg.Iteration,
        })
        if err != nil {
            logger.Error("Failed to add task: %v", err)
        }
    }()
    m.taskInputModal.Close()

// Add to Draw() (after noteInputModal, before dialog)
if m.taskInputModal.IsVisible() {
    m.taskInputModal.Draw(scr, area)
}
```

### Update Store.TaskAdd for Priority Support

```go
// internal/session/task.go - modify TaskAddParams
type TaskAddParams struct {
    Content   string `json:"content"`
    Status    string `json:"status,omitempty"`
    Priority  int    `json:"priority,omitempty"` // NEW: 0=critical, 1=high, 2=medium, 3=low, 4=backlog
    Iteration int    `json:"iteration"`
}

// internal/session/task.go - modify TaskAdd metadata creation (line 65-68)
meta, _ := json.Marshal(map[string]any{
    "status":    status,
    "priority":  params.Priority, // NEW - include priority in event
    "iteration": params.Iteration,
})
```

## Tasks

### 1. Tracer bullet: minimal end-to-end

- [ ] Add `Priority int` field to `TaskAddParams` in `internal/session/task.go`
- [ ] Update `TaskAdd()` to include priority in event metadata
- [ ] Create `internal/tui/task_input_modal.go` with struct, `New()`, `IsVisible()`, `Show()`, `Close()`
- [ ] Add Bubbles textarea, hardcode priority to 2 (medium), focus starts on textarea
- [ ] Add submit button rendering (static text for now, no focus/click yet)
- [ ] Add `taskInputModal *TaskInputModal` field to App and initialize in `NewApp()`
- [ ] Wire ctrl+t in `handleGlobalKeys()` to call `Show()` and return `textarea.Focus()` cmd
- [ ] Handle ctrl+enter to emit `CreateTaskMsg` and close modal
- [ ] Handle ESC to close without saving
- [ ] Add `CreateTaskMsg` handler in `App.Update()` that calls `Store.TaskAdd()`
- [ ] Add `Draw()` method with `styleModalContainer`, render in `App.Draw()`
- [ ] Add priority routing in `handleKeyPress()` to forward keys to modal when visible

### 2. Focus cycling and submit button

- [ ] Add `focusZone` type and `focus` field to struct
- [ ] Implement Tab/Shift+Tab to cycle focus: priority selector -> textarea -> button
- [ ] Blur textarea when focus leaves it, re-focus when it returns
- [ ] Render button with focused/unfocused/disabled styles
- [ ] Handle Enter/Space on button focus to submit
- [ ] Store `buttonArea uv.Rectangle` during Draw for click hit detection
- [ ] Handle mouse click on button area to submit

### 3. Priority selector

- [ ] Add `priorities` slice and `priorityIndex int` field
- [ ] Render priority badges row above textarea (highlight active priority with badge style)
- [ ] Handle Left/Right arrows when priority selector is focused to cycle `priorityIndex`
- [ ] Pass selected priority into `CreateTaskMsg`
- [ ] Style priority badges with appropriate colors (red/orange/yellow/green/gray)

### 4. Polish textarea

- [ ] Configure textarea: multi-line, word wrap, character limit, placeholder text
- [ ] Override textarea KeyMap: remove ctrl+t from LineNext (use only down arrow)
- [ ] Size textarea to fill modal content area (responsive to terminal size)
- [ ] Style textarea to match modal theme (background, cursor color)

### 5. Validation and UX

- [ ] Block submit when content is whitespace-only (dim button, ignore ctrl+enter)
- [ ] Clear textarea and reset priority/focus on close (both cancel and submit)
- [ ] Show hint bar at modal bottom: tab/ctrl+enter/esc

### 6. Guard and edge cases

- [ ] Guard ctrl+t: no-op if dialog, task modal, note modal, note input, task input, or log viewer visible
- [ ] Guard ctrl+t: no-op if session has no active iteration (no iteration to tag)
- [ ] Handle terminal resize while modal is open (recalculate dimensions)

## UI Mockup

```
╭─ New Task ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╮
│                                                  │
│  Priority: critical  high [medium] low  backlog  │
│             🔴       🟠    🟡     🟢     ⚪      │
│                                                  │
│  ┌────────────────────────────────────────────┐  │
│  │ Implement caching layer for API responses  │  │
│  │ to reduce latency on repeated calls.       │  │
│  │                                            │  │
│  │                                            │  │
│  │                                            │  │
│  └────────────────────────────────────────────┘  │
│                                                  │
│                                    [ Add Task ]  │
│                                                  │
│  tab cycle focus · ctrl+enter submit · esc close │
╰──────────────────────────────────────────────────╯
```

Button states:
- **Focused**: `[ Add Task ]` with highlighted border/background
- **Unfocused**: `  Add Task  ` dimmed
- **Disabled** (empty content): `  Add Task  ` muted, non-interactive

## Gotchas

### 1. ctrl+enter requires keyboard enhancements (ALREADY ENABLED)

The app already has `KeyboardEnhancements` enabled at `app.go:511`:

```go
view.KeyboardEnhancements = tea.KeyboardEnhancements{
    ReportEventTypes: true,
}
```

No changes needed - ctrl+enter will work out of the box.

### 2. TaskAddParams lacks Priority field

The current `TaskAddParams` struct doesn't include a `Priority` field. The priority defaults to 2 (medium) via the event reducer logic at `session.go:156-167`. 

**Fix**: Add `Priority int` to `TaskAddParams` and update `TaskAdd()` to include it in the event metadata. The reducer already handles priority from metadata.

### 3. handleGlobalKeys() return value pattern

Same as notes: `handleGlobalKeys()` must return a non-nil cmd (the `textarea.Focus()` cmd) to prevent the keypress from falling through to `dashboard.Update()`.

### 4. Sidebar updates are automatic

Same as notes: After `Store.TaskAdd()` publishes to NATS, the existing event subscription picks it up -> triggers `loadInitialState()` -> sends `StateUpdateMsg` -> calls `sidebar.SetState()`. No manual sidebar refresh needed.

### 5. Task created with status "remaining"

`Store.TaskAdd()` defaults status to "remaining" if not specified. This is correct behavior - new user tasks should start as "remaining" to be picked up by the agent or iteration.

### 6. Existing TaskModal is for viewing, not creating

`TaskModal` exists in `internal/tui/modal.go` - it's for **viewing** task details (like `NoteModal`). The new `TaskInputModal` is for **creating** tasks (like `NoteInputModal`). Naming is consistent:

| Component | Purpose | File |
|-----------|---------|------|
| `TaskModal` | View task details | `modal.go` (exists) |
| `TaskInputModal` | Create new task | `task_input_modal.go` (new) |
| `NoteModal` | View note details | `note_modal.go` (exists) |
| `NoteInputModal` | Create new note | `note_input_modal.go` (exists) |

No naming conflict - proceed with `TaskInputModal`.

### 7. Priority int vs Priority name

The UI shows priority names (critical/high/medium/low/backlog) but the API uses integers (0-4). Map correctly:

| Display | Value |
|---------|-------|
| critical | 0 |
| high | 1 |
| medium | 2 |
| low | 3 |
| backlog | 4 |

### 8. Focus zone naming

NoteInputModal uses `focusTypeSelector` for the note type selector. TaskInputModal should use `focusPrioritySelector` for clarity. Don't reuse the same constant names across files to avoid confusion.

### 9. Update ctrl+n guard

When adding ctrl+t, also update the ctrl+n guard at `app.go:490` to include `a.taskInputModal.IsVisible()`. Otherwise ctrl+n could open while the task input modal is visible.

### 10. Badge styles exist and can be reused

Priority badge styles already exist in `styles.go` and are used by `TaskModal.renderPriorityBadge()` in `modal.go:220-246`. Reuse the same pattern:
- critical (0): `styleBadgeError`
- high (1): `styleBadgeWarning`  
- medium (2): `styleBadgeInfo`
- low (3): `styleBadgeMuted`
- backlog (4): `styleBadgeMuted.Faint(true)`

## Out of Scope

- Edit existing tasks
- Delete tasks
- Task dependencies from modal
- Batch task creation
- Task search/filter from modal
- Status selection (always creates as "remaining")
- Due dates
- Task assignments
