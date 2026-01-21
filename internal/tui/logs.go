package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/mark3labs/iteratr/internal/session"
)

// LogViewer displays scrollable event history with color-coding.
type LogViewer struct {
	state  *session.State
	events []session.Event // Live event stream
	width  int
	height int
	offset int // Scroll offset (number of lines scrolled from top)
}

// NewLogViewer creates a new LogViewer component.
func NewLogViewer() *LogViewer {
	return &LogViewer{}
}

// Update handles messages for the log viewer.
func (l *LogViewer) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		key := msg.String()
		switch key {
		case "j", "down":
			// Scroll down
			l.scrollDown()
		case "k", "up":
			// Scroll up
			l.scrollUp()
		case "g":
			// Go to top
			l.offset = 0
		case "G":
			// Go to bottom
			l.scrollToBottom()
		case "d", "ctrl+d":
			// Half page down
			l.offset += l.height / 2
			l.clampOffset()
		case "u", "ctrl+u":
			// Half page up
			l.offset -= l.height / 2
			if l.offset < 0 {
				l.offset = 0
			}
		}
	}
	return nil
}

// scrollDown scrolls down by one line.
func (l *LogViewer) scrollDown() {
	l.offset++
	l.clampOffset()
}

// scrollUp scrolls up by one line.
func (l *LogViewer) scrollUp() {
	l.offset--
	if l.offset < 0 {
		l.offset = 0
	}
}

// scrollToBottom scrolls to the bottom of the log.
func (l *LogViewer) scrollToBottom() {
	totalLines := len(l.events)
	l.offset = max(0, totalLines-l.height+2) // -2 for padding
}

// clampOffset ensures the offset doesn't exceed the content bounds.
func (l *LogViewer) clampOffset() {
	totalLines := len(l.events)
	maxOffset := max(0, totalLines-l.height+2) // -2 for padding
	if l.offset > maxOffset {
		l.offset = maxOffset
	}
}

// Render returns the log viewer view as a string.
func (l *LogViewer) Render() string {
	if len(l.events) == 0 {
		return styleEmptyState.Render("No events yet")
	}

	var lines []string

	// Calculate visible range based on scroll offset
	visibleHeight := l.height - 2 // Account for padding
	start := l.offset
	end := min(start+visibleHeight, len(l.events))

	// Render visible events
	for i := start; i < end; i++ {
		event := l.events[i]
		lines = append(lines, l.renderEvent(event))
	}

	// Add scroll indicator if there's more content
	totalLines := len(l.events)
	if totalLines > visibleHeight {
		scrollInfo := fmt.Sprintf(" [%d-%d of %d] ", start+1, end, totalLines)
		lines = append(lines, styleDim.Render(scrollInfo))
	}

	return strings.Join(lines, "\n")
}

// renderEvent renders a single event with appropriate styling.
func (l *LogViewer) renderEvent(event session.Event) string {
	// Format timestamp
	timestamp := event.Timestamp.Format("15:04:05")
	timestampStr := styleLogTimestamp.Render(timestamp)

	// Choose style based on event type
	var typeStyle lipgloss.Style
	var typeLabel string

	switch event.Type {
	case "task":
		typeStyle = styleLogTask
		typeLabel = "TASK"
	case "note":
		typeStyle = styleLogNote
		typeLabel = "NOTE"
	case "inbox":
		typeStyle = styleLogInbox
		typeLabel = "INBOX"
	case "iteration":
		typeStyle = styleLogIteration
		typeLabel = "ITER"
	case "control":
		typeStyle = styleLogControl
		typeLabel = "CTRL"
	default:
		typeStyle = styleLogContent
		typeLabel = "EVENT"
	}

	typeStr := typeStyle.Render(fmt.Sprintf("[%s]", typeLabel))

	// Format action
	actionStr := styleDim.Render(event.Action)

	// Format content (truncate if too long)
	content := event.Data
	maxContentWidth := l.width - 30 // Reserve space for timestamp, type, action
	if len(content) > maxContentWidth {
		content = content[:maxContentWidth-3] + "..."
	}
	contentStr := styleLogContent.Render(content)

	return fmt.Sprintf("%s %s %-10s %s", timestampStr, typeStr, actionStr, contentStr)
}

// UpdateSize updates the log viewer dimensions.
func (l *LogViewer) UpdateSize(width, height int) tea.Cmd {
	l.width = width
	l.height = height
	l.clampOffset() // Recalculate offset bounds after resize
	return nil
}

// UpdateState updates the log viewer with new session state.
func (l *LogViewer) UpdateState(state *session.State) tea.Cmd {
	l.state = state
	return nil
}

// AddEvent adds a new event to the log viewer.
// This is called when real-time events are received from NATS.
func (l *LogViewer) AddEvent(event session.Event) tea.Cmd {
	l.events = append(l.events, event)
	// Auto-scroll to bottom when new event arrives
	l.scrollToBottom()
	return nil
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
