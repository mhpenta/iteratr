package tui

import (
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// TaskInputModal is an interactive modal for creating new tasks.
// It displays a textarea for content input, a priority selector, and allows the user to submit tasks.
type TaskInputModal struct {
	visible       bool
	textarea      textarea.Model // Bubbles v2 textarea
	priorityIndex int            // Current selected priority (0-4)
	width         int
	height        int
	buttonArea    uv.Rectangle // Hit area for mouse click on submit button
}

// NewTaskInputModal creates a new TaskInputModal component.
func NewTaskInputModal() *TaskInputModal {
	// Create and configure textarea
	ta := textarea.New()
	ta.Placeholder = "Describe the task..."
	ta.CharLimit = 500
	ta.ShowLineNumbers = false
	ta.Prompt = "" // No prompt character
	ta.SetWidth(50)
	ta.SetHeight(6)

	return &TaskInputModal{
		visible:       false,
		textarea:      ta,
		priorityIndex: 2, // Default to medium
		width:         60,
		height:        18, // Slightly taller than note modal to fit priority row
	}
}

// IsVisible returns whether the modal is currently visible.
func (m *TaskInputModal) IsVisible() bool {
	return m.visible
}

// Show makes the modal visible and focuses the textarea.
func (m *TaskInputModal) Show() tea.Cmd {
	m.visible = true
	return m.textarea.Focus()
}

// Close hides the modal and resets its state.
func (m *TaskInputModal) Close() {
	m.visible = false
	m.reset()
}

// reset clears the textarea and resets the modal to initial state.
// Called on both cancel (ESC) and submit to ensure clean state on next open.
func (m *TaskInputModal) reset() {
	// Clear textarea content
	m.textarea.SetValue("")

	// Reset priority to default (medium)
	m.priorityIndex = 2

	// Blur the textarea to reset its internal state
	m.textarea.Blur()
}
