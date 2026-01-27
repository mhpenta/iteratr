# OpenCode ACP Protocol Research

Research document covering the Agent Client Protocol (ACP) implementation in OpenCode, including all message types, subagent detection, and session tracking mechanisms.

## Overview

OpenCode implements ACP v1, communicating via JSON-RPC over stdin/stdout. The protocol supports request-response patterns and one-way notifications.

**Protocol Version:** 1  
**Transport:** stdin/stdout JSON-RPC

---

## Session Metadata Summary

The ACP protocol surfaces rich metadata during initialization and session creation:

### From `initialize` Response
| Field | Description |
|-------|-------------|
| `protocolVersion` | ACP protocol version (1) |
| `agentInfo.name` | Agent name ("OpenCode") |
| `agentInfo.version` | Agent version (e.g., "1.1.36") |
| `agentCapabilities` | Supported features (MCP, images, sessions) |
| `authMethods[]` | Available authentication methods |

### From `session/new` Response
| Field | Description |
|-------|-------------|
| `sessionId` | Unique session identifier |
| `models.currentModelId` | Active model (`provider/model`) |
| `models.availableModels[]` | All available models with display names |
| `modes.currentModeId` | Active mode (e.g., "build") |
| `modes.availableModes[]` | Available operating modes |

### From `available_commands_update` Notification
| Field | Description |
|-------|-------------|
| `availableCommands[]` | Slash commands with names and descriptions |

Sent immediately after `session/prompt` is called.

---

## Client to Agent Requests

### initialize

Negotiate protocol version and capabilities. Returns agent metadata and supported features.

**Request:**
```json
{
  "protocolVersion": 1,
  "clientCapabilities": {
    "fs": {
      "readTextFile": true,
      "writeTextFile": true
    },
    "terminal": true,
    "_meta": {
      "terminal-auth": true
    }
  },
  "clientInfo": {
    "name": "my-client",
    "version": "1.0.0"
  }
}
```

**Response:**
```json
{
  "protocolVersion": 1,
  "agentCapabilities": {
    "loadSession": true,
    "mcpCapabilities": {
      "http": true,
      "sse": true
    },
    "promptCapabilities": {
      "embeddedContext": true,
      "image": true
    },
    "sessionCapabilities": {
      "fork": {},
      "list": {},
      "resume": {}
    }
  },
  "authMethods": [
    {
      "id": "opencode-login",
      "name": "Login with opencode",
      "description": "Run `opencode auth login` in the terminal"
    }
  ],
  "agentInfo": {
    "name": "OpenCode",
    "version": "1.1.36"
  }
}
```

**Client Capabilities:**
- `fs.readTextFile` - Client can handle file read requests
- `fs.writeTextFile` - Client can handle file write requests
- `terminal` - Client can handle terminal/command execution requests

**Agent Capabilities:**
- `loadSession` - Agent supports loading existing sessions
- `mcpCapabilities.http` - Agent supports HTTP MCP servers
- `mcpCapabilities.sse` - Agent supports SSE MCP servers
- `promptCapabilities.embeddedContext` - Agent accepts embedded resource content
- `promptCapabilities.image` - Agent accepts image content
- `sessionCapabilities.fork` - Agent supports forking sessions
- `sessionCapabilities.list` - Agent supports listing sessions
- `sessionCapabilities.resume` - Agent supports resuming sessions

**Auth Methods:**
- `id` - Identifier to pass to `authenticate` request
- `name` - Human-readable name
- `description` - Instructions for authentication

### authenticate

Select authentication method. Currently unimplemented (throws error).

**Request:**
```json
{
  "authMethodId": "string"
}
```

### session/new

Create a new session. Returns rich metadata about available models, modes, and capabilities.

**Request:**
```json
{
  "cwd": "/path/to/working/directory",
  "mcpServers": [
    { "name": "string", "type": "remote", "url": "string", "headers": [] },
    { "name": "string", "command": "string", "args": [], "env": [] }
  ]
}
```

**Response:**
```json
{
  "sessionId": "ses_401a7ea13ffeDFQ4LWs35Ll3gw",
  "models": {
    "currentModelId": "opencode/big-pickle",
    "availableModels": [
      { "modelId": "anthropic/claude-sonnet-4-5", "name": "Anthropic/Claude Sonnet 4.5 (latest)" },
      { "modelId": "anthropic/claude-opus-4-5", "name": "Anthropic/Claude Opus 4.5 (latest)" },
      { "modelId": "openai/gpt-5.2", "name": "OpenAI/GPT-5.2" },
      { "modelId": "google/gemini-2.5-pro", "name": "Google/Gemini 2.5 Pro" }
      // ... many more models
    ]
  },
  "modes": {
    "currentModeId": "build",
    "availableModes": [
      { "id": "build", "name": "build" },
      { "id": "plan", "name": "plan" }
    ]
  },
  "_meta": {}
}
```

