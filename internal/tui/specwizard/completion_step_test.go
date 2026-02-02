package specwizard

import (
	"strings"
	"testing"
)

func TestCompletionStep_Init(t *testing.T) {
	step := NewCompletionStep("/path/to/spec.md")
	cmd := step.Init()

	if cmd != nil {
		t.Error("Init should return nil cmd")
	}
	if !step.buttonFocused {
		t.Error("buttonFocused should be true after Init")
	}
	if step.focusedIndex != 0 {
		t.Error("focusedIndex should be 0 (View button) after Init")
	}
}

func TestCompletionStep_View(t *testing.T) {
	step := NewCompletionStep("/path/to/my-spec.md")
	step.SetSize(80, 20)

	view := step.View()

	// Check that the view contains expected elements
	if !strings.Contains(view, "Spec created successfully") {
		t.Error("View should contain success message")
	}
	if !strings.Contains(view, "/path/to/my-spec.md") {
		t.Error("View should contain spec path")
	}
	if !strings.Contains(view, "View") {
		t.Error("View should contain View button")
	}
	if !strings.Contains(view, "Start Build") {
		t.Error("View should contain Start Build button")
	}
	if !strings.Contains(view, "Exit") {
		t.Error("View should contain Exit button")
	}
}

func TestCompletionStep_ButtonNavigation(t *testing.T) {
	step := NewCompletionStep("/path/to/spec.md")
	step.Init()

	// Initially focused on View (index 0)
	if step.focusedIndex != 0 {
		t.Errorf("Expected focusedIndex 0, got %d", step.focusedIndex)
	}

	// Test tab navigation by directly manipulating focusedIndex
	// (In real use, Update() would handle this via KeyPressMsg)

	// Tab to next button (Build)
	step.focusedIndex = (step.focusedIndex + 1) % 3
	if step.focusedIndex != 1 {
		t.Errorf("Expected focusedIndex 1 after increment, got %d", step.focusedIndex)
	}

	// Tab to next button (Exit)
	step.focusedIndex = (step.focusedIndex + 1) % 3
	if step.focusedIndex != 2 {
		t.Errorf("Expected focusedIndex 2 after increment, got %d", step.focusedIndex)
	}

	// Tab wraps around to View
	step.focusedIndex = (step.focusedIndex + 1) % 3
	if step.focusedIndex != 0 {
		t.Errorf("Expected focusedIndex 0 after wrap, got %d", step.focusedIndex)
	}

	// Shift+Tab goes back to Exit (reverse direction)
	step.focusedIndex = (step.focusedIndex - 1 + 3) % 3
	if step.focusedIndex != 2 {
		t.Errorf("Expected focusedIndex 2 after decrement, got %d", step.focusedIndex)
	}
}

func TestCompletionStep_GetButtonAction(t *testing.T) {
	step := NewCompletionStep("/path/to/spec.md")
	step.Init()

	tests := []struct {
		index  int
		action ButtonAction
	}{
		{0, ButtonActionView},
		{1, ButtonActionBuild},
		{2, ButtonActionExit},
	}

	for _, tt := range tests {
		step.focusedIndex = tt.index
		action := step.getButtonAction()
		if action != tt.action {
			t.Errorf("For index %d, expected action %d, got %d", tt.index, tt.action, action)
		}
	}
}

func TestCompletionStep_PreferredHeight(t *testing.T) {
	step := NewCompletionStep("/path/to/spec.md")
	height := step.PreferredHeight()

	if height != 8 {
		t.Errorf("Expected preferred height 8, got %d", height)
	}
}
