package specmcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mark3labs/mcp-go/mcp"
)

// handleAskQuestions sends questions to the UI and blocks until answers are received.
// Questions are displayed one at a time with multiple choice options.
// Each question automatically gets "Type your own answer" appended to options.
func (s *Server) handleAskQuestions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments
	args := request.GetArguments()
	if args == nil {
		return mcp.NewToolResultError("no arguments provided"), nil
	}

	// Extract questions array
	questionsRaw, ok := args["questions"]
	if !ok {
		return mcp.NewToolResultError("missing 'questions' parameter"), nil
	}

	// Type assert to []any (mcp-go returns arrays as []any)
	questionsArray, ok := questionsRaw.([]any)
	if !ok {
		return mcp.NewToolResultError("'questions' is not an array"), nil
	}

	if len(questionsArray) == 0 {
		return mcp.NewToolResultError("at least one question is required"), nil
	}

	// Parse all questions
	questions := make([]*Question, 0, len(questionsArray))
	for i, qRaw := range questionsArray {
		qMap, ok := qRaw.(map[string]any)
		if !ok {
			return mcp.NewToolResultError(fmt.Sprintf("question %d is not an object", i)), nil
		}

		q, err := parseQuestion(qMap)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("question %d invalid: %v", i, err)), nil
		}

		questions = append(questions, q)
	}

	// Create answer channel for UI to respond
	answerCh := make(chan []any)

	// Send questions to UI and block waiting for answers
	req := &QuestionRequest{
		Questions: questions,
		AnswerCh:  answerCh,
	}

	select {
	case s.questionCh <- req:
		// Questions sent successfully, wait for answers
	case <-ctx.Done():
		return mcp.NewToolResultError("context cancelled while sending questions"), nil
	}

	// Block until UI sends answers back
	var answers []any
	select {
	case answers = <-answerCh:
		// Answers received
	case <-ctx.Done():
		return mcp.NewToolResultError("context cancelled while waiting for answers"), nil
	}

	// Format and return answers as JSON
	return formatAnswersResponse(answers)
}

// handleFinishSpec saves the completed spec to a file and updates README.md.
// Returns success with file path or error if save fails.
func (s *Server) handleFinishSpec(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments
	args := request.GetArguments()
	if args == nil {
		return mcp.NewToolResultError("no arguments provided"), nil
	}

	// Extract required content parameter
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return mcp.NewToolResultError("missing or empty 'content' parameter"), nil
	}

	// Extract required name parameter
	name, ok := args["name"].(string)
	if !ok || name == "" {
		return mcp.NewToolResultError("missing or empty 'name' parameter"), nil
	}

	// TODO (TAS-11): Implement slugify function with transliteration
	slug := name // Use name directly for now, TAS-11 will implement proper slugify

	// TODO (TAS-12): Validate spec content (check for Overview, Tasks sections)

	// Build spec file path
	specPath := filepath.Join(s.specDir, slug+".md")

	// Check if file exists
	if _, err := os.Stat(specPath); err == nil {
		return mcp.NewToolResultError(fmt.Sprintf("file already exists: %s. Please confirm overwrite or provide a different name.", specPath)), nil
	}

	// Save spec file
	if err := saveSpecFile(specPath, content); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save spec: %v", err)), nil
	}

	// TODO (TAS-14): Update README.md with marker detection/creation

	// Return success with file path
	return mcp.NewToolResultText(fmt.Sprintf("Spec saved successfully to: %s", specPath)), nil
}

// parseQuestion parses a raw question object into a Question struct.
// Validates required fields and returns error if any are missing or invalid.
func parseQuestion(raw map[string]any) (*Question, error) {
	// Extract question field
	questionText, ok := raw["question"].(string)
	if !ok || questionText == "" {
		return nil, fmt.Errorf("missing or empty 'question' field")
	}

	// Extract header field
	header, ok := raw["header"].(string)
	if !ok || header == "" {
		return nil, fmt.Errorf("missing or empty 'header' field")
	}

	// Extract options array
	optionsRaw, ok := raw["options"]
	if !ok {
		return nil, fmt.Errorf("missing 'options' field")
	}

	optionsArray, ok := optionsRaw.([]any)
	if !ok {
		return nil, fmt.Errorf("'options' is not an array")
	}

	options := make([]QuestionOption, 0, len(optionsArray))
	for i, optRaw := range optionsArray {
		optMap, ok := optRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("option %d is not an object", i)
		}

		label, ok := optMap["label"].(string)
		if !ok || label == "" {
			return nil, fmt.Errorf("option %d missing or empty 'label' field", i)
		}

		description, ok := optMap["description"].(string)
		if !ok {
			return nil, fmt.Errorf("option %d missing 'description' field", i)
		}

		options = append(options, QuestionOption{
			Label:       label,
			Description: description,
		})
	}

	// Extract optional multiple field
	multiple := false
	if multipleVal, ok := raw["multiple"].(bool); ok {
		multiple = multipleVal
	}

	return &Question{
		Question: questionText,
		Header:   header,
		Options:  options,
		Multiple: multiple,
	}, nil
}

// formatAnswersResponse formats collected answers into an MCP tool result.
// Answers can be strings (single-select) or []string (multi-select).
// Returns a JSON-formatted result containing the answers array.
func formatAnswersResponse(answers []any) (*mcp.CallToolResult, error) {
	// Validate all answers are non-empty
	for i, answer := range answers {
		switch v := answer.(type) {
		case string:
			if v == "" {
				return mcp.NewToolResultError(fmt.Sprintf("answer %d is empty", i)), nil
			}
		case []string:
			if len(v) == 0 {
				return mcp.NewToolResultError(fmt.Sprintf("answer %d has no selections", i)), nil
			}
			// Check each selection is non-empty
			for j, sel := range v {
				if sel == "" {
					return mcp.NewToolResultError(fmt.Sprintf("answer %d selection %d is empty", i, j)), nil
				}
			}
		default:
			return mcp.NewToolResultError(fmt.Sprintf("answer %d has invalid type: %T", i, answer)), nil
		}
	}

	// Return answers as JSON content
	result := map[string]any{
		"answers": answers,
	}

	return mcp.NewToolResultJSON(result)
}

// saveSpecFile saves the spec content to the given file path.
// Creates parent directory if it doesn't exist.
func saveSpecFile(path string, content string) error {
	// Create parent directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file with 0644 permissions (rw-r--r--)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
