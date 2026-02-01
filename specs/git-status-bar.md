# Git Status Bar

## Overview

Display git repository information in the status bar: branch name, dirty indicator, ahead/behind counts, and short commit hash. Updates on file changes.

## User Story

As a developer, I want to see git status at a glance so I know which branch I'm on and whether there are uncommitted changes.

## Requirements

- Show after session name: `iteratr | my-session | main* abc1234 ↑2↓1 | 00:12:34`
- Branch name with asterisk suffix when dirty (uncommitted changes)
- Short commit hash (7 chars)
- Ahead/behind counts relative to tracking branch (omit if 0/0 or no remote)
- Update when agent modifies files (piggyback on existing `FileChangeMsg`)
- Non-git directories: show nothing (silent omit)
- Detached HEAD: show `HEAD` instead of branch name

## Technical Implementation

### Git Info Package

New `internal/git/info.go`:

```go
type Info struct {
    Branch  string  // Branch name or "HEAD" if detached
    Hash    string  // Short commit hash (7 chars)
    Dirty   bool    // Uncommitted changes exist
    Ahead   int     // Commits ahead of remote
    Behind  int     // Commits behind remote
}

func GetInfo(dir string) (*Info, error)  // Returns nil if not a git repo
```

Implementation uses `git` CLI:
- `git rev-parse --abbrev-ref HEAD` → branch (returns "HEAD" if detached)
- `git rev-parse --short=7 HEAD` → hash
- `git status --porcelain` → dirty (non-empty output = dirty)
- `git rev-list --left-right --count @{u}...HEAD` → ahead/behind (may fail if no upstream)

### Message Type

In `internal/tui/messages.go`:

```go
type GitInfoMsg struct {
    Branch string
    Hash   string
    Dirty  bool
    Ahead  int
    Behind int
    Valid  bool  // false if not a git repo
}
```

### Status Bar Changes

`internal/tui/status.go`:
- Add fields: `gitBranch`, `gitHash`, `gitDirty`, `gitAhead`, `gitBehind`, `gitValid`
- Add `SetGitInfo(msg GitInfoMsg)` method
- Update `buildLeft()` to render git info after session name

Format: `branch* hash ↑N↓M` where:
- `*` only if dirty
- `↑N` only if ahead > 0
- `↓M` only if behind > 0

### Update Flow

```
FileChangeMsg received in App.Update()
  ↓
App calls git.GetInfo(workdir)
  ↓
App sends GitInfoMsg to status bar
  ↓
status.SetGitInfo() updates fields
```

Also fetch on startup (`App.Init()`).

### Throttling

Git commands are fast but avoid spam during rapid file changes:
- Track `lastGitCheck time.Time` in App
- Skip if < 500ms since last check
- Always check on first file change of iteration

## Tasks

### 1. Git info package

- [ ] Create `internal/git/info.go` with `Info` struct
- [ ] Implement `GetInfo(dir string)` using exec.Command
- [ ] Handle non-git directories (return nil, nil)
- [ ] Handle detached HEAD state
- [ ] Handle missing upstream (ahead/behind = 0)
- [ ] Add unit tests with mock git repos

### 2. Message type and status bar fields

- [ ] Add `GitInfoMsg` to `internal/tui/messages.go`
- [ ] Add git fields to `StatusBar` struct
- [ ] Add `SetGitInfo(msg GitInfoMsg)` method
- [ ] Handle `GitInfoMsg` in `StatusBar.Update()`

### 3. Status bar rendering

- [ ] Update `buildLeft()` to include git info after session name
- [ ] Format: `branch* hash ↑N↓M` with conditional parts
- [ ] Style branch name (theme primary color)
- [ ] Style dirty asterisk (theme warning color)

### 4. App integration

- [ ] Fetch git info on startup in `App.Init()`
- [ ] Add `lastGitCheck` field for throttling
- [ ] Update git info on `FileChangeMsg` (with 500ms throttle)
- [ ] Send `GitInfoMsg` to status bar

## UI Mockup

**Clean repo, synced with remote:**
```
iteratr | my-session | main abc1234 | 00:12:34 | Iteration #3 | ...
```

**Dirty repo, ahead of remote:**
```
iteratr | my-session | main* abc1234 ↑2 | 00:12:34 | Iteration #3 | ...
```

**Dirty repo, behind remote:**
```
iteratr | my-session | feature/foo* def5678 ↓3 | 00:12:34 | Iteration #3 | ...
```

**Detached HEAD:**
```
iteratr | my-session | HEAD abc1234 | 00:12:34 | Iteration #3 | ...
```

**Not a git repo:**
```
iteratr | my-session | 00:12:34 | Iteration #3 | ...
```

## Out of Scope (v1)

- Stash count display
- Staged vs unstaged distinction
- Submodule status
- Git config/hooks integration
- Click to open git log

## Open Questions

None - resolved during interview.
