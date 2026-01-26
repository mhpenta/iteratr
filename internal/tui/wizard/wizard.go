package wizard

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	lipglossv2 "charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// WizardResult holds the output values from the wizard.
// These are applied to buildFlags before orchestrator creation.
type WizardResult struct {
	SpecPath    string // Path to selected spec file
	Model       string // Selected model ID (e.g. "anthropic/claude-sonnet-4-5")
	Template    string // Full edited template content
	SessionName string // Validated session name
	Iterations  int    // Max iterations (0 = infinite)
}

// WizardModel is the main BubbleTea model for the build wizard.
// It manages the four-step flow: file picker → model selector → template editor → config.
type WizardModel struct {
	step      int          // Current step (0-3)
	cancelled bool         // User cancelled via ESC
	result    WizardResult // Accumulated result from each step
	width     int          // Terminal width
	height    int          // Terminal height

	// Step components (initialized lazily)
	// filePickerStep   *FilePickerStep
	// modelSelectorStep *ModelSelectorStep
	// templateEditorStep *TemplateEditorStep
	// configStep *ConfigStep
}

// RunWizard is the entry point for the build wizard.
// It creates a standalone BubbleTea program, runs it, and returns the result.
// Returns nil result and error if user cancels or an error occurs.
func RunWizard() (*WizardResult, error) {
	// Create initial model
	m := &WizardModel{
		step:      0,
		cancelled: false,
	}

	// Create BubbleTea program
	p := tea.NewProgram(m)

	// Run the program
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard failed: %w", err)
	}

	// Extract result from final model
	wizModel, ok := finalModel.(*WizardModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}

	// Check if user cancelled
	if wizModel.cancelled {
		return nil, fmt.Errorf("wizard cancelled by user")
	}

	return &wizModel.result, nil
}

// Init initializes the wizard model.
func (m *WizardModel) Init() tea.Cmd {
	// TODO: Initialize first step (file picker)
	return nil
}

// Update handles messages for the wizard.
func (m *WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Global keybindings
		switch msg.String() {
		case "ctrl+c":
			// Always allow Ctrl+C to quit
			m.cancelled = true
			return m, tea.Quit
		case "esc":
			// ESC behavior depends on step
			if m.step == 0 {
				// On first step, confirm exit
				// TODO: Show confirmation dialog
				m.cancelled = true
				return m, tea.Quit
			} else {
				// On other steps, go back
				m.step--
				return m, nil
			}
		case "ctrl+enter":
			// Ctrl+Enter finishes wizard if all steps valid
			// TODO: Validate all steps and quit if valid
			if m.isComplete() {
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		// Store terminal dimensions
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}

	// Forward to current step
	// TODO: Forward messages to active step component
	return m, nil
}

// View renders the wizard UI.
func (m *WizardModel) View() tea.View {
	var view tea.View
	view.AltScreen = true
	view.KeyboardEnhancements = tea.KeyboardEnhancements{
		ReportEventTypes: true, // Required for ctrl+enter
	}

	// TODO: Render current step in modal container using canvas
	// For now, render a simple placeholder
	content := fmt.Sprintf("Build Wizard - Step %d of 4\n\nWidth: %d, Height: %d\n\nPress ESC to cancel", m.step+1, m.width, m.height)

	canvas := uv.NewScreenBuffer(m.width, m.height)
	uv.NewStyledString(content).Draw(canvas, uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: m.width, Y: m.height},
	})

	view.Content = lipglossv2.NewLayer(canvas.Render())
	return view
}

// isComplete checks if all required steps have valid data.
func (m *WizardModel) isComplete() bool {
	// TODO: Validate each step
	return m.result.SpecPath != "" &&
		m.result.Model != "" &&
		m.result.Template != "" &&
		m.result.SessionName != ""
}
