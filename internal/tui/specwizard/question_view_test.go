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