**Models Response Structure:**
- `currentModelId` - Active model in `providerID/modelID` format
- `availableModels[]` - All models the agent can use
  - `modelId` - Unique identifier (`provider/model-name`)
  - `name` - Human-readable display name

**Modes Response Structure:**
- `currentModeId` - Active operating mode
- `availableModes[]` - Available agent modes
  - `id` - Mode identifier
  - `name` - Display name
  - `description` - Optional description

### session/load

Load an existing session with replayed history.

**Request:**
```json
{
  "sessionId": "session_xxx",
  "cwd": "/path/to/working/directory",
  "mcpServers": []
}
```

**Response:** Same structure as `session/new`.

### session/prompt

Send user input to the agent.

**Request:**
```json
{
  "sessionId": "session_xxx",
  "prompt": [
    { "type": "text", "text": "string", "annotations": { "audience": ["user"] } },
    { "type": "image", "uri": "string", "mimeType": "image/png", "data": "base64" },
    { "type": "resource", "resource": { "uri": "string", "mimeType": "string", "text": "string" } },
    { "type": "resource_link", "uri": "string", "name": "string", "mimeType": "string" }
  ]
}
```

**Response:**
```json
{
  "stopReason": "end_turn"
}
```

**Stop Reasons:** `completed`, `cancelled`, `stopped`, `max_output_tokens`, `auth_required`, `tool_choice`, `end_turn`

### session/cancel (Notification)

Cancel ongoing operation. No response expected.

**Notification:**
```json
{
  "sessionId": "session_xxx"
}
```

### setSessionMode

Change the agent's operating mode.

**Request:**
```json
{
  "sessionId": "session_xxx",
  "modeId": "string"
}
```

### unstable_setSessionModel

Change the model (experimental).

**Request:**
```json
{
  "sessionId": "session_xxx",
  "modelId": "providerID/modelID"
}
```

### unstable_listSessions

List available sessions.

**Request:**
```json
{
  "cwd": "/optional/path",
  "cursor": "optional_cursor"
}
```

**Response:**
```json
{
  "sessions": [{
    "sessionId": "session_xxx",
    "cwd": "/path",
    "title": "string",
    "updatedAt": "ISO8601"
  }],
  "nextCursor": "string"
}
```

### unstable_forkSession

Fork an existing session.

**Request:**
```json
{
  "sessionId": "session_xxx",
  "cwd": "/path",
  "mcpServers": []
}
```

### unstable_resumeSession

Resume a session.

**Request:**
```json
{
  "sessionId": "session_xxx",
  "cwd": "/path",
  "mcpServers": []
}
```

---

## Agent to Client Requests

### requestPermission

Request user approval for sensitive operations.

**Request:**
```json
{
  "sessionId": "session_xxx",
  "toolCall": {
    "toolCallId": "string",
    "status": "pending",
    "title": "string",
    "rawInput": {},
    "kind": "execute",
    "locations": [{ "path": "/file/path" }]
  },
  "options": [
    { "optionId": "once", "kind": "allow_once", "name": "Allow once" },
    { "optionId": "always", "kind": "allow_always", "name": "Always allow" },
    { "optionId": "reject", "kind": "reject_once", "name": "Reject" }
  ]
}
```

**Response:**
```json
{
  "outcome": {
    "outcome": "selected",
    "optionId": "once"
  }
}
```

**Permission Option Kinds:** `allow_once`, `allow_always`, `reject_once`, `reject_always`

---

## Agent to Client Notifications (Session Updates)

All session updates follow this structure:

```json
{
  "sessionId": "session_xxx",
  "update": {
    "sessionUpdate": "<update_type>",
    ...
  }
}
```

### user_message_chunk

Stream user message content.

```json
{
  "sessionUpdate": "user_message_chunk",
  "content": {
    "type": "text",
    "text": "string",
    "annotations": { "audience": ["user"] }
  }
}
```

### agent_message_chunk

Stream agent response text.

```json
{
  "sessionUpdate": "agent_message_chunk",
  "content": {
    "type": "text",
    "text": "string",
    "annotations": { "audience": ["assistant"] }
  }
}
```

Also supports `resource_link` and `image` content types.

### agent_thought_chunk

Stream reasoning/thinking content.

```json
{
  "sessionUpdate": "agent_thought_chunk",
  "content": {
    "type": "text",
    "text": "string"
  }
}
```

### tool_call

Tool invocation started (pending state).

```json
{
  "sessionUpdate": "tool_call",
  "toolCallId": "string",
  "title": "string",
  "kind": "execute",
  "status": "pending",
  "locations": [{ "path": "/file/path" }],
  "rawInput": {}
}
```

