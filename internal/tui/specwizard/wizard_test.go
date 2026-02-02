package specwizard

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/tui/wizard"
)

func TestWizardModel_Init(t *testing.T) {
	m := &WizardModel{
		specDir: "./specs",
	}

	cmd := m.Init()

	// Should initialize name step
	if m.nameStep == nil {
		t.Error("Expected nameStep to be initialized")
	}

	// Should return a focus command
	if cmd == nil {
		t.Error("Expected Init to return a focus command")
	}
}

func TestWizardModel_StepNavigation(t *testing.T) {
	m := &WizardModel{
		step:    0,
		specDir: "./specs",
	}
	m.Init()

	// Set valid name
	m.nameStep.input.SetValue("test-spec")

	// Test goNext from step 0
	_, _ = m.goNext()
	if m.step != 1 {
		t.Errorf("Expected step 1, got %d", m.step)
	}
	if m.result.Name != "test-spec" {
		t.Errorf("Expected name 'test-spec', got '%s'", m.result.Name)
	}

	// Test goBack to step 0
	_, _ = m.goBack()
	if m.step != 0 {
		t.Errorf("Expected step 0, got %d", m.step)
	}

	// Test goNext with invalid name
	m.nameStep.input.SetValue("")
	_, _ = m.goNext()
	if m.step != 0 {
		t.Errorf("Expected to stay on step 0 with invalid name, got step %d", m.step)
	}
}

func TestWizardModel_ButtonFocus(t *testing.T) {
	m := &WizardModel{
		step:    0,
		width:   80,
		height:  24,
		specDir: "./specs",
	}
	m.Init()

	// Initially buttons should not be focused
	if m.buttonFocused {
		t.Error("Expected buttons to not be focused initially")
	}

	// Press Tab - should move focus to buttons
	msg := tea.KeyPressMsg{Text: "tab"}
	_, _ = m.Update(msg)
	if !m.buttonFocused {
		t.Error("Expected buttons to be focused after Tab")
	}

	// Press Tab again - should cycle through buttons
	if m.buttonBar == nil {
		t.Fatal("Expected buttonBar to be initialized")
	}
	initialBtn := m.buttonBar.FocusedButton()
	_, _ = m.Update(msg)
	nextBtn := m.buttonBar.FocusedButton()
	if initialBtn == nextBtn {
		t.Error("Expected button focus to cycle on Tab")
	}
}

func TestWizardModel_CancelOnFirstStep(t *testing.T) {
	m := &WizardModel{
		step:    0,
		specDir: "./specs",
	}
	m.Init()

	// Press ESC on first step - should cancel
	msg := tea.KeyPressMsg{Text: "esc"}
	model, cmd := m.Update(msg)
	wm := model.(*WizardModel)

	if !wm.cancelled {
		t.Error("Expected wizard to be cancelled after ESC on first step")
	}
	if cmd == nil || cmd() != tea.Quit() {
		t.Error("Expected Quit command after ESC on first step")
	}
}

func TestWizardModel_BackOnLaterSteps(t *testing.T) {
	m := &WizardModel{
		step:    2,
		specDir: "./specs",
	}
	m.initCurrentStep()

	// Press ESC on later step - should go back
	msg := tea.KeyPressMsg{Text: "esc"}
	model, _ := m.Update(msg)
	wm := model.(*WizardModel)

	if wm.step != 1 {
		t.Errorf("Expected to go back to step 1, got step %d", wm.step)
	}
	if wm.cancelled {
		t.Error("Expected wizard not to be cancelled on ESC from later step")
	}
}

