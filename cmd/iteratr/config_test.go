package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/iteratr/internal/config"
)

func TestConfigCommand(t *testing.T) {
	// Save original XDG_CONFIG_HOME and restore after test
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		if origXDG != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", origXDG)
		} else {
			_ = os.Unsetenv("XDG_CONFIG_HOME")
		}
	}()

	t.Run("runs without error when no config exists", func(t *testing.T) {
		// Create temp directory for test config
		tempDir := t.TempDir()
		_ = os.Setenv("XDG_CONFIG_HOME", tempDir)

		// Change to temp dir so project config also won't be found
		origWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(origWd) }()
		_ = os.Chdir(tempDir)

		// Run config command
		err := runConfig(configCmd, []string{})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("displays global config when it exists", func(t *testing.T) {
		// Create temp directory for test config
		tempDir := t.TempDir()
		_ = os.Setenv("XDG_CONFIG_HOME", tempDir)

		// Create global config
		configDir := filepath.Join(tempDir, "iteratr")
		_ = os.MkdirAll(configDir, 0755)
		globalPath := filepath.Join(configDir, "iteratr.yml")
		configContent := `model: test-model
auto_commit: false
data_dir: test-data
`
		_ = os.WriteFile(globalPath, []byte(configContent), 0644)

		// Change to temp dir so project config won't be found
		origWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(origWd) }()
		_ = os.Chdir(tempDir)

		// Run config command
		err := runConfig(configCmd, []string{})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify config was loaded correctly
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}
		if cfg.Model != "test-model" {
			t.Errorf("Expected model 'test-model', got '%s'", cfg.Model)
		}
		if cfg.AutoCommit != false {
			t.Errorf("Expected auto_commit false, got true")
		}
		if cfg.DataDir != "test-data" {
			t.Errorf("Expected data_dir 'test-data', got '%s'", cfg.DataDir)
		}
	})

	t.Run("displays project config precedence", func(t *testing.T) {
		// Create temp directory for test
		tempDir := t.TempDir()
		_ = os.Setenv("XDG_CONFIG_HOME", tempDir)

		// Create global config
		configDir := filepath.Join(tempDir, "iteratr")
		_ = os.MkdirAll(configDir, 0755)
		globalPath := filepath.Join(configDir, "iteratr.yml")
		globalContent := `model: global-model
data_dir: global-data
`
		_ = os.WriteFile(globalPath, []byte(globalContent), 0644)

		// Create project config
		projectDir := t.TempDir()
		origWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(origWd) }()
		_ = os.Chdir(projectDir)

		projectPath := filepath.Join(projectDir, "iteratr.yml")
		projectContent := `model: project-model
`
		_ = os.WriteFile(projectPath, []byte(projectContent), 0644)

		// Run config command
		err := runConfig(configCmd, []string{})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify project config overrides global
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}
		if cfg.Model != "project-model" {
			t.Errorf("Expected model 'project-model', got '%s'", cfg.Model)
		}
		// data_dir should still come from global since project doesn't set it
		if cfg.DataDir != "global-data" {
			t.Errorf("Expected data_dir 'global-data', got '%s'", cfg.DataDir)
		}
	})

	t.Run("displays env var overrides", func(t *testing.T) {
		// Create temp directory for test
		tempDir := t.TempDir()
		_ = os.Setenv("XDG_CONFIG_HOME", tempDir)

		// Create global config
		configDir := filepath.Join(tempDir, "iteratr")
		_ = os.MkdirAll(configDir, 0755)
		globalPath := filepath.Join(configDir, "iteratr.yml")
		configContent := `model: config-model
`
		_ = os.WriteFile(globalPath, []byte(configContent), 0644)

		// Set ENV var override
		origModel := os.Getenv("ITERATR_MODEL")
		defer func() {
			if origModel != "" {
				_ = os.Setenv("ITERATR_MODEL", origModel)
			} else {
				_ = os.Unsetenv("ITERATR_MODEL")
			}
		}()
		_ = os.Setenv("ITERATR_MODEL", "env-model")

		// Change to temp dir
		origWd, _ := os.Getwd()
		defer func() { _ = os.Chdir(origWd) }()
		_ = os.Chdir(tempDir)

		// Run config command
		err := runConfig(configCmd, []string{})
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		// Verify ENV var overrides config
		cfg, err := config.Load()
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}
		if cfg.Model != "env-model" {
			t.Errorf("Expected model 'env-model', got '%s'", cfg.Model)
		}
	})
}
