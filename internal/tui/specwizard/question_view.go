package specwizard

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mark3labs/iteratr/internal/specmcp"
	"github.com/mark3labs/iteratr/internal/tui"
	"github.com/mark3labs/iteratr/internal/tui/theme"
)

// QuestionOption represents a rendered option item in the ScrollList.
type QuestionOption struct {
	idx         int    // Index in options array
	label       string // Display label (1-5 words)
	description string // Longer description
	isCustom    bool   // True if this is the "Type your own answer..." option
	isSelected  bool   // True if this option is selected (for multi-select)
	isMultiMode bool   // True if rendering in multi-select mode
}

// ID returns the unique identifier for this option.
func (o *QuestionOption) ID() string {
	return o.label
}

// Render returns the rendered string representation for this option.
// Format (single-select):
//
//	label
//	  description (indented, muted color)
//
// Format (multi-select):
//
//	[x] label  (or [ ] if not selected)
//	    description (indented, muted color)
func (o *QuestionOption) Render(width int) string {
	var b strings.Builder

	// For multi-select mode, show checkbox prefix
	if o.isMultiMode {
		checkboxStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#89b4fa"))
		if o.isSelected {
			b.WriteString(checkboxStyle.Render("[x] "))
		} else {
			b.WriteString(checkboxStyle.Render("[ ] "))
		}
	}

	// Render label (main text)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	if o.isCustom {
		// Custom option in muted color
		labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8")).Italic(true)
	}
	b.WriteString(labelStyle.Render(o.label))

	// Render description (if present and not custom option)
	if o.description != "" && !o.isCustom {
		b.WriteString("\n")
		// Add extra indent for multi-select to align with checkbox
		indent := 2
		if o.isMultiMode {
			indent = 6 // Align with text after "[ ] "
		}
		descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6c7086")).PaddingLeft(indent)
		b.WriteString(descStyle.Render(o.description))
	}

	return b.String()
}

// Height returns the number of lines this option occupies.
func (o *QuestionOption) Height() int {
	lines := 1 // Label always takes 1 line
	if o.description != "" && !o.isCustom {
		lines++ // Description adds 1 line
	}
	return lines
}

// QuestionView manages the display of a single question with multiple choice options.
// Supports scrollable list of options with up/down navigation and enter to select.
// For multi-select questions, space toggles selection and enter submits all selected options.
type QuestionView struct {
	question      *specmcp.Question // The question being displayed
	options       []*QuestionOption // All options (including auto-appended custom option)
	scrollList    *tui.ScrollList   // Scrollable list for options
	selectedIdx   int               // Currently focused option index (cursor position)
	selectedSet   map[int]bool      // Set of selected option indices (for multi-select)
	width         int               // Available width
	height        int               // Available height
	focused       bool              // Whether this view has focus
	isMultiSelect bool              // True if this is a multi-select question
}

// NewQuestionView creates a new question view for the given question.
// Automatically appends "Type your own answer..." to the options list.
func NewQuestionView(q *specmcp.Question) *QuestionView {
	// Build options list from question
	options := make([]*QuestionOption, 0, len(q.Options)+1)
	for i, opt := range q.Options {
		options = append(options, &QuestionOption{
			idx:         i,
			label:       opt.Label,
			description: opt.Description,
			isCustom:    false,
		})
	}

	// Auto-append "Type your own answer..." option
	options = append(options, &QuestionOption{
		idx:         len(q.Options),
		label:       "Type your own answer...",
		description: "",
		isCustom:    true,
	})

	// Create scroll list for options
	scrollList := tui.NewScrollList(60, 10)
	scrollList.SetAutoScroll(false) // Manual navigation
	scrollList.SetFocused(true)
	scrollList.SetSelected(0) // Default to first option

	view := &QuestionView{
		question:      q,
		options:       options,
		scrollList:    scrollList,
		selectedIdx:   0,
		selectedSet:   make(map[int]bool),
		width:         60,
		height:        20,
		focused:       true,
		isMultiSelect: q.Multiple,
	}

	// Initialize scroll list items with correct multi-select mode
	view.refreshScrollList()

	return view
}

// refreshScrollList updates the scroll list items to reflect current selection state.
// This is called after toggling selections in multi-select mode.
func (q *QuestionView) refreshScrollList() {
	// Update options with current selection state
	for i, opt := range q.options {
		opt.isSelected = q.selectedSet[i]
		opt.isMultiMode = q.isMultiSelect
	}

	// Convert options to ScrollItem interface and update scroll list
	scrollItems := make([]tui.ScrollItem, len(q.options))
	for i, opt := range q.options {
		scrollItems[i] = opt
	}
	q.scrollList.SetItems(scrollItems)
}

// SetSize updates the dimensions for the question view.
func (q *QuestionView) SetSize(width, height int) {
	q.width = width
	q.height = height

	// Calculate available height for options list
	// Overhead: header (1) + blank (1) + question text (1-2) + blank (1) = 4-5 lines
	// Reserve 5 lines for header/question, rest for options
	optionsHeight := height - 5
	if optionsHeight < 3 {
		optionsHeight = 3
	}

	q.scrollList.SetWidth(width)
	q.scrollList.SetHeight(optionsHeight)
}

// SetFocused sets the focus state of the view.
func (q *QuestionView) SetFocused(focused bool) {
	q.focused = focused
	q.scrollList.SetFocused(focused)
}

