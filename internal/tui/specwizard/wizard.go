package specwizard

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/mark3labs/iteratr/internal/tui/theme"
	"github.com/mark3labs/iteratr/internal/tui/wizard"
)

// ContentChangedMsg is sent when a step's content changes in a way that affects preferred height.
// The wizard handles this by recalculating modal dimensions.
type ContentChangedMsg struct{}

// SpecWizardResult holds the output values from the spec wizard.
type SpecWizardResult struct {
	Name        string // Spec name (validated slug)
	Description string // Full description from textarea
	Model       string // Selected model ID (e.g. "anthropic/claude-sonnet-4-5")
	SpecPath    string // Path to saved spec file (set after agent completion)
}

// WizardModel is the main BubbleTea model for the spec wizard.
// It manages the three input steps, agent phase, and completion.
//
//nolint:unused
type WizardModel struct {
	step      int              // Current step (0-4: name, description, model, agent, completion)
	cancelled bool             // User cancelled via ESC
	result    SpecWizardResult // Accumulated result from each step
	width     int              // Terminal width
	height    int              // Terminal height
	specDir   string           // Spec directory from config

	// Confirmation state (for agent phase cancellation)
	confirmCancelling bool // True if showing "Are you sure you want to cancel?" modal
	confirmFocusYes   bool // True if "Yes" button focused, false if "No" focused

	// Step components
	nameStep        *NameStep                 // Step 0: Name input
	descriptionStep interface{}               // Step 1: Description textarea (placeholder) //nolint:unused
	modelStep       *wizard.ModelSelectorStep // Step 2: Model selector (reused from build wizard)
	agentPhase      interface{}               // Step 3: Agent phase (placeholder) //nolint:unused
	completionStep  *CompletionStep           // Step 4: Completion

	// Button bar with focus tracking (for input steps only)
	buttonBar     *wizard.ButtonBar // Current button bar instance
	buttonFocused bool              // True if buttons have focus (vs step content)
}

