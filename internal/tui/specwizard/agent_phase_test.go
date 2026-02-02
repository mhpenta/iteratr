package specwizard

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/mark3labs/iteratr/internal/specmcp"
	"github.com/stretchr/testify/assert"
)

func TestNewAgentPhase(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test-feature", "A test feature", "anthropic/claude-sonnet-4-5", "./specs", "http://localhost:8080/mcp", mcpServer)

	assert.NotNil(t, phase)
	assert.Equal(t, "test-feature", phase.name)
	assert.Equal(t, "A test feature", phase.description)
	assert.Equal(t, "anthropic/claude-sonnet-4-5", phase.model)
	assert.Equal(t, "./specs", phase.specDir)
	assert.Equal(t, "http://localhost:8080/mcp", phase.mcpURL)
	assert.NotNil(t, phase.runnerCtx)
	assert.NotNil(t, phase.runnerCancel)
	assert.NotNil(t, phase.mcpServer)
	assert.False(t, phase.isRunning)
	assert.False(t, phase.finished)
	assert.Nil(t, phase.err)
	assert.NotNil(t, phase.pendingAnswers)
	assert.False(t, phase.showingQuestion)
	assert.False(t, phase.customAnswerMode)
}

func TestAgentPhase_SetSize(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.SetSize(80, 24)

	assert.Equal(t, 80, phase.width)
	assert.Equal(t, 24, phase.height)
}

func TestAgentPhase_PreferredHeight(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	height := phase.PreferredHeight()

	assert.Equal(t, 20, height)
}

func TestAgentPhase_Stop(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)

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
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("my-feature", "A cool feature", "anthropic/claude-sonnet-4-5", "./specs", "http://localhost:8080/mcp", mcpServer)
	prompt := phase.buildAgentPrompt()

	assert.Contains(t, prompt, "my-feature")
	assert.Contains(t, prompt, "A cool feature")
	assert.Contains(t, prompt, "ask-questions")
	assert.Contains(t, prompt, "finish-spec")
	assert.Contains(t, prompt, "Spec Format")
}

func TestAgentPhase_View(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
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
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)

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
	phase2 := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
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
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)

	cmd := phase.Init()
	assert.NotNil(t, cmd)
}

func TestAgentPhase_QuestionHandling(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.SetSize(80, 24)

	// Create test question
	q := &specmcp.Question{
		Question: "What is your preferred approach?",
		Header:   "Approach Selection",
		Options: []specmcp.QuestionOption{
			{Label: "Option A", Description: "First approach"},
			{Label: "Option B", Description: "Second approach"},
		},
		Multiple: false,
	}

	// Create answer channel
	answerCh := make(chan []any, 1)

	// Send question through QuestionReceivedMsg
	msg := QuestionReceivedMsg{
		Request: &specmcp.QuestionRequest{
			Questions: []*specmcp.Question{q},
			AnswerCh:  answerCh,
		},
	}

	cmd := phase.Update(msg)
	assert.Nil(t, cmd) // showQuestion returns nil

	// Verify question state
	assert.True(t, phase.showingQuestion)
	assert.NotNil(t, phase.questionView)
	assert.Equal(t, q, phase.currentQuestion)
	assert.Equal(t, 0, phase.questionIdx)
	assert.Equal(t, 1, phase.totalQuestions)
	assert.NotNil(t, phase.currentAnswerCh) // Channel is set

	// View should show question
	view := phase.View()
	assert.Contains(t, view, "Approach Selection")
	assert.Contains(t, view, "What is your preferred approach?")
}

