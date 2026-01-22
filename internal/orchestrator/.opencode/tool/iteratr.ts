
import { tool } from "@opencode-ai/plugin"

// Data directory path - resolved at tool generation time
const DATA_DIR = "/tmp/TestGracefulShutdown4023440278/001/.iteratr"
const SESSION = "test-shutdown"

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

export const task_add = tool({
  description: "Add a new task to the iteratr session task list",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
    content: tool.schema.string().describe("Task description"),
    status: tool.schema.enum(["remaining", "blocked"]).default("remaining").describe("Initial status"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    
    try {
      const result = await withRetry(async () => {
        const proc = Bun.spawn(["iteratr", "tool", "task-add", "--name", session, "--content", args.content, "--status", args.status], {
          env: { ITERATR_DATA_DIR: DATA_DIR },
          stdout: "pipe",
          stderr: "pipe"
        })
        const output = await new Response(proc.stdout).text()
        const exitCode = await proc.exited
        if (exitCode !== 0) {
          const errOutput = await new Response(proc.stderr).text()
          throw new Error(errOutput || "task_add failed")
        }
        return output.trim()
      })
      return result
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
    
    try {
      await withRetry(async () => {
        const proc = Bun.spawn(["iteratr", "tool", "task-status", "--name", session, "--id", args.id, "--status", args.status], {
          env: { ITERATR_DATA_DIR: DATA_DIR },
          stdout: "pipe",
          stderr: "pipe"
        })
        const exitCode = await proc.exited
        if (exitCode !== 0) {
          const errOutput = await new Response(proc.stderr).text()
          throw new Error(errOutput || "task_status failed")
        }
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
    
    try {
      const proc = Bun.spawn(["iteratr", "tool", "task-list", "--name", session], {
        env: { ITERATR_DATA_DIR: DATA_DIR },
        stdout: "pipe",
        stderr: "pipe"
      })
      const result = await new Response(proc.stdout).text()
      const exitCode = await proc.exited
      if (exitCode !== 0) {
        const errOutput = await new Response(proc.stderr).text()
        throw new Error(errOutput || "task_list failed")
      }
      return result.trim() || "No tasks yet"
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
  },
})

export const session_complete = tool({
  description: "Signal that ALL tasks are complete and terminate the iteratr session. Only call when every task is done.",
  args: {
    session_name: tool.schema.string().describe("Session name from Context section"),
  },
  async execute(args) {
    const session = args.session_name
    if (!session?.trim()) return "ERROR: session_name required"
    
    try {
      const proc = Bun.spawn(["iteratr", "tool", "session-complete", "--name", session], {
        env: { ITERATR_DATA_DIR: DATA_DIR },
        stdout: "pipe",
        stderr: "pipe"
      })
      const result = await new Response(proc.stdout).text()
      const exitCode = await proc.exited
      if (exitCode !== 0) {
        const errOutput = await new Response(proc.stderr).text()
        throw new Error(errOutput || "session_complete failed")
      }
      return `Session "${session}" marked complete`
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
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
      await withRetry(async () => {
        const proc = Bun.spawn(["iteratr", "tool", "note-add", "--name", session, "--content", args.content, "--type", args.type], {
          env: { ITERATR_DATA_DIR: DATA_DIR },
          stdout: "pipe",
          stderr: "pipe"
        })
        const exitCode = await proc.exited
        if (exitCode !== 0) {
          const errOutput = await new Response(proc.stderr).text()
          throw new Error(errOutput || "note_add failed")
        }
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
    
    try {
      const cmdArgs = ["iteratr", "tool", "note-list", "--name", session]
      if (args.type) {
        cmdArgs.push("--type", args.type)
      }
      
      const proc = Bun.spawn(cmdArgs, {
        env: { ITERATR_DATA_DIR: DATA_DIR },
        stdout: "pipe",
        stderr: "pipe"
      })
      const result = await new Response(proc.stdout).text()
      const exitCode = await proc.exited
      if (exitCode !== 0) {
        const errOutput = await new Response(proc.stderr).text()
        throw new Error(errOutput || "note_list failed")
      }
      return result.trim() || "No notes yet"
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
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
    
    try {
      const proc = Bun.spawn(["iteratr", "tool", "inbox-list", "--name", session], {
        env: { ITERATR_DATA_DIR: DATA_DIR },
        stdout: "pipe",
        stderr: "pipe"
      })
      const result = await new Response(proc.stdout).text()
      const exitCode = await proc.exited
      if (exitCode !== 0) {
        const errOutput = await new Response(proc.stderr).text()
        throw new Error(errOutput || "inbox_list failed")
      }
      return result.trim() || "No unread messages"
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
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
    
    try {
      await withRetry(async () => {
        const proc = Bun.spawn(["iteratr", "tool", "inbox-mark-read", "--name", session, "--id", args.id], {
          env: { ITERATR_DATA_DIR: DATA_DIR },
          stdout: "pipe",
          stderr: "pipe"
        })
        const exitCode = await proc.exited
        if (exitCode !== 0) {
          const errOutput = await new Response(proc.stderr).text()
          throw new Error(errOutput || "inbox_mark_read failed")
        }
      })
      return `Message ${args.id} marked as read`
    } catch (e) {
      return `ERROR: ${(e as Error).message}`
    }
  },
})
