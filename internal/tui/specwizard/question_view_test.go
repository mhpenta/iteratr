package specwizard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/specmcp"
)

func TestNewQuestionView(t *testing.T) {
	q := &specmcp.Question{
		Question: "What color scheme should we use?",
		Header:   "Color Scheme",
		Options: []specmcp.QuestionOption{
			{Label: "Dark mode", Description: "Black background with light text"},
			{Label: "Light mode", Description: "White background with dark text"},
		},
		Multiple: false,
	}

	view := NewQuestionView(q)

	// Should have 3 options: 2 from question + 1 auto-appended custom option
	if len(view.options) != 3 {
		t.Errorf("expected 3 options, got %d", len(view.options))
	}

	// Last option should be custom
	if !view.options[2].isCustom {
		t.Error("last option should be marked as custom")
	}

	if view.options[2].label != "Type your own answer..." {
		t.Errorf("expected custom option label 'Type your own answer...', got %q", view.options[2].label)
	}

	// Should default to first option selected
	if view.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", view.selectedIdx)
	}

	// Should be focused by default
	if !view.focused {
		t.Error("expected view to be focused by default")
	}
}

func TestQuestionView_SetSize(t *testing.T) {
	q := &specmcp.Question{
		Question: "Test question?",
		Header:   "Test",
		Options:  []specmcp.QuestionOption{{Label: "Option A", Description: "Desc A"}},
		Multiple: false,
	}

	view := NewQuestionView(q)
	view.SetSize(80, 25)

	if view.width != 80 {
		t.Errorf("expected width 80, got %d", view.width)
	}

	if view.height != 25 {
		t.Errorf("expected height 25, got %d", view.height)
	}

	// ScrollList should have size set with overhead for header/question
	// Height 25 - 5 overhead = 20 for options
	// (We can't directly test scrollList height, but SetSize should not panic)
}

func TestQuestionView_Navigation(t *testing.T) {
	q := &specmcp.Question{
		Question: "Pick a number",
		Header:   "Number",
		Options: []specmcp.QuestionOption{
			{Label: "One", Description: "1"},
			{Label: "Two", Description: "2"},
			{Label: "Three", Description: "3"},
		},
		Multiple: false,
	}

	view := NewQuestionView(q)

	// Start at index 0
	if view.selectedIdx != 0 {
		t.Fatalf("expected selectedIdx 0, got %d", view.selectedIdx)
	}

	// Press down - should move to index 1
	view.Update(tea.KeyPressMsg{Text: "down"})
	if view.selectedIdx != 1 {
		t.Errorf("expected selectedIdx 1 after down, got %d", view.selectedIdx)
	}

	// Press down again - should move to index 2
	view.Update(tea.KeyPressMsg{Text: "down"})
	if view.selectedIdx != 2 {
		t.Errorf("expected selectedIdx 2 after second down, got %d", view.selectedIdx)
	}

	// Press down again - should move to index 3 (custom option)
	view.Update(tea.KeyPressMsg{Text: "down"})
	if view.selectedIdx != 3 {
		t.Errorf("expected selectedIdx 3 after third down, got %d", view.selectedIdx)
	}

	// Press down at end - should stay at index 3
	view.Update(tea.KeyPressMsg{Text: "down"})
	if view.selectedIdx != 3 {
		t.Errorf("expected selectedIdx to stay at 3, got %d", view.selectedIdx)
	}

	// Press up - should move to index 2
	view.Update(tea.KeyPressMsg{Text: "up"})
	if view.selectedIdx != 2 {
		t.Errorf("expected selectedIdx 2 after up, got %d", view.selectedIdx)
	}

	// Press up twice - should move to index 0
	view.Update(tea.KeyPressMsg{Text: "up"})
	view.Update(tea.KeyPressMsg{Text: "up"})
	if view.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0 after two ups, got %d", view.selectedIdx)
	}

	// Press up at start - should stay at index 0
	view.Update(tea.KeyPressMsg{Text: "up"})
	if view.selectedIdx != 0 {
		t.Errorf("expected selectedIdx to stay at 0, got %d", view.selectedIdx)
	}
}

