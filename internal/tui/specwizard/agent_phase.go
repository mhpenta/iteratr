package specwizard

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/agent"
	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// AgentPhase manages the agent interview phase where the AI agent
// asks questions and generates the spec. It spawns an opencode acp
// subprocess and communicates via the iteratr-spec MCP server.
type AgentPhase struct {
	width  int
	height int

	// Agent state
	runner       *agent.Runner      // opencode acp subprocess runner
	runnerCtx    context.Context    // Context for runner lifecycle
	runnerCancel context.CancelFunc // Cancel function for runner
	status       string             // Current status text (e.g., "Agent is thinking...")
	spinner      *GradientSpinner   // Spinner animation while thinking
	isRunning    bool               // True if agent is currently active
	finished     bool               // True if agent has completed
	err          error              // Error if agent failed

	// Interview state from wizard
	name        string // Spec name
	description string // Spec description
	model       string // Selected model
	specDir     string // Spec directory from config
	mcpURL      string // MCP server URL
}

// NewAgentPhase creates a new agent phase instance.
// mcpURL is the URL for the iteratr-spec MCP server.
func NewAgentPhase(name, description, model, specDir, mcpURL string) *AgentPhase {
	ctx, cancel := context.WithCancel(context.Background())
	return &AgentPhase{
		name:         name,
		description:  description,
		model:        model,
		specDir:      specDir,
		mcpURL:       mcpURL,
		runnerCtx:    ctx,
		runnerCancel: cancel,
		status:       "Starting agent...",
	}
}

// GradientSpinner is a placeholder for the spinner type.
// TODO: Import from internal/tui package once we implement the spinner.
type GradientSpinner struct {
	// Placeholder fields
}

// AgentPhaseMsg is sent by the agent phase to communicate events.
type AgentPhaseMsg struct {
	Type    string // "text", "thinking", "finished", "error"
	Content string // Message content
	Error   error  // Error if Type == "error"
}

// Init initializes the agent phase.
func (a *AgentPhase) Init() tea.Cmd {
	// Start spinner animation
	spinner := NewDefaultGradientSpinner("Starting agent...")
	a.spinner = &spinner

	// Start the agent runner in a goroutine
	return tea.Batch(
		a.spinner.Tick(),
		a.startAgent,
	)
}

// startAgent spawns the opencode acp subprocess and starts the interview.
func (a *AgentPhase) startAgent() tea.Msg {
	logger.Debug("Starting agent phase for spec: %s", a.name)

	// Get current working directory for agent
	workDir := "."

	// Create runner config for stateless spec wizard agent
	cfg := agent.RunnerConfig{
		Model:         a.model,
		WorkDir:       workDir,
		SessionName:   "", // No session persistence for spec wizard
		NATSPort:      0,  // No NATS for spec wizard
		MCPServerURL:  a.mcpURL,
		MCPServerName: "iteratr-spec", // Spec wizard uses iteratr-spec MCP server
		OnText: func(text string) {
			// Text output from agent (not used in spec wizard - agent output hidden)
			logger.Debug("Agent text: %s", text)
		},
		OnToolCall: func(event agent.ToolCallEvent) {
			// Tool calls from agent (ask-questions, finish-spec handled by MCP server)
			logger.Debug("Agent tool call: %s (%s)", event.Title, event.Status)
		},
		OnThinking: func(text string) {
			// Thinking/reasoning from agent - update status
			logger.Debug("Agent thinking: %s", text)
			// TODO: Send thinking msg to update UI status
		},
		OnFinish: func(event agent.FinishEvent) {
			logger.Debug("Agent finished: %s", event.StopReason)
			// TODO: Send finish msg
		},
		OnFileChange: nil, // No file change tracking for spec wizard
	}

	a.runner = agent.NewRunner(cfg)

	// Start the ACP subprocess
	if err := a.runner.Start(a.runnerCtx); err != nil {
		logger.Error("Failed to start agent: %v", err)
		return AgentPhaseMsg{
			Type:  "error",
			Error: fmt.Errorf("failed to start agent: %w", err),
		}
	}

	a.isRunning = true
	a.status = "Agent is analyzing requirements..."

	// Build the agent prompt from the spec template
	prompt := a.buildAgentPrompt()

	// Run the iteration (agent will interview and generate spec)
	go func() {
		if err := a.runner.RunIteration(a.runnerCtx, prompt, ""); err != nil {
			logger.Error("Agent iteration failed: %v", err)
			// TODO: Send error msg
		}
		a.isRunning = false
		a.finished = true
		// TODO: Send finished msg
	}()

	return AgentPhaseMsg{
		Type:    "started",
		Content: "Agent started successfully",
	}
}

