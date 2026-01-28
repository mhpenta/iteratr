package main

import (
	"fmt"
	"os"

	"github.com/mark3labs/iteratr/internal/config"
	"github.com/mark3labs/iteratr/internal/tui/setup"
	"github.com/spf13/cobra"
)

var setupFlags struct {
	project bool
	force   bool
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Create iteratr configuration file",
	Long: `Create an iteratr configuration file with sensible defaults.

By default, creates a global config at ~/.config/iteratr/iteratr.yml.
Use --project to create a project-local config in the current directory.`,
	RunE: runSetup,
}

func init() {
	setupCmd.Flags().BoolVarP(&setupFlags.project, "project", "p", false, "Create config in current directory instead of global location")
	setupCmd.Flags().BoolVarP(&setupFlags.force, "force", "f", false, "Overwrite existing config file")
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Determine target path
	targetPath := config.GlobalPath()
	if setupFlags.project {
		targetPath = config.ProjectPath()
	}

	// Check if config already exists
	if !setupFlags.force && fileExists(targetPath) {
		return fmt.Errorf("config file already exists at %s\n\nUse --force to overwrite", targetPath)
	}

	// Run the TUI wizard to collect user preferences
	result, err := setup.RunSetup(setupFlags.project)
	if err != nil {
		return fmt.Errorf("setup wizard failed: %w", err)
	}

	// Config is written by the wizard itself during the flow
	// Just print success message after wizard exits
	fmt.Printf("\nConfig written to: %s\n\n", targetPath)
	fmt.Println("Run 'iteratr build' to get started.")

	// Suppress unused warning
	_ = result

	return nil
}

// fileExists checks if a file exists (helper for setup command).
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