func TestQuestionView_NavigationWithVimKeys(t *testing.T) {
	q := &specmcp.Question{
		Question: "Pick a letter",
		Header:   "Letter",
		Options: []specmcp.QuestionOption{
			{Label: "A", Description: "First"},
			{Label: "B", Description: "Second"},
		},
		Multiple: false,
	}

	view := NewQuestionView(q)

	// Start at index 0
	if view.selectedIdx != 0 {
		t.Fatalf("expected selectedIdx 0, got %d", view.selectedIdx)
	}

	// Press j (vim down) - should move to index 1
	view.Update(tea.KeyPressMsg{Text: "j"})
	if view.selectedIdx != 1 {
		t.Errorf("expected selectedIdx 1 after 'j', got %d", view.selectedIdx)
	}

	// Press k (vim up) - should move back to index 0
	view.Update(tea.KeyPressMsg{Text: "k"})
	if view.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0 after 'k', got %d", view.selectedIdx)
	}
}

func TestQuestionView_SelectPreDefinedOption(t *testing.T) {
	q := &specmcp.Question{
		Question: "Choose framework",
		Header:   "Framework",
		Options: []specmcp.QuestionOption{
			{Label: "React", Description: "JavaScript library"},
			{Label: "Vue", Description: "Progressive framework"},
		},
		Multiple: false,
	}

	view := NewQuestionView(q)

	// Select first option (React)
	cmd := view.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Fatal("expected command after enter, got nil")
	}

	// Execute command to get message
	msg := cmd()
	answerMsg, ok := msg.(AnswerSelectedMsg)
	if !ok {
		t.Fatalf("expected AnswerSelectedMsg, got %T", msg)
	}

	if answerMsg.Answer != "React" {
		t.Errorf("expected answer 'React', got %q", answerMsg.Answer)
	}
}

func TestQuestionView_SelectCustomOption(t *testing.T) {
	q := &specmcp.Question{
		Question: "What's your favorite?",
		Header:   "Favorite",
		Options: []specmcp.QuestionOption{
			{Label: "Option A", Description: "First choice"},
			{Label: "Option B", Description: "Second choice"},
		},
		Multiple: false,
	}

	view := NewQuestionView(q)

	// Navigate to last option (custom option)
	view.Update(tea.KeyPressMsg{Text: "down"})
	view.Update(tea.KeyPressMsg{Text: "down"})
	view.Update(tea.KeyPressMsg{Text: "down"})

	// Should be at custom option
	if !view.IsCustomSelected() {
		t.Error("expected custom option to be selected")
	}

	// Select custom option
	cmd := view.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Fatal("expected command after enter, got nil")
	}

	// Execute command to get message
	msg := cmd()
	_, ok := msg.(CustomAnswerRequestedMsg)
	if !ok {
		t.Fatalf("expected CustomAnswerRequestedMsg, got %T", msg)
	}
}

func TestQuestionView_SelectWithSpace(t *testing.T) {
	q := &specmcp.Question{
		Question: "Choose one",
		Header:   "Choice",
		Options:  []specmcp.QuestionOption{{Label: "Yes", Description: "Affirmative"}},
		Multiple: false,
	}

	view := NewQuestionView(q)

	// Select with space key
	cmd := view.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if cmd == nil {
		t.Fatal("expected command after space, got nil")
	}

	msg := cmd()
	answerMsg, ok := msg.(AnswerSelectedMsg)
	if !ok {
		t.Fatalf("expected AnswerSelectedMsg, got %T", msg)
	}

	if answerMsg.Answer != "Yes" {
		t.Errorf("expected answer 'Yes', got %q", answerMsg.Answer)
	}
}

func TestQuestionView_View(t *testing.T) {
	q := &specmcp.Question{
		Question: "What should we build?",
		Header:   "Project Type",
		Options: []specmcp.QuestionOption{
			{Label: "Web app", Description: "Browser-based application"},
			{Label: "CLI tool", Description: "Command-line interface"},
		},
		Multiple: false,
	}

	view := NewQuestionView(q)
	rendered := view.View()

	// Should contain header
	if !strings.Contains(rendered, "Project Type") {
		t.Error("view should contain header")
	}

	// Should contain question text
	if !strings.Contains(rendered, "What should we build?") {
		t.Error("view should contain question text")
	}

	// Note: Options are rendered by ScrollList, which may apply formatting
	// We can't easily test for exact option text, but view should not be empty
	if rendered == "" {
		t.Error("view should not be empty")
	}
}

