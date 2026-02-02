package tui

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/session"
)

func TestNewApp(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	app := NewApp(ctx, nil, "test-session", "/tmp", tmpDir, nil, nil, nil)

	if app == nil {
		t.Fatal("expected non-nil app")
		return // Explicit return to help static analysis
	}
	if app.sessionName != "test-session" {
		t.Errorf("session name: got %s, want test-session", app.sessionName)
	}
	if app.dashboard == nil {
		t.Error("expected non-nil dashboard")
	}
	if app.logs == nil {
		t.Error("expected non-nil logs")
	}
	if app.agent == nil {
		t.Error("expected non-nil agent")
	}
}

func TestApp_HandleKeyPress_LogsToggle(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Initially logs not visible
	if app.logsVisible {
		t.Error("logs should not be visible initially")
	}

	// ctrl+x l toggles logs
	msg := tea.KeyPressMsg{Text: "ctrl+x"}
	updatedModel, _ := app.handleKeyPress(msg)
	app = updatedModel.(*App)
	msg = tea.KeyPressMsg{Text: "l"}
	updatedModel, _ = app.handleKeyPress(msg)
	app = updatedModel.(*App)
	if !app.logsVisible {
		t.Error("logs should be visible after ctrl+x l")
	}

	// ctrl+x l again hides
	msg = tea.KeyPressMsg{Text: "ctrl+x"}
	updatedModel, _ = app.handleKeyPress(msg)
	app = updatedModel.(*App)
	msg = tea.KeyPressMsg{Text: "l"}
	updatedModel, _ = app.handleKeyPress(msg)
	app = updatedModel.(*App)
	if app.logsVisible {
		t.Error("logs should be hidden after second ctrl+x l")
	}
}

func TestApp_HandleKeyPress_Quit(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Test ctrl+c
	msg := tea.KeyPressMsg{Text: "ctrl+c"}
	_, cmd := app.handleKeyPress(msg)

	if !app.quitting {
		t.Error("expected quitting to be true")
	}
	if cmd == nil {
		t.Error("expected quit command")
	}
}

// TestApp_RenderActiveView removed - renderActiveView() method was removed in Phase 12.4
// View rendering now uses Draw pattern with Ultraviolet Screen buffer

func TestApp_Update_WindowSize(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	msg := tea.WindowSizeMsg{
		Width:  100,
		Height: 50,
	}

	updatedModel, _ := app.Update(msg)
	updatedApp := updatedModel.(*App)

	if updatedApp.width != 100 {
		t.Errorf("width: got %d, want 100", updatedApp.width)
	}
	if updatedApp.height != 50 {
		t.Errorf("height: got %d, want 50", updatedApp.height)
	}
}

func TestApp_Update_AgentOutput(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	msg := AgentOutputMsg{
		Content: "Test output",
	}

	_, cmd := app.Update(msg)
	// Command can be nil - just verify it doesn't panic
	_ = cmd
}

func TestApp_Update_IterationStart(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	msg := IterationStartMsg{
		Number: 5,
	}

	_, cmd := app.Update(msg)
	// Command can be nil - just verify it doesn't panic
	_ = cmd
}

func TestApp_Update_StateUpdate(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	state := &session.State{
		Session: "test-session",
		Tasks: map[string]*session.Task{
			"t1": {ID: "t1", Content: "Task 1", Status: "remaining"},
		},
	}

	msg := StateUpdateMsg{
		State: state,
	}

	_, cmd := app.Update(msg)
	// Command can be nil - just verify it doesn't panic
	_ = cmd
}

func TestApp_View(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.width = 100
	app.height = 50

	view := app.View()

	// Verify view properties are set correctly
	if !view.AltScreen {
		t.Error("expected AltScreen to be enabled")
	}

	if view.MouseMode != tea.MouseModeCellMotion {
		t.Errorf("mouse mode: got %v, want MouseModeCellMotion", view.MouseMode)
	}

	if !view.ReportFocus {
		t.Error("expected ReportFocus to be enabled")
	}
}

func TestApp_View_Quitting(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)
	app.quitting = true

	view := app.View()

	// Just verify we get a view back
	_ = view
}

// TestApp_RenderViewTabs removed - renderViewTabs() method was removed in Phase 12.4
// View navigation now handled by Footer component

// TestApp_RenderHeader removed - renderHeader() method was removed in Phase 12.4
// Header now handled by Header component with Draw pattern

// TestApp_RenderFooter removed - renderFooter() method was removed in Phase 12.4
// Footer now handled by Footer component with Draw pattern

