# Theme System Refactoring

## Overview

Refactor color/style management from scattered definitions to a centralized theme package. Standardize on lipgloss v2 throughout. Enable future theming (light mode, custom palettes).

## User Story

As a developer, I want consistent color access across all components so UI changes are centralized and future theming is possible.

## Requirements

- Single source of truth for all colors
- Zero hardcoded hex values outside theme package
- All components use `theme.Current().S()` pattern
- Standardize on lipgloss v2 (remove v1 usage)
- Prepare for future light/dark mode support
- No visual changes to current appearance

## Technical Implementation

### Architecture (Crush-inspired)

```
internal/tui/theme/
├── theme.go              # Theme struct, color definitions
├── manager.go            # Singleton manager, Current()
├── styles.go             # Pre-built Styles struct + builder
├── catppuccin_mocha.go   # Default theme palette
└── util.go               # Color utilities (interpolate, parse)
```

### Theme Struct

```go
type Theme struct {
    Name   string
    IsDark bool

    // Semantic colors
    Primary, Secondary, Tertiary lipgloss.Color

    // Background hierarchy (dark→light)
    BgBase, BgMantle, BgGutter lipgloss.Color
    BgSurface0, BgSurface1, BgSurface2 lipgloss.Color
    BgOverlay lipgloss.Color

    // Foreground hierarchy (dim→bright)
    FgMuted, FgSubtle, FgBase, FgBright lipgloss.Color

    // Status
    Success, Warning, Error, Info lipgloss.Color

    // Diff
    DiffInsertBg, DiffDeleteBg, DiffEqualBg, DiffMissingBg lipgloss.Color

    // Borders
    BorderMuted, BorderDefault, BorderFocused lipgloss.Color

    // Lazy-built styles
    styles     *Styles
    stylesOnce sync.Once
}

func (t *Theme) S() *Styles {
    t.stylesOnce.Do(func() { t.styles = t.buildStyles() })
    return t.styles
}
```

### Styles Struct (80+ pre-built styles)

Categories:
- Base text: `Base`, `Muted`, `Subtle`, `Bright`
- Status: `Success`, `Warning`, `Error`, `Info`
- Header/Footer: `HeaderTitle`, `HeaderSeparator`, `FooterKey`, `StatusBar`
- Status indicators: `StatusRemaining`, `StatusInProgress`, `StatusCompleted`, `StatusBlocked`
- Tool calls: `ToolIconPending`, `ToolIconSuccess`, `ToolName`, `ToolOutput`
- Code blocks: `CodeLineNum`, `CodeContent`, `CodeTruncation`
- Diff view: `DiffLineNumInsert`, `DiffContentDelete`, etc.
- Thinking: `ThinkingBox`, `ThinkingContent`, `ThinkingDuration`
- Notes: `NoteTypeLearning`, `NoteTypeStuck`, `NoteContent`
- Logs: `LogTimestamp`, `LogTask`, `LogContent`
- Modals: `ModalContainer`, `ModalTitle`, `ModalLabel`
- Badges: `Badge`, `BadgeSuccess`, `BadgeWarning`
- Panels: `PanelTitle`, `PanelTitleFocused`, `PanelRule`
- Messages: `AssistantBorder`, `UserBorder`
- Inputs: `TextInputStyles` (bubbles textinput.Styles)
- Buttons: `ButtonNormal`, `ButtonDisabled`, `ButtonFocused`

### Singleton Manager

```go
var (
    manager     *Manager
    managerOnce sync.Once
)

func Current() *Theme {
    return DefaultManager().Current()
}

func DefaultManager() *Manager {
    managerOnce.Do(func() {
        manager = &Manager{themes: make(map[string]*Theme)}
        manager.Register(NewCatppuccinMocha())
        manager.SetTheme("catppuccin-mocha")
    })
    return manager
}
```

### Component Access Pattern

```go
import "iteratr/internal/tui/theme"

func (c *Component) View() string {
    t := theme.Current()
    s := t.S()
    
    // Use pre-built style
    header := s.HeaderTitle.Render("Title")
    
    // Use color directly
    custom := lipgloss.NewStyle().Foreground(t.Primary).Render("text")
    
    return header + custom
}
```

## Gotchas & Mitigations

### 1. Lipgloss v1 → v2 Migration

**Problem**: Current codebase mixes v1 (`github.com/charmbracelet/lipgloss`) and v2 (`charm.land/lipgloss/v2`). Some files import BOTH versions (e.g., `wizard/model_selector.go`, `wizard/config_step.go`).