func TestQuestionView_SelectedAnswer(t *testing.T) {
	q := &specmcp.Question{
		Question: "Pick color",
		Header:   "Color",
		Options: []specmcp.QuestionOption{
			{Label: "Red", Description: "#FF0000"},
			{Label: "Blue", Description: "#0000FF"},
		},
		Multiple: false,
	}

	view := NewQuestionView(q)

	// Initially at first option
	if view.SelectedAnswer() != "Red" {
		t.Errorf("expected selected answer 'Red', got %q", view.SelectedAnswer())
	}

	// Move to second option
	view.Update(tea.KeyPressMsg{Text: "down"})
	if view.SelectedAnswer() != "Blue" {
		t.Errorf("expected selected answer 'Blue', got %q", view.SelectedAnswer())
	}

	// Move to custom option
	view.Update(tea.KeyPressMsg{Text: "down"})
	if view.SelectedAnswer() != "" {
		t.Errorf("expected empty answer for custom option, got %q", view.SelectedAnswer())
	}
}

func TestQuestionView_IsCustomSelected(t *testing.T) {
	q := &specmcp.Question{
		Question: "Test?",
		Header:   "Test",
		Options:  []specmcp.QuestionOption{{Label: "A", Description: "Option A"}},
		Multiple: false,
	}

	view := NewQuestionView(q)

	// Start at first option (not custom)
	if view.IsCustomSelected() {
		t.Error("IsCustomSelected should be false at first option")
	}

	// Move to custom option
	view.Update(tea.KeyPressMsg{Text: "down"})
	view.Update(tea.KeyPressMsg{Text: "down"})

	if !view.IsCustomSelected() {
		t.Error("IsCustomSelected should be true at custom option")
	}
}

func TestQuestionView_SetFocused(t *testing.T) {
	q := &specmcp.Question{
		Question: "Test?",
		Header:   "Test",
		Options:  []specmcp.QuestionOption{{Label: "A", Description: "Opt A"}},
		Multiple: false,
	}

	view := NewQuestionView(q)

	// Should be focused by default
	if !view.focused {
		t.Error("expected view to be focused by default")
	}

	// Blur
	view.SetFocused(false)
	if view.focused {
		t.Error("expected view to be unfocused after SetFocused(false)")
	}

	// Focus again
	view.SetFocused(true)
	if !view.focused {
		t.Error("expected view to be focused after SetFocused(true)")
	}
}

func TestQuestionView_PreferredHeight(t *testing.T) {
	tests := []struct {
		name           string
		optionsCount   int
		expectedMin    int
		expectedMax    int
		hasDescription bool
	}{
		{
			name:           "few options without descriptions",
			optionsCount:   2,
			expectedMin:    8,  // 5 overhead + 2 options (1 line each) + 1 custom option
			expectedMax:    10, // allow some variation
			hasDescription: false,
		},
		{
			name:           "few options with descriptions",
			optionsCount:   2,
			expectedMin:    9,  // 5 overhead + 2 options (2 lines each) + 1 custom option (1 line)
			expectedMax:    12, // allow variation
			hasDescription: true,
		},
		{
			name:           "many options capped at 15",
			optionsCount:   20,
			expectedMin:    15, // Should cap options height at 15
			expectedMax:    20, // 5 overhead + 15 cap
			hasDescription: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := make([]specmcp.QuestionOption, tt.optionsCount)
			for i := 0; i < tt.optionsCount; i++ {
				options[i] = specmcp.QuestionOption{
					Label: "Option",
				}
				if tt.hasDescription {
					options[i].Description = "Description text"
				}
			}

			q := &specmcp.Question{
				Question: "Test question?",
				Header:   "Test",
				Options:  options,
				Multiple: false,
			}

			view := NewQuestionView(q)
			height := view.PreferredHeight()

			if height < tt.expectedMin {
				t.Errorf("expected height >= %d, got %d", tt.expectedMin, height)
			}

			if height > tt.expectedMax {
				t.Errorf("expected height <= %d, got %d", tt.expectedMax, height)
			}
		})
	}
}

