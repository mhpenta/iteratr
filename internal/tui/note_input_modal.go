package tui

import (
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

// NoteInputModal is an interactive modal for creating new notes.
// It displays a textarea for content input and allows the user to submit notes.
type NoteInputModal struct {
	visible  bool
	textarea textarea.Model
	noteType string // Current selected type (hardcoded to "learning" for now)
	width    int
	height   int
}

// NewNoteInputModal creates a new NoteInputModal component.
func NewNoteInputModal() *NoteInputModal {
	// Create and configure textarea
	ta := textarea.New()
	ta.Placeholder = "Enter your note..."
	ta.CharLimit = 500
	ta.ShowLineNumbers = false
	ta.Prompt = "" // No prompt character
	ta.SetWidth(50)
	ta.SetHeight(6)

	return &NoteInputModal{
		visible:  false,
		textarea: ta,
		noteType: "learning", // Hardcoded for tracer bullet
		width:    60,
		height:   16,
	}
}

// IsVisible returns whether the modal is currently visible.
func (m *NoteInputModal) IsVisible() bool {
	return m.visible
}

// Show makes the modal visible and focuses the textarea.
func (m *NoteInputModal) Show() tea.Cmd {
	m.visible = true
	return m.textarea.Focus()
}

// Close hides the modal.
func (m *NoteInputModal) Close() {
	m.visible = false
}
