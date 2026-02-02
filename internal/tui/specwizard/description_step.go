package specwizard

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// DescriptionStep manages the spec description textarea UI step.
type DescriptionStep struct {
	textarea textarea.Model // Multi-line description input
	width    int            // Available width
	height   int            // Available height
}

// NewDescriptionStep creates a new description step.
func NewDescriptionStep() *DescriptionStep {
	// Initialize textarea
	ta := textarea.New()
	ta.Placeholder = "Provide as much detail as possible..."
	ta.Prompt = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0 // No character limit

	// Configure styles for textarea (using lipgloss v2)
	styles := textarea.Styles{
		Focused: textarea.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color("#b4befe")),
		},
		Blurred: textarea.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")),
		},
		Cursor: textarea.CursorStyle{
			Color: lipgloss.Color("#cba6f7"),
			Shape: tea.CursorBar,
			Blink: true,
		},
	}
	ta.SetStyles(styles)
	ta.SetWidth(50)
	ta.SetHeight(8)

	return &DescriptionStep{
		textarea: ta,
		width:    60,
		height:   20,
	}
}

// Init initializes the description step and focuses the textarea.
func (d *DescriptionStep) Init() tea.Cmd {
	return d.textarea.Focus()
}

// Focus gives focus to the description step.
func (d *DescriptionStep) Focus() tea.Cmd {
	return d.textarea.Focus()
}

// Blur removes focus from the description step textarea.
func (d *DescriptionStep) Blur() {
	d.textarea.Blur()
}

// SetSize updates the dimensions for the description step.
func (d *DescriptionStep) SetSize(width, height int) {
	d.width = width
	d.height = height

	// Reserve space for label, hint, and spacing (5 lines)
	textareaHeight := height - 5
	if textareaHeight < 5 {
		textareaHeight = 5
	}

	d.textarea.SetWidth(width - 10)
	d.textarea.SetHeight(textareaHeight)
}

// Update handles messages for the description step.
func (d *DescriptionStep) Update(msg tea.Msg) tea.Cmd {
	// Handle keyboard input
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "tab":
			// Signal to move to buttons
			return func() tea.Msg {
				return TabExitForwardMsg{}
			}

		case "shift+tab":
			// Signal to move to buttons from end
			return func() tea.Msg {
				return TabExitBackwardMsg{}
			}

		case "ctrl+d":
			// Ctrl+D signals completion (common in CLI workflows)
			if d.IsValid() {
				return func() tea.Msg {
					return DescriptionCompleteMsg{}
				}
			}
			return nil
		}
	}

	// Forward messages to textarea
	var cmd tea.Cmd
	d.textarea, cmd = d.textarea.Update(msg)

	return cmd
}

// View renders the description step.
func (d *DescriptionStep) View() string {
	var b strings.Builder

	// Label
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))
	b.WriteString(labelStyle.Render("Feature Description"))
	b.WriteString("\n")

	// Hint text
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).Faint(true)
	b.WriteString(hintStyle.Render("(provide as much detail as possible)"))
	b.WriteString("\n\n")

	// Textarea
	b.WriteString(d.textarea.View())
	b.WriteString("\n")

	return b.String()
}

// IsValid returns true if the description is not empty.
func (d *DescriptionStep) IsValid() bool {
	desc := strings.TrimSpace(d.textarea.Value())
	return desc != ""
}

// Description returns the trimmed description text.
func (d *DescriptionStep) Description() string {
	return strings.TrimSpace(d.textarea.Value())
}

// PreferredHeight returns the preferred height for this step's content.
func (d *DescriptionStep) PreferredHeight() int {
	// Fixed content overhead:
	// - "Feature Description" label: 1
	// - hint text: 1
	// - blank line: 1
	// - textarea: 8 lines (default)
	// - blank line after: 1
	// Total: 12 lines
	return 12
}

// DescriptionCompleteMsg is sent when the description is complete and valid.
type DescriptionCompleteMsg struct{}