**Mitigation**: 
- Remove all v1 imports from our code
- Update all imports to `charm.land/lipgloss/v2`
- v2 API is mostly compatible; main changes are import path

### 2. Glamour v2 Available (Crush Pattern)

**Problem**: `github.com/charmbracelet/glamour` v1 depends on lipgloss v1.

**Solution**: Crush uses `charm.land/glamour/v2` (pseudo-version). We can do the same:
- Replace `github.com/charmbracelet/glamour` with `charm.land/glamour/v2 v2.0.0-20260123212943-6014aa153a9b`
- Update `markdown.go` import from `github.com/charmbracelet/glamour` to `charm.land/glamour/v2`
- Remove `github.com/charmbracelet/lipgloss` v1 from go.mod entirely
- **Result**: Single lipgloss version (v2) throughout

### 3. Editor Package Has No Lipgloss Dependency

**Problem**: Initially thought `charmbracelet/x/editor` might conflict.

**Mitigation**: Non-issue. Editor only depends on Go stdlib - safe to keep.

### 4. Package-Level Variable Initialization

**Problem**: Current `styles.go` uses `var styleX = lipgloss.NewStyle()...` at package level. If theme isn't initialized, this panics.

**Mitigation**:
- Theme manager uses `sync.Once` - safe for concurrent init
- `Current()` always returns valid theme (default registered in `managerOnce`)
- Styles are lazy-built on first `S()` call, not at package load

### 5. Circular Import Risk

**Problem**: If `theme` imports `tui` and `tui` imports `theme`, circular dependency.

**Mitigation**:
- Theme package has zero dependencies on `tui` package
- Theme only imports lipgloss, sync, fmt
- All TUI components import theme (one-way dependency)

### 6. GradientSpinner Color Parameters

**Problem**: `NewGradientSpinner(colorA, colorB, label)` takes string colors, currently hardcoded at call sites.

**Mitigation**:
- Add `NewDefaultGradientSpinner(label)` that uses `theme.Current().Primary/Secondary`
- Or change signature to accept `lipgloss.Color` and convert internally

### 7. interpolateColor Function Location

**Problem**: `interpolateColor` in `anim.go` is used by both `GradientSpinner` and `renderModalTitle`.

**Mitigation**:
- Move to `theme/util.go` as exported `InterpolateColor`
- Both `anim.go` and `styles.go` import from theme package

### 8. Bubbles v2 Component Styles

**Problem**: `textinput.Styles`, `spinner.Style` require specific struct types from bubbles.

**Mitigation**:
- `Styles` struct includes `TextInputStyles textinput.Styles`
- Builder populates these from theme colors
- Components use `t.S().TextInputStyles` directly

### 9. Test Isolation

**Problem**: Tests may need different themes or mock colors.

**Mitigation**:
- Add `SetThemeForTesting(t *Theme)` that bypasses manager
- Or use `manager.Register()` + `manager.SetTheme()` in test setup

### 10. renderModalTitle Gradient

**Problem**: Uses `colorPrimary` and `colorSecondary` directly for gradient.

**Mitigation**:
- Move to theme package or pass colors as parameters
- Function becomes `RenderModalTitle(title, width string, t *Theme)`

### 11. Dual Lipgloss Imports in Same File

**Problem**: `wizard/model_selector.go` and `wizard/config_step.go` import BOTH lipgloss v1 and v2:
```go
import (
    lipglossv2 "charm.land/lipgloss/v2"
    "github.com/charmbracelet/lipgloss"  // v1!
)
```

**Mitigation**:
- Remove all v1 imports
- Replace `lipgloss.` calls with `lipglossv2.` (or just `lipgloss` after rename)
- Standardize import alias: use `lipgloss` for v2 (no alias needed)

### 12. textarea.DefaultDarkStyles() Pattern

**Problem**: `task_input_modal.go` uses `textarea.DefaultDarkStyles()` from bubbles v2, then modifies cursor color with v1 lipgloss.Color:
```go
styles := textarea.DefaultDarkStyles()
styles.Cursor.Color = lipgloss.Color(colorSecondary)  // v1 type!
```

**Mitigation**:
- Replace with theme-based styles: `t.S().TextAreaStyles`
- Or update to use v2 types: `lipgloss.Color(...)` from v2 package

## Tracer Bullet

Validate architecture with minimal end-to-end slice before full implementation:

1. Create `theme/` package with minimal `Theme` struct (just `Primary`, `BgBase`, `FgBase`)
2. Create `Manager` with `Current()` returning hardcoded CatppuccinMocha
3. Create `Styles` with just `HeaderTitle` style
4. Update ONE component (`status.go`) to use `theme.Current().S().HeaderTitle`
5. Run app, verify header renders correctly
6. If works → proceed with full implementation

