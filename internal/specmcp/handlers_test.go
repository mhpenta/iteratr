package specmcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// extractText extracts text from CallToolResult.Content[0]
func extractText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		return textContent.Text
	}
	return ""
}

// extractJSON extracts JSON data from CallToolResult.Content[0]
func extractJSON(result *mcp.CallToolResult) (map[string]any, bool) {
	if len(result.Content) == 0 {
		return nil, false
	}

	// Check if it's a TextContent with JSON
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		var data map[string]any
		if err := json.Unmarshal([]byte(textContent.Text), &data); err == nil {
			return data, true
		}
	}

	return nil, false
}

func TestFormatAnswersResponse(t *testing.T) {
	tests := []struct {
		name        string
		answers     []any
		wantErr     bool
		errContains string
		validate    func(t *testing.T, result map[string]any)
	}{
		{
			name:    "single string answer",
			answers: []any{"My custom answer"},
			wantErr: false,
			validate: func(t *testing.T, result map[string]any) {
				answers, ok := result["answers"].([]any)
				require.True(t, ok, "answers should be an array")
				require.Len(t, answers, 1)
				assert.Equal(t, "My custom answer", answers[0])
			},
		},
		{
			name:    "multiple string answers",
			answers: []any{"Answer 1", "Answer 2", "Answer 3"},
			wantErr: false,
			validate: func(t *testing.T, result map[string]any) {
				answers, ok := result["answers"].([]any)
				require.True(t, ok)
				require.Len(t, answers, 3)
				assert.Equal(t, "Answer 1", answers[0])
				assert.Equal(t, "Answer 2", answers[1])
				assert.Equal(t, "Answer 3", answers[2])
			},
		},
		{
			name:    "multi-select answer",
			answers: []any{[]string{"Option A", "Option B"}},
			wantErr: false,
			validate: func(t *testing.T, result map[string]any) {
				answers, ok := result["answers"].([]any)
				require.True(t, ok)
				require.Len(t, answers, 1)

				// JSON unmarshaling converts []string to []any
				selections, ok := answers[0].([]any)
				require.True(t, ok, "multi-select should be an array")
				require.Len(t, selections, 2)
				assert.Equal(t, "Option A", selections[0])
				assert.Equal(t, "Option B", selections[1])
			},
		},
		{
			name:    "mixed single and multi-select",
			answers: []any{"Single answer", []string{"Multi 1", "Multi 2"}, "Another single"},
			wantErr: false,
			validate: func(t *testing.T, result map[string]any) {
				answers, ok := result["answers"].([]any)
				require.True(t, ok)
				require.Len(t, answers, 3)

				assert.Equal(t, "Single answer", answers[0])

				selections, ok := answers[1].([]any)
				require.True(t, ok)
				assert.Len(t, selections, 2)

				assert.Equal(t, "Another single", answers[2])
			},
		},
		{
			name:        "empty string answer",
			answers:     []any{""},
			wantErr:     true,
			errContains: "answer 0 is empty",
		},
		{
			name:        "empty multi-select",
			answers:     []any{[]string{}},
			wantErr:     true,
			errContains: "answer 0 has no selections",
		},
		{
			name:        "multi-select with empty selection",
			answers:     []any{[]string{"Valid", ""}},
			wantErr:     true,
			errContains: "answer 0 selection 1 is empty",
		},
		{
			name:        "invalid answer type",
			answers:     []any{123},
			wantErr:     true,
			errContains: "answer 0 has invalid type: int",
		},
		{
			name:        "mixed valid and invalid",
			answers:     []any{"Valid", 123},
			wantErr:     true,
			errContains: "answer 1 has invalid type: int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatAnswersResponse(tt.answers)
			require.NoError(t, err, "formatAnswersResponse should not return error")
			require.NotNil(t, result)

			if tt.wantErr {
				assert.True(t, result.IsError, "result should be an error")
				require.Len(t, result.Content, 1)

				// Extract error text from content
				text := extractText(result)
				assert.Contains(t, text, tt.errContains)
			} else {
				assert.False(t, result.IsError, "result should not be an error")
				require.Len(t, result.Content, 1)

				// Parse JSON content
				text := extractText(result)

				var parsed map[string]any
				err := json.Unmarshal([]byte(text), &parsed)
				require.NoError(t, err, "content should be valid JSON")

				if tt.validate != nil {
					tt.validate(t, parsed)
				}
			}
		})
	}
}

