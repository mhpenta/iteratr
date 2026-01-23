# Agent Message Display

## Overview

Overhaul `AgentOutput` to match crush's rich message rendering: type-specific message items, thinking blocks, enhanced tool display, streaming animations, metadata footer, and width-based caching.

## User Story

As a user watching the agent work, I want clearly differentiated, well-styled message blocks (text, thinking, tools, errors) so I can quickly understand what the agent is doing and scan output efficiently.

## Requirements

- Assistant text rendered as markdown with syntax highlighting
- Thinking/reasoning blocks: collapsible, shows last 10 lines when collapsed, footer with duration
- Tool calls: status icons (●/✓/×), header with params, expandable output (10 lines default), error/canceled states
- Streaming animation (gradient spinner) while waiting for content
- Assistant info footer: `◇ Model via Provider ⏱ Duration`
- Width-based render caching (only re-render when width changes)
- Error/canceled finish states displayed inline

## Technical Implementation

### Current Architecture
- `AgentOutput` in `internal/tui/agent.go` - flat `[]AgentMessage` rendered through viewport
- `AgentMessage` struct: Type (Text/Tool/Divider), Content, Tool, ToolStatus, ToolOutput
- `RunnerConfig` callbacks: `OnText(string)`, `OnToolCall(ToolCallEvent)`
- `ToolCallEvent`: ToolCallID, Title, Status (pending/in_progress/completed), RawInput, Output, Kind

### Target Architecture
- Replace `AgentMessage` with interface-based `MessageItem` hierarchy
- Type-specific structs: `TextMessageItem`, `ThinkingMessageItem`, `ToolMessageItem`
- Width-based caching wrapper
- New callbacks: `OnThinking`, `OnFinish` from ACP
- Gradient spinner animation for streaming state

### New Dependencies
- `charm.land/glamour/v2` - markdown rendering
- `github.com/alecthomas/chroma/v2` - syntax highlighting for tool code output

### Key Files
- `internal/tui/agent.go` - main refactor target
- `internal/tui/messages.go` - new: MessageItem interface + type structs
- `internal/tui/markdown.go` - new: glamour wrapper
- `internal/tui/highlight.go` - new: chroma syntax highlighting
- `internal/tui/styles.go` - new styles
- `internal/tui/anim.go` - gradient spinner
- `internal/agent/types.go` - new event types (FinishEvent)
- `internal/agent/acp.go` - parse `agent_thought_chunk`
- `internal/agent/runner.go` - new callbacks (OnThinking, OnFinish)
- `internal/orchestrator/orchestrator.go` - wire new callbacks to TUI

## Tasks

### 1. Message item interface and types

- [ ] Create `internal/tui/messages.go` with `MessageItem` interface: `ID() string`, `Render(width int) string`, `Height() int`
- [ ] Add `Expandable` interface: `IsExpanded() bool`, `ToggleExpanded()`
- [ ] Define `TextMessageItem` struct: id, content string, cachedRender string, cachedWidth int
- [ ] Define `ThinkingMessageItem` struct: id, content string, collapsed bool (default true), duration time.Duration, finished bool
- [ ] Define `ToolMessageItem` struct: id, toolName string, status ToolStatus enum, input map[string]any, output string, expanded bool, maxLines int (default 10)
- [ ] Define `InfoMessageItem` struct: id, model string, provider string, duration time.Duration
- [ ] Define `DividerMessageItem` struct: id, iteration int
- [ ] Define `ToolStatus` type: `ToolStatusPending`, `ToolStatusRunning`, `ToolStatusSuccess`, `ToolStatusError`, `ToolStatusCanceled`

### 2. Styles for new message types