// Update handles messages for the question view.
func (q *QuestionView) Update(msg tea.Msg) tea.Cmd {
	// Handle keyboard input
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if q.selectedIdx > 0 {
				q.selectedIdx--
				q.scrollList.SetSelected(q.selectedIdx)
				q.scrollList.ScrollToItem(q.selectedIdx)
			}
			return nil

		case "down", "j":
			if q.selectedIdx < len(q.options)-1 {
				q.selectedIdx++
				q.scrollList.SetSelected(q.selectedIdx)
				q.scrollList.ScrollToItem(q.selectedIdx)
			}
			return nil

		case " ", "space":
			// Space key behavior depends on multi-select mode
			if q.isMultiSelect {
				// Multi-select: toggle selection of current option
				if q.selectedIdx >= 0 && q.selectedIdx < len(q.options) {
					selectedOpt := q.options[q.selectedIdx]
					// Don't allow toggling custom option in multi-select
					if !selectedOpt.isCustom {
						if q.selectedSet[q.selectedIdx] {
							delete(q.selectedSet, q.selectedIdx)
						} else {
							q.selectedSet[q.selectedIdx] = true
						}
						// Update scroll list items to reflect new selection state
						q.refreshScrollList()
					}
				}
				return nil
			} else {
				// Single-select: same as enter - submit immediately
				return q.handleSubmit()
			}

		case "enter":
			// Enter key submits answer(s)
			return q.handleSubmit()
		}
	}

	// Forward to scroll list for scrolling support
	return q.scrollList.Update(msg)
}

// handleSubmit handles the enter key press to submit answer(s).
// For single-select: returns the currently focused option.
// For multi-select: returns all selected options, or custom answer request if none selected.
func (q *QuestionView) handleSubmit() tea.Cmd {
	if q.isMultiSelect {
		// Multi-select: gather all selected options
		if len(q.selectedSet) == 0 {
			// No selections - treat as custom answer request
			return func() tea.Msg {
				return CustomAnswerRequestedMsg{}
			}
		}

		// Collect selected option labels
		var answers []string
		for idx := range q.selectedSet {
			if idx >= 0 && idx < len(q.options) {
				opt := q.options[idx]
				if !opt.isCustom {
					answers = append(answers, opt.label)
				}
			}
		}

		// Return multi-answer message
		return func() tea.Msg {
			return MultiAnswerSelectedMsg{
				Answers: answers,
			}
		}
	} else {
		// Single-select: return currently focused option
		if q.selectedIdx >= 0 && q.selectedIdx < len(q.options) {
			selectedOpt := q.options[q.selectedIdx]
			if selectedOpt.isCustom {
				// User wants to type custom answer
				return func() tea.Msg {
					return CustomAnswerRequestedMsg{}
				}
			}
			// Pre-defined option selected
			return func() tea.Msg {
				return AnswerSelectedMsg{
					Answer: selectedOpt.label,
				}
			}
		}
		return nil
	}
}

// View renders the question view.
func (q *QuestionView) View() string {
	var b strings.Builder

	t := theme.Current()

	// Render header (short label)
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Primary)).Bold(true)
	b.WriteString(headerStyle.Render(q.question.Header))
	b.WriteString("\n\n")

	// Render full question text
	questionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))
	b.WriteString(questionStyle.Render(q.question.Question))
	b.WriteString("\n\n")

	// Render options using scroll list
	b.WriteString(q.scrollList.View())

	return b.String()
}

// SelectedAnswer returns the currently selected answer (label text).
// Returns empty string if custom option is selected (parent should prompt for input).
func (q *QuestionView) SelectedAnswer() string {
	if q.selectedIdx >= 0 && q.selectedIdx < len(q.options) {
		opt := q.options[q.selectedIdx]
		if opt.isCustom {
			return "" // Custom answer - parent should prompt
		}
		return opt.label
	}
	return ""
}

// IsCustomSelected returns true if the "Type your own answer..." option is selected.
func (q *QuestionView) IsCustomSelected() bool {
	if q.selectedIdx >= 0 && q.selectedIdx < len(q.options) {
		return q.options[q.selectedIdx].isCustom
	}
	return false
}

// PreferredHeight returns the preferred height for this question view.
// Calculates based on header + question + options count.
func (q *QuestionView) PreferredHeight() int {
	// Header: 1 line
	// Blank: 1 line
	// Question text: assume 1-2 lines (wrap long questions)
	// Blank: 1 line
	// Options: sum of all option heights (cap at 15 for modal sizing)
	// Total overhead: 4-5 lines

	overhead := 5

	// Calculate total options height
	optionsHeight := 0
	for _, opt := range q.options {
		optionsHeight += opt.Height()
	}

	// Cap options height for reasonable modal size
	if optionsHeight > 15 {
		optionsHeight = 15
	}

	return overhead + optionsHeight
}

// CustomAnswerRequestedMsg is sent when the user selects "Type your own answer...".
// Parent should display text input to collect the custom response.
type CustomAnswerRequestedMsg struct{}

// AnswerSelectedMsg is sent when the user selects a pre-defined answer option (single-select).
type AnswerSelectedMsg struct {
	Answer string // The selected option label
}

// MultiAnswerSelectedMsg is sent when the user submits multiple selected options (multi-select).
type MultiAnswerSelectedMsg struct {
	Answers []string // All selected option labels
}
