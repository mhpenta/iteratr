package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/mark3labs/iteratr/internal/config"
	"github.com/spf13/cobra"
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

	globalPath := config.GlobalPath()
	projectPath := config.ProjectPath()
	absProjectPath, err := filepath.Abs(projectPath)
	if err != nil {
		absProjectPath = projectPath
	}

	globalExists := fileExists(globalPath)
	projectExists := fileExists(projectPath)

	// Build configuration values table
	configRows := [][]string{
		{"model", cfg.Model},
		{"auto_commit", strconv.FormatBool(cfg.AutoCommit)},
		{"data_dir", cfg.DataDir},
		{"log_level", cfg.LogLevel},
		{"log_file", cfg.LogFile},
		{"iterations", strconv.Itoa(cfg.Iterations)},
		{"headless", strconv.FormatBool(cfg.Headless)},
		{"template", cfg.Template},
	}

	configTable := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorBorder)).
		Headers("Key", "Value").
		Rows(configRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().
					Foreground(colorPrimary).
					Bold(true).
					Padding(0, 1)
			}
			style := lipgloss.NewStyle().Padding(0, 1)
			if col == 0 {
				return style.Foreground(colorBase)
			}
			return style.Foreground(colorMuted)
		})

	titleStyle := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	fmt.Println(titleStyle.Render("Configuration"))
	fmt.Println(configTable)
	fmt.Println()

	// Build config files table
	fileRows := [][]string{}
	if globalExists {
		fileRows = append(fileRows, []string{"Global", globalPath, "✓"})
	} else {
		fileRows = append(fileRows, []string{"Global", globalPath, "not found"})
	}
	if projectExists {
		fileRows = append(fileRows, []string{"Project", absProjectPath, "✓"})
	} else {
		fileRows = append(fileRows, []string{"Project", absProjectPath, "not found"})
	}

	filesTable := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorBorder)).
		Headers("Type", "Path", "Status").
		Rows(fileRows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().
					Foreground(colorPrimary).
					Bold(true).
					Padding(0, 1)
			}
			style := lipgloss.NewStyle().Padding(0, 1)
			if col == 2 {
				// Status column - color based on found/not found
				if row < len(fileRows) && fileRows[row][2] == "✓" {
					return style.Foreground(colorSuccess)
				}
				return style.Foreground(colorWarning)
			}
			if col == 0 {
				return style.Foreground(colorBase)
			}
			return style.Foreground(colorMuted)
		})

	fmt.Println(titleStyle.Render("Config Files"))
	fmt.Println(filesTable)

	// Show environment overrides if any
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

	var envRows [][]string
	for _, ev := range envVars {
		if val := os.Getenv(ev.name); val != "" {
			envRows = append(envRows, []string{ev.name, val})
		}
	}

	if len(envRows) > 0 {
		fmt.Println()
		envTable := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderStyle(lipgloss.NewStyle().Foreground(colorBorder)).
			Headers("Variable", "Value").
			Rows(envRows...).
			StyleFunc(func(row, col int) lipgloss.Style {
				if row == table.HeaderRow {
					return lipgloss.NewStyle().
						Foreground(colorPrimary).
						Bold(true).
						Padding(0, 1)
				}
				style := lipgloss.NewStyle().Padding(0, 1)
				if col == 0 {
					return style.Foreground(colorBase)
				}
				return style.Foreground(colorMuted)
			})

		fmt.Println(titleStyle.Render("Environment Overrides"))
		fmt.Println(envTable)
	}

	// Helpful note if no config files exist
	if !globalExists && !projectExists {
		fmt.Println()
		noteStyle := lipgloss.NewStyle().Foreground(colorWarning)
		fmt.Println(noteStyle.Render("No config files found. Run 'iteratr setup' to create one."))
	}

	return nil
}