func TestParseQuestion(t *testing.T) {
	tests := []struct {
		name        string
		raw         map[string]any
		want        *Question
		wantErr     bool
		errContains string
	}{
		{
			name: "valid question with options",
			raw: map[string]any{
				"question": "What is your preferred approach?",
				"header":   "Approach Selection",
				"options": []any{
					map[string]any{
						"label":       "Option A",
						"description": "Use approach A",
					},
					map[string]any{
						"label":       "Option B",
						"description": "Use approach B",
					},
				},
			},
			want: &Question{
				Question: "What is your preferred approach?",
				Header:   "Approach Selection",
				Options: []QuestionOption{
					{Label: "Option A", Description: "Use approach A"},
					{Label: "Option B", Description: "Use approach B"},
				},
				Multiple: false,
			},
		},
		{
			name: "question with multiple flag",
			raw: map[string]any{
				"question": "Select all that apply",
				"header":   "Multi-select",
				"options": []any{
					map[string]any{
						"label":       "Option 1",
						"description": "First option",
					},
				},
				"multiple": true,
			},
			want: &Question{
				Question: "Select all that apply",
				Header:   "Multi-select",
				Options: []QuestionOption{
					{Label: "Option 1", Description: "First option"},
				},
				Multiple: true,
			},
		},
		{
			name: "missing question field",
			raw: map[string]any{
				"header": "Header",
				"options": []any{
					map[string]any{
						"label":       "Option",
						"description": "Desc",
					},
				},
			},
			wantErr:     true,
			errContains: "missing or empty 'question' field",
		},
		{
			name: "missing header field",
			raw: map[string]any{
				"question": "Question text",
				"options": []any{
					map[string]any{
						"label":       "Option",
						"description": "Desc",
					},
				},
			},
			wantErr:     true,
			errContains: "missing or empty 'header' field",
		},
		{
			name: "missing options field",
			raw: map[string]any{
				"question": "Question text",
				"header":   "Header",
			},
			wantErr:     true,
			errContains: "missing 'options' field",
		},
		{
			name: "option missing label",
			raw: map[string]any{
				"question": "Question text",
				"header":   "Header",
				"options": []any{
					map[string]any{
						"description": "Desc",
					},
				},
			},
			wantErr:     true,
			errContains: "option 0 missing or empty 'label' field",
		},
		{
			name: "option missing description",
			raw: map[string]any{
				"question": "Question text",
				"header":   "Header",
				"options": []any{
					map[string]any{
						"label": "Label",
					},
				},
			},
			wantErr:     true,
			errContains: "option 0 missing 'description' field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseQuestion(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestHandleAskQuestions_ChannelBlocking(t *testing.T) {
	tests := []struct {
		name           string
		questions      []any
		simulateAnswer func(t *testing.T, req *QuestionRequest)
		wantErr        bool
		errContains    string
		validate       func(t *testing.T, result *mcp.CallToolResult)
	}{
		{
			name: "single question with answer",
			questions: []any{
				map[string]any{
					"question": "What is your name?",
					"header":   "Name",
					"options": []any{
						map[string]any{"label": "John", "description": "Name is John"},
						map[string]any{"label": "Jane", "description": "Name is Jane"},
					},
				},
			},
			simulateAnswer: func(t *testing.T, req *QuestionRequest) {
				require.Len(t, req.Questions, 1)
				assert.Equal(t, "What is your name?", req.Questions[0].Question)
				assert.Equal(t, "Name", req.Questions[0].Header)
				require.Len(t, req.Questions[0].Options, 2)
				// Send answer back
				req.AnswerCh <- []any{"John"}
			},
			validate: func(t *testing.T, result *mcp.CallToolResult) {
				require.False(t, result.IsError)
				require.Len(t, result.Content, 1)
				// Extract JSON content
				content, ok := extractJSON(result)
				require.True(t, ok, "content should be JSON object")
				answers, ok := content["answers"].([]any)
				require.True(t, ok, "should have answers array")
				require.Len(t, answers, 1)
				assert.Equal(t, "John", answers[0])
			},
		},
		{
			name: "multiple questions with mixed answers",
			questions: []any{
				map[string]any{
					"question": "Pick one",
					"header":   "Single",
					"options": []any{
						map[string]any{"label": "A", "description": "Option A"},
					},
				},
				map[string]any{
					"question": "Pick many",
					"header":   "Multi",
					"options": []any{
						map[string]any{"label": "X", "description": "Option X"},
						map[string]any{"label": "Y", "description": "Option Y"},
					},
					"multiple": true,
				},
			},
			simulateAnswer: func(t *testing.T, req *QuestionRequest) {
				require.Len(t, req.Questions, 2)
				assert.False(t, req.Questions[0].Multiple)
				assert.True(t, req.Questions[1].Multiple)
				// Send mixed answer types
				req.AnswerCh <- []any{"A", []string{"X", "Y"}}
			},
			validate: func(t *testing.T, result *mcp.CallToolResult) {
				require.False(t, result.IsError)
				content, ok := extractJSON(result)
				require.True(t, ok)
				answers, ok := content["answers"].([]any)
				require.True(t, ok)
				require.Len(t, answers, 2)
				assert.Equal(t, "A", answers[0])
				multiAnswer, ok := answers[1].([]any)
				require.True(t, ok)
				require.Len(t, multiAnswer, 2)
			},
		},
		{
			name: "context cancelled before question sent",
			questions: []any{
				map[string]any{
					"question": "Question",
					"header":   "Header",
					"options":  []any{map[string]any{"label": "A", "description": "Desc"}},
				},
			},
			simulateAnswer: func(t *testing.T, req *QuestionRequest) {
				// Don't read from channel - let it block
				t.Fatal("should not reach here")
			},
			wantErr:     true,
			errContains: "context cancelled",
		},
		{
			name: "context cancelled while waiting for answer",
			questions: []any{
				map[string]any{
					"question": "Question",
					"header":   "Header",
					"options":  []any{map[string]any{"label": "A", "description": "Desc"}},
				},
			},
			simulateAnswer: func(t *testing.T, req *QuestionRequest) {
				// Read question but don't send answer
				require.Len(t, req.Questions, 1)
				// Don't send answer - let context cancel
			},
			wantErr:     true,
			errContains: "context cancelled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create server with question channel
			srv := New("/tmp/test-specs")

			// Create request
			request := mcp.CallToolRequest{}
			request.Params.Arguments = map[string]any{
				"questions": tt.questions,
			}

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			// Handle questions in goroutine
			resultCh := make(chan *mcp.CallToolResult)
			go func() {
				result, err := srv.handleAskQuestions(ctx, request)
				require.NoError(t, err) // Handler should never return error, only error results
				resultCh <- result
			}()

			// Simulate UI receiving and answering questions
			if !tt.wantErr || tt.errContains == "context cancelled while waiting for answers" {
				select {
				case req := <-srv.QuestionChannel():
					tt.simulateAnswer(t, req)
				case <-time.After(50 * time.Millisecond):
					t.Fatal("timeout waiting for question")
				}
			} else {
				// For cancel before send test, just cancel immediately
				cancel()
			}

			// Wait for result
			select {
			case result := <-resultCh:
				if tt.wantErr {
					assert.True(t, result.IsError, "expected error result")
					assert.Contains(t, extractText(result), tt.errContains)
				} else {
					if tt.validate != nil {
						tt.validate(t, result)
					}
				}
			case <-time.After(200 * time.Millisecond):
				t.Fatal("timeout waiting for result")
			}
		})
	}
}

func TestHandleAskQuestions_InvalidQuestions(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]any
		errContains string
	}{
		{
			name:        "no arguments",
			args:        nil,
			errContains: "no arguments provided",
		},
		{
			name:        "missing questions parameter",
			args:        map[string]any{},
			errContains: "missing 'questions' parameter",
		},
		{
			name:        "questions not an array",
			args:        map[string]any{"questions": "not an array"},
			errContains: "'questions' is not an array",
		},
		{
			name:        "empty questions array",
			args:        map[string]any{"questions": []any{}},
			errContains: "at least one question is required",
		},
		{
			name: "question not an object",
			args: map[string]any{
				"questions": []any{"not an object"},
			},
			errContains: "question 0 is not an object",
		},
		{
			name: "question missing required field",
			args: map[string]any{
				"questions": []any{
					map[string]any{
						"header":  "Header",
						"options": []any{},
					},
				},
			},
			errContains: "question 0 invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New("/tmp/test-specs")
			request := mcp.CallToolRequest{}
			request.Params.Arguments = tt.args

			ctx := context.Background()
			result, err := srv.handleAskQuestions(ctx, request)
			require.NoError(t, err) // Handler should never return error
			assert.True(t, result.IsError)
			assert.Contains(t, extractText(result), tt.errContains)
		})
	}
}