func TestViewType_Constants(t *testing.T) {
	// Verify view type constants are distinct
	views := []ViewType{
		ViewDashboard,
		ViewLogs,
	}

	seen := make(map[ViewType]bool)
	for _, view := range views {
		if seen[view] {
			t.Errorf("duplicate view type: %v", view)
		}
		seen[view] = true
	}

	if len(seen) != 2 {
		t.Errorf("expected 2 distinct view types, got %d", len(seen))
	}
}

func TestApp_HandleKeyPress_SidebarToggle(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Initially sidebar should be visible (default state from persistent storage)
	if !app.sidebarVisible {
		t.Error("expected sidebar to be visible initially")
	}

	// Press ctrl+x b to toggle sidebar hidden
	msg := tea.KeyPressMsg{Text: "ctrl+x"}
	updatedModel, _ := app.handleKeyPress(msg)
	app = updatedModel.(*App)
	msg = tea.KeyPressMsg{Text: "b"}
	updatedModel, _ = app.handleKeyPress(msg)
	app = updatedModel.(*App)

	if app.sidebarVisible {
		t.Error("expected sidebar to be hidden after ctrl+x b")
	}

	// Press ctrl+x b again to toggle sidebar visible
	msg = tea.KeyPressMsg{Text: "ctrl+x"}
	updatedModel, _ = app.handleKeyPress(msg)
	app = updatedModel.(*App)
	msg = tea.KeyPressMsg{Text: "b"}
	updatedModel, _ = app.handleKeyPress(msg)
	app = updatedModel.(*App)

	if !app.sidebarVisible {
		t.Error("expected sidebar to be visible after second ctrl+x b")
	}
}

func TestApp_HandleKeyPress_PrefixKeySequence(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Initially not in prefix mode
	if app.awaitingPrefixKey {
		t.Error("expected awaitingPrefixKey to be false initially")
	}

	// Press ctrl+x to enter prefix mode
	msg := tea.KeyPressMsg{Text: "ctrl+x"}
	updatedModel, _ := app.handleKeyPress(msg)
	app = updatedModel.(*App)

	if !app.awaitingPrefixKey {
		t.Error("expected awaitingPrefixKey to be true after ctrl+x")
	}
	if !app.status.prefixMode {
		t.Error("expected status bar prefixMode to be true after ctrl+x")
	}

	// Press 'l' to toggle logs (ctrl+x l)
	msg = tea.KeyPressMsg{Text: "l"}
	updatedModel, _ = app.handleKeyPress(msg)
	app = updatedModel.(*App)

	if app.awaitingPrefixKey {
		t.Error("expected awaitingPrefixKey to be false after completing sequence")
	}
	if app.status.prefixMode {
		t.Error("expected status bar prefixMode to be false after completing sequence")
	}
	if !app.logsVisible {
		t.Error("expected logs to be visible after ctrl+x l")
	}
}

func TestApp_HandleKeyPress_PrefixKeySequence_Sidebar(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Initially sidebar is visible (default state)
	if !app.sidebarVisible {
		t.Error("expected sidebar to be visible initially")
	}

	// Press ctrl+x then 'b' to toggle sidebar hidden
	msg := tea.KeyPressMsg{Text: "ctrl+x"}
	updatedModel, _ := app.handleKeyPress(msg)
	app = updatedModel.(*App)

	msg = tea.KeyPressMsg{Text: "b"}
	updatedModel, _ = app.handleKeyPress(msg)
	app = updatedModel.(*App)

	if app.sidebarVisible {
		t.Error("expected sidebar to be hidden after ctrl+x b")
	}
	if app.awaitingPrefixKey {
		t.Error("expected awaitingPrefixKey to be false after completing sequence")
	}
}

func TestApp_HandleKeyPress_PrefixKeySequence_Cancel(t *testing.T) {
	ctx := context.Background()
	app := NewApp(ctx, nil, "test-session", "/tmp", t.TempDir(), nil, nil, nil)

	// Press ctrl+x to enter prefix mode
	msg := tea.KeyPressMsg{Text: "ctrl+x"}
	updatedModel, _ := app.handleKeyPress(msg)
	app = updatedModel.(*App)

	if !app.awaitingPrefixKey {
		t.Error("expected awaitingPrefixKey to be true after ctrl+x")
	}

	// Press esc to cancel prefix mode
	msg = tea.KeyPressMsg{Text: "esc"}
	updatedModel, _ = app.handleKeyPress(msg)
	app = updatedModel.(*App)

	if app.awaitingPrefixKey {
		t.Error("expected awaitingPrefixKey to be false after esc")
	}
	if app.logsVisible {
		t.Error("expected logs to remain hidden after canceling prefix mode")
	}
}