func TestQuestionOption_Render(t *testing.T) {
	tests := []struct {
		name          string
		option        QuestionOption
		width         int
		shouldHave    []string
		shouldNotHave []string
	}{
		{
			name: "regular option with description",
			option: QuestionOption{
				idx:         0,
				label:       "Dark theme",
				description: "Black background with light text",
				isCustom:    false,
			},
			width:      80,
			shouldHave: []string{"Dark theme", "Black background"},
		},
		{
			name: "custom option",
			option: QuestionOption{
				idx:         1,
				label:       "Type your own answer...",
				description: "",
				isCustom:    true,
			},
			width:         80,
			shouldHave:    []string{"Type your own answer..."},
			shouldNotHave: []string{}, // Description should be skipped for custom
		},
		{
			name: "option without description",
			option: QuestionOption{
				idx:         0,
				label:       "Yes",
				description: "",
				isCustom:    false,
			},
			width:      80,
			shouldHave: []string{"Yes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := tt.option.Render(tt.width)

			for _, text := range tt.shouldHave {
				if !strings.Contains(rendered, text) {
					t.Errorf("expected rendered output to contain %q, got: %s", text, rendered)
				}
			}

			for _, text := range tt.shouldNotHave {
				if strings.Contains(rendered, text) {
					t.Errorf("expected rendered output NOT to contain %q, got: %s", text, rendered)
				}
			}
		})
	}
}

func TestQuestionOption_Height(t *testing.T) {
	tests := []struct {
		name           string
		option         QuestionOption
		expectedHeight int
	}{
		{
			name: "option with description",
			option: QuestionOption{
				idx:         0,
				label:       "Option A",
				description: "This is a description",
				isCustom:    false,
			},
			expectedHeight: 2, // label + description
		},
		{
			name: "option without description",
			option: QuestionOption{
				idx:         0,
				label:       "Option B",
				description: "",
				isCustom:    false,
			},
			expectedHeight: 1, // label only
		},
		{
			name: "custom option",
			option: QuestionOption{
				idx:         0,
				label:       "Type your own answer...",
				description: "ignored",
				isCustom:    true,
			},
			expectedHeight: 1, // custom options don't show description
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			height := tt.option.Height()
			if height != tt.expectedHeight {
				t.Errorf("expected height %d, got %d", tt.expectedHeight, height)
			}
		})
	}
}

func TestQuestionOption_ID(t *testing.T) {
	opt := QuestionOption{
		idx:         0,
		label:       "Test Option",
		description: "Test description",
		isCustom:    false,
	}

	id := opt.ID()
	if id != "Test Option" {
		t.Errorf("expected ID 'Test Option', got %q", id)
	}
}

// Multi-select tests

func TestQuestionView_MultiSelect_Initialization(t *testing.T) {
	q := &specmcp.Question{
		Question: "Select frameworks",
		Header:   "Frameworks",
		Options: []specmcp.QuestionOption{
			{Label: "React", Description: "JavaScript library"},
			{Label: "Vue", Description: "Progressive framework"},
			{Label: "Angular", Description: "Full framework"},
		},
		Multiple: true,
	}

	view := NewQuestionView(q)

	// Should be in multi-select mode
	if !view.isMultiSelect {
		t.Error("expected view to be in multi-select mode")
	}

	// Should have empty selection set initially
	if len(view.selectedSet) != 0 {
		t.Errorf("expected empty selection set, got %d selections", len(view.selectedSet))
	}

	// Options should have isMultiMode set
	for i, opt := range view.options {
		if !opt.isMultiMode && !opt.isCustom {
			t.Errorf("option %d should have isMultiMode=true", i)
		}
	}
}

