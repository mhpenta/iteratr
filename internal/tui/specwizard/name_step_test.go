package specwizard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestNameStepInit(t *testing.T) {
	step := NewNameStep()
	cmd := step.Init()

	// Should return a focus command
	if cmd == nil {
		t.Fatal("Init should return a focus command")
	}
}

func TestNameStepView(t *testing.T) {
	step := NewNameStep()
	step.Init()

	view := step.View()

	// Should contain label
	if !strings.Contains(view, "Spec Name") {
		t.Error("View should contain 'Spec Name' label")
	}

	// Should contain hint
	if !strings.Contains(view, "lowercase, hyphens only") {
		t.Error("View should contain hint text")
	}

	// Should contain placeholder
	if !strings.Contains(view, "my-feature-name") {
		t.Error("View should contain placeholder")
	}
}

func TestNameStepViewWithError(t *testing.T) {
	step := NewNameStep()
	step.Init()

	// Set an error
	step.validError = "Test error message"
	view := step.View()

	// Should contain error message
	if !strings.Contains(view, "Test error message") {
		t.Error("View should display validation error")
	}
}

func TestNameStepValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantValid bool
		wantError string
	}{
		{
			name:      "valid simple name",
			input:     "my-feature",
			wantValid: true,
		},
		{
			name:      "valid with numbers",
			input:     "feature-123",
			wantValid: true,
		},
		{
			name:      "valid single word",
			input:     "feature",
			wantValid: true,
		},
		{
			name:      "valid multiple hyphens",
			input:     "my-great-feature-name",
			wantValid: true,
		},
		{
			name:      "empty string",
			input:     "",
			wantValid: false,
			wantError: "cannot be empty",
		},
		{
			name:      "whitespace only",
			input:     "   ",
			wantValid: false,
			wantError: "cannot be empty",
		},
		{
			name:      "uppercase letters",
			input:     "My-Feature",
			wantValid: false,
			wantError: "lowercase",
		},
		{
			name:      "contains spaces",
			input:     "my feature",
			wantValid: false,
			wantError: "lowercase",
		},
		{
			name:      "contains underscore",
			input:     "my_feature",
			wantValid: false,
			wantError: "lowercase",
		},
		{
			name:      "starts with hyphen",
			input:     "-myfeature",
			wantValid: false,
			wantError: "start or end",
		},
		{
			name:      "ends with hyphen",
			input:     "myfeature-",
			wantValid: false,
			wantError: "start or end",
		},
		{
			name:      "consecutive hyphens",
			input:     "my--feature",
			wantValid: false,
			wantError: "consecutive hyphens",
		},
		{
			name:      "special characters",
			input:     "my-feature!",
			wantValid: false,
			wantError: "lowercase",
		},
		{
			name:      "too long",
			input:     strings.Repeat("a", 101),
			wantValid: false,
			wantError: "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := NewNameStep()
			step.input.SetValue(tt.input)

			valid := step.validate()

			if valid != tt.wantValid {
				t.Errorf("validate() = %v, want %v", valid, tt.wantValid)
			}

			if !tt.wantValid && tt.wantError != "" {
				if !strings.Contains(strings.ToLower(step.validError), strings.ToLower(tt.wantError)) {
					t.Errorf("validError = %q, want it to contain %q", step.validError, tt.wantError)
				}
			}
		})
	}
}

func TestNameStepIsValid(t *testing.T) {
	step := NewNameStep()

	// Invalid initially (empty)
	if step.IsValid() {
		t.Error("IsValid should return false for empty input")
	}

	// Valid with proper input
	step.input.SetValue("my-feature")
	if !step.IsValid() {
		t.Error("IsValid should return true for valid input")
	}
}

func TestNameStepName(t *testing.T) {
	step := NewNameStep()
	step.input.SetValue("  my-feature  ")

	name := step.Name()

	// Should trim whitespace
	if name != "my-feature" {
		t.Errorf("Name() = %q, want %q", name, "my-feature")
	}
}

func TestNameStepEnterKey(t *testing.T) {
	step := NewNameStep()
	step.Init()

	// Set valid input
	step.input.SetValue("my-feature")

	// Simulate enter key
	cmd := step.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Enter key should return a command")
	}

	// Execute command to get message
	msg := cmd()
	if _, ok := msg.(NameCompleteMsg); !ok {
		t.Errorf("Enter key should return NameCompleteMsg, got %T", msg)
	}
}

func TestNameStepEnterKeyWithInvalidInput(t *testing.T) {
	step := NewNameStep()
	step.Init()

	// Set invalid input (uppercase)
	step.input.SetValue("My-Feature")

	// Simulate enter key
	cmd := step.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	// Should not advance (cmd may be nil or not return NameCompleteMsg)
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(NameCompleteMsg); ok {
			t.Error("Enter key should not return NameCompleteMsg for invalid input")
		}
	}

	// Should have error set
	if step.validError == "" {
		t.Error("validError should be set after enter with invalid input")
	}
}

func TestNameStepTabKey(t *testing.T) {
	step := NewNameStep()
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

func TestNameStepShiftTabKey(t *testing.T) {
	step := NewNameStep()
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

func TestNameStepErrorClearsOnInput(t *testing.T) {
	step := NewNameStep()
	step.Init()

	// Set an error
	step.validError = "Some error"

	// Simulate typing
	step.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})

	// Error should be cleared
	if step.validError != "" {
		t.Error("validError should be cleared when user types")
	}
}

func TestNameStepFocusAndBlur(t *testing.T) {
	step := NewNameStep()

	// Focus
	cmd := step.Focus()
	if cmd == nil {
		t.Error("Focus should return a command")
	}

	// Input should be focused after Init
	step.Init()
	if !step.input.Focused() {
		t.Error("Input should be focused after Init")
	}

	// Blur
	step.Blur()
	if step.input.Focused() {
		t.Error("Input should not be focused after Blur")
	}
}

func TestNameStepSetSize(t *testing.T) {
	step := NewNameStep()

	step.SetSize(100, 50)

	if step.width != 100 {
		t.Errorf("width = %d, want 100", step.width)
	}
	if step.height != 50 {
		t.Errorf("height = %d, want 50", step.height)
	}
}

func TestNameStepPreferredHeight(t *testing.T) {
	step := NewNameStep()

	height := step.PreferredHeight()

	// Should return fixed height for content
	if height != 5 {
		t.Errorf("PreferredHeight() = %d, want 5", height)
	}
}