// buildAgentPrompt constructs the agent prompt for the spec wizard.
// Includes the feature name, description, and full spec format from AGENTS.md.
func (a *AgentPhase) buildAgentPrompt() string {
	// TODO: Read spec format from AGENTS.md and include it
	// For now, use a simplified prompt
	return fmt.Sprintf(`Follow the user instructions and interview me in detail using the ask-questions tool about literally anything: technical implementation, UI & UX, concerns, tradeoffs, etc. but make sure the questions are not obvious. Be very in-depth and continue interviewing me continually until it's complete. Then, write the spec using the finish-spec tool.

Feature: %s
Description: %s

## Spec Format

Each spec should include:
- **Overview** - What the feature does
- **User Story** - Who benefits and why
- **Requirements** - Detailed requirements gathered from stakeholders
- **Technical Implementation** - Routes, components, data flow
- **Tasks** - Byte-sized implementation tasks
- **UI Mockup** - ASCII or description of the interface
- **Out of Scope** - What's explicitly not included in v1
- **Open Questions** - Unresolved decisions for future discussion
`, a.name, a.description)
}

// Update handles messages for the agent phase.
func (a *AgentPhase) Update(msg tea.Msg) tea.Cmd {
	// Handle spinner animation
	if a.spinner != nil {
		if cmd := a.spinner.Update(msg); cmd != nil {
			return cmd
		}
	}

	// Handle agent phase messages
	switch msg := msg.(type) {
	case AgentPhaseMsg:
		switch msg.Type {
		case "error":
			a.err = msg.Error
			a.finished = true
			a.isRunning = false
			a.spinner = nil
			return nil
		case "finished":
			a.finished = true
			a.isRunning = false
			a.spinner = nil
			return nil
		}
	}

	return nil
}

// View renders the agent phase UI.
func (a *AgentPhase) View() string {
	var sections []string

	// Title
	title := "Spec Wizard - Interview"
	sections = append(sections, theme.Current().S().ModalTitle.Render(title))
	sections = append(sections, "")

	// Status with spinner (if running)
	if a.isRunning && a.spinner != nil {
		spinnerView := a.spinner.View()
		statusLine := fmt.Sprintf("%s %s", spinnerView, a.status)
		sections = append(sections, statusLine)
	} else if a.err != nil {
		// Error state
		errorMsg := theme.Current().S().Error.Render(fmt.Sprintf("Error: %v", a.err))
		sections = append(sections, errorMsg)
	} else if a.finished {
		// Finished state
		successMsg := theme.Current().S().Success.Render("Interview complete! Generating spec...")
		sections = append(sections, successMsg)
	}

	sections = append(sections, "")

	// Instructions
	instructions := theme.Current().S().Dim.Render("The agent will ask you questions about the feature. Answer honestly and in detail.")
	sections = append(sections, instructions)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Center content on screen
	return lipgloss.Place(a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		content,
	)
}

// SetSize updates the agent phase dimensions.
func (a *AgentPhase) SetSize(width, height int) {
	a.width = width
	a.height = height
}

// PreferredHeight returns the preferred content height for the agent phase.
func (a *AgentPhase) PreferredHeight() int {
	return 20 // Fixed height for agent phase
}

// Stop stops the agent phase and cleans up resources.
func (a *AgentPhase) Stop() {
	if a.runnerCancel != nil {
		a.runnerCancel()
	}
	if a.runner != nil {
		a.runner.Stop()
	}
}

// NewDefaultGradientSpinner creates a gradient spinner with a label.
// TODO: Move to internal/tui package and import here.
func NewDefaultGradientSpinner(label string) GradientSpinner {
	return GradientSpinner{}
}

// Tick returns a command to animate the spinner.
func (s *GradientSpinner) Tick() tea.Cmd {
	// TODO: Implement spinner animation
	return nil
}

// Update handles spinner animation messages.
func (s *GradientSpinner) Update(msg tea.Msg) tea.Cmd {
	// TODO: Implement spinner animation
	return nil
}

// View renders the spinner.
func (s *GradientSpinner) View() string {
	// TODO: Implement spinner rendering
	return "..."
}