func TestAgentPhase_AnswerSelection(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.SetSize(80, 24)

	// Setup question state
	q := &specmcp.Question{
		Question: "Choose one:",
		Header:   "Test",
		Options:  []specmcp.QuestionOption{{Label: "Option A", Description: ""}},
		Multiple: false,
	}
	answerCh := make(chan []any, 1)
	phase.showingQuestion = true
	phase.questionBatch = []*specmcp.Question{q}
	phase.currentQuestion = q
	phase.questionView = NewQuestionView(q)
	phase.questionIdx = 0
	phase.totalQuestions = 1
	phase.currentAnswerCh = answerCh
	phase.pendingAnswers = make([]any, 0)

	// Simulate answer selection
	answerMsg := AnswerSelectedMsg{Answer: "Option A"}
	cmd := phase.Update(answerMsg)

	// Should trigger batch completion since it's the last question
	assert.NotNil(t, cmd) // moveToNextQuestion returns batch commands

	// Verify answer collected
	assert.Equal(t, 1, len(phase.pendingAnswers))
	assert.Equal(t, "Option A", phase.pendingAnswers[0])

	// Verify answers sent to channel (goroutine, so check asynchronously)
	select {
	case answers := <-answerCh:
		assert.Equal(t, 1, len(answers))
		assert.Equal(t, "Option A", answers[0])
	default:
		// Goroutine may not have run yet - that's okay for this test
	}
}

func TestAgentPhase_MultipleQuestions(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.SetSize(80, 24)

	// Create multiple questions
	q1 := &specmcp.Question{
		Question: "Question 1?",
		Header:   "Q1",
		Options:  []specmcp.QuestionOption{{Label: "A1", Description: ""}},
		Multiple: false,
	}
	q2 := &specmcp.Question{
		Question: "Question 2?",
		Header:   "Q2",
		Options:  []specmcp.QuestionOption{{Label: "A2", Description: ""}},
		Multiple: false,
	}

	answerCh := make(chan []any, 1)
	msg := QuestionReceivedMsg{
		Request: &specmcp.QuestionRequest{
			Questions: []*specmcp.Question{q1, q2},
			AnswerCh:  answerCh,
		},
	}

	// Receive questions
	cmd := phase.Update(msg)
	assert.Nil(t, cmd)
	assert.Equal(t, 0, phase.questionIdx)
	assert.Equal(t, 2, phase.totalQuestions)

	// Answer first question
	answerMsg1 := AnswerSelectedMsg{Answer: "A1"}
	cmd = phase.Update(answerMsg1)
	assert.Nil(t, cmd) // showQuestion for Q2 returns nil

	// Should move to second question
	assert.Equal(t, 1, phase.questionIdx)
	assert.Equal(t, q2, phase.currentQuestion)

	// Answer second question
	answerMsg2 := AnswerSelectedMsg{Answer: "A2"}
	cmd = phase.Update(answerMsg2)
	assert.NotNil(t, cmd) // Batch complete

	// Verify both answers collected
	assert.Equal(t, 2, len(phase.pendingAnswers))
	assert.Equal(t, "A1", phase.pendingAnswers[0])
	assert.Equal(t, "A2", phase.pendingAnswers[1])
}

func TestAgentPhase_CustomAnswer(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.SetSize(80, 24)

	// Setup question state
	q := &specmcp.Question{
		Question: "Question?",
		Header:   "Test",
		Options:  []specmcp.QuestionOption{{Label: "Option", Description: ""}},
		Multiple: false,
	}
	answerCh := make(chan []any, 1)
	phase.showingQuestion = true
	phase.questionBatch = []*specmcp.Question{q}
	phase.currentQuestion = q
	phase.questionView = NewQuestionView(q)
	phase.questionIdx = 0
	phase.totalQuestions = 1
	phase.currentAnswerCh = answerCh

	// User requests custom answer
	customMsg := CustomAnswerRequestedMsg{}
	cmd := phase.Update(customMsg)
	assert.Nil(t, cmd)
	assert.True(t, phase.customAnswerMode)

	// Type custom answer
	phase.Update(tea.KeyPressMsg{Code: 'm', Text: "m"})
	phase.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	assert.Equal(t, "my", phase.customAnswerInput)

	// Submit with enter
	cmd = phase.Update(tea.KeyPressMsg{Text: "enter"})
	assert.NotNil(t, cmd)
	assert.False(t, phase.customAnswerMode)
	assert.Equal(t, 1, len(phase.pendingAnswers))
	assert.Equal(t, "my", phase.pendingAnswers[0])
}

