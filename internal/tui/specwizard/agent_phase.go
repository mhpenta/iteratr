package specwizard

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/agent"
	"github.com/mark3labs/iteratr/internal/logger"
	"github.com/mark3labs/iteratr/internal/specmcp"
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

	// Question handling
	mcpServer         *specmcp.Server     // MCP server instance for question channel
	questionBatch     []*specmcp.Question // Current batch of questions from MCP
	currentQuestion   *specmcp.Question   // Current question being displayed
	questionView      *QuestionView       // View for displaying question
	questionIdx       int                 // Index of current question in batch
	totalQuestions    int                 // Total questions in current batch
	pendingAnswers    []any               // Collected answers for current batch
	currentAnswerCh   chan<- []any        // Channel to send answers back to MCP handler
	showingQuestion   bool                // True if currently displaying a question
	customAnswerMode  bool                // True if user wants to type custom answer
	customAnswerInput string              // Custom answer text input

	// Agent callback channel for sending events from goroutines
	msgChan agentPhaseMsgChan // Buffered channel for agent callbacks
}

// NewAgentPhase creates a new agent phase instance.
// mcpURL is the URL for the iteratr-spec MCP server.
// mcpServer is the MCP server instance used to receive questions via its QuestionChannel.
func NewAgentPhase(name, description, model, specDir, mcpURL string, mcpServer *specmcp.Server) *AgentPhase {
	ctx, cancel := context.WithCancel(context.Background())
	return &AgentPhase{
		name:           name,
		description:    description,
		model:          model,
		specDir:        specDir,
		mcpURL:         mcpURL,
		mcpServer:      mcpServer,
		runnerCtx:      ctx,
		runnerCancel:   cancel,
		status:         "Starting agent...",
		pendingAnswers: make([]any, 0),
		msgChan:        make(agentPhaseMsgChan, 10), // Buffered to prevent blocking callbacks
	}
}

// GradientSpinner is a placeholder for the spinner type.
// TODO: Import from internal/tui package once we implement the spinner.
type GradientSpinner struct {
	// Placeholder fields
}

// AgentPhaseMsg is sent by the agent phase to communicate events.
type AgentPhaseMsg struct {
	Type    string // "text", "thinking", "finished", "error", "started"
	Content string // Message content
	Error   error  // Error if Type == "error"
}

// agentPhaseMsgChan is a buffered channel for sending agent messages from callbacks.
// Callbacks run in goroutines, so we need a way to send tea.Msg back to the UI.
type agentPhaseMsgChan chan AgentPhaseMsg

// toCmd converts the channel to a tea.Cmd that waits for the next message.
func (ch agentPhaseMsgChan) toCmd() tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// QuestionReceivedMsg is sent when the MCP server receives questions from the agent.
type QuestionReceivedMsg struct {
	Request *specmcp.QuestionRequest
}

// Init initializes the agent phase.
func (a *AgentPhase) Init() tea.Cmd {
	// Start spinner animation
	spinner := NewDefaultGradientSpinner("Starting agent...")
	a.spinner = &spinner

	// Start the agent runner in a goroutine and listen for questions from MCP
	return tea.Batch(
		a.spinner.Tick(),
		a.startAgent,
		a.listenForQuestions(),
		a.msgChan.toCmd(), // Listen for agent callback messages
	)
}

