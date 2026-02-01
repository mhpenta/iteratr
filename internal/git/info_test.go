package git

import (
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
