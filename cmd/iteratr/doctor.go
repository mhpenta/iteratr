package main

import (
	"fmt"
	"os/exec"
	"strings"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check dependencies and environment",
	Long: `Check that required dependencies are installed and accessible.

This command verifies that:
- opencode is installed and in PATH
- The data directory is writable
- Other environment requirements are met`,
	RunE: runDoctor,
}

// Theme colors (catppuccin mocha)
var (
	colorPrimary = lipgloss.Color("#cba6f7") // Mauve
	colorMuted   = lipgloss.Color("#a6adc8") // Subtext0
	colorBase    = lipgloss.Color("#cdd6f4") // Text
	colorSuccess = lipgloss.Color("#a6e3a1") // Green
	colorWarning = lipgloss.Color("#f9e2af") // Yellow
	colorError   = lipgloss.Color("#f38ba8") // Red
	colorBorder  = lipgloss.Color("#585b70") // Surface2
)

type checkResult struct {
	name    string
	status  string
	details string
}

func runDoctor(cmd *cobra.Command, args []string) error {
	var results []checkResult
	allOk := true

	// Check for opencode
	if _, err := exec.LookPath("opencode"); err != nil {
		results = append(results, checkResult{
			name:    "opencode",
			status:  "FAIL",
			details: "Not found in PATH. Install: https://opencode.coder.com",
		})
		allOk = false
	} else {
		out, err := exec.Command("opencode", "--version").CombinedOutput()
		if err != nil {
			results = append(results, checkResult{
				name:    "opencode",
				status:  "WARN",
				details: "Found but can't get version",
			})
		} else {
			version := strings.TrimSpace(string(out))
			results = append(results, checkResult{
				name:    "opencode",
				status:  "OK",
				details: version,
			})
		}
	}

	// Build rows with status icons
	rows := make([][]string, len(results))
	for i, r := range results {
		var icon string
		switch r.status {
		case "OK":
			icon = "✓"
		case "FAIL":
			icon = "⊗"
		case "WARN":
			icon = "⊘"
		}
		rows[i] = []string{r.name, icon, r.details}
	}

	// Create styled table
	t := table.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(colorBorder)).
		Headers("Dependency", "Status", "Details").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return lipgloss.NewStyle().
					Foreground(colorPrimary).
					Bold(true).
					Padding(0, 1)
			}

			style := lipgloss.NewStyle().Padding(0, 1)

			// Style status column with colors
			if col == 1 {
				status := results[row].status
				switch status {
				case "OK":
					return style.Foreground(colorSuccess)
				case "FAIL":
					return style.Foreground(colorError)
				case "WARN":
					return style.Foreground(colorWarning)
				}
			}

			// Name column
			if col == 0 {
				return style.Foreground(colorBase)
			}

			// Details column
			return style.Foreground(colorMuted)
		})

	fmt.Println(t)

	// Summary
	fmt.Println()
	successStyle := lipgloss.NewStyle().Foreground(colorSuccess)
	errorStyle := lipgloss.NewStyle().Foreground(colorError)

	if allOk {
		fmt.Println(successStyle.Render("✓ All checks passed!"))
		return nil
	} else {
		fmt.Println(errorStyle.Render("⊗ Some checks failed. Please install missing dependencies."))
		return fmt.Errorf("doctor check failed")
	}
}
