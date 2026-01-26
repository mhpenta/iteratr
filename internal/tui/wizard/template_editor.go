package wizard

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/mark3labs/iteratr/internal/template"
)

// TemplateEditorStep manages the template editor UI step.
type TemplateEditorStep struct {
	textarea textarea.Model // Multi-line textarea for template editing
	width    int            // Available width
	height   int            // Available height
}

// NewTemplateEditorStep creates a new template editor step.
func NewTemplateEditorStep() *TemplateEditorStep {
	// Create and configure textarea
	ta := textarea.New()
	ta.Placeholder = "Edit template..."
	ta.ShowLineNumbers = false
	ta.Prompt = "" // No prompt character
	ta.SetWidth(60)
	ta.SetHeight(10)

	// No character limit for template
	ta.CharLimit = 0

	// Override textarea KeyMap to prevent conflicts
	// Remove ctrl+n from LineNext (if needed for wizard shortcuts)
	ta.KeyMap.LineNext = key.NewBinding(key.WithKeys("down"))

	// Style textarea with dark theme
	styles := textarea.DefaultDarkStyles()
	styles.Cursor.Color = lipgloss.Color("#cba6f7") // Primary color
	styles.Cursor.Shape = tea.CursorBlock
	styles.Cursor.Blink = true
	ta.SetStyles(styles)

	// Pre-populate with default template
	ta.SetValue(template.DefaultTemplate)

	return &TemplateEditorStep{
		textarea: ta,
		width:    60,
		height:   20,
	}
}

// Init initializes the template editor and focuses the textarea.
func (t *TemplateEditorStep) Init() tea.Cmd {
	return t.textarea.Focus()
}

// SetSize updates the dimensions for the template editor.
func (t *TemplateEditorStep) SetSize(width, height int) {
	t.width = width
	t.height = height

	// Update textarea size (leave room for variable reference)
	t.textarea.SetWidth(width - 4)

	// Reserve 4 lines for variable reference at bottom
	textareaHeight := height - 4
	if textareaHeight < 5 {
		textareaHeight = 5
	}
	t.textarea.SetHeight(textareaHeight)
}

// Update handles messages for the template editor step.
func (t *TemplateEditorStep) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	t.textarea, cmd = t.textarea.Update(msg)
	return cmd
}

// View renders the template editor step.
func (t *TemplateEditorStep) View() string {
	var b strings.Builder

	// Render textarea
	b.WriteString(t.textarea.View())
	b.WriteString("\n\n")

	// Show placeholder variables reference
	varStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))
	b.WriteString(varStyle.Render("Variables: {{session}} {{iteration}} {{spec}} {{notes}} {{tasks}} {{history}} {{extra}} {{port}} {{binary}}"))
	b.WriteString("\n\n")

	// Hint bar
	hintBar := renderHintBar(
		"enter", "next",
		"ctrl+enter", "finish",
		"esc", "back",
	)
	b.WriteString(hintBar)

	return b.String()
}

// Content returns the current template content.
func (t *TemplateEditorStep) Content() string {
	return t.textarea.Value()
}

// TemplateEditedMsg is sent when the template is modified (optional, for future use).
type TemplateEditedMsg struct {
	Content string
}
