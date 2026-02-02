package specwizard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestDescriptionStepInit(t *testing.T) {
	step := NewDescriptionStep()
	cmd := step.Init()

	// Should return a focus command
	if cmd == nil {
		t.Fatal("Init should return a focus command")
	}
}

func TestDescriptionStepView(t *testing.T) {
	step := NewDescriptionStep()
	step.Init()

	view := step.View()

	// Should contain label
	if !strings.Contains(view, "Feature Description") {
		t.Error("View should contain 'Feature Description' label")
	}

	// Should contain hint
	if !strings.Contains(view, "provide as much detail as possible") {
		t.Error("View should contain hint text")
	}
}

func TestDescriptionStepIsValid(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
	}{
		{
			name:      "valid description",
			input:     "This is a detailed description of the feature.",
			wantValid: true,
		},
		{
			name:      "valid multi-line description",
			input:     "Line 1\nLine 2\nLine 3",
			wantValid: true,
		},
		{
			name:      "empty string",
			input:     "",
			wantValid: false,
		},
		{
			name:      "whitespace only",
			input:     "   \n  \n   ",
			wantValid: false,
		},
		{
			name:      "single character",
			input:     "a",
			wantValid: true,
		},
		{
			name:      "long description",
			input:     strings.Repeat("This is a very long description. ", 100),
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := NewDescriptionStep()
			step.textarea.SetValue(tt.input)

			valid := step.IsValid()

			if valid != tt.wantValid {
				t.Errorf("IsValid() = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

func TestDescriptionStepDescription(t *testing.T) {
	step := NewDescriptionStep()
	input := "  This is a description with leading and trailing whitespace.  \n\n"
	step.textarea.SetValue(input)

	desc := step.Description()

	// Should trim whitespace
	expected := "This is a description with leading and trailing whitespace."
	if desc != expected {
		t.Errorf("Description() = %q, want %q", desc, expected)
	}
}

func TestDescriptionStepCtrlD(t *testing.T) {
	step := NewDescriptionStep()
	step.Init()

	// Set valid input
	step.textarea.SetValue("A valid description")

	// Simulate Ctrl+D key
	cmd := step.Update(tea.KeyPressMsg{Text: "ctrl+d"})

	if cmd == nil {
		t.Fatal("Ctrl+D key should return a command for valid input")
	}

	// Execute command to get message
	msg := cmd()
	if _, ok := msg.(DescriptionCompleteMsg); !ok {
		t.Errorf("Ctrl+D key should return DescriptionCompleteMsg, got %T", msg)
	}
}

func TestDescriptionStepCtrlDWithEmptyInput(t *testing.T) {
	step := NewDescriptionStep()
	step.Init()

	// Empty input
	step.textarea.SetValue("")

	// Simulate Ctrl+D key
	cmd := step.Update(tea.KeyPressMsg{Text: "ctrl+d"})

	// Should not advance for empty input
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(DescriptionCompleteMsg); ok {
			t.Error("Ctrl+D key should not return DescriptionCompleteMsg for empty input")
		}
	}
}

func TestDescriptionStepTabKey(t *testing.T) {
	step := NewDescriptionStep()
	step.Init()

	// Simulate tab key
	cmd := step.Update(tea.KeyPressMsg{Text: "tab"})

	if cmd == nil {
		t.Fatal("Tab key should return a command")
	}

	// Execute command to get message
	msg := cmd()
	if _, ok := msg.(TabExitForwardMsg); !ok {
		t.Errorf("Tab key should return TabExitForwardMsg, got %T", msg)
	}
}

func TestDescriptionStepShiftTabKey(t *testing.T) {
	step := NewDescriptionStep()
	step.Init()

	// Simulate shift+tab key
	cmd := step.Update(tea.KeyPressMsg{Text: "shift+tab"})

	if cmd == nil {
		t.Fatal("Shift+Tab key should return a command")
	}

	// Execute command to get message
	msg := cmd()
	if _, ok := msg.(TabExitBackwardMsg); !ok {
		t.Errorf("Shift+Tab key should return TabExitBackwardMsg, got %T", msg)
	}
}

func TestDescriptionStepFocusAndBlur(t *testing.T) {
	step := NewDescriptionStep()

	// Focus
	cmd := step.Focus()
	if cmd == nil {
		t.Error("Focus should return a command")
	}

	// Textarea should be focused after Init
	step.Init()
	if !step.textarea.Focused() {
		t.Error("Textarea should be focused after Init")
	}

	// Blur
	step.Blur()
	if step.textarea.Focused() {
		t.Error("Textarea should not be focused after Blur")
	}
}

func TestDescriptionStepSetSize(t *testing.T) {
	step := NewDescriptionStep()

	step.SetSize(100, 30)

	if step.width != 100 {
		t.Errorf("width = %d, want 100", step.width)
	}
	if step.height != 30 {
		t.Errorf("height = %d, want 30", step.height)
	}
}

func TestDescriptionStepPreferredHeight(t *testing.T) {
	step := NewDescriptionStep()

	height := step.PreferredHeight()

	// Should return fixed height for content
	if height != 12 {
		t.Errorf("PreferredHeight() = %d, want 12", height)
	}
}

func TestDescriptionStepMultiLineInput(t *testing.T) {
	step := NewDescriptionStep()
	step.Init()

	// Set multi-line input
	multiLineText := "Line 1\nLine 2\nLine 3"
	step.textarea.SetValue(multiLineText)

	desc := step.Description()

	// Should preserve line breaks
	if desc != multiLineText {
		t.Errorf("Description() = %q, want %q", desc, multiLineText)
	}

	// Should be valid
	if !step.IsValid() {
		t.Error("Multi-line description should be valid")
	}
}

func TestDescriptionStepNoCharacterLimit(t *testing.T) {
	step := NewDescriptionStep()

	// Verify no character limit is set (CharLimit should be 0)
	if step.textarea.CharLimit != 0 {
		t.Errorf("CharLimit = %d, want 0 (unlimited)", step.textarea.CharLimit)
	}
}

func TestDescriptionStepNoLineNumbers(t *testing.T) {
	step := NewDescriptionStep()

	// Verify line numbers are disabled
	if step.textarea.ShowLineNumbers {
		t.Error("ShowLineNumbers should be false")
	}
}