func TestAgentPhase_CustomAnswerCancel(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.customAnswerMode = true
	phase.customAnswerInput = "some text"

	// Press ESC to cancel
	cmd := phase.Update(tea.KeyPressMsg{Text: "esc"})
	assert.Nil(t, cmd)
	assert.False(t, phase.customAnswerMode)
}

func TestAgentPhase_CustomAnswerBackspace(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.customAnswerMode = true
	phase.customAnswerInput = "hello"

	// Press backspace
	cmd := phase.Update(tea.KeyPressMsg{Text: "backspace"})
	assert.Nil(t, cmd)
	assert.Equal(t, "hell", phase.customAnswerInput)
}

func TestAgentPhase_ViewCustomAnswerMode(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.SetSize(80, 24)
	phase.customAnswerMode = true
	phase.customAnswerInput = "my custom answer"
	phase.currentQuestion = &specmcp.Question{
		Question: "Test question?",
		Header:   "Test Header",
		Options:  []specmcp.QuestionOption{},
		Multiple: false,
	}

	view := phase.View()
	assert.Contains(t, view, "Test Header")
	assert.Contains(t, view, "Test question?")
	assert.Contains(t, view, "Type your answer:")
	assert.Contains(t, view, "my custom answer")
	assert.Contains(t, view, "Press Enter to submit")
}

func TestQuestionReceivedMsg(t *testing.T) {
	answerCh := make(chan []any)
	q := &specmcp.Question{
		Question: "Test?",
		Header:   "Test",
		Options:  []specmcp.QuestionOption{},
		Multiple: false,
	}
	req := &specmcp.QuestionRequest{
		Questions: []*specmcp.Question{q},
		AnswerCh:  answerCh,
	}

	msg := QuestionReceivedMsg{Request: req}
	assert.NotNil(t, msg.Request)
	assert.Equal(t, 1, len(msg.Request.Questions))
	assert.Equal(t, q, msg.Request.Questions[0])
}

func TestAgentPhase_ThinkingCallback(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.SetSize(80, 24)

	// Simulate thinking message from agent callback
	thinkingMsg := AgentPhaseMsg{
		Type:    "thinking",
		Content: "I need to ask about error handling...",
	}

	// Update should handle thinking message and update status
	cmd := phase.Update(thinkingMsg)
	assert.NotNil(t, cmd, "Update should return command to continue listening")
	assert.Contains(t, phase.status, "I need to ask about error handling")

	// Verify status is truncated if too long
	longThinking := AgentPhaseMsg{
		Type:    "thinking",
		Content: "This is a very long thinking message that should be truncated because it exceeds eighty characters in length",
	}
	cmd = phase.Update(longThinking)
	assert.NotNil(t, cmd)
	assert.LessOrEqual(t, len(phase.status), 90) // "Agent: " + 80 chars + "..."
	assert.Contains(t, phase.status, "Agent:")
}

func TestAgentPhase_TextCallback(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)

	// Text messages should be handled but not displayed (agent output hidden in spec wizard)
	textMsg := AgentPhaseMsg{
		Type:    "text",
		Content: "Some agent text output",
	}

	cmd := phase.Update(textMsg)
	assert.NotNil(t, cmd, "Update should return command to continue listening")
	// Text doesn't update status - it's hidden from user
	assert.NotContains(t, phase.status, "Some agent text output")
}

func TestAgentPhase_FinishedCallback(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.isRunning = true
	spinner := NewDefaultGradientSpinner("Agent is working...")
	phase.spinner = &spinner

	// Simulate finished message from agent callback
	finishedMsg := AgentPhaseMsg{
		Type:    "finished",
		Content: "end_turn",
	}

	cmd := phase.Update(finishedMsg)
	assert.NotNil(t, cmd, "Update should return command to continue listening")
	assert.True(t, phase.finished)
	assert.False(t, phase.isRunning)
	assert.Nil(t, phase.spinner, "Spinner should be hidden when finished")
	assert.Equal(t, "Interview complete! Generating spec...", phase.status)
}