func TestWizardModel_WindowResize(t *testing.T) {
	m := &WizardModel{
		step:    0,
		width:   80,
		height:  24,
		specDir: "./specs",
	}
	m.Init()

	// Send window resize message
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	_, _ = m.Update(msg)

	if m.width != 120 {
		t.Errorf("Expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("Expected height 40, got %d", m.height)
	}
}

func TestWizardModel_IsStepValid(t *testing.T) {
	tests := []struct {
		name      string
		step      int
		nameValue string
		modelID   string
		wantValid bool
	}{
		{
			name:      "step 0 with valid name",
			step:      0,
			nameValue: "test-spec",
			wantValid: true,
		},
		{
			name:      "step 0 with empty name",
			step:      0,
			nameValue: "",
			wantValid: false,
		},
		{
			name:      "step 0 with invalid name (uppercase)",
			step:      0,
			nameValue: "TestSpec",
			wantValid: false,
		},
		{
			name:      "step 1 always valid",
			step:      1,
			wantValid: true,
		},
		{
			name:      "step 2 with model selected",
			step:      2,
			modelID:   "anthropic/claude-sonnet-4-5",
			wantValid: true,
		},
		{
			name:      "step 2 without model",
			step:      2,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &WizardModel{
				step:    tt.step,
				specDir: "./specs",
			}
			m.initCurrentStep()

			// Set test data
			if tt.step == 0 && m.nameStep != nil {
				m.nameStep.input.SetValue(tt.nameValue)
			}
			// Skip step 2 validation test - requires model selector to actually load models
			if tt.step == 2 {
				// Model selector needs actual models loaded, skip this test
				return
			}

			got := m.isStepValid()
			if got != tt.wantValid {
				t.Errorf("isStepValid() = %v, want %v", got, tt.wantValid)
			}
		})
	}
}

func TestWizardModel_CalculateModalDimensions(t *testing.T) {
	tests := []struct {
		name             string
		width            int
		height           int
		step             int
		minModalWidth    int
		maxModalWidth    int
		minModalHeight   int
		minContentWidth  int
		minContentHeight int
	}{
		{
			name:             "normal terminal size",
			width:            80,
			height:           24,
			step:             0,
			minModalWidth:    60,
			maxModalWidth:    100,
			minModalHeight:   15,
			minContentWidth:  40,
			minContentHeight: 5,
		},
		{
			name:             "small terminal",
			width:            40,
			height:           15,
			step:             0,
			minModalWidth:    60, // Should clamp to min
			maxModalWidth:    100,
			minModalHeight:   15,
			minContentWidth:  40,
			minContentHeight: 5,
		},
		{
			name:             "large terminal",
			width:            200,
			height:           60,
			step:             0,
			minModalWidth:    60,
			maxModalWidth:    100, // Should clamp to max for input steps
			minModalHeight:   15,
			minContentWidth:  40,
			minContentHeight: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &WizardModel{
				width:   tt.width,
				height:  tt.height,
				step:    tt.step,
				specDir: "./specs",
			}
			m.initCurrentStep()

			modalWidth, modalHeight, contentWidth, contentHeight := m.calculateModalDimensions()

			if modalWidth < tt.minModalWidth {
				t.Errorf("modalWidth %d < min %d", modalWidth, tt.minModalWidth)
			}
			if modalWidth > tt.maxModalWidth {
				t.Errorf("modalWidth %d > max %d", modalWidth, tt.maxModalWidth)
			}
			if modalHeight < tt.minModalHeight {
				t.Errorf("modalHeight %d < min %d", modalHeight, tt.minModalHeight)
			}
			if contentWidth < tt.minContentWidth {
				t.Errorf("contentWidth %d < min %d", contentWidth, tt.minContentWidth)
			}
			if contentHeight < tt.minContentHeight {
				t.Errorf("contentHeight %d < min %d", contentHeight, tt.minContentHeight)
			}
		})
	}
}