## Tasks

### 1. Tracer Bullet (validate architecture)
- [ ] Create `internal/tui/theme/` directory
- [ ] Create `theme.go` with minimal Theme struct (Primary, BgBase, FgBase only)
- [ ] Create `manager.go` with Current() singleton
- [ ] Create `catppuccin_mocha.go` with 3 colors
- [ ] Create `styles.go` with Styles struct containing only HeaderTitle
- [ ] Update `status.go` to use `theme.Current().S().HeaderTitle`
- [ ] Verify app compiles and runs correctly

### 2. Complete Theme Package
- [ ] Add all color fields to Theme struct (18 colors)
- [ ] Add diff colors (4 colors)
- [ ] Add border colors (3 colors)
- [ ] Update CatppuccinMocha with full palette

### 3. Complete Styles Builder
- [ ] Add all 80+ style fields to Styles struct
- [ ] Implement buildStyles() method
- [ ] Add TextInputStyles for bubbles components
- [ ] Add button styles (Normal, Disabled, Focused)

### 4. Add Utilities
- [ ] Move interpolateColor to theme/util.go as InterpolateColor
- [ ] Move parseHexColor, formatHexColor to util.go
- [ ] Add AsString(color) helper for any remaining string conversions

### 5. Migrate Main TUI (lipgloss v2 + theme)
- [ ] Update `styles.go` - remove colors/styles, keep only renderModalTitle (updated to use theme)
- [ ] Update `anim.go` - use theme colors, add NewDefaultGradientSpinner
- [ ] Update `agent.go` - use theme.Current().S().TextInputStyles
- [ ] Update `status.go` - full migration to theme.S()
- [ ] Update `footer.go` - use theme.S()
- [ ] Update `sidebar.go` - use theme.S()
- [ ] Update `modal.go` - use theme.S()
- [ ] Update `note_modal.go` - use theme.S()
- [ ] Update `dialog.go` - use theme.S()
- [ ] Update `logs.go` - use theme.S()
- [ ] Update `dashboard.go` - use theme.S()
- [ ] Update `messages.go` - use theme.S()
- [ ] Update `scrolllist.go` - use theme.S() if needed
- [ ] Update `task_input_modal.go` - use theme.S().TextInputStyles
- [ ] Update `note_input_modal.go` - use theme.S().TextInputStyles
- [ ] Update `draw.go` - use theme colors if needed
- [ ] Update `notes.go` - use theme.S()

### 6. Migrate Wizard Components
- [ ] Delete `wizard/styles.go`
- [ ] Update `wizard/wizard.go` - use theme package
- [ ] Update `wizard/file_picker.go` - replace 3 hardcoded colors
- [ ] Update `wizard/model_selector.go` - replace 11 hardcoded colors
- [ ] Update `wizard/button_bar.go` - replace 6 hardcoded colors, use theme.S().Button*
- [ ] Update `wizard/config_step.go` - replace 8 hardcoded colors
- [ ] Update `wizard/template_editor.go` - replace 2 hardcoded colors

### 7. Update Dependencies & Imports (v1 → v2)
- [ ] Update go.mod: replace `github.com/charmbracelet/glamour` with `charm.land/glamour/v2 v2.0.0-20260123212943-6014aa153a9b`
- [ ] Update go.mod: remove `github.com/charmbracelet/lipgloss` v1 (direct dependency)
- [ ] Update `markdown.go`: change import to `charm.land/glamour/v2`
- [ ] Replace `github.com/charmbracelet/lipgloss` with `charm.land/lipgloss/v2` in all files
- [ ] Update any v1-specific API usage
- [ ] Run `go mod tidy` to clean up indirect dependencies

### 8. Cleanup & Verification
- [ ] Remove backward compatibility code from styles.go
- [ ] Run all tests
- [ ] Visual verification - compare before/after screenshots
- [ ] Run linter

## UI Mockup

No visual changes - this is an internal refactoring. UI should look identical before and after.

## Out of Scope (v1)

- Light mode theme
- Custom user themes
- Theme configuration file
- Runtime theme switching UI
- Color scale generation (opencode-style seed→scale)

## Open Questions

1. Should `renderModalTitle` move to theme package or stay in styles.go with theme dependency?
   - **Decision**: Keep in styles.go, pass theme as parameter

2. Should we add a `theme.Colors()` shortcut that returns just colors without styles?
   - **Decision**: No, `Current()` provides direct color access via `t.Primary` etc.
