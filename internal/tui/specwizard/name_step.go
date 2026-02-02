package specwizard

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// NameStep manages the spec name input UI step.
type NameStep struct {
	input      textinput.Model // Spec name input
	validError string          // Validation error message
	width      int             // Available width
	height     int             // Available height
}

// NewNameStep creates a new name step.
func NewNameStep() *NameStep {
	// Initialize name input
	input := textinput.New()
	input.Placeholder = "my-feature-name"
	input.Prompt = ""

	// Configure styles for textinput (using lipgloss v2)
	styles := textinput.Styles{
		Focused: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4")),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color("#b4befe")),
		},
		Blurred: textinput.StyleState{
			Text:        lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Placeholder: lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")),
			Prompt:      lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")),
		},
		Cursor: textinput.CursorStyle{
			Color: lipgloss.Color("#cba6f7"),
			Shape: tea.CursorBar,
			Blink: true,
		},
	}
	input.SetStyles(styles)
	input.SetWidth(50)

	return &NameStep{
		input:  input,
		width:  60,
		height: 10,
	}
}

// Init initializes the name step and focuses the input.
func (n *NameStep) Init() tea.Cmd {
	return n.input.Focus()
}

// Focus gives focus to the name step.
func (n *NameStep) Focus() tea.Cmd {
	return n.input.Focus()
}

// Blur removes focus from the name step input.
func (n *NameStep) Blur() {
	n.input.Blur()
}

// SetSize updates the dimensions for the name step.
func (n *NameStep) SetSize(width, height int) {
	n.width = width
	n.height = height
	n.input.SetWidth(width - 10)
}

// Update handles messages for the name step.
func (n *NameStep) Update(msg tea.Msg) tea.Cmd {
	// Handle keyboard input
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "enter":
			// Validate and advance if valid
			if n.validate() {
				// Name is valid - this will be handled by parent wizard
				return func() tea.Msg {
					return NameCompleteMsg{}
				}
			}
			return nil

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
		}
	}

	// Forward messages to input
	var cmd tea.Cmd
	n.input, cmd = n.input.Update(msg)

	// Clear error on input change
	if _, ok := msg.(tea.KeyPressMsg); ok {
		n.validError = ""
	}

	return cmd
}

// View renders the name step.
func (n *NameStep) View() string {
	var b strings.Builder

	// Label
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))
	b.WriteString(labelStyle.Render("Spec Name"))
	b.WriteString("\n")

	// Hint text
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).Faint(true)
	b.WriteString(hintStyle.Render("(lowercase, hyphens only)"))
	b.WriteString("\n\n")

	// Input field
	b.WriteString(n.input.View())
	b.WriteString("\n")

	// Show validation error if present
	if n.validError != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
		b.WriteString(errorStyle.Render("✗ " + n.validError))
		b.WriteString("\n")
	}

	return b.String()
}

// validate validates the spec name input.
// Returns true if the name is valid, false otherwise.
// Sets validError if invalid.
func (n *NameStep) validate() bool {
	name := strings.TrimSpace(n.input.Value())

	// Check if empty
	if name == "" {
		n.validError = "Spec name cannot be empty"
		return false
	}

	// Check length (reasonable max for filenames)
	if len(name) > 100 {
		n.validError = "Spec name too long (max 100 characters)"
		return false
	}

	// Validate slug format: lowercase alphanumeric + hyphens only
	// Must not start or end with hyphen
	if name[0] == '-' || name[len(name)-1] == '-' {
		n.validError = "Name cannot start or end with a hyphen"
		return false
	}

	// Check each character
	for _, r := range name {
		// Only allow lowercase letters, digits, and hyphens
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			n.validError = "Use only lowercase letters, numbers, and hyphens"
			return false
		}
	}

	// Check for consecutive hyphens (not strictly required but good practice)
	if strings.Contains(name, "--") {
		n.validError = "Cannot contain consecutive hyphens"
		return false
	}

	// Valid!
	return true
}

// IsValid returns true if the current name is valid.
func (n *NameStep) IsValid() bool {
	return n.validate()
}

// Name returns the validated spec name.
func (n *NameStep) Name() string {
	return strings.TrimSpace(n.input.Value())
}

// PreferredHeight returns the preferred height for this step's content.
func (n *NameStep) PreferredHeight() int {
	// Fixed content lines:
	// - "Spec Name" label: 1
	// - hint text: 1
	// - blank line: 1
	// - input: 1
	// - error line (reserve even if not showing): 1
	// Total: 5 lines
	return 5
}

// NameCompleteMsg is sent when the name is complete and valid.
type NameCompleteMsg struct{}

// TabExitForwardMsg is sent when Tab is pressed.
// Parent should move focus to buttons.
type TabExitForwardMsg struct{}

// TabExitBackwardMsg is sent when Shift+Tab is pressed.
// Parent should move focus to buttons (from end).
type TabExitBackwardMsg struct{}