func TestQuestionView_MultiSelect_ToggleSelection(t *testing.T) {
	q := &specmcp.Question{
		Question: "Pick colors",
		Header:   "Colors",
		Options: []specmcp.QuestionOption{
			{Label: "Red", Description: "Primary color"},
			{Label: "Blue", Description: "Primary color"},
			{Label: "Green", Description: "Primary color"},
		},
		Multiple: true,
	}

	view := NewQuestionView(q)

	// Initially no selections
	if len(view.selectedSet) != 0 {
		t.Fatalf("expected no selections initially, got %d", len(view.selectedSet))
	}

	// Press space to select first option (Red)
	view.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if !view.selectedSet[0] {
		t.Error("expected first option to be selected after space")
	}
	if len(view.selectedSet) != 1 {
		t.Errorf("expected 1 selection, got %d", len(view.selectedSet))
	}

	// Press space again to deselect
	view.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if view.selectedSet[0] {
		t.Error("expected first option to be deselected after second space")
	}
	if len(view.selectedSet) != 0 {
		t.Errorf("expected 0 selections, got %d", len(view.selectedSet))
	}

	// Navigate down and select second option (Blue)
	view.Update(tea.KeyPressMsg{Text: "down"})
	view.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if !view.selectedSet[1] {
		t.Error("expected second option to be selected")
	}

	// Navigate down and select third option (Green) - both should be selected
	view.Update(tea.KeyPressMsg{Text: "down"})
	view.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if !view.selectedSet[1] || !view.selectedSet[2] {
		t.Error("expected both second and third options to be selected")
	}
	if len(view.selectedSet) != 2 {
		t.Errorf("expected 2 selections, got %d", len(view.selectedSet))
	}
}

func TestQuestionView_MultiSelect_SubmitMultiple(t *testing.T) {
	q := &specmcp.Question{
		Question: "Choose languages",
		Header:   "Languages",
		Options: []specmcp.QuestionOption{
			{Label: "Go", Description: "Compiled language"},
			{Label: "Python", Description: "Interpreted language"},
			{Label: "Rust", Description: "Systems language"},
		},
		Multiple: true,
	}

	view := NewQuestionView(q)

	// Select first and third options
	view.Update(tea.KeyPressMsg{Code: ' ', Text: " "}) // Select Go
	view.Update(tea.KeyPressMsg{Text: "down"})
	view.Update(tea.KeyPressMsg{Text: "down"})
	view.Update(tea.KeyPressMsg{Code: ' ', Text: " "}) // Select Rust

	// Press enter to submit
	cmd := view.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Fatal("expected command after enter, got nil")
	}

	// Execute command to get message
	msg := cmd()
	multiMsg, ok := msg.(MultiAnswerSelectedMsg)
	if !ok {
		t.Fatalf("expected MultiAnswerSelectedMsg, got %T", msg)
	}

	// Should have both selected answers
	if len(multiMsg.Answers) != 2 {
		t.Errorf("expected 2 answers, got %d", len(multiMsg.Answers))
	}

	// Check that answers contain selected options (order may vary)
	hasGo := false
	hasRust := false
	for _, ans := range multiMsg.Answers {
		if ans == "Go" {
			hasGo = true
		}
		if ans == "Rust" {
			hasRust = true
		}
	}

	if !hasGo || !hasRust {
		t.Errorf("expected answers to contain 'Go' and 'Rust', got %v", multiMsg.Answers)
	}
}

func TestQuestionView_MultiSelect_SubmitEmpty(t *testing.T) {
	q := &specmcp.Question{
		Question: "Pick one or more",
		Header:   "Options",
		Options: []specmcp.QuestionOption{
			{Label: "A", Description: "First"},
			{Label: "B", Description: "Second"},
		},
		Multiple: true,
	}

	view := NewQuestionView(q)

	// Don't select anything, just press enter
	cmd := view.Update(tea.KeyPressMsg{Text: "enter"})
	if cmd == nil {
		t.Fatal("expected command after enter, got nil")
	}

	// Execute command to get message
	msg := cmd()
	_, ok := msg.(CustomAnswerRequestedMsg)
	if !ok {
		t.Fatalf("expected CustomAnswerRequestedMsg when no selections, got %T", msg)
	}
}

