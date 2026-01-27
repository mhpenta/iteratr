package tui

import (
	"context"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/mark3labs/iteratr/internal/agent"
)

// SubagentModal displays a full-screen modal that loads and replays a subagent session.
// It reuses the existing ScrollList and MessageItem infrastructure from AgentOutput.
type SubagentModal struct {
	// Content display (reuses AgentOutput infrastructure)
	scrollList *ScrollList
	messages   []MessageItem
	toolIndex  map[string]int // toolCallId → message index

	// Session metadata
	sessionID    string
	subagentType string
	workDir      string

	// ACP subprocess
	cmd *exec.Cmd
	// conn will be stored as interface{} to avoid exposing internal agent types
	// Actual implementation will use agent.acpConn internally

	// State
	loading bool
	err     error // Non-nil shows error message in modal
	width   int
	height  int

	// Spinner for loading state (created lazily when needed)
	spinner *GradientSpinner

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// NewSubagentModal creates a new SubagentModal.
func NewSubagentModal(sessionID, subagentType, workDir string) *SubagentModal {
	ctx, cancel := context.WithCancel(context.Background())
	spinner := NewDefaultGradientSpinner("Loading session...")
	return &SubagentModal{
		sessionID:    sessionID,
		subagentType: subagentType,
		workDir:      workDir,
		messages:     make([]MessageItem, 0),
		toolIndex:    make(map[string]int),
		loading:      true,
		ctx:          ctx,
		cancel:       cancel,
		spinner:      &spinner,
	}
}

// Start spawns the ACP subprocess, initializes it, and begins loading the session.
// Returns a command that will start the session loading process.
func (m *SubagentModal) Start() tea.Cmd {
	// This will be implemented in task TAS-16
	return nil
}

// Draw renders the modal as a full-screen overlay.
func (m *SubagentModal) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	// This will be implemented in task TAS-19
	return nil
}

// Update handles keyboard input for scrolling.
func (m *SubagentModal) Update(msg tea.Msg) tea.Cmd {
	// This will be implemented in task TAS-20
	return nil
}

// HandleUpdate processes streaming messages from the subagent session.
// Returns a command to continue streaming if Continue is true.
func (m *SubagentModal) HandleUpdate(msg tea.Msg) tea.Cmd {
	// This will be implemented in task TAS-17 (continuous streaming)
	return nil
}

// Close terminates the ACP subprocess and cleans up resources.
func (m *SubagentModal) Close() {
	// This will be implemented in task TAS-21
}

// appendText adds agent text to the modal's message list.
func (m *SubagentModal) appendText(content string) {
	// This will be implemented in task TAS-18
}

// appendToolCall adds or updates a tool call message.
func (m *SubagentModal) appendToolCall(event agent.ToolCallEvent) {
	// This will be implemented in task TAS-18
}

// appendThinking adds agent thinking content to the modal's message list.
func (m *SubagentModal) appendThinking(content string) {
	// This will be implemented in task TAS-18
}

// appendUserMessage adds a user message to the modal's message list.
func (m *SubagentModal) appendUserMessage(text string) {
	// This will be implemented in task TAS-18
}