func TestWizardModel_ButtonActivation(t *testing.T) {
	m := &WizardModel{
		step:    0,
		specDir: "./specs",
	}
	m.Init()
	m.nameStep.input.SetValue("test-spec")

	// Activate Back button on step 0 - should cancel
	model, cmd := m.activateButton(wizard.ButtonBack)
	wm := model.(*WizardModel)
	if !wm.cancelled {
		t.Error("Expected wizard to be cancelled when activating Back on step 0")
	}
	if cmd == nil || cmd() != tea.Quit() {
		t.Error("Expected Quit command")
	}

	// Reset and test Next button
	m = &WizardModel{
		step:    0,
		specDir: "./specs",
	}
	m.Init()
	m.nameStep.input.SetValue("test-spec")

	// Activate Next button - should advance
	model, _ = m.activateButton(wizard.ButtonNext)
	wm = model.(*WizardModel)
	if wm.step != 1 {
		t.Errorf("Expected to advance to step 1, got step %d", wm.step)
	}
	if wm.result.Name != "test-spec" {
		t.Errorf("Expected name 'test-spec', got '%s'", wm.result.Name)
	}
}

func TestWizardModel_View(t *testing.T) {
	m := &WizardModel{
		step:    0,
		width:   80,
		height:  24,
		specDir: "./specs",
	}
	m.Init()

	view := m.View()

	// Should have alt screen enabled
	if !view.AltScreen {
		t.Error("Expected AltScreen to be enabled")
	}

	// Should have mouse mode enabled
	if view.MouseMode != tea.MouseModeCellMotion {
		t.Error("Expected MouseModeCellMotion to be enabled")
	}

	// Content should not be nil
	if view.Content == nil {
		t.Error("Expected view content to not be nil")
	}
}

func TestWizardModel_PreferredHeight(t *testing.T) {
	m := &WizardModel{
		step:    0,
		specDir: "./specs",
	}
	m.initCurrentStep()

	height := m.getStepPreferredHeight()
	if height <= 0 {
		t.Errorf("Expected positive preferred height, got %d", height)
	}

	// Name step should have a specific height
	if height != m.nameStep.PreferredHeight() {
		t.Errorf("Expected preferred height %d, got %d", m.nameStep.PreferredHeight(), height)
	}
}

// TestWizardModel_ConfirmationModal_Show tests that ESC on agent phase shows confirmation modal
func TestWizardModel_ConfirmationModal_Show(t *testing.T) {
	m := &WizardModel{
		step:    3, // Agent phase
		width:   80,
		height:  24,
		specDir: "./specs",
	}

	// Press ESC during agent phase - should show confirmation modal
	msg := tea.KeyPressMsg{Text: "esc"}
	model, cmd := m.Update(msg)
	wm := model.(*WizardModel)

	if !wm.confirmCancelling {
		t.Error("Expected confirmation modal to be shown after ESC on agent phase")
	}
	if wm.cancelled {
		t.Error("Expected wizard not to be cancelled immediately, should show confirmation first")
	}
	if cmd != nil {
		t.Error("Expected no command when showing confirmation modal")
	}
	if wm.confirmFocusYes {
		t.Error("Expected focus to be on 'No' button by default")
	}
}

// TestWizardModel_ConfirmationModal_Dismiss tests dismissing the confirmation modal
func TestWizardModel_ConfirmationModal_Dismiss(t *testing.T) {
	m := &WizardModel{
		step:              3,
		confirmCancelling: true, // Modal already showing
		confirmFocusYes:   false,
		width:             80,
		height:            24,
		specDir:           "./specs",
	}

	// Press ESC while modal showing - should dismiss it
	msg := tea.KeyPressMsg{Text: "esc"}
	model, cmd := m.Update(msg)
	wm := model.(*WizardModel)

	if wm.confirmCancelling {
		t.Error("Expected confirmation modal to be dismissed after ESC")
	}
	if wm.cancelled {
		t.Error("Expected wizard not to be cancelled after dismissing modal")
	}
	if cmd != nil {
		t.Error("Expected no command when dismissing modal")
	}
}