- [ ] Add `styleThinkingBox` in `styles.go`: Background(colorSurface0), PaddingLeft(1), MarginBottom(1)
- [ ] Add `styleThinkingContent`: Foreground(colorSubtext1), Italic(true)
- [ ] Add `styleThinkingTruncationHint`: Foreground(colorSubtext0), Italic(true)
- [ ] Add `styleThinkingFooter`: Foreground(colorSubtext0)
- [ ] Add `styleThinkingDuration`: Foreground(colorSecondary)
- [ ] Add `styleToolIconPending`: Foreground(colorWarning) (renders "●")
- [ ] Add `styleToolIconSuccess`: Foreground(colorSuccess) (renders "✓")
- [ ] Add `styleToolIconError`: Foreground(colorError) (renders "×")
- [ ] Add `styleToolIconCanceled`: Foreground(colorMuted) (renders "×")
- [ ] Add `styleToolName`: Foreground(colorSecondary), Bold(true)
- [ ] Add `styleToolParams`: Foreground(colorSubtext0)
- [ ] Add `styleToolOutput`: Background(colorSurface0), PaddingLeft(2)
- [ ] Add `styleToolTruncation`: Foreground(colorSubtext0), Italic(true)
- [ ] Add `styleToolError`: Foreground(colorError)
- [ ] Add `styleInfoIcon`: Foreground(colorMuted) (renders "◇")
- [ ] Add `styleInfoModel`: Foreground(colorSecondary)
- [ ] Add `styleInfoProvider`: Foreground(colorSubtext0)
- [ ] Add `styleInfoDuration`: Foreground(colorInfo)
- [ ] Add `styleAssistantBorder`: Border left, BorderForeground(colorPrimary), PaddingLeft(1)
- [ ] Add `styleFinishError`: Foreground(colorError)
- [ ] Add `styleFinishCanceled`: Foreground(colorMuted), Italic(true)

### 3. Render methods for each message type

- [ ] Implement `TextMessageItem.Render(width)`: word wrap content, apply `styleAssistantBorder`, cap at min(width-2, 120)
- [ ] Implement `ThinkingMessageItem.Render(width)`: if collapsed and >10 lines show last 10 with "… (N lines hidden)" hint; add footer "Thought for Xs" if finished; wrap in `styleThinkingBox`
- [ ] Implement `ToolMessageItem.Render(width)`: header = `[icon] [name] [formatted params]`; body = output lines capped at maxLines with truncation hint; code output (View/Edit tools with filePath) uses `syntaxHighlight()`; plain output uses `styleToolOutput` background
- [ ] Implement `InfoMessageItem.Render(width)`: format `◇ Model via Provider ⏱ Duration`
- [ ] Implement `DividerMessageItem.Render(width)`: centered label with horizontal rules (keep existing logic)
- [ ] Implement `ToolMessageItem` param formatting: show primary param (command/filePath) then `(key=val, ...)` for remaining, truncate to width

### 4. Width-based render caching

- [ ] Add `cachedRender string` and `cachedWidth int` fields to each MessageItem struct
- [ ] In each `Render(width)` method: return cachedRender if width == cachedWidth, otherwise render and cache
- [ ] Invalidate cache (reset cachedWidth to 0) when content changes (e.g., tool output arrives, thinking content appends)

### 5. Refactor AgentOutput to use MessageItem slice

- [ ] Change `messages []AgentMessage` to `messages []MessageItem` in `AgentOutput`
- [ ] Update `refreshContent()` to call `msg.Render(contentWidth)` for each item
- [ ] Update `AppendText()` to create/append-to `TextMessageItem`
- [ ] Update `AppendToolCall()` to create/update `ToolMessageItem` (map status strings to ToolStatus enum)
- [ ] Update `AddIterationDivider()` to create `DividerMessageItem`
- [ ] Remove old `renderMessage()`, `renderTextMessage()`, `renderToolMessage()`, `renderDivider()` methods
- [ ] Remove old `AgentMessage` struct and `MessageType` enum

### 6. Thinking block support - ACP layer

