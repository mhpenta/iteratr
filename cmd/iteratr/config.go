package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/iteratr/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Display current configuration",
	Long: `Display the current resolved configuration showing values from all sources.

Configuration precedence (highest to lowest):
  1. Environment variables (ITERATR_*)
  2. Project config (./iteratr.yml)
  3. Global config (~/.config/iteratr/iteratr.yml)
  4. Defaults`,
	RunE: runConfig,
}

func runConfig(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Display configuration
	fmt.Println("Current Configuration:")
	fmt.Println("=====================")
	fmt.Println()

	// Marshal to YAML for pretty printing
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to format config: %w", err)
	}
	fmt.Print(string(data))
	fmt.Println()

	// Show config file locations
	fmt.Println("Config File Locations:")
	fmt.Println("---------------------")

	globalPath := config.GlobalPath()
	projectPath := config.ProjectPath()

	// Resolve project path to absolute for clearer display
	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		absProjectPath = projectPath
	}

	globalExists := fileExists(globalPath)
	projectExists := fileExists(projectPath)

	if globalExists {
		fmt.Printf("Global:  %s ✓\n", globalPath)
	} else {
		fmt.Printf("Global:  %s (not found)\n", globalPath)
	}

	if projectExists {
		fmt.Printf("Project: %s ✓\n", absProjectPath)
	} else {
		fmt.Printf("Project: %s (not found)\n", absProjectPath)
	}

	fmt.Println()

	// Show which environment variables are set
	envVars := []struct {
		name string
		key  string
	}{
		{"ITERATR_MODEL", "model"},
		{"ITERATR_AUTO_COMMIT", "auto_commit"},
		{"ITERATR_DATA_DIR", "data_dir"},
		{"ITERATR_LOG_LEVEL", "log_level"},
		{"ITERATR_LOG_FILE", "log_file"},
		{"ITERATR_ITERATIONS", "iterations"},
		{"ITERATR_HEADLESS", "headless"},
		{"ITERATR_TEMPLATE", "template"},
	}

	hasEnvOverrides := false
	var setEnvVars []string

	for _, ev := range envVars {
		if val := os.Getenv(ev.name); val != "" {
			hasEnvOverrides = true
			setEnvVars = append(setEnvVars, fmt.Sprintf("  %s=%s", ev.name, val))
		}
	}

	if hasEnvOverrides {
		fmt.Println("Environment Overrides:")
		fmt.Println("---------------------")
		for _, line := range setEnvVars {
			fmt.Println(line)
		}
		fmt.Println()
	}

	// Helpful note
	if !globalExists && !projectExists {
		fmt.Println("Note: No config files found. Run 'iteratr setup' to create one.")
	}

	return nil
}
