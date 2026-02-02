package specmcp

import (
	"encoding/json"
	"testing"

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
		want        *question
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
			want: &question{
				Question: "What is your preferred approach?",
				Header:   "Approach Selection",
				Options: []questionOption{
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
			want: &question{
				Question: "Select all that apply",
				Header:   "Multi-select",
				Options: []questionOption{
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
