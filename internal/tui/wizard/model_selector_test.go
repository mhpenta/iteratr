package wizard

import (
	"os"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/config"
)

// TestModelSelectorPreFillFromConfig verifies that the model selector
// pre-selects the model from config after models are loaded.
func TestModelSelectorPreFillFromConfig(t *testing.T) {
	// Create a temporary config directory with correct structure
	tmpDir := t.TempDir()

	// Write a test config with a specific model
	testModel := "test/model-from-config"
	cfg := &config.Config{
		Model:      testModel,
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 0,
	}

	// Set XDG_CONFIG_HOME to temp dir so config.Load() finds our test config
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	if err := os.Setenv("XDG_CONFIG_HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set XDG_CONFIG_HOME: %v", err)
	}

	// Write config using WriteGlobal (which uses XDG_CONFIG_HOME)
	if err := config.WriteGlobal(cfg); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create a model selector
	selector := NewModelSelectorStep()

	// Simulate models loaded (including our test model)
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: testModel, name: testModel}, // Our configured model
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}

	// Send ModelsLoadedMsg
	msg := ModelsLoadedMsg{models: testModels}
	cmd := selector.Update(msg)

	// Verify command is returned (ContentChangedMsg)
	if cmd == nil {
		t.Fatal("Expected cmd from Update, got nil")
	}

	// Execute the command to get the message
	resultMsg := cmd()
	if _, ok := resultMsg.(ContentChangedMsg); !ok {
		t.Errorf("Expected ContentChangedMsg, got %T", resultMsg)
	}

	// Verify the test model is selected
	selectedModel := selector.SelectedModel()
	if selectedModel != testModel {
		t.Errorf("Expected selected model %q, got %q", testModel, selectedModel)
	}

	// Verify the selectedIdx is correct (index 1 in testModels)
	if selector.selectedIdx != 1 {
		t.Errorf("Expected selectedIdx 1, got %d", selector.selectedIdx)
	}
}

// TestModelSelectorNoConfig verifies that the model selector defaults to
// first model when no config exists.
func TestModelSelectorNoConfig(t *testing.T) {
	// Ensure no config exists by using empty temp dir
	tmpDir := t.TempDir()
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	if err := os.Setenv("XDG_CONFIG_HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set XDG_CONFIG_HOME: %v", err)
	}

	// Also ensure no project config
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	// Create a model selector
	selector := NewModelSelectorStep()

	// Simulate models loaded
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}

	// Send ModelsLoadedMsg
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify first model is selected by default
	selectedModel := selector.SelectedModel()
	if selectedModel != testModels[0].id {
		t.Errorf("Expected selected model %q, got %q", testModels[0].id, selectedModel)
	}

	// Verify selectedIdx is 0
	if selector.selectedIdx != 0 {
		t.Errorf("Expected selectedIdx 0, got %d", selector.selectedIdx)
	}
}

// TestModelSelectorConfigModelNotInList verifies fallback behavior when
// configured model is not in the available models list.
func TestModelSelectorConfigModelNotInList(t *testing.T) {
	// Create a temporary config directory
	tmpDir := t.TempDir()

	// Write a test config with a model that won't be in the list
	cfg := &config.Config{
		Model:      "nonexistent/model",
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 0,
	}

	// Set XDG_CONFIG_HOME to temp dir
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	if err := os.Setenv("XDG_CONFIG_HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set XDG_CONFIG_HOME: %v", err)
	}

	// Write config using WriteGlobal
	if err := config.WriteGlobal(cfg); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create a model selector
	selector := NewModelSelectorStep()

	// Simulate models loaded (not including the configured model)
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}

	// Send ModelsLoadedMsg
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify first model is selected as fallback
	selectedModel := selector.SelectedModel()
	if selectedModel != testModels[0].id {
		t.Errorf("Expected selected model %q, got %q", testModels[0].id, selectedModel)
	}

	// Verify selectedIdx is 0
	if selector.selectedIdx != 0 {
		t.Errorf("Expected selectedIdx 0, got %d", selector.selectedIdx)
	}
}

// TestModelSelectorUserOverride verifies that user can navigate and select
// a different model than the pre-selected one.
func TestModelSelectorUserOverride(t *testing.T) {
	// Create a temporary config directory
	tmpDir := t.TempDir()

	// Write a test config
	cfg := &config.Config{
		Model:      "anthropic/claude-sonnet-4-5",
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 0,
	}

	// Set XDG_CONFIG_HOME
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()
	if err := os.Setenv("XDG_CONFIG_HOME", tmpDir); err != nil {
		t.Fatalf("Failed to set XDG_CONFIG_HOME: %v", err)
	}

	// Write config using WriteGlobal
	if err := config.WriteGlobal(cfg); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create selector
	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify first model is pre-selected
	if selector.SelectedModel() != "anthropic/claude-sonnet-4-5" {
		t.Fatal("Expected first model to be pre-selected")
	}

	// Simulate user pressing "down" to move to second model
	keyMsg := tea.KeyPressMsg{Code: tea.KeyDown}
	_ = selector.Update(keyMsg)

	// Verify second model is now selected
	selectedModel := selector.SelectedModel()
	if selectedModel != "openai/gpt-4" {
		t.Errorf("Expected selected model %q after down key, got %q", "openai/gpt-4", selectedModel)
	}

	// Simulate user pressing "enter" to confirm selection
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	cmd := selector.Update(enterMsg)

	// Verify ModelSelectedMsg is returned
	if cmd == nil {
		t.Fatal("Expected cmd from enter key, got nil")
	}

	resultMsg := cmd()
	selectedMsg, ok := resultMsg.(ModelSelectedMsg)
	if !ok {
		t.Fatalf("Expected ModelSelectedMsg, got %T", resultMsg)
	}

	// Verify the correct model is in the message
	if selectedMsg.ModelID != "openai/gpt-4" {
		t.Errorf("Expected ModelID %q, got %q", "openai/gpt-4", selectedMsg.ModelID)
	}
}