// listenForQuestions listens for questions from the MCP server and sends them to the UI.
func (a *AgentPhase) listenForQuestions() tea.Cmd {
	return func() tea.Msg {
		select {
		case req := <-a.mcpServer.QuestionChannel():
			return QuestionReceivedMsg{Request: req}
		case <-a.runnerCtx.Done():
			return nil
		}
	}
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
			// Send text message to UI (non-blocking)
			select {
			case a.msgChan <- AgentPhaseMsg{Type: "text", Content: text}:
			default:
				logger.Warn("Agent message channel full, dropping text message")
			}
		},
		OnToolCall: func(event agent.ToolCallEvent) {
			// Tool calls from agent (ask-questions, finish-spec handled by MCP server)
			logger.Debug("Agent tool call: %s (%s)", event.Title, event.Status)
		},
		OnThinking: func(text string) {
			// Thinking/reasoning from agent - update status
			logger.Debug("Agent thinking: %s", text)
			// Send thinking message to update UI status (non-blocking)
			select {
			case a.msgChan <- AgentPhaseMsg{Type: "thinking", Content: text}:
			default:
				logger.Warn("Agent message channel full, dropping thinking message")
			}
		},
		OnFinish: func(event agent.FinishEvent) {
			logger.Debug("Agent finished: %s", event.StopReason)
			// Send finish message
			var msg AgentPhaseMsg
			if event.Error != "" {
				msg = AgentPhaseMsg{
					Type:  "error",
					Error: fmt.Errorf("%s", event.Error),
				}
			} else {
				msg = AgentPhaseMsg{
					Type:    "finished",
					Content: event.StopReason,
				}
			}
			select {
			case a.msgChan <- msg:
			default:
				logger.Warn("Agent message channel full, dropping finish message")
			}
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
	// Handle custom answer input mode
	if a.customAnswerMode {
		return a.handleCustomAnswerInput(msg)
	}

	// Handle question view if showing question
	if a.showingQuestion && a.questionView != nil {
		cmd := a.questionView.Update(msg)

		// Check for answer submission
		switch msg := msg.(type) {
		case AnswerSelectedMsg:
			// Single answer received
			a.pendingAnswers = append(a.pendingAnswers, msg.Answer)
			return a.moveToNextQuestion()
		case MultiAnswerSelectedMsg:
			// Multiple answers received
			a.pendingAnswers = append(a.pendingAnswers, msg.Answers)
			return a.moveToNextQuestion()
		case CustomAnswerRequestedMsg:
			// User wants to type custom answer
			a.customAnswerMode = true
			a.customAnswerInput = ""
			return nil
		}

		return cmd
	}

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
		case "thinking":
			// Update status with thinking text (truncate if too long)
			thinkingText := msg.Content
			if len(thinkingText) > 80 {
				thinkingText = thinkingText[:77] + "..."
			}
			a.status = fmt.Sprintf("Agent: %s", thinkingText)
			// Continue listening for more messages
			return a.msgChan.toCmd()

		case "text":
			// Text output from agent (hidden from user in spec wizard, just log)
			logger.Debug("Agent text: %s", msg.Content)
			// Continue listening for more messages
			return a.msgChan.toCmd()

		case "error":
			a.err = msg.Error
			a.finished = true
			a.isRunning = false
			a.spinner = nil
			// Continue listening (but won't receive more since finished)
			return a.msgChan.toCmd()

		case "finished":
			a.finished = true
			a.isRunning = false
			a.spinner = nil
			a.status = "Interview complete! Generating spec..."
			// Continue listening (but won't receive more since finished)
			return a.msgChan.toCmd()

		case "started":
			// Agent started successfully - continue listening
			return a.msgChan.toCmd()
		}

	case QuestionReceivedMsg:
		// New questions batch received from MCP server
		logger.Debug("Received %d questions from agent", len(msg.Request.Questions))
		a.questionBatch = msg.Request.Questions
		a.currentAnswerCh = msg.Request.AnswerCh
		a.questionIdx = 0
		a.totalQuestions = len(msg.Request.Questions)
		a.pendingAnswers = make([]any, 0, a.totalQuestions)

		// Display first question
		return a.showQuestion(msg.Request.Questions[0])
	}

	return nil
}