- [ ] Add `"agent_thought_chunk"` case to `sessionUpdate` switch in `acp.go` (line ~283)
- [ ] Define `agentThoughtChunk` struct: `SessionUpdate string`, `Content contentPart` (same shape as `agentMessageChunk`)
- [ ] Add `OnThinking func(string)` callback to `RunnerConfig` in `runner.go`
- [ ] Add `onThinking func(string)` field to `Runner` struct, wire in `NewRunner()`
- [ ] In `prompt()`: parse `agent_thought_chunk`, call `onThinking(chunk.Content.Text)`

### 7. Finish event support - runner layer

ACP has no `agent_finish` notification. Completion is signaled by `PromptResponse` return with `StopReason` (end_turn, max_tokens, cancelled, refusal, max_turn_requests). Errors are JSON-RPC errors.

- [ ] Define `FinishEvent` struct in `types.go`: StopReason string, Error string, Duration time.Duration, Model string, Provider string
- [ ] Add `OnFinish func(FinishEvent)` callback to `RunnerConfig`
- [ ] Add `onFinish` field to `Runner` struct, wire in `NewRunner()`
- [ ] In `RunIteration()`: capture `startTime := time.Now()` before `conn.prompt()`
- [ ] After `conn.prompt()` returns: compute duration, call `onFinish(FinishEvent{StopReason: "end_turn", Duration: elapsed, Model: r.model})`
- [ ] On prompt error: call `onFinish(FinishEvent{StopReason: "error", Error: err.Error(), Duration: elapsed})`
- [ ] On ctx cancel: call `onFinish(FinishEvent{StopReason: "cancelled", Duration: elapsed})`
- [ ] Parse `StopReason` from `PromptResponse` in `acp.go` if available (add `stopReason` field to prompt response struct)

### 8. TUI message types for new events

- [ ] Add `AgentThinkingMsg` struct: `Content string`
- [ ] Add `AgentFinishMsg` struct: `Reason string`, `Error string`, `Model string`, `Provider string`, `Duration time.Duration`
- [ ] Handle `AgentThinkingMsg` in `app.go` Update: call `agent.AppendThinking(msg.Content)`
- [ ] Handle `AgentFinishMsg` in `app.go` Update: call `agent.AppendFinish(msg)`

### 9. AgentOutput methods for thinking and finish

- [ ] Add `AppendThinking(content string)` method: if last message is ThinkingMessageItem append to it, else create new one; invalidate cache
- [ ] Add `AppendFinish(msg AgentFinishMsg)` method: set finished=true on last ThinkingMessageItem (with duration); append InfoMessageItem (model/provider/duration); if error/canceled append styled finish reason to last TextMessageItem
- [ ] Add `MarkToolError(toolCallID, errMsg string)` method: find tool by ID, set status to ToolStatusError, set output to error message
- [ ] Add `MarkToolCanceled(toolCallID string)` method: find tool by ID, set status to ToolStatusCanceled

### 10. Wire new callbacks in orchestrator

- [ ] Add `OnThinking` callback to RunnerConfig in orchestrator: send `tui.AgentThinkingMsg{Content: content}`
- [ ] Add `OnFinish` callback to RunnerConfig in orchestrator: send `tui.AgentFinishMsg{...}`
- [ ] Capture model/provider from config to pass in FinishMsg
- [ ] Wire same callbacks in headless mode (print thinking dimmed, print finish summary)

### 11. Tool expand/collapse interaction

- [ ] Add key handling in `AgentOutput.Update()`: when focused tool message receives Space/Enter, call `ToggleExpanded()`
- [ ] Track `focusedIndex int` in AgentOutput for which message has keyboard focus
- [ ] Add Up/Down key handling to move focusedIndex between expandable messages
- [ ] On toggle: invalidate cache for that message, call `refreshContent()`

### 12. Gradient spinner animation

