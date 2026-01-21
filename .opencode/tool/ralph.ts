
import { tool } from "@opencode-ai/plugin"
import { resolve, dirname } from "path"

// Resolve absolute path to store - tools run from .opencode/tool/ so we go up 2 levels
const STORE = resolve(dirname(import.meta.path), "../../.ralph/store")

// Retry helper with exponential backoff
async function withRetry<T>(fn: () => Promise<T>, maxRetries = 3): Promise<T> {
  let lastError: Error | null = null
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn()
    } catch (e) {
      lastError = e as Error
      if (i < maxRetries - 1) {
        await new Promise(r => setTimeout(r, 100 * Math.pow(2, i)))
      }
    }
  }
  throw lastError
}

// Get current iteration from store
async function getCurrentIteration(session: string): Promise<number> {
  const cmd = `xs cat ${STORE} | from json --objects | where topic == "ralph.${session}.iteration" | where {|f| $f.meta.action? == "start"} | last | get meta.n`
  try {
    const result = await Bun.$`nu -c ${cmd}`.text()
    return parseInt(result.trim()) || 1
  } catch {
    return 1
  }
}

export const task_add = tool({
  description: "Add a new task to the ralph session task list",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
    content: tool.schema.string().describe("Task description"),
    status: tool.schema.enum(["remaining", "blocked"]).default("remaining").describe("Initial status"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const topic = `ralph.${session}.task`
    try {
      const meta = JSON.stringify({ action: "add", status: args.status })
      const result = await withRetry(async () => {
        return await Bun.$`echo ${args.content} | xs append ${STORE} ${topic} --meta ${meta}`.text()
      })
      return result.trim()
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
  },
})

export const task_status = tool({
  description: "Update a task status by ID. Use IDs from task_list output.",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
    id: tool.schema.string().describe("Task ID (full or 8+ char prefix)"),
    status: tool.schema.enum(["in_progress", "completed", "blocked"]).describe("New status"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const topic = `ralph.${session}.task`
    try {
      const iteration = await getCurrentIteration(session)
      const meta = JSON.stringify({ action: "status", id: args.id, status: args.status, iteration })
      await withRetry(async () => {
        await Bun.$`xs append ${STORE} ${topic} --meta ${meta}`.text()
      })
      return `Task ${args.id} marked as ${args.status}`
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
  },
})

export const task_list = tool({
  description: "Get current task list grouped by status. Shows task IDs needed for task_status.",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const topic = `ralph.${session}.task`
    const cmd = `
      let topic = "${topic}"
      let frames = (xs cat ${STORE} | from json --objects | where topic == $topic)
      
      if ($frames | is-empty) {
        "No tasks yet"
      } else {
        let tasks = ($frames | reduce -f {} {|frame, state|
          let action = ($frame.meta.action? | default "add")
          
          if $action == "add" {
            let content = (xs cas ${STORE} $frame.hash)
            let status = ($frame.meta.status? | default "remaining")
            let iteration = ($frame.meta.iteration? | default null)
            $state | upsert $frame.id {
              id: $frame.id
              content: $content
              status: $status
              iteration: $iteration
            }
          } else if $action == "status" {
            let target_id = $frame.meta.id
            let new_status = $frame.meta.status
            let iteration = ($frame.meta.iteration? | default null)
            # Find task by exact match or prefix (8+ chars)
            let matching_id = ($state | columns | where {|id| $id == $target_id or ($id | str starts-with $target_id)} | first | default null)
            if ($matching_id | is-not-empty) {
              $state | upsert $matching_id {|task|
                $task | get $matching_id | upsert status $new_status | upsert iteration $iteration
              }
            } else {
              $state
            }
          } else {
            $state
          }
        })
        
        let task_list = ($tasks | values)
        let grouped = if ($task_list | is-empty) { {} } else { $task_list | group-by status }
        
        {
          completed: ($grouped | get -o completed | default [])
          in_progress: ($grouped | get -o in_progress | default [])
          blocked: ($grouped | get -o blocked | default [])
          remaining: ($grouped | get -o remaining | default [])
        } | to json
      }
    `
    const result = await Bun.$`nu -c ${cmd}`.text()
    return result.trim() || "No tasks yet"
  },
})

export const session_complete = tool({
  description: "Signal that ALL tasks are complete and terminate the ralph session. Only call when every task is done.",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    
    try {
      // Write timestamp-based marker - ralph.nu will check for any session_complete frame
      const ts = Date.now()
      const meta = JSON.stringify({ action: "session_complete", ts })
      const topic = `ralph.${session}.control`
      const result = await withRetry(async () => {
        return await Bun.$`echo "complete" | xs append ${STORE} ${topic} --meta ${meta}`.text()
      })
      return `Session "${session}" marked complete (ts=${ts}, store=${STORE}, result=${result.slice(0,50)}...)`
    } catch (e) {
      return `ERROR: ${(e as Error).message} (store=${STORE})`
    }
  },
})