// handleCustomAnswerInput handles keyboard input in custom answer mode.
func (a *AgentPhase) handleCustomAnswerInput(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "enter":
			// Submit custom answer if non-empty
			if a.customAnswerInput != "" {
				a.pendingAnswers = append(a.pendingAnswers, a.customAnswerInput)
				a.customAnswerMode = false
				return a.moveToNextQuestion()
			}
			// Empty input - do nothing
			return nil

		case "esc":
			// Cancel custom answer - go back to question view
			a.customAnswerMode = false
			return nil

		case "backspace":
			if len(a.customAnswerInput) > 0 {
				a.customAnswerInput = a.customAnswerInput[:len(a.customAnswerInput)-1]
			}
			return nil

		default:
			// Add character to input
			if len(keyMsg.String()) == 1 {
				a.customAnswerInput += keyMsg.String()
			}
			return nil
		}
	}
	return nil
}

// showQuestion displays a question in the UI.
func (a *AgentPhase) showQuestion(q *specmcp.Question) tea.Cmd {
	a.currentQuestion = q
	a.questionView = NewQuestionView(q)
	a.questionView.SetSize(a.width, a.height)
	a.showingQuestion = true
	a.spinner = nil // Hide spinner while showing question

	logger.Debug("Showing question: %s (%d/%d)", q.Header, a.questionIdx+1, a.totalQuestions)
	return nil
}

// moveToNextQuestion moves to the next question or completes the batch.
func (a *AgentPhase) moveToNextQuestion() tea.Cmd {
	a.questionIdx++

	if a.questionIdx < a.totalQuestions {
		// More questions in this batch - show next
		logger.Debug("Moving to next question (%d/%d)", a.questionIdx+1, a.totalQuestions)
		return a.showQuestion(a.questionBatch[a.questionIdx])
	}

	// All questions answered - send answers back to MCP handler
	logger.Debug("All questions answered, sending %d answers back to MCP", len(a.pendingAnswers))

	// Send answers through the channel (non-blocking)
	go func() {
		a.currentAnswerCh <- a.pendingAnswers
	}()

	// Reset question state and show spinner again
	a.showingQuestion = false
	a.questionView = nil
	a.currentQuestion = nil
	a.currentAnswerCh = nil
	a.status = "Agent is analyzing your answers..."
	spinner := NewDefaultGradientSpinner(a.status)
	a.spinner = &spinner

	// Continue listening for more questions and agent messages
	return tea.Batch(
		a.spinner.Tick(),
		a.listenForQuestions(),
		a.msgChan.toCmd(),
	)
}

// View renders the agent phase UI.
func (a *AgentPhase) View() string {
	// Show custom answer input mode
	if a.customAnswerMode {
		return a.renderCustomAnswerInput()
	}

	// Show question view if displaying a question
	if a.showingQuestion && a.questionView != nil {
		return a.questionView.View()
	}

	// Show spinner/status while agent is thinking
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

// renderCustomAnswerInput renders the custom answer input view.
func (a *AgentPhase) renderCustomAnswerInput() string {
	var sections []string

	t := theme.Current()

	// Title
	title := "Spec Wizard - Interview"
	sections = append(sections, t.S().ModalTitle.Render(title))
	sections = append(sections, "")

	// Question header
	if a.currentQuestion != nil {
		headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)
		sections = append(sections, headerStyle.Render(a.currentQuestion.Header))
		sections = append(sections, "")

		// Question text
		questionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
		sections = append(sections, questionStyle.Render(a.currentQuestion.Question))
		sections = append(sections, "")
	}

	// Custom answer label
	label := "Type your answer:"
	sections = append(sections, label)
	sections = append(sections, "")

	// Text input box
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Primary)).
		Padding(0, 1).
		Width(a.width - 10)

	inputContent := a.customAnswerInput
	if inputContent == "" {
		inputContent = t.S().Dim.Render("(type your answer...)")
	}
	sections = append(sections, inputStyle.Render(inputContent))
	sections = append(sections, "")

	// Instructions
	instructions := t.S().Dim.Render("Press Enter to submit • ESC to go back")
	sections = append(sections, instructions)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

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
