package specwizard

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAgentPhase(t *testing.T) {
	phase := NewAgentPhase("test-feature", "A test feature", "anthropic/claude-sonnet-4-5", "./specs", "http://localhost:8080/mcp")

	assert.NotNil(t, phase)
	assert.Equal(t, "test-feature", phase.name)
	assert.Equal(t, "A test feature", phase.description)
	assert.Equal(t, "anthropic/claude-sonnet-4-5", phase.model)
	assert.Equal(t, "./specs", phase.specDir)
	assert.Equal(t, "http://localhost:8080/mcp", phase.mcpURL)
	assert.NotNil(t, phase.runnerCtx)
	assert.NotNil(t, phase.runnerCancel)
	assert.False(t, phase.isRunning)
	assert.False(t, phase.finished)
	assert.Nil(t, phase.err)
}

func TestAgentPhase_SetSize(t *testing.T) {
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp")
	phase.SetSize(80, 24)

	assert.Equal(t, 80, phase.width)
	assert.Equal(t, 24, phase.height)
}

func TestAgentPhase_PreferredHeight(t *testing.T) {
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp")
	height := phase.PreferredHeight()

	assert.Equal(t, 20, height)
}

func TestAgentPhase_Stop(t *testing.T) {
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp")

	// Should not panic even if runner not started
	phase.Stop()

	// Verify context was cancelled
	select {
	case <-phase.runnerCtx.Done():
		// Context was cancelled as expected
	default:
		t.Error("Expected context to be cancelled after Stop()")
	}
}

func TestAgentPhase_BuildAgentPrompt(t *testing.T) {
	phase := NewAgentPhase("my-feature", "A cool feature", "anthropic/claude-sonnet-4-5", "./specs", "http://localhost:8080/mcp")
	prompt := phase.buildAgentPrompt()

	assert.Contains(t, prompt, "my-feature")
	assert.Contains(t, prompt, "A cool feature")
	assert.Contains(t, prompt, "ask-questions")
	assert.Contains(t, prompt, "finish-spec")
	assert.Contains(t, prompt, "Spec Format")
}

func TestAgentPhase_View(t *testing.T) {
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp")
	phase.SetSize(80, 24)

	// Test initial state
	view := phase.View()
	assert.Contains(t, view, "Spec Wizard - Interview")

	// Test error state
	phase.err = assert.AnError
	phase.finished = true
	view = phase.View()
	assert.Contains(t, view, "Error")

	// Test finished state
	phase.err = nil
	phase.finished = true
	view = phase.View()
	assert.Contains(t, view, "Interview complete")
}

func TestAgentPhase_Update(t *testing.T) {
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp")

	// Test error message
	msg := AgentPhaseMsg{
		Type:  "error",
		Error: assert.AnError,
	}
	_ = phase.Update(msg)
	assert.Equal(t, assert.AnError, phase.err)
	assert.True(t, phase.finished)
	assert.False(t, phase.isRunning)
	assert.Nil(t, phase.spinner)

	// Test finished message
	phase2 := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp")
	phase2.isRunning = true
	msg2 := AgentPhaseMsg{
		Type: "finished",
	}
	_ = phase2.Update(msg2)
	assert.True(t, phase2.finished)
	assert.False(t, phase2.isRunning)
	assert.Nil(t, phase2.spinner)
}

func TestAgentPhaseMsg(t *testing.T) {
	// Test message types
	tests := []struct {
		name    string
		msgType string
		content string
		err     error
	}{
		{"text message", "text", "some text", nil},
		{"thinking message", "thinking", "thinking...", nil},
		{"finished message", "finished", "", nil},
		{"error message", "error", "", assert.AnError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := AgentPhaseMsg{
				Type:    tt.msgType,
				Content: tt.content,
				Error:   tt.err,
			}
			assert.Equal(t, tt.msgType, msg.Type)
			assert.Equal(t, tt.content, msg.Content)
			assert.Equal(t, tt.err, msg.Error)
		})
	}
}

func TestAgentPhase_Init(t *testing.T) {
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp")

	cmd := phase.Init()
	assert.NotNil(t, cmd)
}