- [ ] Create `GradientSpinner` struct in `anim.go`: frame int, size int (default 15), colorA/colorB lipgloss.Color, label string
- [ ] Add `GradientSpinnerMsg` tick message type
- [ ] Implement `GradientSpinner.View()`: render size-char string with gradient between colorA/colorB, shifting by frame; prepend label if set
- [ ] Implement `GradientSpinner.Tick() tea.Cmd`: return tick command at 80ms interval
- [ ] Implement `GradientSpinner.Update(msg)`: increment frame on tick
- [ ] Add `spinner *GradientSpinner` and `isStreaming bool` fields to `AgentOutput`
- [ ] Start spinner when first `AppendText("")` or `AppendThinking("")` arrives (streaming begins)
- [ ] Stop spinner when content arrives or finish event received
- [ ] In `refreshContent()`: if isStreaming and no content yet, prepend spinner view with label ("Thinking..." or "Generating...")

### 13. Markdown rendering for text content

- [ ] Add glamour dependency: `go get charm.land/glamour/v2`
- [ ] Add chroma dependency: `go get github.com/alecthomas/chroma/v2`
- [ ] Create `internal/tui/markdown.go` with `renderMarkdown(content string, width int) string`
- [ ] Create glamour renderer with `glamour.WithWordWrap(width)` and dark style config
- [ ] Create `internal/tui/highlight.go` with `syntaxHighlight(source, fileName string) string`
- [ ] Use `lexers.Match(fileName)` for language detection, fallback to `lexers.Analyse(source)`, then `lexers.Fallback`
- [ ] Use `formatters.Get("terminal16m")` for true color output
- [ ] Use `renderMarkdown()` in `TextMessageItem.Render()` instead of plain `wrapText()`
- [ ] Use `syntaxHighlight()` in `ToolMessageItem.Render()` for code output (when tool has filePath in input)
- [ ] Preserve `wrapText()` as fallback if glamour returns error

### 14. Error state for tool calls

- [ ] Add "error" status support to `ToolCallEvent.Status` (alongside pending/in_progress/completed)
- [ ] Handle "error" status in `AppendToolCall()`: set ToolStatusError, store error output
- [ ] Render error tools: `× ToolName` header + red-styled error message body
- [ ] Add "canceled" status handling: set ToolStatusCanceled on context cancel

### 15. Tests

- [ ] Unit test `TextMessageItem.Render()`: verify word wrap, border, width capping at 120
- [ ] Unit test `ThinkingMessageItem.Render()`: verify collapse at >10 lines, expand shows all, truncation hint text, footer with duration
- [ ] Unit test `ToolMessageItem.Render()`: verify icon per status, param formatting, output truncation at maxLines, expand shows all
- [ ] Unit test `InfoMessageItem.Render()`: verify format string
- [ ] Unit test width caching: verify second call with same width returns cached, different width re-renders
- [ ] Update existing `app_test.go` and `integration_test.go` to use new message types
- [ ] Test `AppendThinking()`: appends to existing thinking item, creates new if last isn't thinking
- [ ] Test `AppendFinish()`: creates InfoMessageItem, marks thinking finished

## Out of Scope

- User message rendering (agent-only output currently)
- Text selection/highlighting
- Per-message focus/blur styling (panel-level focus is sufficient for v1)
- File/image attachment display
- Compact mode for nested tools (no nesting in current agent)
- Click-to-expand (keyboard only for v1)

## Open Questions

None.

## Resolved

- ACP emits `"agent_thought_chunk"` for thinking (same shape as `agent_message_chunk`)
- ACP has no finish notification; `StopReason` comes from `PromptResponse` return (end_turn, max_tokens, cancelled, refusal, max_turn_requests)
- Model/provider info passed from RunnerConfig (already available as `r.model`); provider parsed from model string (e.g., "anthropic/claude-sonnet-4-5" → provider="Anthropic")
- Markdown rendering: use **glamour** (`glamour.WithStyles()`, `glamour.WithWordWrap(width)`) — same as crush
- Tool output: use **chroma** for syntax highlighting (auto language detection via `lexers.Match(fileName)` with `lexers.Analyse(source)` fallback); plain text output uses background styling only