func TestHandleFinishSpec_Validation(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]any
		wantErr     bool
		errContains string
	}{
		{
			name: "valid spec content",
			args: map[string]any{
				"content": `# My Feature

## Overview
Feature description here.

## Tasks
- [ ] Task 1
- [ ] Task 2
`,
				"name": "my-feature",
			},
			wantErr: false,
		},
		{
			name: "missing content parameter",
			args: map[string]any{
				"name": "my-feature",
			},
			wantErr:     true,
			errContains: "missing or empty 'content' parameter",
		},
		{
			name: "missing name parameter",
			args: map[string]any{
				"content": "# Overview\n## Tasks\n",
			},
			wantErr:     true,
			errContains: "missing or empty 'name' parameter",
		},
		{
			name: "empty content",
			args: map[string]any{
				"content": "",
				"name":    "my-feature",
			},
			wantErr:     true,
			errContains: "missing or empty 'content' parameter",
		},
		{
			name: "content missing Overview section",
			args: map[string]any{
				"content": `# My Feature

## Tasks
- [ ] Task 1
`,
				"name": "my-feature",
			},
			wantErr:     true,
			errContains: "invalid spec content: missing required section: Overview",
		},
		{
			name: "content missing Tasks section",
			args: map[string]any{
				"content": `# My Feature

## Overview
Feature description.
`,
				"name": "my-feature",
			},
			wantErr:     true,
			errContains: "invalid spec content: missing required section: Tasks",
		},
		{
			name: "content missing both sections",
			args: map[string]any{
				"content": `# My Feature

## Introduction
Some intro.
`,
				"name": "my-feature",
			},
			wantErr:     true,
			errContains: "invalid spec content: missing required sections: Overview, Tasks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test
			tmpDir := t.TempDir()
			srv := New(tmpDir)

			request := mcp.CallToolRequest{}
			request.Params.Arguments = tt.args

			ctx := context.Background()
			result, err := srv.handleFinishSpec(ctx, request)
			require.NoError(t, err) // Handler should never return error

			if tt.wantErr {
				assert.True(t, result.IsError, "expected error result")
				assert.Contains(t, extractText(result), tt.errContains)
			} else {
				assert.False(t, result.IsError, "expected success result")
				// Verify file was created
				text := extractText(result)
				assert.Contains(t, text, "Spec saved successfully")
			}
		})
	}
}