// TestWizardModel_ConfirmationModal_ConfirmNo tests confirming 'No' (don't cancel)
func TestWizardModel_ConfirmationModal_ConfirmNo(t *testing.T) {
	m := &WizardModel{
		step:              3,
		confirmCancelling: true,
		confirmFocusYes:   false, // Focus on "No"
		width:             80,
		height:            24,
		specDir:           "./specs",
	}

	// Press Enter with "No" focused - should dismiss modal and continue
	msg := tea.KeyPressMsg{Text: "enter"}
	model, cmd := m.Update(msg)
	wm := model.(*WizardModel)

	if wm.confirmCancelling {
		t.Error("Expected confirmation modal to be dismissed after confirming No")
	}
	if wm.cancelled {
		t.Error("Expected wizard not to be cancelled after confirming No")
	}
	if cmd != nil {
		t.Error("Expected no command when confirming No")
	}
}

// TestWizardModel_ConfirmationModal_ConfirmYes tests confirming 'Yes' (cancel wizard)
func TestWizardModel_ConfirmationModal_ConfirmYes(t *testing.T) {
	m := &WizardModel{
		step:              3,
		confirmCancelling: true,
		confirmFocusYes:   true, // Focus on "Yes"
		width:             80,
		height:            24,
		specDir:           "./specs",
	}

	// Press Enter with "Yes" focused - should cancel wizard
	msg := tea.KeyPressMsg{Text: "enter"}
	model, cmd := m.Update(msg)
	wm := model.(*WizardModel)

	if !wm.cancelled {
		t.Error("Expected wizard to be cancelled after confirming Yes")
	}
	if cmd == nil || cmd() != tea.Quit() {
		t.Error("Expected Quit command after confirming Yes")
	}
}

// TestWizardModel_ConfirmationModal_ToggleFocus tests toggling focus between Yes/No
func TestWizardModel_ConfirmationModal_ToggleFocus(t *testing.T) {
	tests := []struct {
		name         string
		initialFocus bool
		key          string
		expectFocus  bool
	}{
		{
			name:         "Tab from No to Yes",
			initialFocus: false,
			key:          "tab",
			expectFocus:  true,
		},
		{
			name:         "Tab from Yes to No",
			initialFocus: true,
			key:          "tab",
			expectFocus:  false,
		},
		{
			name:         "Left from Yes to No",
			initialFocus: true,
			key:          "left",
			expectFocus:  false,
		},
		{
			name:         "Right from No to Yes",
			initialFocus: false,
			key:          "right",
			expectFocus:  true,
		},
		{
			name:         "Shift+Tab from No to Yes",
			initialFocus: false,
			key:          "shift+tab",
			expectFocus:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &WizardModel{
				step:              3,
				confirmCancelling: true,
				confirmFocusYes:   tt.initialFocus,
				width:             80,
				height:            24,
				specDir:           "./specs",
			}

			msg := tea.KeyPressMsg{Text: tt.key}
			model, _ := m.Update(msg)
			wm := model.(*WizardModel)

			if wm.confirmFocusYes != tt.expectFocus {
				t.Errorf("Expected confirmFocusYes=%v, got %v", tt.expectFocus, wm.confirmFocusYes)
			}
			if !wm.confirmCancelling {
				t.Error("Expected confirmation modal to still be showing")
			}
		})
	}
}

