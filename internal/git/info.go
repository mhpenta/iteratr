package git

import (
	"os/exec"
	"strconv"
	"strings"
)

// Info contains git repository status information.
type Info struct {
	Branch string // Branch name or "HEAD" if detached
	Hash   string // Short commit hash (7 chars)
	Dirty  bool   // Uncommitted changes exist
	Ahead  int    // Commits ahead of remote
	Behind int    // Commits behind remote
}

// GetInfo retrieves git repository information for the given directory.
// Returns nil, nil if the directory is not a git repository.
func GetInfo(dir string) (*Info, error) {
	// Check if this is a git repository
	if !isGitRepo(dir) {
		return nil, nil
	}

	info := &Info{}

	// Get branch name (returns "HEAD" if detached)
	branch, err := runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	info.Branch = branch

	// Get short commit hash
	hash, err := runGit(dir, "rev-parse", "--short=7", "HEAD")
	if err != nil {
		return nil, err
	}
	info.Hash = hash

	// Check for dirty state
	status, err := runGit(dir, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	info.Dirty = status != ""

	// Get ahead/behind counts (may fail if no upstream)
	counts, err := runGit(dir, "rev-list", "--left-right", "--count", "@{u}...HEAD")
	if err == nil && counts != "" {
		parts := strings.Fields(counts)
		if len(parts) == 2 {
			info.Behind, _ = strconv.Atoi(parts[0])
			info.Ahead, _ = strconv.Atoi(parts[1])
		}
	}
	// If no upstream, ahead/behind stay 0

	return info, nil
}

// isGitRepo checks if the directory is inside a git repository.
func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	err := cmd.Run()
	return err == nil
}

// runGit executes a git command and returns trimmed stdout.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