func TestValidateSpecContent(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantErr     bool
		errContains string
	}{
		{
			name: "valid spec with both sections",
			content: `# My Feature Spec

## Overview
This is a feature spec that does something cool.

## Tasks
- [ ] Task 1
- [ ] Task 2
`,
			wantErr: false,
		},
		{
			name: "valid spec with Overview as h1",
			content: `# Overview
Feature description here.

## Tasks
- [ ] Do something
`,
			wantErr: false,
		},
		{
			name: "valid spec with sections in different order",
			content: `# Feature Name

## Tasks
- [ ] Build it

## Overview
What we're building.

## Technical Details
More info here.
`,
			wantErr: false,
		},
		{
			name: "valid spec with extra spaces after section",
			content: `## Overview  
Some text here

## Tasks  
- [ ] Task
`,
			wantErr: false,
		},
		{
			name: "sections without hash marks (not detected - acceptable)",
			content: `Overview
This is the overview.

Tasks
- [ ] Something
`,
			wantErr:     true,
			errContains: "missing required sections: Overview, Tasks",
		},
		{
			name: "missing Overview section",
			content: `# My Feature

## Tasks
- [ ] Task 1
- [ ] Task 2
`,
			wantErr:     true,
			errContains: "missing required section: Overview",
		},
		{
			name: "missing Tasks section",
			content: `# My Feature

## Overview
Feature description here.

## Implementation
Some details.
`,
			wantErr:     true,
			errContains: "missing required section: Tasks",
		},
		{
			name: "missing both sections",
			content: `# My Feature

## Introduction
Some intro text.

## Conclusion
Some conclusion.
`,
			wantErr:     true,
			errContains: "missing required sections: Overview, Tasks",
		},
		{
			name:        "empty content",
			content:     "",
			wantErr:     true,
			errContains: "content is empty",
		},
		{
			name: "sections as part of other words (false positive test)",
			content: `# My Feature

## OverviewExtra
This should not match.

## TasksManager
This should not match either.
`,
			wantErr:     true,
			errContains: "missing required sections: Overview, Tasks",
		},
		{
			name:    "sections in code blocks detected (loose validation - acceptable)",
			content: "# My Feature\n\n```\n## Overview\nThis is in a code block\n## Tasks\nAlso in code block\n```\n",
			wantErr: false, // Loose validation doesn't parse code blocks
		},
		{
			name: "case sensitive - lowercase sections",
			content: `# My Feature

## overview
This should not match.

## tasks
This should not match either.
`,
			wantErr:     true,
			errContains: "missing required sections: Overview, Tasks",
		},
		{
			name: "sections in middle of line (should not match)",
			content: `# My Feature

Some text ## Overview here
And ## Tasks here too
`,
			wantErr:     true,
			errContains: "missing required sections: Overview, Tasks",
		},
		{
			name: "multiple Overview and Tasks sections (should still pass)",
			content: `## Overview
First overview

## Tasks
First tasks

## Overview
Second overview (maybe nested)

## Tasks  
More tasks
`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSpecContent(tt.content)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
