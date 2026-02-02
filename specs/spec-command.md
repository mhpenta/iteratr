# Spec Command

## Overview

New `iteratr spec` subcommand with wizard UI for creating feature specs via AI-assisted interview. Spawns opencode acp with custom MCP server exposing question/finish tools.

## User Story

Developer wants to create a well-structured spec without manually writing markdown. Wizard collects name/description, then AI agent interviews user in depth about requirements, edge cases, and tradeoffs before generating complete spec.

## Requirements

### Wizard Steps
1. **Name Input** - Single-line text, strict slug format (lowercase alphanumeric + hyphens only)
2. **Description Textarea** - Multi-line, no limit, hint: "provide as much detail as possible"
3. **Model Selector** - Same component as build wizard, fetches from `opencode models`
4. Auto-start agent phase after model selection (no explicit confirmation)

### Agent Phase UI
- Spinner with status text while agent thinking (e.g., "Agent is analyzing...")
- Agent text output hidden from user
- Questions displayed one at a time (not batch)
- No timeout on user responses
- ESC triggers "Are you sure you want to cancel?" confirmation

### MCP Server: iteratr-spec
Separate from build's `iteratr-tools`, registers two tools:

**ask-questions**
```
Parameters:
  questions: array of {
    question: string     // Full question text
    header: string       // Short label (max 30 chars)
    options: array of {
      label: string      // Display text (1-5 words)
      description: string
    }
    multiple?: bool      // Allow multi-select (default: false)
  }

Behavior:
- Show questions one at a time
- Auto-append "Type your own answer" option to all questions
- Reject empty custom responses (re-prompt)
- Block until all questions answered
- Return array of answers (strings or string arrays for multi-select)
```

**finish-spec**
```
Parameters:
  content: string   // Full spec markdown content
  name: string      // Spec name for filename

Behavior:
- Slugify name (spaces->hyphens, transliterate accents)
- Validate loosely: check for Overview, Tasks sections
- If file exists: return error requesting overwrite confirmation
- Save to {spec_dir}/{slug}.md
- Update README.md (see below)
- Return success with file path
```

### README.md Update
- Look for `<!-- SPECS -->` marker
- If found: insert row after marker
- If not found: append marker + new table after existing content
- Table format: `| Name | Description | Date |`
- Create README with header + table if missing

### Completion Screen
Three buttons after spec saved:
- **View**: Open in $EDITOR, or print path if $EDITOR unset
- **Start Build**: Execute `iteratr build --spec <path>` directly
- **Exit**: Return to shell

### Configuration
- `spec_dir` in iteratr.yml (default: `./specs`)
- `ITERATR_SPEC_DIR` env var

### Error Handling
- opencode acp start failure: show error message, exit wizard
- Agent ends without calling finish-spec: discard everything, show error
- File exists on save: MCP returns error, agent should ask user to confirm overwrite or provide new name

### Agent Prompt
```
Follow the user instructions and interview me in detail using the ask-questions 
tool about literally anything: technical implementation, UI & UX, concerns, 
tradeoffs, etc. but make sure the questions are not obvious. Be very in-depth 
and continue interviewing me continually until it's complete. Then, write the 
spec using the finish-spec tool.

Feature: {name}
Description: {description}

## Spec Format
[Include full spec format from AGENTS.md]
```

## Technical Implementation

### New Files
- `cmd/iteratr/spec.go` - Cobra command setup
- `internal/tui/specwizard/wizard.go` - Main wizard model
- `internal/tui/specwizard/name_step.go` - Name input step
- `internal/tui/specwizard/description_step.go` - Textarea step
- `internal/tui/specwizard/agent_phase.go` - Agent interaction view
- `internal/tui/specwizard/completion_step.go` - Final actions view
- `internal/tui/specwizard/question_view.go` - Single question component
- `internal/specmcp/server.go` - MCP server setup
- `internal/specmcp/tools.go` - Tool registration
- `internal/specmcp/handlers.go` - Tool handlers

### Reused Components
- `internal/tui/wizard/model_selector.go` - Model selection step
- `internal/tui/wizard/button_bar.go` - Navigation buttons
- `internal/agent/runner.go` - ACP spawning (adapted for stateless use)
- `internal/agent/acp.go` - ACP protocol

### Data Flow
```
User Input -> Wizard Steps -> Model Selected
                              |
                         Spawn opencode acp
                         Start MCP server (iteratr-spec)
                              |
                         Agent prompts <- MCP URL
                              |
Agent calls ask-questions -> MCP blocks -> UI shows question
                              |
User answers -> MCP returns -> Agent continues
                              |
Agent calls finish-spec -> Save file -> Update README
                              |
                         Completion screen
                              |
                      View / Build / Exit
```

### Config Changes
Add to `internal/config/config.go`:
```go
type Config struct {
    // ... existing fields
    SpecDir string `mapstructure:"spec_dir"`
}
```
Default: `./specs`, env: `ITERATR_SPEC_DIR`

## UI Mockup