### tool_call_update

Tool status change.

**In Progress:**
```json
{
  "sessionUpdate": "tool_call_update",
  "toolCallId": "string",
  "status": "in_progress",
  "kind": "execute",
  "title": "string",
  "locations": [],
  "rawInput": {}
}
```

**Completed:**
```json
{
  "sessionUpdate": "tool_call_update",
  "toolCallId": "string",
  "status": "completed",
  "kind": "execute",
  "title": "string",
  "rawInput": {},
  "rawOutput": {
    "output": "string",
    "metadata": {}
  },
  "content": [
    { "type": "content", "content": { "type": "text", "text": "string" } },
    { "type": "diff", "path": "/file", "oldText": "", "newText": "" }
  ]
}
```

**Failed:**
```json
{
  "sessionUpdate": "tool_call_update",
  "toolCallId": "string",
  "status": "failed",
  "kind": "execute",
  "title": "string",
  "rawInput": {},
  "rawOutput": { "error": "string" },
  "content": []
}
```

**Tool Call Statuses:** `pending`, `running`, `in_progress`, `completed`, `failed`, `partially_completed`

### plan

Todo list updates.

```json
{
  "sessionUpdate": "plan",
  "entries": [{
    "priority": "medium",
    "status": "pending",
    "content": "string"
  }]
}
```

**Plan Entry Status:** `pending`, `in_progress`, `completed`  
**Plan Entry Priority:** `high`, `medium`, `low`

### available_commands_update

Available slash commands. Sent immediately after `session/prompt` request.

```json
{
  "sessionUpdate": "available_commands_update",
  "availableCommands": [
    { "name": "init", "description": "create/update AGENTS.md" },
    { "name": "review", "description": "review changes [commit|branch|pr], defaults to uncommitted" },
    { "name": "validate_plan", "description": "Validate implementation against plan, verify success criteria, identify issues" },
    { "name": "research_codebase", "description": "Document codebase as-is" },
    { "name": "implement_plan", "description": "Implement technical plans from thoughts/shared/plans with verification" },
    { "name": "describe_pr", "description": "Generate comprehensive PR descriptions following repository templates" },
    { "name": "create_plan", "description": "Create detailed implementation plans through interactive research and iteration" },
    { "name": "compact", "description": "compact the session" }
  ]
}
```

**Note:** Commands are project-specific and defined in `.opencode/` configuration.

---

## Tool Kinds

| Kind | Tools |
|------|-------|
| `execute` | bash |
| `fetch` | webfetch |
| `edit` | edit, patch, write |
| `search` | grep, glob, context7_* |
| `read` | list, read |
| `other` | everything else |

---

## Content Block Types

Used in prompts and message chunks:

| Type | Fields |
|------|--------|
| `text` | `text`, `annotations?` |
| `image` | `uri?`, `mimeType`, `data?` (base64) |
| `resource` | `resource: { uri, mimeType, text?, blob? }` |
| `resource_link` | `uri`, `name?`, `mimeType?` |

---

## Subagent Detection and Tracking

### Session Parent-Child Relationships

Sessions have a `parentID` field indicating hierarchy:

```typescript
interface SessionInfo {
  id: string
  parentID?: string  // If set, this is a child/subagent session
  title: string
  // ...
}
```

**Detection:**
- `parentID` exists = child session spawned by subagent
- `parentID` undefined = root/primary session

### Task Tool - Subagent Spawning

The `task` tool spawns subagents with these parameters:

**Input:**
```json
{
  "description": "Short task description",
  "prompt": "Detailed task instructions",
  "subagent_type": "general|explore|codebase-analyzer|...",
  "session_id": "optional - continue existing session",
  "command": "optional - triggering command"
}
```

**Output (in rawOutput):**
```
<task_metadata>
session_id: session_xxx
</task_metadata>
```

**Metadata:**
```json
{
  "summary": [],
  "sessionId": "session_xxx",
  "model": {
    "modelID": "string",
    "providerID": "string"
  }
}
```

### Available Subagent Types

| Type | Purpose |
|------|---------|
| `general` | General-purpose multi-step tasks |
| `explore` | Fast codebase exploration |
| `web-search-researcher` | Web search and research |
| `thoughts-locator` | Find relevant thoughts documents |
| `thoughts-analyzer` | Deep dive on research topics |
| `codebase-pattern-finder` | Find similar implementations |
| `codebase-locator` | Locate files and components |
| `codebase-analyzer` | Analyze implementation details |
| `component-tree` | Update component-tree.md |

### Session Title Patterns

Titles indicate session type:

- Root sessions: `"New session - {timestamp}"`
- Child sessions: `"Child session - {timestamp}"`
- Subagent sessions: `"{description} (@{agent.name} subagent)"`

### Copilot Plugin Detection

