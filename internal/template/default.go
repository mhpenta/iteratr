package template

// DefaultTemplate is the embedded default prompt template.
// It uses {{variable}} placeholders for dynamic content injection.
const DefaultTemplate = `# iteratr Session
Session: {{session}} | Iteration: #{{iteration}}

{{history}}

## Spec
{{spec}}

{{tasks}}

{{notes}}

## Rules
- ONE task per iteration - complete fully, then STOP
- Test changes before marking complete
- Write iteration-summary before stopping
- Call session-complete only when ALL tasks done
- Respect user-added tasks even if not in spec

## Workflow
1. **Sync tasks with spec** (EVERY iteration, BEFORE picking a task):
   a. List ALL requirements/items from the spec (e.g., files to fix, features to implement)
   b. List ALL existing non-cancelled tasks
   c. For EACH spec requirement: check if a matching task exists
      - If NO matching task exists: use task-add to create it
   d. For EACH existing task: check if it still matches the current spec
      - If task references outdated info (e.g., "9 files" but spec now says "15 files"):
        cancel the outdated task and add a new task with correct info
      - If task has no corresponding spec requirement: cancel it (unless user-added)
   e. Only proceed to step 2 after sync is complete
2. Pick ONE ready task (highest priority, no blockers) using task-next tool
3. Mark task as in_progress using task-update tool
4. Implement + test
5. Mark task as completed using task-update tool
6. Write iteration-summary using iteration-summary tool
7. STOP (do not pick another task)

## If Stuck
- Add a note using note-add tool with type "stuck" describing the issue
- Mark task blocked or fix before completing
- If blocked by another task: use task-update tool to set depends_on

## Subagents
Spin up subagents (via Task tool) to parallelize work. Each subagent has fresh context, so "one task per agent" is preserved.

**DO parallelize when:**
- Tasks are independent (no shared files)
- Tasks have no uncommitted dependencies between them
- Read-only research while you implement

**DO NOT parallelize when:**
- Tasks modify the same files (causes conflicts)
- Task B depends on Task A's uncommitted changes
- Uncertain about conflicts - err sequential

Mark all delegated tasks in_progress, then completed when subagents finish.
{{extra}}`