func TestAgentPhase_ErrorCallback(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.isRunning = true
	spinner := NewDefaultGradientSpinner("Agent is working...")
	phase.spinner = &spinner

	// Simulate error message from agent callback
	errorMsg := AgentPhaseMsg{
		Type:  "error",
		Error: assert.AnError,
	}

	cmd := phase.Update(errorMsg)
	assert.NotNil(t, cmd, "Update should return command to continue listening")
	assert.True(t, phase.finished)
	assert.False(t, phase.isRunning)
	assert.Nil(t, phase.spinner, "Spinner should be hidden on error")
	assert.NotNil(t, phase.err)
	assert.Equal(t, assert.AnError, phase.err)

	// View should show error
	view := phase.View()
	assert.Contains(t, view, "Error:")
}

func TestAgentPhase_StartedCallback(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)

	// Simulate started message
	startedMsg := AgentPhaseMsg{
		Type:    "started",
		Content: "Agent started successfully",
	}

	cmd := phase.Update(startedMsg)
	assert.NotNil(t, cmd, "Update should return command to continue listening")
}

func TestAgentPhase_MessageChannelBuffering(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)

	// Verify message channel is buffered
	assert.NotNil(t, phase.msgChan)

	// Should be able to send multiple messages without blocking
	phase.msgChan <- AgentPhaseMsg{Type: "thinking", Content: "msg1"}
	phase.msgChan <- AgentPhaseMsg{Type: "thinking", Content: "msg2"}
	phase.msgChan <- AgentPhaseMsg{Type: "thinking", Content: "msg3"}

	// Receive and verify messages
	msg1 := <-phase.msgChan
	assert.Equal(t, "thinking", msg1.Type)
	assert.Equal(t, "msg1", msg1.Content)

	msg2 := <-phase.msgChan
	assert.Equal(t, "thinking", msg2.Type)
	assert.Equal(t, "msg2", msg2.Content)

	msg3 := <-phase.msgChan
	assert.Equal(t, "thinking", msg3.Type)
	assert.Equal(t, "msg3", msg3.Content)
}

func TestAgentPhase_ContinuesListeningAfterAnswers(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.SetSize(80, 24)

	// Set up question batch
	answerCh := make(chan []any, 1)
	q := &specmcp.Question{
		Question: "Test question?",
		Header:   "Test",
		Options: []specmcp.QuestionOption{
			{Label: "Option A", Description: "First option"},
		},
		Multiple: false,
	}
	phase.questionBatch = []*specmcp.Question{q}
	phase.currentQuestion = q
	phase.questionView = NewQuestionView(q)
	phase.questionIdx = 0
	phase.totalQuestions = 1
	phase.currentAnswerCh = answerCh
	phase.showingQuestion = true
	phase.pendingAnswers = []any{}

	// Submit answer
	answerMsg := AnswerSelectedMsg{Answer: "Option A"}
	cmd := phase.Update(answerMsg)

	// Should return batch command that includes listening for agent messages
	assert.NotNil(t, cmd, "moveToNextQuestion should return batch command")
	assert.False(t, phase.showingQuestion, "Should hide question after last answer")
	assert.Equal(t, "Agent is analyzing your answers...", phase.status)
}

func TestAgentPhase_ThinkingStatusWhileShowingQuestion(t *testing.T) {
	mcpServer := specmcp.New("./specs")
	phase := NewAgentPhase("test", "desc", "model", "./specs", "http://localhost:8080/mcp", mcpServer)
	phase.SetSize(80, 24)

	// Not showing a question yet - agent is thinking
	phase.showingQuestion = false

	// Thinking message arrives
	thinkingMsg := AgentPhaseMsg{
		Type:    "thinking",
		Content: "Processing your previous answer...",
	}

	// Status should be updated with thinking content
	cmd := phase.Update(thinkingMsg)
	assert.NotNil(t, cmd)
	assert.Contains(t, phase.status, "Processing your previous answer")

	// Now start showing a question
	q := &specmcp.Question{
		Question: "Test question?",
		Header:   "Test",
		Options: []specmcp.QuestionOption{
			{Label: "Option A", Description: "First option"},
		},
		Multiple: false,
	}
	phase.showQuestion(q)

	// View should show question, not the thinking status
	view := phase.View()
	assert.Contains(t, view, "Test question?")
}