export const note_add = tool({
  description: "Add a note for future iterations (learnings, tips, blockers, decisions)",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
    content: tool.schema.string().describe("Note content"),
    type: tool.schema.enum(["learning", "stuck", "tip", "decision"]).describe("Note category"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    try {
      const iteration = await getCurrentIteration(session)
      const meta = JSON.stringify({ action: "add", type: args.type, iteration })
      const topic = `ralph.${session}.note`
      await withRetry(async () => {
        await Bun.$`echo ${args.content} | xs append ${STORE} ${topic} --meta ${meta}`.text()
      })
      const preview = args.content.length > 50 ? args.content.slice(0, 50) + "..." : args.content
      return `Note added: [${args.type}] ${preview}`
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
  },
})

export const note_list = tool({
  description: "List notes from this session",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
    type: tool.schema.enum(["learning", "stuck", "tip", "decision"]).optional().describe("Filter by type"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const typeFilter = args.type ? `| where {|n| $n.type == "${args.type}"}` : ""
    const topic = `ralph.${session}.note`
    const cmd = `
      xs cat ${STORE} | from json --objects | where topic == "${topic}" | each {|f|
        {
          id: $f.id
          type: ($f.meta.type? | default "note")
          iteration: ($f.meta.iteration? | default null)
          content: (xs cas ${STORE} $f.hash)
        }
      } ${typeFilter} | to json
    `
    const result = await Bun.$`nu -c ${cmd}`.text()
    return result.trim() || "No notes yet"
  },
})

export const inbox_list = tool({
  description: "Get unread inbox messages. Check this at start of each iteration.",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const topic = `ralph.${session}.inbox`
    const cmd = `
      let topic = "${topic}"
      let frames = (xs cat ${STORE} | from json --objects | where topic == $topic)
      
      if ($frames | is-empty) {
        "No unread messages"
      } else {
        let messages = ($frames | reduce -f {} {|frame, state|
          let action = ($frame.meta.action? | default "add")
          
          if $action == "mark_read" {
            let target_id = $frame.meta.id
            let matching_id = ($state | columns | where {|id| $id == $target_id or ($id | str starts-with $target_id)} | first | default null)
            if ($matching_id | is-not-empty) {
              $state | upsert $matching_id {|m|
                $m | get $matching_id | upsert status "read"
              }
            } else {
              $state
            }
          } else if $action == "add" or ($frame.meta.status? == "unread") {
            let content = (xs cas ${STORE} $frame.hash)
            $state | upsert $frame.id {
              id: $frame.id
              content: $content
              status: "unread"
              timestamp: ($frame.meta.timestamp? | default "")
            }
          } else {
            $state
          }
        })
        
        let unread = ($messages | values | where status == "unread")
        if ($unread | is-empty) {
          "No unread messages"
        } else {
          $unread | to json
        }
      }
    `
    const result = await Bun.$`nu -c ${cmd}`.text()
    return result.trim() || "No unread messages"
  },
})

export const inbox_mark_read = tool({
  description: "Mark an inbox message as read after processing",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
    id: tool.schema.string().describe("Message ID from inbox_list"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    const topic = `ralph.${session}.inbox`
    try {
      const meta = JSON.stringify({ action: "mark_read", id: args.id })
      await withRetry(async () => {
        await Bun.$`xs append ${STORE} ${topic} --meta ${meta}`.text()
      })
      return `Message ${args.id} marked as read`
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
  },
})
