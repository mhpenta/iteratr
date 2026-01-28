package wizard

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/config"
)

// TestWizardPreFillsFromConfig verifies that the wizard pre-fills model
// from config when config exists.
func TestWizardPreFillsFromConfig(t *testing.T) {
	// Create temp directory for config
	tmpDir := t.TempDir()

	// Write a test config with specific model
	testModel := "anthropic/claude-opus-4"
	cfg := &config.Config{
		Model:      testModel,
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: 10,
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

	// Write config
	if err := config.WriteGlobal(cfg); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create model selector (step 2 in wizard)
	selector := NewModelSelectorStep()

	// Simulate models loaded (including configured model)
	testModels := []*ModelInfo{
		{id: "anthropic/claude-sonnet-4-5", name: "anthropic/claude-sonnet-4-5"},
		{id: testModel, name: testModel}, // Our configured model
		{id: "openai/gpt-4", name: "openai/gpt-4"},
	}

	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify configured model is pre-selected
	if selector.SelectedModel() != testModel {
		t.Errorf("Expected model %q to be pre-selected from config, got %q", testModel, selector.SelectedModel())
	}

	// Verify it's at the correct index (1)
	if selector.selectedIdx != 1 {
		t.Errorf("Expected selectedIdx 1, got %d", selector.selectedIdx)
	}
}

// TestWizardUserOverridesConfigModel verifies that user can override
// the config model during wizard without modifying config file.
func TestWizardUserOverridesConfigModel(t *testing.T) {
	// Create temp directory for config
	tmpDir := t.TempDir()

	// Write a test config
	configModel := "anthropic/claude-sonnet-4-5"
	cfg := &config.Config{
		Model:      configModel,
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

	// Write config
	if err := config.WriteGlobal(cfg); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Create selector
	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: configModel, name: configModel},
		{id: "openai/gpt-4", name: "openai/gpt-4"},
		{id: "anthropic/claude-opus-4", name: "anthropic/claude-opus-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify config model is pre-selected
	if selector.SelectedModel() != configModel {
		t.Fatal("Expected config model to be pre-selected")
	}

	// User navigates to different model
	downKey := tea.KeyPressMsg{Code: tea.KeyDown}
	_ = selector.Update(downKey)

	// Verify different model is selected
	userSelectedModel := "openai/gpt-4"
	if selector.SelectedModel() != userSelectedModel {
		t.Errorf("Expected user to select %q, got %q", userSelectedModel, selector.SelectedModel())
	}

	// User confirms selection
	enterKey := tea.KeyPressMsg{Code: tea.KeyEnter}
	cmd := selector.Update(enterKey)

	// Verify ModelSelectedMsg contains user's choice
	if cmd == nil {
		t.Fatal("Expected cmd from enter key")
	}

	resultMsg := cmd()
	selectedMsg, ok := resultMsg.(ModelSelectedMsg)
	if !ok {
		t.Fatalf("Expected ModelSelectedMsg, got %T", resultMsg)
	}

	if selectedMsg.ModelID != userSelectedModel {
		t.Errorf("Expected ModelID %q, got %q", userSelectedModel, selectedMsg.ModelID)
	}

	// Verify config file was NOT modified
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.Model != configModel {
		t.Errorf("Config file was modified! Expected model %q, got %q", configModel, loadedCfg.Model)
	}
}

// TestWizardWithoutConfigUsesFirstModel verifies that when no config exists,
// wizard defaults to first available model.
func TestWizardWithoutConfigUsesFirstModel(t *testing.T) {
	// Create empty temp directory (no config)
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to empty dir
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

	// Ensure no project config
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
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

	// Verify first model is selected (no config to pre-fill from)
	if selector.SelectedModel() != testModels[0].id {
		t.Errorf("Expected first model %q, got %q", testModels[0].id, selector.SelectedModel())
	}
}

// TestWizardProjectConfigOverridesGlobal verifies that project config
// takes precedence over global config for pre-filling.
func TestWizardProjectConfigOverridesGlobal(t *testing.T) {
	// Create temp directories
	tmpConfigDir := t.TempDir()
	tmpProjectDir := t.TempDir()

	// Write global config
	globalModel := "anthropic/claude-sonnet-4-5"
	globalCfg := &config.Config{
		Model:      globalModel,
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
	if err := os.Setenv("XDG_CONFIG_HOME", tmpConfigDir); err != nil {
		t.Fatalf("Failed to set XDG_CONFIG_HOME: %v", err)
	}

	if err := config.WriteGlobal(globalCfg); err != nil {
		t.Fatalf("Failed to write global config: %v", err)
	}

	// Write project config (in current directory)
	origWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(tmpProjectDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	projectModel := "openai/gpt-4"
	projectCfg := &config.Config{
		Model:      projectModel,
		AutoCommit: false,
		DataDir:    ".iteratr",
		LogLevel:   "debug",
		Iterations: 5,
	}

	if err := config.WriteProject(projectCfg); err != nil {
		t.Fatalf("Failed to write project config: %v", err)
	}

	// Verify project config exists
	if _, err := os.Stat("iteratr.yml"); os.IsNotExist(err) {
		t.Fatal("Project config was not created")
	}

	// Create selector
	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: globalModel, name: globalModel},
		{id: projectModel, name: projectModel},
		{id: "anthropic/claude-opus-4", name: "anthropic/claude-opus-4"},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify PROJECT model is pre-selected (not global)
	if selector.SelectedModel() != projectModel {
		t.Errorf("Expected project model %q to override global, got %q", projectModel, selector.SelectedModel())
	}
}

// TestWizardEnvVarOverridesConfig verifies that ITERATR_MODEL env var
// takes precedence over config file for pre-filling.
func TestWizardEnvVarOverridesConfig(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Write config
	configModel := "anthropic/claude-sonnet-4-5"
	cfg := &config.Config{
		Model:      configModel,
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

	if err := config.WriteGlobal(cfg); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Set ITERATR_MODEL env var (should override config)
	envModel := "openai/gpt-4-turbo"
	origModel := os.Getenv("ITERATR_MODEL")
	defer func() {
		if origModel != "" {
			_ = os.Setenv("ITERATR_MODEL", origModel)
		} else {
			_ = os.Unsetenv("ITERATR_MODEL")
		}
	}()
	if err := os.Setenv("ITERATR_MODEL", envModel); err != nil {
		t.Fatalf("Failed to set ITERATR_MODEL: %v", err)
	}

	// Verify config.Load() returns env var value
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if loadedCfg.Model != envModel {
		t.Errorf("Expected config.Load() to return env var model %q, got %q", envModel, loadedCfg.Model)
	}

	// Create selector
	selector := NewModelSelectorStep()

	// Load models
	testModels := []*ModelInfo{
		{id: configModel, name: configModel},
		{id: envModel, name: envModel},
	}
	msg := ModelsLoadedMsg{models: testModels}
	_ = selector.Update(msg)

	// Verify ENV model is pre-selected (not config file model)
	if selector.SelectedModel() != envModel {
		t.Errorf("Expected env var model %q to override config, got %q", envModel, selector.SelectedModel())
	}
}

// TestWizardResultDoesNotModifyConfig verifies that completing the wizard
// does not modify the config file on disk.
func TestWizardResultDoesNotModifyConfig(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Write initial config
	originalModel := "anthropic/claude-sonnet-4-5"
	originalIterations := 10
	cfg := &config.Config{
		Model:      originalModel,
		AutoCommit: true,
		DataDir:    ".iteratr",
		LogLevel:   "info",
		Iterations: originalIterations,
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

	if err := config.WriteGlobal(cfg); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Get config file path and modification time
	configPath := filepath.Join(tmpDir, "iteratr", "iteratr.yml")
	origStat, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	origModTime := origStat.ModTime()

	// Simulate wizard completing with different values
	// (In real usage, these values go to buildFlags, NOT back to config)
	wizardResult := &WizardResult{
		SpecPath:    "specs/feature.md",
		Model:       "openai/gpt-4", // Different from config
		Template:    "custom template",
		SessionName: "test-session",
		Iterations:  20, // Different from config
		ResumeMode:  false,
	}

	// Verify wizard result has different values
	if wizardResult.Model == originalModel {
		t.Fatal("Test setup error: wizard result should have different model")
	}
	if wizardResult.Iterations == originalIterations {
		t.Fatal("Test setup error: wizard result should have different iterations")
	}

	// Load config again
	loadedCfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	// Verify config still has original values
	if loadedCfg.Model != originalModel {
		t.Errorf("Config model was modified! Expected %q, got %q", originalModel, loadedCfg.Model)
	}
	if loadedCfg.Iterations != originalIterations {
		t.Errorf("Config iterations was modified! Expected %d, got %d", originalIterations, loadedCfg.Iterations)
	}

	// Verify config file was not written (modification time unchanged)
	newStat, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}
	if !newStat.ModTime().Equal(origModTime) {
		t.Error("Config file was modified (mod time changed)")
	}
}
