# Sidebar Toggle

## Overview

`ctrl+x b` toggles sidebar (Tasks/Notes) visibility. Messages panel expands to fill space when hidden.

## User Story

As a user with limited terminal width, I want to hide the sidebar to maximize message viewing area, and quickly restore it when needed.

## Requirements

### Toggle Behavior
- `ctrl+x b` keybind (replaces non-working `ctrl+x s`)
- Global keybind - works from any focused component
- Instant toggle (no animation)
- No visible indicator when hidden - full space to messages panel
- Messages panel dynamically expands/shrinks with sidebar visibility

### Focus Management
- If focus in sidebar when hiding: move focus to messages panel
- Focus on messages panel preserved when showing sidebar

### State Restoration
- When re-shown: restore exact scroll position, tab selection, selected item
- Sidebar component stays alive in memory while hidden (no re-init)

### Persistence
- Store in `.iteratr/ui-state.json` (structured for future expansion)
- Format: `{"sidebar": {"visible": true}}`
- Carry across sessions (user preference, not session-scoped)
- Last-write-wins acceptable for concurrent access

### Responsive Behavior
- Narrow terminals (< 100 chars): always force-hide sidebar (existing compact mode)
- Wide terminals (>= 100 chars): respect user's manual toggle choice
- If user manually hid on wide terminal, keep hidden when resizing
- If auto-hidden due to narrow terminal without manual toggle, auto-restore when widening past threshold

### Discoverability
- Status bar hint: show `C-x b: sidebar` only when sidebar hidden

## Technical Implementation

### Files to Modify

**internal/tui/app.go**
- Add `sidebarUserHidden bool` field to track manual toggle vs auto-hide
- Modify `ctrl+x s` handler to `ctrl+x b`, set `sidebarUserHidden` flag
- Update `handleSidebarToggle()` to persist state
- In `WindowSizeMsg` handler: check user preference before auto-showing
- Pass `sidebarVisible` state to status bar

**internal/tui/layout.go**
- Modify `CalculateLayout()` to accept `sidebarHidden bool` param
- When hidden in desktop mode: return empty sidebar rect, expand main

**internal/tui/statusbar.go**
- Add `sidebarHidden bool` field
- Render `C-x b: sidebar` hint when hidden

**internal/state/ui_state.go** (new file)
- Define `UIState` struct: `type UIState struct { Sidebar SidebarState }`
- Define `SidebarState`: `type SidebarState struct { Visible bool }`
- `Load()`: read from `.iteratr/ui-state.json`, return defaults if missing
- `Save()`: write JSON to `.iteratr/ui-state.json`

### Data Flow

```
User presses ctrl+x b
    ↓
App.Update() handles keypress
    ↓
Toggle sidebarVisible, set sidebarUserHidden = true
    ↓
Persist to ui-state.json
    ↓
If hiding && sidebar focused: move focus to messages
    ↓
Mark layoutDirty = true
    ↓
App.View() recalculates layout with hidden flag
    ↓
Messages panel expands to fill sidebar space
```

### State Machine

```
States: Visible, UserHidden, AutoHidden

Transitions:
  Visible + ctrl+x b      → UserHidden
  Visible + width < 100   → AutoHidden
  UserHidden + ctrl+x b   → Visible
  UserHidden + width < 100 → UserHidden (stays hidden)
  AutoHidden + ctrl+x b   → Visible (clears auto flag)
  AutoHidden + width >= 100 → Visible (if not UserHidden)
```

## Tasks

### 1. UI State Persistence
- [ ] Create `internal/state/ui_state.go` with Load/Save functions
- [ ] Add `UIState` and `SidebarState` structs
- [ ] Write to `.iteratr/ui-state.json` on toggle
- [ ] Load on app startup

### 2. Keybind and Toggle Logic
- [ ] Replace `ctrl+x s` with `ctrl+x b` in App.Update()
- [ ] Add `sidebarUserHidden` field to App struct
- [ ] Update toggle handler to set user flag and persist
- [ ] Move focus to messages panel when hiding focused sidebar

### 3. Layout Modification
- [ ] Add `sidebarHidden` param to `CalculateLayout()`
- [ ] When hidden in desktop mode: expand main rect, empty sidebar rect
- [ ] Update `propagateSizes()` to skip sidebar when hidden
- [ ] Update layout tests for hidden state

### 4. Responsive Behavior
- [ ] In `WindowSizeMsg`: check `sidebarUserHidden` before auto-showing
- [ ] When narrowing below threshold: set auto-hidden (not user-hidden)
- [ ] When widening past threshold: restore only if not user-hidden

### 5. Status Bar Hint
- [ ] Add `SetSidebarHidden(bool)` method to StatusBar
- [ ] Render `C-x b: sidebar` hint on right side when hidden
- [ ] Call from App when toggling sidebar

## UI Mockup

**Desktop mode with sidebar visible (current)**
```
┌─────────────────────────────┬───────────────┐
│                             │   ▲ Tasks     │
│   Agent Messages            │   ○ Task 1    │
│                             │   ○ Task 2    │
│                             ├───────────────┤
│                             │   ▼ Notes     │
│                             │   Note 1      │
├─────────────────────────────┴───────────────┤
│ Status: Running  Model: opus               │
└─────────────────────────────────────────────┘
```

**Desktop mode with sidebar hidden (new)**
```
┌─────────────────────────────────────────────┐
│                                             │
│   Agent Messages                            │
│   (full width)                              │
│                                             │
│                                             │
│                                             │
├─────────────────────────────────────────────┤
│ Status: Running  Model: opus   C-x b: sidebar│
└─────────────────────────────────────────────┘
```

## Out of Scope

- Sidebar resize/drag functionality
- Multiple sidebar positions (left vs right)
- Sidebar width persistence
- Keyboard shortcut customization
- Partial collapse (icon-only rail)

## Open Questions

- None currently