func TestApp_ResponsiveSidebarBehavior(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	tests := []struct {
		name                string
		initialWidth        int
		targetWidth         int
		userHiddenBefore    bool
		sidebarVisibleAfter bool
		userHiddenAfter     bool
	}{
		{
			name:                "narrowing below threshold auto-hides sidebar",
			initialWidth:        120,
			targetWidth:         80,
			userHiddenBefore:    false,
			sidebarVisibleAfter: false,
			userHiddenAfter:     false, // Auto-hidden, not user-hidden
		},
		{
			name:                "widening past threshold auto-restores sidebar",
			initialWidth:        80,
			targetWidth:         120,
			userHiddenBefore:    false,
			sidebarVisibleAfter: true,
			userHiddenAfter:     false,
		},
		{
			name:                "user-hidden sidebar stays hidden when narrowing",
			initialWidth:        120,
			targetWidth:         80,
			userHiddenBefore:    true,
			sidebarVisibleAfter: false,
			userHiddenAfter:     true, // Remains user-hidden
		},
		{
			name:                "user-hidden sidebar stays hidden when widening",
			initialWidth:        80,
			targetWidth:         120,
			userHiddenBefore:    true,
			sidebarVisibleAfter: false,
			userHiddenAfter:     true, // Remains user-hidden
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(ctx, nil, "test-session", "/tmp", tmpDir, nil, nil, nil)

			// Set initial width
			msg := tea.WindowSizeMsg{Width: tt.initialWidth, Height: 30}
			updatedModel, _ := app.Update(msg)
			app = updatedModel.(*App)

			// Set user-hidden state if needed
			if tt.userHiddenBefore {
				app.sidebarVisible = false
				app.sidebarUserHidden = true
			} else {
				app.sidebarVisible = true
				app.sidebarUserHidden = false
			}

			// Resize to target width
			msg = tea.WindowSizeMsg{Width: tt.targetWidth, Height: 30}
			updatedModel, _ = app.Update(msg)
			app = updatedModel.(*App)

			// Check results
			if app.sidebarVisible != tt.sidebarVisibleAfter {
				t.Errorf("sidebarVisible: got %v, want %v", app.sidebarVisible, tt.sidebarVisibleAfter)
			}
			if app.sidebarUserHidden != tt.userHiddenAfter {
				t.Errorf("sidebarUserHidden: got %v, want %v", app.sidebarUserHidden, tt.userHiddenAfter)
			}
		})
	}
}

func TestApp_ManualTogglePreservedAcrossResizes(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	app := NewApp(ctx, nil, "test-session", "/tmp", tmpDir, nil, nil, nil)

	// Start with wide terminal
	msg := tea.WindowSizeMsg{Width: 120, Height: 30}
	updatedModel, _ := app.Update(msg)
	app = updatedModel.(*App)

	// User manually hides sidebar
	keyMsg := tea.KeyPressMsg{Text: "ctrl+x"}
	updatedModel, _ = app.handleKeyPress(keyMsg)
	app = updatedModel.(*App)
	keyMsg = tea.KeyPressMsg{Text: "b"}
	updatedModel, _ = app.handleKeyPress(keyMsg)
	app = updatedModel.(*App)

	if app.sidebarVisible {
		t.Error("sidebar should be hidden after manual toggle")
	}
	if !app.sidebarUserHidden {
		t.Error("sidebarUserHidden should be true after manual hide")
	}

	// Narrow terminal (should stay hidden)
	msg = tea.WindowSizeMsg{Width: 80, Height: 30}
	updatedModel, _ = app.Update(msg)
	app = updatedModel.(*App)

	if app.sidebarVisible {
		t.Error("sidebar should remain hidden when narrowing")
	}
	if !app.sidebarUserHidden {
		t.Error("sidebarUserHidden should remain true")
	}

	// Widen terminal again (should still stay hidden)
	msg = tea.WindowSizeMsg{Width: 120, Height: 30}
	updatedModel, _ = app.Update(msg)
	app = updatedModel.(*App)

	if app.sidebarVisible {
		t.Error("sidebar should remain hidden when widening (user preference)")
	}
	if !app.sidebarUserHidden {
		t.Error("sidebarUserHidden should remain true")
	}

	// User manually shows sidebar
	keyMsg = tea.KeyPressMsg{Text: "ctrl+x"}
	updatedModel, _ = app.handleKeyPress(keyMsg)
	app = updatedModel.(*App)
	keyMsg = tea.KeyPressMsg{Text: "b"}
	updatedModel, _ = app.handleKeyPress(keyMsg)
	app = updatedModel.(*App)

	if !app.sidebarVisible {
		t.Error("sidebar should be visible after manual toggle")
	}
	if app.sidebarUserHidden {
		t.Error("sidebarUserHidden should be false after manual show")
	}
}
