package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/mark3labs/iteratr/internal/session"
)

// TaskSidebar displays tasks in a compact sidebar with status bar at bottom.
type TaskSidebar struct {
	state        *session.State
	width        int
	height       int
	cursor       int // Selected task index
	scrollOffset int // For scrolling when list exceeds available height
	focused      bool
}

// NewTaskSidebar creates a new TaskSidebar component.
func NewTaskSidebar() *TaskSidebar {
	return &TaskSidebar{
		cursor:  0,
		focused: false,
	}
}

// Update handles messages for the sidebar.
func (s *TaskSidebar) Update(msg tea.Msg) tea.Cmd {
	if !s.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return s.handleKeyPress(msg)
	}
	return nil
}

// handleKeyPress handles keyboard input for task navigation.
func (s *TaskSidebar) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	tasks := s.getTasks()
	maxIndex := len(tasks) - 1
	if maxIndex < 0 {
		return nil
	}

	switch msg.String() {
	case "j", "down":
		if s.cursor < maxIndex {
			s.cursor++
			s.adjustScroll()
		}
	case "k", "up":
		if s.cursor > 0 {
			s.cursor--
			s.adjustScroll()
		}
	case "g":
		s.cursor = 0
		s.scrollOffset = 0
	case "G":
		s.cursor = maxIndex
		s.adjustScroll()
	}

	return nil
}

// adjustScroll adjusts scroll offset to keep cursor visible.
func (s *TaskSidebar) adjustScroll() {
	// Each task is 1 line, reserve 5 for header, borders, and status bar
	visibleLines := s.height - 5
	if visibleLines < 1 {
		visibleLines = 1
	}

	if s.cursor >= s.scrollOffset+visibleLines {
		s.scrollOffset = s.cursor - visibleLines + 1
	} else if s.cursor < s.scrollOffset {
		s.scrollOffset = s.cursor
	}
}

// getTasks returns all tasks in display order (in_progress first, then remaining, blocked, completed).
func (s *TaskSidebar) getTasks() []*session.Task {
	if s.state == nil {
		return nil
	}

	var inProgress, remaining, blocked, completed []*session.Task
	for _, task := range s.state.Tasks {
		switch task.Status {
		case "in_progress":
			inProgress = append(inProgress, task)
		case "remaining":
			remaining = append(remaining, task)
		case "blocked":
			blocked = append(blocked, task)
		case "completed":
			completed = append(completed, task)
		}
	}

	// Concatenate in order
	var tasks []*session.Task
	tasks = append(tasks, inProgress...)
	tasks = append(tasks, remaining...)
	tasks = append(tasks, blocked...)
	tasks = append(tasks, completed...)
	return tasks
}

// Render returns the sidebar view as a string.
func (s *TaskSidebar) Render() string {
	// Guard against zero dimensions (not yet sized)
	if s.width < 10 || s.height < 5 {
		return ""
	}

	// Header
	header := styleSidebarHeader.Width(s.width - 2).Render("Tasks")

	// Task list
	taskList := s.renderTaskList()

	// Calculate heights
	headerHeight := 2                         // header + border
	listHeight := s.height - headerHeight - 2 // -2 for borders
	if listHeight < 1 {
		listHeight = 1
	}

	// Ensure task list fills available space
	taskLines := strings.Split(taskList, "\n")
	for len(taskLines) < listHeight {
		taskLines = append(taskLines, "")
	}
	if len(taskLines) > listHeight {
		taskLines = taskLines[:listHeight]
	}
	taskList = strings.Join(taskLines, "\n")

	// Build sidebar content
	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		taskList,
	)

	// Apply sidebar border style
	return styleSidebarBorder.Width(s.width).Height(s.height).Render(content)
}

// renderTaskList renders the task items.
func (s *TaskSidebar) renderTaskList() string {
	tasks := s.getTasks()
	if len(tasks) == 0 {
		return styleDim.Render("  No tasks")
	}

	var lines []string
	visibleLines := s.height - 5 // Reserve for header and status bar
	if visibleLines < 1 {
		visibleLines = 1
	}

	for i, task := range tasks {
		// Skip tasks before scroll offset
		if i < s.scrollOffset {
			continue
		}
		// Stop if we've rendered enough visible lines
		if len(lines) >= visibleLines {
			break
		}

		line := s.renderTask(task, i == s.cursor)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// renderTask renders a single task line.
func (s *TaskSidebar) renderTask(task *session.Task, isSelected bool) string {
	// Status indicator
	var indicator string
	var indicatorStyle lipgloss.Style

	switch task.Status {
	case "in_progress":
		indicator = "►"
		indicatorStyle = styleStatusInProgress
	case "remaining":
		indicator = "○"
		indicatorStyle = styleStatusRemaining
	case "completed":
		indicator = "✓"
		indicatorStyle = styleStatusCompleted
	case "blocked":
		indicator = "⊘"
		indicatorStyle = styleStatusBlocked
	default:
		indicator = "○"
		indicatorStyle = styleStatusRemaining
	}

	// Truncate content to fit width (leave room for indicator and padding)
	maxContentWidth := s.width - 6 // 2 for border, 2 for indicator+space, 2 padding
	if maxContentWidth < 10 {
		maxContentWidth = 10
	}

	content := task.Content
	if len(content) > maxContentWidth {
		content = content[:maxContentWidth-3] + "..."
	}

	// Build line
	styledIndicator := indicatorStyle.Render(indicator)
	line := fmt.Sprintf(" %s %s", styledIndicator, content)

	// Apply selection style
	if isSelected && s.focused {
		line = styleTaskSelected.Width(s.width - 2).Render(line)
	} else if isSelected {
		// Subtle highlight when not focused
		line = styleDim.Render(line)
	}

	return line
}

// SetFocused sets whether the sidebar has keyboard focus.
func (s *TaskSidebar) SetFocused(focused bool) {
	s.focused = focused
}

// IsFocused returns whether the sidebar has keyboard focus.
func (s *TaskSidebar) IsFocused() bool {
	return s.focused
}

// UpdateSize updates the sidebar dimensions.
func (s *TaskSidebar) UpdateSize(width, height int) tea.Cmd {
	s.width = width
	s.height = height
	return nil
}

// UpdateState updates the sidebar with new session state.
func (s *TaskSidebar) UpdateState(state *session.State) tea.Cmd {
	s.state = state
	return nil
}

// Sidebar styles
var (
	styleSidebarBorder = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), true, true, true, false). // No left border
				BorderForeground(colorMuted)

	styleSidebarHeader = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(colorMuted).
				PaddingLeft(1)
)