The copilot plugin marks requests with `x-initiator` header:

```typescript
headers["x-initiator"] = isAgent ? "agent" : "user"
```

Detection logic:
- Check if last message role !== "user" 
- Check if session has `parentID`

### Querying Child Sessions

```typescript
const childSessions = await Session.children(parentSessionId)
```

### Detecting Subagent Calls via ACP

Watch `tool_call` and `tool_call_update` for:

```javascript
// Detection
if (update.sessionUpdate === "tool_call" || update.sessionUpdate === "tool_call_update") {
  const isSubagent = rawInput?.subagent_type !== undefined
  
  if (isSubagent) {
    const subagentType = rawInput.subagent_type
    const description = rawInput.description
    const spawnedSessionId = metadata?.sessionId  // Available on completion
  }
}
```

### Limitations

**Not exposed via ACP:**
- No dedicated "subagent spawned" notification
- No subagent session start/end events
- `parentID` not in session responses
- No way to subscribe to child session updates

**Workarounds:**
- Parse `task` tool calls for subagent info
- Extract `session_id` from `<task_metadata>` in output
- Track `metadata.sessionId` on tool completion

---

## Message Flow Summary

| Message | Direction | Type | Purpose |
|---------|-----------|------|---------|
| initialize | Client → Agent | Request | Protocol negotiation |
| authenticate | Client → Agent | Request | Auth (stub) |
| session/new | Client → Agent | Request | Create session |
| session/load | Client → Agent | Request | Load session |
| session/prompt | Client → Agent | Request | Send input |
| session/cancel | Client → Agent | Notification | Cancel operation |
| setSessionMode | Client → Agent | Request | Change mode |
| unstable_* | Client → Agent | Request | Experimental features |
| requestPermission | Agent → Client | Request | Ask approval |
| session/update | Agent → Client | Notification | Progress updates |

---

## References

- [Agent Client Protocol Spec](https://agentclientprotocol.com/)
- OpenCode source: `src/acp/agent.ts`
- Session management: `src/session/index.ts`
- Task tool: `src/tool/task.ts`
- Copilot plugin: `src/plugin/copilot.ts`

---

## Appendix: Test Results

Test script: `research/acp-subagent-test.ts`

### Observed Subagent Lifecycle via ACP

**1. tool_call (pending)**
```json
{
  "toolCallId": "call_e11d74682ace4ddaaa97e1c0",
  "title": "task",
  "kind": "other",
  "status": "pending",
  "rawInput": {}
}
```

**2. tool_call_update (in_progress)** - first update includes rawInput
```json
{
  "toolCallId": "call_e11d74682ace4ddaaa97e1c0",
  "status": "in_progress",
  "subagent_type": "general",
  "description": "Read go.mod module name"
}
```

**3. tool_call_update (in_progress)** - multiple updates as subagent works

**4. tool_call_update (completed)** - final update with results
```json
{
  "toolCallId": "call_e11d74682ace4ddaaa97e1c0",
  "status": "completed",
  "subagent_type": "general",
  "rawOutput": {
    "output": "\ngithub.com/mark3labs/iteratr\n\n<task_metadata>\nsession_id: ses_401a7c450ffeiS9TVG427W6P52\n</task_metadata>",
    "metadata": {
      "summary": [
        {
          "id": "prt_bfe584ecd001w05mjmer8CRVuB",
          "tool": "read",
          "state": {
            "status": "completed",
            "title": "go.mod"
          }
        }
      ],
      "sessionId": "ses_401a7c450ffeiS9TVG427W6P52",
      "model": {
        "modelID": "big-pickle",
        "providerID": "opencode"
      },
      "truncated": false
    }
  },
  "content": [
    {
      "type": "content",
      "content": {
        "type": "text",
        "text": "\ngithub.com/mark3labs/iteratr\n\n<task_metadata>\nsession_id: ses_401a7c450ffeiS9TVG427W6P52\n</task_metadata>"
      }
    }
  ]
}
```

### Key Observations

1. **No dedicated subagent notification** - Subagent spawning is just a `tool_call` for the `task` tool
2. **rawInput contains subagent info** - `subagent_type`, `description`, `prompt` visible in first `in_progress` update
3. **Multiple in_progress updates** - Sent as subagent performs work (each tool call in subagent triggers an update)
4. **Session ID in metadata** - On completion, `rawOutput.metadata.sessionId` contains the spawned child session
5. **Session ID in rawOutput** - Also available in `<task_metadata>` block within the output string
6. **Tool summary available** - `rawOutput.metadata.summary` lists all tools the subagent invoked
7. **Model info available** - `rawOutput.metadata.model` shows which model ran the subagent
8. **No parent/child relationship exposed** - Can't query child sessions or subscribe to their updates via ACP
