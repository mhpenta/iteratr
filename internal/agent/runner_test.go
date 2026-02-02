package agent

import (
	"testing"
)

func TestNewRunner_MCPServerName(t *testing.T) {
	tests := []struct {
		name              string
		mcpServerName     string
		expectedFieldName string
	}{
		{
			name:              "default to iteratr-tools when empty",
			mcpServerName:     "",
			expectedFieldName: "",
		},
		{
			name:              "custom name iteratr-spec",
			mcpServerName:     "iteratr-spec",
			expectedFieldName: "iteratr-spec",
		},
		{
			name:              "custom name iteratr-tools explicitly",
			mcpServerName:     "iteratr-tools",
			expectedFieldName: "iteratr-tools",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := RunnerConfig{
				Model:         "test/model",
				WorkDir:       "/tmp",
				MCPServerURL:  "http://localhost:8080/mcp",
				MCPServerName: tt.mcpServerName,
			}

			runner := NewRunner(cfg)

			if runner.mcpServerName != tt.expectedFieldName {
				t.Errorf("NewRunner().mcpServerName = %q, want %q", runner.mcpServerName, tt.expectedFieldName)
			}
		})
	}
}

func TestExtractProvider(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{
			name:     "anthropic model",
			model:    "anthropic/claude-sonnet-4-5",
			expected: "Anthropic",
		},
		{
			name:     "openai model",
			model:    "openai/gpt-4",
			expected: "Openai",
		},
		{
			name:     "model without slash",
			model:    "claude-sonnet-4-5",
			expected: "",
		},
		{
			name:     "empty string",
			model:    "",
			expected: "",
		},
		{
			name:     "single letter provider",
			model:    "a/model",
			expected: "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractProvider(tt.model)
			if got != tt.expected {
				t.Errorf("extractProvider(%q) = %q, want %q", tt.model, got, tt.expected)
			}
		})
	}
}
