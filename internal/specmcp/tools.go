package specmcp

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// registerTools registers all MCP tools with the spec wizard server.
// Tools are registered using mcp-go schema builders to provide native MCP protocol access.
func (s *Server) registerTools() error {
	// ask-questions: array of question objects with options
	s.mcpServer.AddTool(
		mcp.NewTool("ask-questions",
			mcp.WithDescription("Ask the user one or more questions with multiple choice options"),
			mcp.WithArray("questions", mcp.Required(),
				mcp.Items(map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question": map[string]any{
							"type":        "string",
							"description": "Full question text",
						},
						"header": map[string]any{
							"type":        "string",
							"description": "Short label for the question (max 30 chars)",
						},
						"options": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"label": map[string]any{
										"type":        "string",
										"description": "Display text (1-5 words)",
									},
									"description": map[string]any{
										"type":        "string",
										"description": "Detailed description of the option",
									},
								},
								"required": []string{"label", "description"},
							},
						},
						"multiple": map[string]any{
							"type":        "boolean",
							"description": "Allow multi-select (default: false)",
						},
					},
					"required": []string{"question", "header", "options"},
				})),
		),
		s.handleAskQuestions,
	)

	// finish-spec: save the spec to file
	s.mcpServer.AddTool(
		mcp.NewTool("finish-spec",
			mcp.WithDescription("Save the completed spec to a file and update README.md"),
			mcp.WithString("content", mcp.Required(), mcp.Description("Full spec markdown content")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Spec name for filename")),
		),
		s.handleFinishSpec,
	)

	return nil
}
