package specwizard

import (
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// ButtonAction represents which action button was clicked.
type ButtonAction int

const (
	ButtonActionNone ButtonAction = iota
	ButtonActionView
	ButtonActionBuild
	ButtonActionExit
)

// CompletionStep is the final step showing success message and action buttons.
type CompletionStep struct {
	width    int
	height   int
	specPath string // Path to the saved spec file

	// Button management
	buttonFocused bool
	focusedIndex  int // 0=View, 1=Build, 2=Exit
}

// NewCompletionStep creates a new completion step.
func NewCompletionStep(specPath string) *CompletionStep {
	return &CompletionStep{
		specPath: specPath,
	}
}

// Init initializes the completion step.
func (m *CompletionStep) Init() tea.Cmd {
	m.buttonFocused = true
	m.focusedIndex = 0 // Start with View button focused
	return nil
}

// Update handles messages for the completion step.
func (m *CompletionStep) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Handle button-focused keyboard input
		if m.buttonFocused {
			switch msg.String() {
			case "tab", "right":
				m.focusedIndex = (m.focusedIndex + 1) % 3
				return nil
			case "shift+tab", "left":
				m.focusedIndex = (m.focusedIndex - 1 + 3) % 3
				return nil
			case "enter", " ":
				// Activate focused button
				action := m.getButtonAction()
				return m.executeAction(action)
			case "esc":
				// ESC exits
				return tea.Quit
			}
		}

	case tea.MouseClickMsg:
		// Mouse click handling could be added here in the future
		// For now, keyboard navigation is sufficient
	}
	return nil
}

// View renders the completion step.
func (m *CompletionStep) View() string {
	var sections []string

	// Success message
	successMsg := theme.Current().S().Success.Render("✓ Spec created successfully!")
	sections = append(sections, successMsg)
	sections = append(sections, "")

	// Spec file path
	pathLabel := theme.Current().S().ModalLabel.Render("Spec saved to:")
	pathValue := theme.Current().S().ModalValue.Render(m.specPath)
	sections = append(sections, fmt.Sprintf("%s\n%s", pathLabel, pathValue))
	sections = append(sections, "")

	// Action prompt
	prompt := theme.Current().S().ModalLabel.Render("What would you like to do?")
	sections = append(sections, prompt)
	sections = append(sections, "")

	// Render buttons
	sections = append(sections, m.renderButtons())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// SetSize updates the dimensions of the completion step.
func (m *CompletionStep) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// PreferredHeight returns the preferred content height for this step.
func (m *CompletionStep) PreferredHeight() int {
	// Success message: 1 line
	// Blank: 1 line
	// Path label: 1 line
	// Path value: 1 line
	// Blank: 1 line
	// Action prompt: 1 line
	// Blank: 1 line
	// Button bar: 1 line
	return 8
}

// renderButtons renders the three action buttons with focus styling.
func (m *CompletionStep) renderButtons() string {
	// Define button styles
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#cdd6f4")).
		Background(lipgloss.Color("#313244")).
		Padding(0, 2).
		MarginLeft(1).
		MarginRight(1)

	focusedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1e1e2e")).
		Background(lipgloss.Color("#b4befe")).
		Bold(true).
		Padding(0, 2).
		MarginLeft(1).
		MarginRight(1)

	// Button labels
	labels := []string{"View", "Start Build", "Exit"}

	// Render each button with appropriate style
	var buttons []string
	for i, label := range labels {
		if i == m.focusedIndex {
			buttons = append(buttons, focusedStyle.Render(label))
		} else {
			buttons = append(buttons, normalStyle.Render(label))
		}
	}

	// Join buttons and center them
	buttonRow := lipgloss.JoinHorizontal(lipgloss.Left, buttons...)
	return lipgloss.Place(m.width, 1, lipgloss.Center, lipgloss.Center, buttonRow)
}

// getButtonAction returns the action for the currently focused button.
func (m *CompletionStep) getButtonAction() ButtonAction {
	switch m.focusedIndex {
	case 0:
		return ButtonActionView
	case 1:
		return ButtonActionBuild
	case 2:
		return ButtonActionExit
	default:
		return ButtonActionNone
	}
}

// executeAction performs the action for the given button.
func (m *CompletionStep) executeAction(action ButtonAction) tea.Cmd {
	switch action {
	case ButtonActionView:
		return m.openInEditor()
	case ButtonActionBuild:
		return m.startBuild()
	case ButtonActionExit:
		return tea.Quit
	default:
		return nil
	}
}

// openInEditor opens the spec file in $EDITOR.
// If $EDITOR is not set, prints the file path instead.
func (m *CompletionStep) openInEditor() tea.Cmd {
	return func() tea.Msg {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			// No editor set - just print path and exit
			fmt.Printf("Spec saved to: %s\n", m.specPath)
			return tea.Quit()
		}

		// Open in editor
		cmd := exec.Command(editor, m.specPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Run editor and wait for it to exit
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error opening editor: %v\n", err)
		}

		return tea.Quit()
	}
}

// startBuild executes "iteratr build --spec <path>".
func (m *CompletionStep) startBuild() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("iteratr", "build", "--spec", m.specPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// Run build and wait for it to exit
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting build: %v\n", err)
		}

		return tea.Quit()
	}
}
