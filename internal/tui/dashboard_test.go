package tui

import (
	"testing"

	"github.com/mark3labs/iteratr/internal/session"
)

// TestDashboard_Render removed - Dashboard now uses Draw() method with Screen/Draw pattern
// Rendering is tested through integration tests in app_test.go

func TestDashboard_RenderSessionInfo(t *testing.T) {
	d := &Dashboard{
		sessionName: "my-session",
		iteration:   42,
		sidebar:     NewSidebar(),
	}

	output := d.renderSessionInfo()
	if output == "" {
		t.Error("expected non-empty session info")
	}

	// Basic smoke test - should contain session name
	// We don't want to test lipgloss styling details, just that content is present
	// Note: lipgloss styles are stripped in tests, so we just check structure
}

func TestDashboard_GetTaskStats(t *testing.T) {
	tests := []struct {
		name     string
		state    *session.State
		wantZero bool
		expected progressStats
	}{
		{
			name: "counts tasks correctly",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "remaining"},
					"t2": {ID: "t2", Status: "in_progress"},
					"t3": {ID: "t3", Status: "completed"},
					"t4": {ID: "t4", Status: "blocked"},
					"t5": {ID: "t5", Status: "completed"},
				},
			},
			expected: progressStats{
				Total:      5,
				Remaining:  1,
				InProgress: 1,
				Completed:  2,
				Blocked:    1,
			},
		},
		{
			name:     "handles empty task list",
			state:    &session.State{Tasks: map[string]*session.Task{}},
			wantZero: true,
			expected: progressStats{
				Total:      0,
				Remaining:  0,
				InProgress: 0,
				Completed:  0,
				Blocked:    0,
			},
		},
		{
			name: "handles only completed tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "completed"},
					"t2": {ID: "t2", Status: "completed"},
				},
			},
			expected: progressStats{
				Total:      2,
				Remaining:  0,
				InProgress: 0,
				Completed:  2,
				Blocked:    0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dashboard{state: tt.state}
			stats := d.getTaskStats()

			if stats.Total != tt.expected.Total {
				t.Errorf("Total: got %d, want %d", stats.Total, tt.expected.Total)
			}
			if stats.Remaining != tt.expected.Remaining {
				t.Errorf("Remaining: got %d, want %d", stats.Remaining, tt.expected.Remaining)
			}
			if stats.InProgress != tt.expected.InProgress {
				t.Errorf("InProgress: got %d, want %d", stats.InProgress, tt.expected.InProgress)
			}
			if stats.Completed != tt.expected.Completed {
				t.Errorf("Completed: got %d, want %d", stats.Completed, tt.expected.Completed)
			}
			if stats.Blocked != tt.expected.Blocked {
				t.Errorf("Blocked: got %d, want %d", stats.Blocked, tt.expected.Blocked)
			}
		})
	}
}

func TestDashboard_RenderProgressIndicator(t *testing.T) {
	tests := []struct {
		name  string
		state *session.State
	}{
		{
			name: "renders progress with tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "completed"},
					"t2": {ID: "t2", Status: "remaining"},
				},
			},
		},
		{
			name: "renders progress with no tasks",
			state: &session.State{
				Tasks: map[string]*session.Task{},
			},
		},
		{
			name: "renders progress with all completed",
			state: &session.State{
				Tasks: map[string]*session.Task{
					"t1": {ID: "t1", Status: "completed"},
					"t2": {ID: "t2", Status: "completed"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Dashboard{state: tt.state}
			output := d.renderProgressIndicator()
			if output == "" {
				t.Error("expected non-empty progress indicator")
			}
		})
	}
}

// TestDashboard_RenderCurrentTask removed - renderCurrentTask() method no longer exists
// Current task is now shown in StatusBar component

func TestDashboard_UpdateState(t *testing.T) {
	d := NewDashboard(nil)

	state := &session.State{
		Session: "new-session",
		Tasks:   map[string]*session.Task{},
	}

	d.UpdateState(state)

	if d.state != state {
		t.Error("state was not updated")
	}
	if d.sessionName != "new-session" {
		t.Errorf("session name: got %s, want new-session", d.sessionName)
	}
}

func TestDashboard_SetIteration(t *testing.T) {
	d := NewDashboard(nil)

	d.SetIteration(10)

	if d.iteration != 10 {
		t.Errorf("iteration: got %d, want 10", d.iteration)
	}
}

func TestDashboard_UpdateSize(t *testing.T) {
	d := NewDashboard(nil)

	d.UpdateSize(100, 50)

	if d.width != 100 {
		t.Errorf("width: got %d, want 100", d.width)
	}
	if d.height != 50 {
		t.Errorf("height: got %d, want 50", d.height)
	}
}

func TestNewDashboard(t *testing.T) {
	d := NewDashboard(nil)

	if d == nil {
		t.Fatal("expected non-nil dashboard")
	}
	if d.sessionName != "" {
		t.Errorf("expected empty session name, got %s", d.sessionName)
	}
	if d.iteration != 0 {
		t.Errorf("expected iteration 0, got %d", d.iteration)
	}
}

// TestDashboard_RenderTaskStats removed - renderTaskStats() method no longer exists
// Task stats are shown in Sidebar component and renderProgressIndicator()