// RunWizard is the entry point for the spec wizard.
// It creates a standalone BubbleTea program, runs it, and returns the result.
// specDir is the spec directory from config.
// Returns nil result and error if user cancels or an error occurs.
func RunWizard(specDir string) (*SpecWizardResult, error) {
	// Create initial model
	m := &WizardModel{
		step:    0,
		specDir: specDir,
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
	// Initialize name step (step 0)
	m.nameStep = NewNameStep()
	return m.nameStep.Init()
}

// Update handles messages for the wizard.
func (m *WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Handle confirmation modal input first (overrides everything)
		if m.confirmCancelling {
			switch msg.String() {
			case "esc":
				// ESC dismisses the confirmation modal
				m.confirmCancelling = false
				return m, nil
			case "tab", "left", "right", "shift+tab":
				// Toggle focus between Yes/No
				m.confirmFocusYes = !m.confirmFocusYes
				return m, nil
			case "enter", " ":
				// Activate focused button
				if m.confirmFocusYes {
					// Yes - cancel the wizard
					m.cancelled = true
					return m, tea.Quit
				}
				// No - dismiss confirmation and continue
				m.confirmCancelling = false
				return m, nil
			case "y":
				// Quick "y" key to confirm
				m.cancelled = true
				m.confirmCancelling = false
				return m, tea.Quit
			case "n":
				// Quick "n" key to dismiss
				m.confirmCancelling = false
				return m, nil
			}
			// Any other key - ignore
			return m, nil
		}

		// Handle button-focused keyboard input (only for input steps 0-2)
		if m.buttonFocused && m.buttonBar != nil && m.step <= 2 {
			switch msg.String() {
			case "tab", "right":
				// Cycle to next button, wrap to content if at end
				if !m.buttonBar.FocusNext() {
					m.buttonFocused = false
					m.buttonBar.Blur()
					return m, m.focusStepContentFirst()
				}
				return m, nil
			case "shift+tab", "left":
				// Cycle to previous button, wrap to content if at start
				if !m.buttonBar.FocusPrev() {
					m.buttonFocused = false
					m.buttonBar.Blur()
					return m, m.focusStepContentLast()
				}
				return m, nil
			case "enter", " ":
				// Activate focused button
				return m.activateButton(m.buttonBar.FocusedButton())
			}
		}

		// Global keybindings
		switch msg.String() {
		case "ctrl+c":
			// Always allow Ctrl+C to quit
			m.cancelled = true
			return m, tea.Quit
		case "esc":
			// ESC behavior depends on step
			if m.step == 0 {
				// First step - cancel wizard
				m.cancelled = true
				return m, tea.Quit
			} else if m.step <= 2 {
				// Input steps - go back
				return m.goBack()
			} else if m.step == 3 {
				// Agent phase - show confirmation modal
				m.confirmCancelling = true
				m.confirmFocusYes = false // Default focus on "No"
				return m, nil
			} else if m.step == 4 {
				// Completion step - let it handle ESC (exits)
				return m, nil
			}
		case "tab":
			// Tab moves focus to buttons (unless already there or not an input step)
			if !m.buttonFocused && m.step <= 2 {
				m.buttonFocused = true
				m.blurStepContent()
				m.ensureButtonBar()
				m.buttonBar.FocusFirst() // Start at first button (Back/Cancel) for sequential cycling
				return m, nil
			}
		case "shift+tab":
			// Shift+Tab from content wraps to buttons (from the end)
			if !m.buttonFocused && m.step <= 2 {
				m.buttonFocused = true
				m.blurStepContent()
				m.ensureButtonBar()
				m.buttonBar.FocusLast() // Start at last button (Next) for reverse cycling
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		// Store terminal dimensions
		m.width = msg.Width
		m.height = msg.Height
		// Update size of current step
		m.updateCurrentStepSize()
		return m, nil

	case NameCompleteMsg:
		// Name entered in step 0 - auto-advance if valid
		if m.nameStep != nil && m.nameStep.IsValid() {
			m.result.Name = m.nameStep.Name()
			m.step++
			m.buttonFocused = false
			m.initCurrentStep()
			return m, nil
		}
		return m, nil

	case TabExitForwardMsg:
		// Tab pressed on input - move to buttons
		m.buttonFocused = true
		m.blurStepContent()
		m.ensureButtonBar()
		m.buttonBar.FocusFirst() // Start at first button (Back/Cancel) for sequential cycling
		return m, nil

	case TabExitBackwardMsg:
		// Shift+Tab pressed on input - move to buttons from end
		m.buttonFocused = true
		m.blurStepContent()
		m.ensureButtonBar()
		m.buttonBar.FocusLast() // Start at last button (Next) for reverse cycling
		return m, nil

	case ContentChangedMsg:
		// A step's content changed, recalculate modal dimensions
		m.updateCurrentStepSize()
		return m, nil

	case wizard.ModelSelectedMsg:
		// Model selected in step 2
		m.result.Model = msg.ModelID
		m.step++
		m.buttonFocused = false
		m.initCurrentStep()
		// TODO: Start agent phase
		return m, nil
	}

	// Forward to current step (only if not button focused)
	if m.buttonFocused {
		return m, nil
	}

	var cmd tea.Cmd
	switch m.step {
	case 0:
		if m.nameStep != nil {
			cmd = m.nameStep.Update(msg)
		}
	case 1:
		// TODO: Update description step
	case 2:
		if m.modelStep != nil {
			cmd = m.modelStep.Update(msg)
		}
	case 3:
		// TODO: Update agent phase
	case 4:
		if m.completionStep != nil {
			cmd = m.completionStep.Update(msg)
		}
	}

	return m, cmd
}

// activateButton performs the action for the given button.
func (m *WizardModel) activateButton(btnID wizard.ButtonID) (tea.Model, tea.Cmd) {
	switch btnID {
	case wizard.ButtonBack:
		if m.step == 0 {
			// Cancel wizard
			m.cancelled = true
			return m, tea.Quit
		}
		return m.goBack()
	case wizard.ButtonNext:
		return m.goNext()
	}
	return m, nil
}

// goBack returns to the previous step.
func (m *WizardModel) goBack() (tea.Model, tea.Cmd) {
	if m.step > 0 {
		m.step--
		m.buttonFocused = false
		m.initCurrentStep()
	}
	return m, nil
}

// goNext advances to the next step if valid.
func (m *WizardModel) goNext() (tea.Model, tea.Cmd) {
	if !m.isStepValid() {
		return m, nil
	}

	switch m.step {
	case 0:
		// Name step - save and advance
		if m.nameStep != nil && m.nameStep.IsValid() {
			m.result.Name = m.nameStep.Name()
			m.step++
			m.buttonFocused = false
			m.initCurrentStep()
			return m, nil
		}
	case 1:
		// Description step - save and advance
		// TODO: Implement description step
		m.step++
		m.buttonFocused = false
		m.initCurrentStep()
		return m, m.modelStep.Init()
	case 2:
		// Model step - save and advance
		if m.modelStep != nil {
			modelID := m.modelStep.SelectedModel()
			if modelID != "" {
				m.result.Model = modelID
				m.step++
				m.buttonFocused = false
				m.initCurrentStep()
				// TODO: Start agent phase
				return m, nil
			}
		}
	}
	return m, nil
}

// blurStepContent removes focus from the current step's content.
func (m *WizardModel) blurStepContent() {
	switch m.step {
	case 0:
		if m.nameStep != nil {
			m.nameStep.Blur()
		}
	case 1:
		// TODO: Blur description step
	}
}

// focusStepContentFirst gives focus to the current step's first focusable item.
func (m *WizardModel) focusStepContentFirst() tea.Cmd {
	switch m.step {
	case 0:
		if m.nameStep != nil {
			return m.nameStep.Focus()
		}
	case 1:
		// TODO: Focus description step
	}
	return nil
}

// focusStepContentLast gives focus to the current step's last focusable item.
func (m *WizardModel) focusStepContentLast() tea.Cmd {
	switch m.step {
	case 0:
		if m.nameStep != nil {
			return m.nameStep.Focus()
		}
	case 1:
		// TODO: Focus description step
	}
	return nil
}

// ensureButtonBar creates the button bar if it doesn't exist.
func (m *WizardModel) ensureButtonBar() {
	modalWidth := m.width - 6
	if modalWidth < 60 {
		modalWidth = 60
	}
	if modalWidth > 100 {
		modalWidth = 100
	}

	var buttons []wizard.Button
	nextLabel := "Next →"
	isValid := m.isStepValid()

	switch m.step {
	case 0:
		buttons = wizard.CreateCancelNextButtons(isValid, nextLabel)
	default:
		buttons = wizard.CreateBackNextButtons(true, isValid, nextLabel)
	}

	m.buttonBar = wizard.NewButtonBar(buttons)
	m.buttonBar.SetWidth(modalWidth)
}

// initCurrentStep initializes the current step component if not already initialized.
func (m *WizardModel) initCurrentStep() {
	switch m.step {
	case 0:
		if m.nameStep == nil {
			m.nameStep = NewNameStep()
		}
	case 1:
		// TODO: Initialize description step
	case 2:
		if m.modelStep == nil {
			m.modelStep = wizard.NewModelSelectorStep()
		}
	case 3:
		// TODO: Initialize agent phase
	case 4:
		if m.completionStep == nil && m.result.SpecPath != "" {
			m.completionStep = NewCompletionStep(m.result.SpecPath)
		}
	}
	m.updateCurrentStepSize()
}

// getStepPreferredHeight returns the preferred content height for the current step.
func (m *WizardModel) getStepPreferredHeight() int {
	switch m.step {
	case 0:
		if m.nameStep != nil {
			return m.nameStep.PreferredHeight()
		}
	case 1:
		// TODO: Return description step preferred height
		return 15
	case 2:
		if m.modelStep != nil {
			return m.modelStep.PreferredHeight()
		}
	case 3:
		// TODO: Return agent phase preferred height
		return 20
	case 4:
		if m.completionStep != nil {
			return m.completionStep.PreferredHeight()
		}
	}
	return 15 // Default fallback
}

// calculateModalDimensions calculates the modal dimensions based on terminal size and content.
// Returns modalWidth, modalHeight, contentWidth, contentHeight.
func (m *WizardModel) calculateModalDimensions() (int, int, int, int) {
	// Calculate modal width
	modalWidth := m.width - 6
	if modalWidth < 60 {
		modalWidth = 60
	}
	if modalWidth > 100 {
		modalWidth = 100
	}

	// Content width = modal width minus padding (2 each side) minus border (1 each side)
	contentWidth := modalWidth - 6
	if contentWidth < 40 {
		contentWidth = 40
	}

	// Calculate max modal height (don't overflow screen)
	maxModalHeight := m.height - 4
	if maxModalHeight < 15 {
		maxModalHeight = 15
	}

	// Modal overhead:
	// - padding top/bottom: 2
	// - border top/bottom: 2
	// - title line: 1
	// - blank after title: 1
	// - blank before buttons: 1
	// - button bar: 1 (for input steps 0-2 only)
	// Total overhead: 8 for input steps, 6 for agent/completion
	modalOverhead := 8
	if m.step >= 3 {
		modalOverhead = 6 // Agent phase and completion don't have button bars in modal
	}

	// Get preferred content height from current step
	preferredContentHeight := m.getStepPreferredHeight()

	// Calculate ideal modal height based on content
	idealModalHeight := preferredContentHeight + modalOverhead

	// Clamp modal height between min and max
	modalHeight := idealModalHeight
	if modalHeight > maxModalHeight {
		modalHeight = maxModalHeight
	}
	if modalHeight < 15 {
		modalHeight = 15
	}

	// Calculate actual content height
	contentHeight := modalHeight - modalOverhead
	if contentHeight < 5 {
		contentHeight = 5
	}

	return modalWidth, modalHeight, contentWidth, contentHeight
}

// updateCurrentStepSize updates the size of the current step component.
func (m *WizardModel) updateCurrentStepSize() {
	_, _, contentWidth, contentHeight := m.calculateModalDimensions()

	switch m.step {
	case 0:
		if m.nameStep != nil {
			m.nameStep.SetSize(contentWidth, contentHeight)
		}
	case 1:
		// TODO: Set size for description step
	case 2:
		if m.modelStep != nil {
			m.modelStep.SetSize(contentWidth, contentHeight)
		}
	case 3:
		// TODO: Set size for agent phase
	case 4:
		if m.completionStep != nil {
			m.completionStep.SetSize(contentWidth, contentHeight)
		}
	}
}

// View renders the wizard UI.
func (m *WizardModel) View() tea.View {
	var view tea.View
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion // Enable mouse clicks
	view.KeyboardEnhancements = tea.KeyboardEnhancements{
		ReportEventTypes: true, // Required for ctrl+enter if needed
	}

	// Render current step content
	var stepContent string
	switch m.step {
	case 0:
		if m.nameStep != nil {
			stepContent = m.nameStep.View()
		}
	case 1:
		// TODO: Render description step
		stepContent = "Description step (TODO)"
	case 2:
		if m.modelStep != nil {
			stepContent = m.modelStep.View()
		}
	case 3:
		// TODO: Render agent phase
		stepContent = "Agent phase (TODO)"
	case 4:
		if m.completionStep != nil {
			stepContent = m.completionStep.View()
		}
	}

	// Wrap in modal container with title (only for input steps 0-2)
	var content string
	if m.step <= 2 {
		content = m.renderModal(stepContent)
	} else if m.step == 4 {
		// Completion step renders its own modal
		content = m.renderModal(stepContent)
	} else {
		// Agent phase uses full screen
		content = stepContent
	}

	canvas := uv.NewScreenBuffer(m.width, m.height)
	uv.NewStyledString(content).Draw(canvas, uv.Rectangle{
		Min: uv.Position{X: 0, Y: 0},
		Max: uv.Position{X: m.width, Y: m.height},
	})

	// If confirmation modal is showing, overlay it on top
	if m.confirmCancelling {
		// Draw confirmation modal on the canvas
		confirmModal := m.renderConfirmationModal()
		uv.NewStyledString(confirmModal).Draw(canvas, uv.Rectangle{
			Min: uv.Position{X: 0, Y: 0},
			Max: uv.Position{X: m.width, Y: m.height},
		})
	}

	view.Content = lipgloss.NewLayer(canvas.Render())
	return view
}

// renderModal wraps the step content in a modal container with title, buttons, and step indicator.
func (m *WizardModel) renderModal(stepContent string) string {
	// Calculate modal dimensions dynamically based on content and terminal size
	modalWidth, modalHeight, _, _ := m.calculateModalDimensions()

	var sections []string

	// Title with step indicator and step name
	stepNames := []string{
		"Name",
		"Description",
		"Model",
		"Interview",
		"Complete",
	}
	totalSteps := 5
	if m.step == 4 {
		// Completion step doesn't show step indicator
		title := "Spec Wizard - Complete"
		sections = append(sections, theme.Current().S().ModalTitle.Render(title))
	} else {
		title := fmt.Sprintf("Spec Wizard - Step %d of %d: %s", m.step+1, totalSteps, stepNames[m.step])
		sections = append(sections, theme.Current().S().ModalTitle.Render(title))
	}
	sections = append(sections, "")

	// Step content
	sections = append(sections, stepContent)

	// Add spacing before buttons (only for input steps 0-2)
	if m.step <= 2 {
		sections = append(sections, "")

		// Calculate button Y position relative to modal content
		stepLines := strings.Count(stepContent, "\n") + 1
		buttonContentY := 1 + 1 + stepLines + 1 // title + blank + content + blank

		// Add button bar based on current step
		buttonBar := m.createButtonBar(modalWidth, buttonContentY)
		sections = append(sections, buttonBar)
	}

	// Join all sections
	content := strings.Join(sections, "\n")

	// Apply modal container style with fixed dimensions
	modalStyle := theme.Current().S().ModalContainer.Width(modalWidth).Height(modalHeight)

	modalContent := modalStyle.Render(content)

	// Center the modal on screen
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modalContent,
	)
}

// createButtonBar creates the button bar for the current step.
// Buttons are context-aware based on step and validation state.
func (m *WizardModel) createButtonBar(modalWidth, contentStartY int) string {
	var buttons []wizard.Button
	nextLabel := "Next →"

	// Determine next button label and validation state
	isValid := m.isStepValid()

	switch m.step {
	case 0:
		// First step: Cancel + Next
		buttons = wizard.CreateCancelNextButtons(isValid, nextLabel)
	default:
		// Other input steps: Back + Next
		buttons = wizard.CreateBackNextButtons(true, isValid, nextLabel)
	}

	// Create button bar (we'll restore focus state separately)
	m.buttonBar = wizard.NewButtonBar(buttons)
	m.buttonBar.SetWidth(modalWidth)

	// Restore focus state if buttons were focused
	if m.buttonFocused {
		// Focus was on buttons - restore focus to first button
		m.buttonBar.FocusFirst()
	}

	return m.buttonBar.Render()
}

// isStepValid checks if the current step has valid data.
// Used to enable/disable the Next button.
func (m *WizardModel) isStepValid() bool {
	switch m.step {
	case 0:
		// Name step: valid if name passes validation
		if m.nameStep != nil {
			return m.nameStep.IsValid()
		}
		return false
	case 1:
		// Description step: always valid (can be empty)
		return true
	case 2:
		// Model step: valid if a model is selected
		if m.modelStep != nil {
			return m.modelStep.SelectedModel() != ""
		}
		return false
	}
	return false
}

// renderConfirmationModal renders the "Are you sure you want to cancel?" modal
// overlay for agent phase cancellation.
func (m *WizardModel) renderConfirmationModal() string {
	// Modal dimensions
	modalWidth := 60
	modalHeight := 8

	t := theme.Current()

	// Build content
	var lines []string
	lines = append(lines, "")
	lines = append(lines, "  Are you sure you want to cancel?")
	lines = append(lines, "")
	lines = append(lines, "  This will discard the interview and exit the wizard.")
	lines = append(lines, "")

	// Render buttons
	yesStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.Primary))
	noStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(t.FgMuted))

	// Apply focus styling
	if m.confirmFocusYes {
		yesStyle = yesStyle.
			Background(lipgloss.Color(t.Primary)).
			Foreground(lipgloss.Color(t.BgBase)).
			Bold(true)
	} else {
		noStyle = noStyle.
			Background(lipgloss.Color(t.Primary)).
			Foreground(lipgloss.Color(t.BgBase)).
			Bold(true)
	}

	yesBtn := yesStyle.Render("Yes")
	noBtn := noStyle.Render("No")

	// Center buttons
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, "  ", noBtn)
	buttonLine := lipgloss.NewStyle().Width(modalWidth - 4).Align(lipgloss.Center).Render(buttons)
	lines = append(lines, buttonLine)

	content := strings.Join(lines, "\n")

	// Apply modal container style
	modalStyle := t.S().ModalContainer.Width(modalWidth).Height(modalHeight)
	modalContent := modalStyle.Render(content)

	// Center on screen
	return lipgloss.Place(m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		modalContent,
	)
}
