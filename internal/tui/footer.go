package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	uv "github.com/charmbracelet/ultraviolet"
)

// Footer renders the bottom footer bar with navigation hints.
type Footer struct {
	width      int
	activeView ViewType
	layoutMode LayoutMode
}

// NewFooter creates a new Footer component.
func NewFooter() *Footer {
	return &Footer{
		layoutMode: LayoutDesktop,
	}
}

// Draw renders the footer to the screen at the given area.
// Returns nil cursor since footer is non-interactive.
func (f *Footer) Draw(scr uv.Screen, area uv.Rectangle) *tea.Cursor {
	if area.Dy() < 1 {
		return nil
	}

	// Build footer content based on available width
	content := f.buildFooterContent(area.Dx())

	// Render to screen using DrawStyled helper
	DrawStyled(scr, area, styleFooter, content)

	return nil
}

// buildFooterContent creates the footer text with navigation hints.
func (f *Footer) buildFooterContent(availableWidth int) string {
	var parts []string

	// View navigation shortcuts
	views := []struct {
		key  string
		name string
		view ViewType
	}{
		{"1", "Dashboard", ViewDashboard},
		{"2", "Logs", ViewLogs},
		{"3", "Notes", ViewNotes},
		{"4", "Inbox", ViewInbox},
	}

	for _, v := range views {
		key := styleFooterKey.Render(fmt.Sprintf("[%s]", v.key))
		var label string
		if v.view == f.activeView {
			// Highlight active view
			label = styleFooterActive.Render(v.name)
		} else {
			label = styleFooterLabel.Render(v.name)
		}
		parts = append(parts, key+" "+label)
	}

	// In compact mode, add sidebar toggle hint
	if f.layoutMode == LayoutCompact {
		sidebarHint := styleFooterKey.Render("[s]") + styleFooterLabel.Render("Sidebar")
		parts = append(parts, sidebarHint)
	}

	// Add help and quit hints
	helpHint := styleFooterKey.Render("[?]") + styleFooterLabel.Render("Help")
	quitHint := styleFooterKey.Render("[q]") + styleFooterLabel.Render("Quit")

	// Build left side (view navigation + optional sidebar toggle)
	leftParts := parts
	left := strings.Join(leftParts, "  ")

	// Build right side (help + quit)
	rightParts := []string{helpHint, quitHint}
	right := strings.Join(rightParts, "  ")

	// Calculate spacing to fill width
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := availableWidth - leftWidth - rightWidth - 2 // -2 for side padding
	if padding < 2 {
		padding = 2
	}

	// Combine with spacing
	content := left + strings.Repeat(" ", padding) + right

	// If content is too wide, use condensed version
	if lipgloss.Width(content) > availableWidth {
		content = f.buildCondensedContent(availableWidth)
	}

	return content
}

// buildCondensedContent creates a shorter version for narrow terminals.
func (f *Footer) buildCondensedContent(availableWidth int) string {
	// Minimal version: [1-4]Views [?]Help [q]Quit
	views := styleFooterKey.Render("[1-4]") + styleFooterLabel.Render("Views")
	help := styleFooterKey.Render("[?]") + styleFooterLabel.Render("Help")
	quit := styleFooterKey.Render("[q]") + styleFooterLabel.Render("Quit")

	parts := []string{views, help, quit}
	content := strings.Join(parts, " ")

	// If still too wide, use ultra-minimal version
	if lipgloss.Width(content) > availableWidth {
		content = styleFooterKey.Render("[1-4]") + " " +
			styleFooterKey.Render("[?]") + " " +
			styleFooterKey.Render("[q]")
	}

	return content
}

// SetSize updates the footer width.
func (f *Footer) SetSize(width, height int) {
	f.width = width
}

// SetActiveView updates which view is currently active.
func (f *Footer) SetActiveView(view ViewType) {
	f.activeView = view
}

// SetLayoutMode updates the layout mode (desktop/compact).
func (f *Footer) SetLayoutMode(mode LayoutMode) {
	f.layoutMode = mode
}

// Update handles messages. Footer is mostly static.
func (f *Footer) Update(msg tea.Msg) tea.Cmd {
	return nil
}

// Compile-time interface check
var _ Component = (*Footer)(nil)
