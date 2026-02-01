package git

import (
	"os"
	"testing"
)

func TestGetInfo_CurrentRepo(t *testing.T) {
	// Test against the current repository (iteratr itself)
	info, err := GetInfo("../..")
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if info == nil {
		t.Fatal("Expected info for git repo, got nil")
	}

	// Should have a branch name
	if info.Branch == "" {
		t.Error("Expected non-empty branch name")
	}

	// Should have a hash
	if info.Hash == "" {
		t.Error("Expected non-empty hash")
	}
	if len(info.Hash) != 7 {
		t.Errorf("Expected 7-char hash, got %d chars: %s", len(info.Hash), info.Hash)
	}

	t.Logf("Branch: %s, Hash: %s, Dirty: %v, Ahead: %d, Behind: %d",
		info.Branch, info.Hash, info.Dirty, info.Ahead, info.Behind)
}

func TestGetInfo_NonGitDir(t *testing.T) {
	// /tmp is typically not a git repository
	info, err := GetInfo("/tmp")
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if info != nil {
		t.Error("Expected nil for non-git directory")
	}
}

func TestGetInfo_NoUpstream(t *testing.T) {
	// Create a temp git repo with no upstream
	dir := t.TempDir()

	// Initialize git repo
	if _, err := runGit(dir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	// Configure user for commits
	if _, err := runGit(dir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if _, err := runGit(dir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	// Create initial commit (required for HEAD to exist)
	if _, err := runGit(dir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Get info - should succeed with ahead/behind = 0 (no upstream)
	info, err := GetInfo(dir)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if info == nil {
		t.Fatal("Expected info, got nil")
	}

	// Ahead/behind should be 0 when no upstream is configured
	if info.Ahead != 0 {
		t.Errorf("Expected Ahead=0 with no upstream, got %d", info.Ahead)
	}
	if info.Behind != 0 {
		t.Errorf("Expected Behind=0 with no upstream, got %d", info.Behind)
	}

	// Branch should be master or main (depends on git version)
	if info.Branch != "master" && info.Branch != "main" {
		t.Errorf("Expected branch master or main, got %s", info.Branch)
	}

	t.Logf("Branch: %s, Hash: %s, Ahead: %d, Behind: %d",
		info.Branch, info.Hash, info.Ahead, info.Behind)
}

// setupTestRepo creates a temporary git repo with optional configuration.
// Returns the directory path.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	if _, err := runGit(dir, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	if _, err := runGit(dir, "config", "user.email", "test@test.com"); err != nil {
		t.Fatalf("git config email failed: %v", err)
	}
	if _, err := runGit(dir, "config", "user.name", "Test"); err != nil {
		t.Fatalf("git config name failed: %v", err)
	}

	return dir
}

func TestGetInfo_DetachedHead(t *testing.T) {
	dir := setupTestRepo(t)

	// Create initial commit
	if _, err := runGit(dir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Get the commit hash
	hash, err := runGit(dir, "rev-parse", "HEAD")
	if err != nil {
		t.Fatalf("git rev-parse failed: %v", err)
	}

	// Checkout detached HEAD
	if _, err := runGit(dir, "checkout", "--detach", hash); err != nil {
		t.Fatalf("git checkout --detach failed: %v", err)
	}

	info, err := GetInfo(dir)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if info == nil {
		t.Fatal("Expected info, got nil")
	}

	// Branch should be "HEAD" when detached
	if info.Branch != "HEAD" {
		t.Errorf("Expected Branch='HEAD' for detached head, got %s", info.Branch)
	}

	t.Logf("Branch: %s, Hash: %s", info.Branch, info.Hash)
}

func TestGetInfo_DirtyRepo(t *testing.T) {
	dir := setupTestRepo(t)

	// Create initial commit
	if _, err := runGit(dir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Should be clean initially
	info, err := GetInfo(dir)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if info.Dirty {
		t.Error("Expected clean repo, got dirty")
	}

	// Create untracked file (makes repo dirty)
	testFile := dir + "/test.txt"
	if err := writeFile(testFile, "test content"); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Should now be dirty
	info, err = GetInfo(dir)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if !info.Dirty {
		t.Error("Expected dirty repo after adding untracked file")
	}

	t.Logf("Dirty: %v", info.Dirty)
}

func TestGetInfo_CleanRepo(t *testing.T) {
	dir := setupTestRepo(t)

	// Create initial commit
	if _, err := runGit(dir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	info, err := GetInfo(dir)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if info == nil {
		t.Fatal("Expected info, got nil")
	}

	// Should be clean
	if info.Dirty {
		t.Error("Expected clean repo")
	}

	// Hash should be 7 chars
	if len(info.Hash) != 7 {
		t.Errorf("Expected 7-char hash, got %d chars: %s", len(info.Hash), info.Hash)
	}

	t.Logf("Branch: %s, Hash: %s, Dirty: %v", info.Branch, info.Hash, info.Dirty)
}

func TestGetInfo_StagedChanges(t *testing.T) {
	dir := setupTestRepo(t)

	// Create initial commit
	if _, err := runGit(dir, "commit", "--allow-empty", "-m", "initial"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Create and stage a file
	testFile := dir + "/staged.txt"
	if err := writeFile(testFile, "staged content"); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if _, err := runGit(dir, "add", "staged.txt"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}

	// Should be dirty (staged changes count as dirty)
	info, err := GetInfo(dir)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if !info.Dirty {
		t.Error("Expected dirty repo with staged changes")
	}

	t.Logf("Dirty (staged): %v", info.Dirty)
}

func TestGetInfo_ModifiedTrackedFile(t *testing.T) {
	dir := setupTestRepo(t)

	// Create and commit a file
	testFile := dir + "/tracked.txt"
	if err := writeFile(testFile, "initial content"); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if _, err := runGit(dir, "add", "tracked.txt"); err != nil {
		t.Fatalf("git add failed: %v", err)
	}
	if _, err := runGit(dir, "commit", "-m", "add tracked file"); err != nil {
		t.Fatalf("git commit failed: %v", err)
	}

	// Should be clean
	info, err := GetInfo(dir)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if info.Dirty {
		t.Error("Expected clean repo after commit")
	}

	// Modify the tracked file
	if err := writeFile(testFile, "modified content"); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Should now be dirty
	info, err = GetInfo(dir)
	if err != nil {
		t.Fatalf("GetInfo failed: %v", err)
	}
	if !info.Dirty {
		t.Error("Expected dirty repo after modifying tracked file")
	}

	t.Logf("Dirty (modified): %v", info.Dirty)
}

// writeFile is a helper to write content to a file.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