func TestQuestionView_MultiSelect_CannotToggleCustomOption(t *testing.T) {
	q := &specmcp.Question{
		Question: "Select items",
		Header:   "Items",
		Options: []specmcp.QuestionOption{
			{Label: "Item 1", Description: "First item"},
			{Label: "Item 2", Description: "Second item"},
		},
		Multiple: true,
	}

	view := NewQuestionView(q)

	// Navigate to custom option (last in list)
	view.Update(tea.KeyPressMsg{Text: "down"})
	view.Update(tea.KeyPressMsg{Text: "down"})
	view.Update(tea.KeyPressMsg{Text: "down"})

	// Should be at custom option
	if !view.IsCustomSelected() {
		t.Fatal("expected to be at custom option")
	}

	// Try to toggle with space - should do nothing
	view.Update(tea.KeyPressMsg{Code: ' ', Text: " "})

	// Custom option should not be in selected set
	customIdx := len(view.options) - 1
	if view.selectedSet[customIdx] {
		t.Error("custom option should not be selectable in multi-select mode")
	}
}

func TestQuestionView_MultiSelect_CheckboxRendering(t *testing.T) {
	q := &specmcp.Question{
		Question: "Test checkboxes",
		Header:   "Test",
		Options: []specmcp.QuestionOption{
			{Label: "Option A", Description: "First"},
			{Label: "Option B", Description: "Second"},
		},
		Multiple: true,
	}

	view := NewQuestionView(q)

	// Select first option
	view.Update(tea.KeyPressMsg{Code: ' ', Text: " "})

	// Check that options have selection state
	if !view.options[0].isSelected {
		t.Error("first option should be marked as selected")
	}
	if view.options[1].isSelected {
		t.Error("second option should not be marked as selected")
	}

	// Render first option - should show [x]
	rendered := view.options[0].Render(80)
	if !strings.Contains(rendered, "[x]") {
		t.Error("selected option should render with [x] checkbox")
	}

	// Render second option - should show [ ]
	rendered = view.options[1].Render(80)
	if !strings.Contains(rendered, "[ ]") {
		t.Error("unselected option should render with [ ] checkbox")
	}
}

func TestQuestionView_SingleSelect_NoCheckbox(t *testing.T) {
	q := &specmcp.Question{
		Question: "Pick one",
		Header:   "Single",
		Options: []specmcp.QuestionOption{
			{Label: "Option A", Description: "First"},
		},
		Multiple: false,
	}

	view := NewQuestionView(q)

	// Check that view is not in multi-select mode
	if view.isMultiSelect {
		t.Error("view should not be in multi-select mode")
	}

	// Check that option does not have multi-mode set
	if view.options[0].isMultiMode {
		t.Error("option should not have isMultiMode=true")
	}

	// Render option - should NOT show checkbox
	rendered := view.options[0].Render(80)
	if strings.Contains(rendered, "[ ]") || strings.Contains(rendered, "[x]") {
		t.Errorf("single-select option should not render with checkbox, got: %q", rendered)
	}
}

func TestQuestionView_MultiSelect_SpaceDoesNotSubmit(t *testing.T) {
	q := &specmcp.Question{
		Question: "Multi choice",
		Header:   "Test",
		Options: []specmcp.QuestionOption{
			{Label: "A", Description: "First"},
		},
		Multiple: true,
	}

	view := NewQuestionView(q)

	// Press space - should toggle, not submit
	cmd := view.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if cmd != nil {
		t.Error("space in multi-select should not return a command (submit)")
	}

	// Verify selection was toggled
	if !view.selectedSet[0] {
		t.Error("space should have toggled selection")
	}
}

func TestQuestionView_SingleSelect_SpaceSubmits(t *testing.T) {
	q := &specmcp.Question{
		Question: "Single choice",
		Header:   "Test",
		Options: []specmcp.QuestionOption{
			{Label: "A", Description: "First"},
		},
		Multiple: false,
	}

	view := NewQuestionView(q)

	// Press space - should submit immediately (same as enter)
	cmd := view.Update(tea.KeyPressMsg{Code: ' ', Text: " "})
	if cmd == nil {
		t.Fatal("space in single-select should return submit command")
	}

	msg := cmd()
	answerMsg, ok := msg.(AnswerSelectedMsg)
	if !ok {
		t.Fatalf("expected AnswerSelectedMsg, got %T", msg)
	}

	if answerMsg.Answer != "A" {
		t.Errorf("expected answer 'A', got %q", answerMsg.Answer)
	}
}