// TestWizardModel_ConfirmationModal_QuickKeys tests quick 'y' and 'n' keys
func TestWizardModel_ConfirmationModal_QuickKeys(t *testing.T) {
	tests := []struct {
		name            string
		key             string
		expectCanceled  bool
		expectModalOpen bool
	}{
		{
			name:            "Quick 'y' key cancels wizard",
			key:             "y",
			expectCanceled:  true,
			expectModalOpen: false,
		},
		{
			name:            "Quick 'n' key dismisses modal",
			key:             "n",
			expectCanceled:  false,
			expectModalOpen: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &WizardModel{
				step:              3,
				confirmCancelling: true,
				confirmFocusYes:   false,
				width:             80,
				height:            24,
				specDir:           "./specs",
			}

			msg := tea.KeyPressMsg{Text: tt.key}
			model, cmd := m.Update(msg)
			wm := model.(*WizardModel)

			if wm.cancelled != tt.expectCanceled {
				t.Errorf("Expected cancelled=%v, got %v", tt.expectCanceled, wm.cancelled)
			}
			if wm.confirmCancelling != tt.expectModalOpen {
				t.Errorf("Expected confirmCancelling=%v, got %v", tt.expectModalOpen, wm.confirmCancelling)
			}
			if tt.expectCanceled && (cmd == nil || cmd() != tea.Quit()) {
				t.Error("Expected Quit command when cancelling")
			}
		})
	}
}

// TestWizardModel_ConfirmationModal_IgnoresOtherKeys tests that other keys are ignored
func TestWizardModel_ConfirmationModal_IgnoresOtherKeys(t *testing.T) {
	m := &WizardModel{
		step:              3,
		confirmCancelling: true,
		confirmFocusYes:   false,
		width:             80,
		height:            24,
		specDir:           "./specs",
	}

	// Press an unhandled key - should be ignored
	msg := tea.KeyPressMsg{Text: "x"}
	model, cmd := m.Update(msg)
	wm := model.(*WizardModel)

	if !wm.confirmCancelling {
		t.Error("Expected confirmation modal to still be showing")
	}
	if wm.cancelled {
		t.Error("Expected wizard not to be cancelled")
	}
	if cmd != nil {
		t.Error("Expected no command for unhandled key")
	}
}

// TestWizardModel_ConfirmationModal_BlocksStepInput tests that confirmation modal blocks step input
func TestWizardModel_ConfirmationModal_BlocksStepInput(t *testing.T) {
	m := &WizardModel{
		step:              3,
		confirmCancelling: true,
		confirmFocusYes:   false,
		width:             80,
		height:            24,
		specDir:           "./specs",
	}

	// Press a key that would normally be handled by step - should be ignored
	msg := tea.KeyPressMsg{Text: "down"}
	_, cmd := m.Update(msg)

	// Command should be nil because modal intercepts all input
	if cmd != nil {
		t.Error("Expected modal to block step input")
	}
}

// TestWizardModel_ConfirmationModal_ViewOverlay tests that confirmation modal renders as overlay
func TestWizardModel_ConfirmationModal_ViewOverlay(t *testing.T) {
	m := &WizardModel{
		step:              3,
		confirmCancelling: true,
		confirmFocusYes:   false,
		width:             80,
		height:            24,
		specDir:           "./specs",
	}

	view := m.View()

	// View should render (basic smoke test)
	if view.Content == nil {
		t.Error("Expected view content to not be nil")
	}

	// View should still have alt screen and mouse mode
	if !view.AltScreen {
		t.Error("Expected AltScreen to be enabled")
	}
}

// TestWizardModel_ConfirmationModal_OnlyOnAgentPhase tests that confirmation only shows on step 3
func TestWizardModel_ConfirmationModal_OnlyOnAgentPhase(t *testing.T) {
	// Test that ESC on other steps doesn't trigger confirmation modal
	tests := []struct {
		step int
		name string
	}{
		{step: 0, name: "step 0 (name)"},
		{step: 1, name: "step 1 (description)"},
		{step: 2, name: "step 2 (model)"},
		{step: 4, name: "step 4 (completion)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &WizardModel{
				step:    tt.step,
				width:   80,
				height:  24,
				specDir: "./specs",
			}
			m.initCurrentStep()

			msg := tea.KeyPressMsg{Text: "esc"}
			model, _ := m.Update(msg)
			wm := model.(*WizardModel)

			if wm.confirmCancelling {
				t.Errorf("Expected confirmation modal NOT to show on step %d", tt.step)
			}
		})
	}
}
