package specmcp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gosimple/slug"
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

	// Slugify name with transliteration
	slug := slugify(name)

	// Validate spec content
	if err := validateSpecContent(content); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid spec content: %v", err)), nil
	}

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

// validateSpecContent performs loose validation on spec markdown content.
// Checks for presence of required sections: Overview and Tasks.
// Returns error if any required sections are missing.
func validateSpecContent(content string) error {
	if content == "" {
		return fmt.Errorf("content is empty")
	}

	// Check for Overview section
	hasOverview := containsSection(content, "Overview") || containsSection(content, "## Overview")

	// Check for Tasks section
	hasTasks := containsSection(content, "Tasks") || containsSection(content, "## Tasks")

	// Report missing sections
	if !hasOverview && !hasTasks {
		return fmt.Errorf("missing required sections: Overview, Tasks")
	}
	if !hasOverview {
		return fmt.Errorf("missing required section: Overview")
	}
	if !hasTasks {
		return fmt.Errorf("missing required section: Tasks")
	}

	return nil
}

// containsSection checks if the content contains a markdown section header.
// Looks for the section name as a heading (with # prefix or standalone).
func containsSection(content, section string) bool {
	if len(content) == 0 {
		return false
	}

	// Look for section as markdown heading: "# Section", "## Section", etc.
	// Pattern: newline or start-of-file, then "#" (one or more), space, section name, then newline or space

	// Check at start of file
	if hasPrefix(content, "#") {
		// Find the section name after the # marks
		idx := 0
		for idx < len(content) && content[idx] == '#' {
			idx++
		}
		// Skip whitespace after #
		for idx < len(content) && content[idx] == ' ' {
			idx++
		}
		// Check if section name matches
		if hasPrefix(content[idx:], section) {
			afterSection := idx + len(section)
			if afterSection >= len(content) || content[afterSection] == '\n' || content[afterSection] == ' ' {
				return true
			}
		}
	}

	// Check after newlines
	searchStr := "\n#"
	for i := 0; i <= len(content)-len(searchStr); i++ {
		if content[i:i+len(searchStr)] == searchStr {
			// Found "\n#" - now check if section name follows
			idx := i + 1 // Skip the newline
			// Skip # marks
			for idx < len(content) && content[idx] == '#' {
				idx++
			}
			// Skip whitespace
			for idx < len(content) && content[idx] == ' ' {
				idx++
			}
			// Check if section name matches
			if hasPrefix(content[idx:], section) {
				afterSection := idx + len(section)
				if afterSection >= len(content) || content[afterSection] == '\n' || content[afterSection] == ' ' {
					return true
				}
			}
		}
	}

	return false
}

// hasPrefix checks if string starts with prefix
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// slugify converts a string to a URL-friendly slug format.
// Performs transliteration of Unicode characters to ASCII,
// converts to lowercase, replaces spaces with hyphens,
// and removes non-alphanumeric characters except hyphens.
func slugify(s string) string {
	// Use gosimple/slug for transliteration and normalization
	// This handles: lowercase conversion, space->hyphen, unicode->ASCII,
	// and removes characters that aren't alphanumeric or hyphens
	return slug.Make(s)
}
