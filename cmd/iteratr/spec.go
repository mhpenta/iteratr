package main

import (
	"fmt"

	"github.com/mark3labs/iteratr/internal/config"
	"github.com/spf13/cobra"
)

var specCmd = &cobra.Command{
	Use:   "spec",
	Short: "Create a feature spec using AI-assisted wizard",
	Long: `Create a feature specification using an AI-assisted interview wizard.

The spec command launches an interactive wizard that collects basic information
about your feature, then uses an AI agent to interview you in depth about
requirements, edge cases, and technical implementation details. The agent
generates a complete spec file and updates the specs README.

The wizard will:
1. Ask for a spec name (slug format)
2. Collect a detailed description
3. Select an AI model to use
4. Interview you interactively about requirements
5. Generate and save the complete spec file`,
	RunE: runSpec,
}

func runSpec(cmd *cobra.Command, args []string) error {
	// Load config via Viper
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if config exists or if model is set via ENV var
	if !config.Exists() && cfg.Model == "" {
		return fmt.Errorf("no configuration found\n\nRun 'iteratr setup' to create a config file, or set ITERATR_MODEL environment variable")
	}

	// Validate that model is set (required for AI agent)
	if cfg.Model == "" {
		return fmt.Errorf("model not configured\n\nSet model via:\n  - iteratr setup (creates config file)\n  - ITERATR_MODEL environment variable")
	}

	// TODO: Launch wizard
	// The wizard will handle:
	// 1. Name input step
	// 2. Description textarea step
	// 3. Model selection step
	// 4. Agent phase (spawn opencode acp with MCP server)
	// 5. Question handling (ask-questions tool)
	// 6. Spec generation (finish-spec tool)
	// 7. Completion step (View/Start Build/Exit)

	return fmt.Errorf("spec wizard not yet implemented")
}
