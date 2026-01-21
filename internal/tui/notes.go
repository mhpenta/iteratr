package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/mark3labs/iteratr/internal/session"
)

// NotesPanel displays notes grouped by type with color-coding.
type NotesPanel struct {
	state  *session.State
	width  int
	height int
	offset int // Scroll offset for long note lists
}

// NewNotesPanel creates a new NotesPanel component.
func NewNotesPanel() *NotesPanel {
	return &NotesPanel{}
}

// Update handles messages for the notes panel.
func (n *NotesPanel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		key := msg.String()
		switch key {
		case "j", "down":
			n.offset++
			n.clampOffset()
		case "k", "up":
			n.offset--
			if n.offset < 0 {
				n.offset = 0
			}
		case "g":
			n.offset = 0
		case "G":
			n.scrollToBottom()
		}
	}
	return nil
}

// Render returns the notes panel view as a string.
func (n *NotesPanel) Render() string {
	if n.state == nil || len(n.state.Notes) == 0 {
		return styleEmptyState.Render("No notes recorded yet")
	}

	// Group notes by type
	notesByType := make(map[string][]*session.Note)
	for _, note := range n.state.Notes {
		notesByType[note.Type] = append(notesByType[note.Type], note)
	}

	var sections []string

	// Render notes by type in consistent order
	types := []string{"learning", "decision", "tip", "stuck"}
	for _, noteType := range types {
		notes := notesByType[noteType]
		if len(notes) == 0 {
			continue
		}

		// Render type header with color-coding
		header := n.renderTypeHeader(noteType, len(notes))
		sections = append(sections, header)

		// Render individual notes
		for _, note := range notes {
			noteStr := n.renderNote(note)
			sections = append(sections, noteStr)
		}

		// Add spacing between type groups
		sections = append(sections, "")
	}

	// Join all sections
	content := strings.Join(sections, "\n")

	// Handle scrolling if content exceeds available height
	lines := strings.Split(content, "\n")
	if len(lines) > n.height-2 {
		start := n.offset
		end := min(start+n.height-2, len(lines))
		lines = lines[start:end]

		// Add scroll indicator
		scrollInfo := fmt.Sprintf(" [%d-%d of %d] ", start+1, end, len(strings.Split(content, "\n")))
		lines = append(lines, styleDim.Render(scrollInfo))
	}

	return strings.Join(lines, "\n")
}

// renderTypeHeader renders a color-coded header for a note type.
func (n *NotesPanel) renderTypeHeader(noteType string, count int) string {
	var style lipgloss.Style
	var label string

	switch noteType {
	case "learning":
		style = styleNoteTypeLearning
		label = "LEARNING"
	case "stuck":
		style = styleNoteTypeStuck
		label = "STUCK"
	case "tip":
		style = styleNoteTypeTip
		label = "TIP"
	case "decision":
		style = styleNoteTypeDecision
		label = "DECISION"
	default:
		style = styleHighlight
		label = strings.ToUpper(noteType)
	}

	headerText := fmt.Sprintf("%s (%d)", label, count)
	return style.Render(headerText)
}

// renderNote renders a single note with iteration number.
func (n *NotesPanel) renderNote(note *session.Note) string {
	// Format iteration number
	iterStr := fmt.Sprintf("[#%d]", note.Iteration)
	iterFormatted := styleNoteIteration.Render(iterStr)

	// Format content with word wrapping if needed
	content := note.Content
	maxWidth := n.width - 10 // Reserve space for indent and iteration
	if len(content) > maxWidth {
		// Simple word wrapping - split on spaces
		words := strings.Fields(content)
		var lines []string
		var currentLine string

		for _, word := range words {
			if len(currentLine)+len(word)+1 <= maxWidth {
				if currentLine == "" {
					currentLine = word
				} else {
					currentLine += " " + word
				}
			} else {
				if currentLine != "" {
					lines = append(lines, currentLine)
				}
				currentLine = word
			}
		}
		if currentLine != "" {
			lines = append(lines, currentLine)
		}

		// Format first line with iteration number
		firstLine := fmt.Sprintf("  %s %s", iterFormatted, lines[0])
		result := []string{styleNoteContent.Render(firstLine)}

		// Format continuation lines without iteration number
		for i := 1; i < len(lines); i++ {
			contLine := fmt.Sprintf("      %s", lines[i])
			result = append(result, styleNoteContent.Render(contLine))
		}

		return strings.Join(result, "\n")
	}

	// Single line note
	noteStr := fmt.Sprintf("  %s %s", iterFormatted, content)
	return styleNoteContent.Render(noteStr)
}

// scrollToBottom scrolls to the bottom of the notes.
func (n *NotesPanel) scrollToBottom() {
	if n.state == nil {
		return
	}

	// Count total lines (headers + notes + spacing)
	notesByType := make(map[string][]*session.Note)
	for _, note := range n.state.Notes {
		notesByType[note.Type] = append(notesByType[note.Type], note)
	}

	totalLines := 0
	types := []string{"learning", "decision", "tip", "stuck"}
	for _, noteType := range types {
		notes := notesByType[noteType]
		if len(notes) > 0 {
			totalLines++ // header
			totalLines += len(notes)
			totalLines++ // spacing
		}
	}

	n.offset = max(0, totalLines-n.height+2)
}

// clampOffset ensures the offset doesn't exceed content bounds.
func (n *NotesPanel) clampOffset() {
	if n.state == nil {
		n.offset = 0
		return
	}

	// Count total lines
	notesByType := make(map[string][]*session.Note)
	for _, note := range n.state.Notes {
		notesByType[note.Type] = append(notesByType[note.Type], note)
	}

	totalLines := 0
	types := []string{"learning", "decision", "tip", "stuck"}
	for _, noteType := range types {
		notes := notesByType[noteType]
		if len(notes) > 0 {
			totalLines++ // header
			totalLines += len(notes)
			totalLines++ // spacing
		}
	}

	maxOffset := max(0, totalLines-n.height+2)
	if n.offset > maxOffset {
		n.offset = maxOffset
	}
}

// UpdateSize updates the notes panel dimensions.
func (n *NotesPanel) UpdateSize(width, height int) tea.Cmd {
	n.width = width
	n.height = height
	n.clampOffset()
	return nil
}

// UpdateState updates the notes panel with new session state.
func (n *NotesPanel) UpdateState(state *session.State) tea.Cmd {
	n.state = state
	n.clampOffset()
	return nil
}
