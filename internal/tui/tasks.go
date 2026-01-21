package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/mark3labs/iteratr/internal/session"
)

// TaskList displays tasks grouped by status with filtering and navigation.
type TaskList struct {
	state  *session.State
	width  int
	height int
}

// NewTaskList creates a new TaskList component.
func NewTaskList() *TaskList {
	return &TaskList{}
}

// Update handles messages for the task list.
func (t *TaskList) Update(msg tea.Msg) tea.Cmd {
	// TODO: Implement task list updates (j/k navigation, filtering)
	return nil
}

// Render returns the task list view as a string.
func (t *TaskList) Render() string {
	if t.state == nil {
		return styleEmptyState.Render("No session loaded")
	}

	// Group tasks by status
	remaining := []*session.Task{}
	inProgress := []*session.Task{}
	completed := []*session.Task{}
	blocked := []*session.Task{}

	for _, task := range t.state.Tasks {
		switch task.Status {
		case "remaining":
			remaining = append(remaining, task)
		case "in_progress":
			inProgress = append(inProgress, task)
		case "completed":
			completed = append(completed, task)
		case "blocked":
			blocked = append(blocked, task)
		}
	}

	var sections []string

	// Render each status group
	if len(inProgress) > 0 {
		sections = append(sections, t.renderGroup("IN PROGRESS", inProgress, styleStatusInProgress))
	}
	if len(remaining) > 0 {
		sections = append(sections, t.renderGroup("REMAINING", remaining, styleStatusRemaining))
	}
	if len(blocked) > 0 {
		sections = append(sections, t.renderGroup("BLOCKED", blocked, styleStatusBlocked))
	}
	if len(completed) > 0 {
		sections = append(sections, t.renderGroup("COMPLETED", completed, styleStatusCompleted))
	}

	if len(sections) == 0 {
		return styleEmptyState.Render("No tasks yet")
	}

	return strings.Join(sections, "\n\n")
}

// renderGroup renders a group of tasks with a status header.
func (t *TaskList) renderGroup(title string, tasks []*session.Task, statusStyle lipgloss.Style) string {
	// Render header with count
	header := styleGroupHeader.Render(fmt.Sprintf("%s (%d)", title, len(tasks)))

	// Render tasks
	var taskLines []string
	for _, task := range tasks {
		taskLines = append(taskLines, t.renderTask(task, statusStyle))
	}

	return header + "\n" + strings.Join(taskLines, "\n")
}

// renderTask renders a single task with ID prefix and content.
func (t *TaskList) renderTask(task *session.Task, statusStyle lipgloss.Style) string {
	// Get 8 character ID prefix
	idPrefix := task.ID
	if len(idPrefix) > 8 {
		idPrefix = idPrefix[:8]
	}

	// Render ID and content
	id := styleTaskID.Render(fmt.Sprintf("[%s]", idPrefix))
	content := styleTaskContent.Render(task.Content)

	// Combine with status indicator
	bullet := statusStyle.Render("●")
	return fmt.Sprintf("  %s %s %s", bullet, id, content)
}

// UpdateSize updates the task list dimensions.
func (t *TaskList) UpdateSize(width, height int) tea.Cmd {
	t.width = width
	t.height = height
	return nil
}

// UpdateState updates the task list with new session state.
func (t *TaskList) UpdateState(state *session.State) tea.Cmd {
	t.state = state
	return nil
}