### Name Step
```
+- Spec Wizard - Step 1 of 3: Name --------------------+
|                                                      |
|  Enter spec name (lowercase, hyphens only):          |
|  +------------------------------------------------+  |
|  | my-feature-name                                |  |
|  +------------------------------------------------+  |
|                                                      |
|                          [ Cancel ]  [ Next -> ]     |
+------------------------------------------------------+
```

### Description Step
```
+- Spec Wizard - Step 2 of 3: Description -------------+
|                                                      |
|  Describe the feature in detail:                     |
|  +------------------------------------------------+  |
|  | I want to add a new subcommand `spec` that     |  |
|  | first shows a wizard similar to the wizard in  |  |
|  | the `build` subcommand. It asks for a name     |  |
|  | then a description...                          |  |
|  +------------------------------------------------+  |
|                                                      |
|                          [ <- Back ]  [ Next -> ]    |
+------------------------------------------------------+
```

### Agent Phase (Thinking)
```
+- Spec Wizard - Interview ----------------------------+
|                                                      |
|                                                      |
|          [spinner] Agent is analyzing requirements...|
|                                                      |
|                                                      |
|                          [ Cancel ]                  |
+------------------------------------------------------+
```

### Agent Phase (Question)
```
+- Spec Wizard - Interview ----------------------------+
|                                                      |
|  Error Handling                                      |
|                                                      |
|  What should happen if the API request fails?        |
|                                                      |
|  > * Retry with exponential backoff                  |
|      Automatic retry up to 3 times                   |
|    o Show error and let user retry                   |
|      Display error modal with retry button           |
|    o Fail silently with fallback                     |
|      Use cached data if available                    |
|    o Type your own answer...                         |
|                                                      |
|                          [ Submit ]                  |
+------------------------------------------------------+
```

### Completion
```
+- Spec Wizard - Complete -----------------------------+
|                                                      |
|  [check] Spec saved to specs/my-feature-name.md      |
|  [check] Updated specs/README.md                     |
|                                                      |
|                                                      |
|          [ View ]  [ Start Build ]  [ Exit ]         |
+------------------------------------------------------+
```

## Tasks

### 1. Configuration
- [ ] Add `spec_dir` field to Config struct with default `./specs`
- [ ] Add ITERATR_SPEC_DIR env var binding in Viper setup

### 2. Cobra Command
- [ ] Create `cmd/iteratr/spec.go` with basic cobra command skeleton
- [ ] Wire command into root command

### 3. MCP Server
- [ ] Create `internal/specmcp/server.go` with HTTP server setup (copy pattern from mcpserver)
- [ ] Create `internal/specmcp/tools.go` with ask-questions and finish-spec tool registration
- [ ] Create `internal/specmcp/handlers.go` with handler stubs

### 4. Ask Questions Handler
- [ ] Implement question channel/blocking mechanism for MCP->UI communication
- [ ] Implement answer collection and response formatting
- [ ] Add multi-select support

### 5. Finish Spec Handler
- [ ] Implement slugify function with transliteration
- [ ] Implement loose validation (check for Overview, Tasks sections)
- [ ] Implement file save with overwrite detection
- [ ] Implement README.md update with marker detection/creation

### 6. Wizard Framework
- [ ] Create `internal/tui/specwizard/wizard.go` main model (3 input steps + agent phase + completion)
- [ ] Implement step navigation, button bar, modal rendering (reuse wizard patterns)

### 7. Input Steps
- [ ] Create name_step.go with textinput and slug validation
- [ ] Create description_step.go with textarea component
- [ ] Integrate existing model_selector.go as step 3

### 8. Agent Phase
- [ ] Create agent_phase.go with spinner view
- [ ] Implement opencode acp spawning (stateless, no session store)
- [ ] Wire MCP server URL into ACP session
- [ ] Handle agent text/thinking callbacks (update spinner status)

### 9. Question View
- [ ] Create question_view.go with scrollable options list
- [ ] Implement custom answer text input mode
- [ ] Implement multi-select toggle behavior
- [ ] Wire question channel to receive questions from MCP handler

### 10. Completion Step
- [ ] Create completion_step.go with success message
- [ ] Implement View button ($EDITOR or path fallback)
- [ ] Implement Start Build button (exec iteratr build)
- [ ] Implement Exit button

### 11. Cancellation Flow
- [ ] Add confirmation modal for ESC during agent phase
- [ ] Implement clean shutdown of opencode acp process
- [ ] Implement MCP server shutdown

### 12. Integration & Testing
- [ ] Wire all components together in spec.go
- [ ] Manual E2E test: full wizard flow
- [ ] Test error cases: missing opencode, agent failure, file exists

## Out of Scope

- CLI flags for non-interactive use (always wizard)
- Session persistence for spec interviews
- Resuming interrupted spec sessions
- Editing existing specs through wizard
- Multiple spec generation in single session

## Open Questions

1. Should finish-spec support a `confirmed_overwrite: bool` param, or require agent to call with different name?
2. Should README migration from old format be a separate command?
3. Future: support for templates (different spec formats for different project types)?
