package setup

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestModelStep_CustomMode(t *testing.T) {
	// Create a new model step
	step := NewModelStep()

	// Simulate models loaded (skip actual fetch)
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "test/model-1", name: "test/model-1"},
		{id: "test/model-2", name: "test/model-2"},
	}
	step.filterModels()

	// Initially should not be in custom mode
	if step.isCustomMode {
		t.Error("Expected isCustomMode to be false initially")
	}

	// Press 'c' to enter custom mode
	step.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})

	// Should now be in custom mode
	if !step.isCustomMode {
		t.Error("Expected isCustomMode to be true after pressing 'c'")
	}

	// View should show custom input
	view := step.View()
	if !strings.Contains(view, "Enter Custom Model") {
		t.Error("Expected view to contain 'Enter Custom Model'")
	}

	// Simulate typing a custom model
	for _, r := range "my-custom/model" {
		step.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	// Verify custom input value
	if step.customInput.Value() != "my-custom/model" {
		t.Errorf("Expected custom input value to be 'my-custom/model', got '%s'", step.customInput.Value())
	}

	// Press Enter to confirm
	cmd := step.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Execute the command to get the message
	if cmd == nil {
		t.Fatal("Expected cmd to be non-nil after pressing Enter")
	}

	msg := cmd()
	modelMsg, ok := msg.(ModelSelectedMsg)
	if !ok {
		t.Fatalf("Expected ModelSelectedMsg, got %T", msg)
	}

	if modelMsg.ModelID != "my-custom/model" {
		t.Errorf("Expected ModelID to be 'my-custom/model', got '%s'", modelMsg.ModelID)
	}
}

func TestModelStep_CustomModeCancel(t *testing.T) {
	// Create a new model step
	step := NewModelStep()

	// Simulate models loaded
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "test/model-1", name: "test/model-1"},
	}
	step.filterModels()

	// Enter custom mode
	step.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})

	if !step.isCustomMode {
		t.Error("Expected isCustomMode to be true after pressing 'c'")
	}

	// Type something
	for _, r := range "partial" {
		step.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}

	// Press ESC to cancel
	step.Update(tea.KeyPressMsg{Code: tea.KeyEscape})

	// Should exit custom mode
	if step.isCustomMode {
		t.Error("Expected isCustomMode to be false after pressing ESC")
	}

	// Custom input should be cleared
	if step.customInput.Value() != "" {
		t.Errorf("Expected custom input to be cleared, got '%s'", step.customInput.Value())
	}
}

func TestModelStep_CustomModeEmptyInput(t *testing.T) {
	// Create a new model step
	step := NewModelStep()

	// Simulate models loaded
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "test/model-1", name: "test/model-1"},
	}
	step.filterModels()

	// Enter custom mode
	step.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})

	// Press Enter without typing anything
	cmd := step.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Should not return a command (empty input ignored)
	if cmd != nil {
		t.Error("Expected cmd to be nil when pressing Enter with empty input")
	}
}

func TestModelStep_PreferredHeight_CustomMode(t *testing.T) {
	step := NewModelStep()

	// Not in custom mode initially - add multiple models so height differs from custom mode
	step.loading = false
	step.allModels = []*ModelInfo{
		{id: "test/model-1", name: "test/model-1"},
		{id: "test/model-2", name: "test/model-2"},
		{id: "test/model-3", name: "test/model-3"},
		{id: "test/model-4", name: "test/model-4"},
		{id: "test/model-5", name: "test/model-5"},
	}
	step.filterModels()

	normalHeight := step.PreferredHeight()

	// Enter custom mode
	step.isCustomMode = true

	customHeight := step.PreferredHeight()

	// Custom mode should have fixed height of 5
	if customHeight != 5 {
		t.Errorf("Expected custom mode height to be 5, got %d", customHeight)
	}

	// Heights should be different (normal mode has 5 models + 4 overhead = 9)
	if normalHeight == customHeight {
		t.Errorf("Expected normal and custom mode to have different heights, both got %d", normalHeight)
	}
}
