#!/usr/bin/env bun
/**
 * Test script to observe file modification tracking via ACP
 * Usage: bun research/acp-file-tracking-test.ts
 * 
 * This script prompts the agent to create and edit files, then logs
 * all tool_call messages to see what metadata is surfaced for file operations.
 */

import { spawn } from "bun";
import { rmSync } from "fs";

// JSON-RPC message ID counter
let messageId = 0;

// Track all file-related tool calls
const fileOperations: Array<{
  toolCallId: string;
  tool: string;
  kind: string;
  status: string;
  locations: any[];
  rawInput: any;
  rawOutput: any;
  content: any[];
  timestamp: string;
}> = [];

function createRequest(method: string, params: any = {}) {
  return {
    jsonrpc: "2.0",
    id: ++messageId,
    method,
    params,
  };
}

function log(direction: ">>>" | "<<<" | "---", label: string, data?: any) {
  const timestamp = new Date().toISOString().split("T")[1].slice(0, -1);
  console.log(`\n[${timestamp}] ${direction} ${label}`);
  if (data !== undefined) {
    console.log(JSON.stringify(data, null, 2));
  }
}

function isFileModifyingTool(toolName: string, kind?: string): boolean {
  const editTools = ["edit", "write", "patch", "multiedit"];
  const editKinds = ["edit"];
  return editTools.includes(toolName?.toLowerCase()) || editKinds.includes(kind?.toLowerCase());
}

async function main() {
  // Clean up any leftover test files
  const testFiles = [
    "research/test-file-1.txt",
    "research/test-file-2.txt",
  ];
  for (const f of testFiles) {
    try { rmSync(f); } catch {}
  }

  log("---", "Starting opencode acp subprocess");

  const proc = spawn(["opencode", "acp"], {
    stdin: "pipe",
    stdout: "pipe",
    stderr: "inherit",
    cwd: process.cwd(),
  });

  const writer = proc.stdin;
  const reader = proc.stdout;

  // Pending requests waiting for response
  const pending = new Map<number, { resolve: Function; reject: Function }>();

  // Buffer for incoming data
  let buffer = "";

  // Send a JSON-RPC request and wait for response
  async function sendRequest(method: string, params: any = {}): Promise<any> {
    const request = createRequest(method, params);
    const message = JSON.stringify(request);

    log(">>>", `REQUEST: ${method}`, request);

    writer.write(message + "\n");
    writer.flush();

    return new Promise((resolve, reject) => {
      pending.set(request.id, { resolve, reject });
      setTimeout(() => {
        if (pending.has(request.id)) {
          pending.delete(request.id);
          reject(new Error(`Timeout waiting for response to ${method}`));
        }
      }, 180000);
    });
  }

  // Process incoming messages
  async function processMessages() {
    const decoder = new TextDecoder();

    for await (const chunk of reader) {
      buffer += decoder.decode(chunk);

      let newlineIdx;
      while ((newlineIdx = buffer.indexOf("\n")) !== -1) {
        const line = buffer.slice(0, newlineIdx).trim();
        buffer = buffer.slice(newlineIdx + 1);

        if (!line) continue;

        try {
          const msg = JSON.parse(line);

          if (msg.id !== undefined && pending.has(msg.id)) {
            log("<<<", `RESPONSE (id=${msg.id})`, msg);
            const { resolve, reject } = pending.get(msg.id)!;
            pending.delete(msg.id);
            if (msg.error) {
              reject(new Error(`RPC Error: ${msg.error.message}`));
            } else {
              resolve(msg.result);
            }
          } else if (msg.method) {
            const update = msg.params?.update;
            const isToolCall =
              update?.sessionUpdate === "tool_call" ||
              update?.sessionUpdate === "tool_call_update";

            const toolName = update?.title || msg.params?.toolCall?.title;
            const toolKind = update?.kind;

            if (isToolCall) {
              // Log ALL tool calls with full details for file-modifying tools
              const isFileTool = isFileModifyingTool(toolName, toolKind);
              
              if (isFileTool || toolName === "read") {
                log("<<<", `*** FILE TOOL [${update?.status}]: ${toolName} (kind: ${toolKind}) ***`, {
                  toolCallId: update?.toolCallId,
                  status: update?.status,
                  kind: update?.kind,
                  title: update?.title,
                  locations: update?.locations,
                  rawInput: update?.rawInput,
                  rawOutput: update?.rawOutput,
                  content: update?.content,
                  metadata: update?.metadata,
                });

                // Track completed file operations
                if (update?.status === "completed" && isFileTool) {
                  fileOperations.push({
                    toolCallId: update.toolCallId,
                    tool: toolName,
                    kind: toolKind,
                    status: update.status,
                    locations: update.locations || [],
                    rawInput: update.rawInput || {},
                    rawOutput: update.rawOutput || {},
                    content: update.content || [],
                    timestamp: new Date().toISOString(),
                  });
                }
              } else {
                // Brief log for other tools
                log("<<<", `TOOL [${update?.status}]: ${toolName}`, {
                  toolCallId: update?.toolCallId,
                  kind: update?.kind,
                });
              }
            } else if (msg.method === "session/update") {
              if (update?.sessionUpdate === "agent_message_chunk") {
                const text = update.content?.text?.slice(0, 100);
                log("<<<", `AGENT: "${text}${text?.length >= 100 ? "..." : ""}"`);
              } else if (update?.sessionUpdate === "agent_thought_chunk") {
                // Skip thought chunks for brevity
              } else if (update?.sessionUpdate === "plan") {
                log("<<<", `PLAN UPDATE`, update.entries);
              } else {
                log("<<<", `SESSION: ${update?.sessionUpdate}`);
              }
            } else if (msg.method === "session/request_permission") {
              log("<<<", `PERMISSION REQUEST`, msg.params);
              // Auto-allow for testing
              const response = {
                jsonrpc: "2.0",
                id: msg.id,
                result: { outcome: { outcome: "selected", optionId: "once" } },
              };
              log(">>>", "AUTO-ALLOW", response);
              writer.write(JSON.stringify(response) + "\n");
              writer.flush();
            }
          }
        } catch (e) {
          log("---", `PARSE ERROR: ${e}`, line);
        }
      }
    }
  }

  // Start processing messages in background
  const messageProcessor = processMessages();

  try {
    // Step 1: Initialize
    log("---", "Step 1: Initialize");
    await sendRequest("initialize", {
      protocolVersion: 1,
      clientCapabilities: {
        fs: { readTextFile: true, writeTextFile: true },
        terminal: true,
      },
      clientInfo: {
        name: "acp-file-tracking-test",
        version: "0.0.1",
      },
    });

    // Step 2: Create new session
    log("---", "Step 2: Create session");
    const sessionResult = await sendRequest("session/new", {
      cwd: process.cwd(),
      mcpServers: [],
    });

    const sessionId = sessionResult.sessionId;
    log("---", `Session created: ${sessionId}`);

    // Step 3: Send prompt to create and edit files
    log("---", "Step 3: Sending prompt to create and edit files");
    const promptResult = await sendRequest("session/prompt", {
      sessionId,
      prompt: [
        {
          type: "text",
          text: `Do these tasks in order:
1. Create a new file at "research/test-file-1.txt" with the content "Hello World - Line 1"
2. Read the file you just created to verify it exists
3. Edit that file to add a second line "Hello World - Line 2"
4. Create another file at "research/test-file-2.txt" with content "Second file"

Use the Write tool to create files and the Edit tool to modify them. Be brief in your responses.`,
        },
      ],
    });

    log("---", "Prompt completed", promptResult);

    // Wait for trailing notifications
    await Bun.sleep(2000);

    // Print summary of file operations
    log("---", "=== FILE OPERATIONS SUMMARY ===");
    console.log(`\nTotal file-modifying operations: ${fileOperations.length}\n`);
    
    for (const op of fileOperations) {
      console.log(`\n--- ${op.tool.toUpperCase()} (${op.kind}) ---`);
      console.log(`Tool Call ID: ${op.toolCallId}`);
      console.log(`Status: ${op.status}`);
      
      // Extract file paths
      const paths: string[] = [];
      if (op.rawInput?.filePath) paths.push(op.rawInput.filePath);
      if (op.rawInput?.path) paths.push(op.rawInput.path);
      for (const loc of op.locations) {
        if (loc.path) paths.push(loc.path);
      }
      for (const c of op.content) {
        if (c.type === "diff" && c.path) paths.push(c.path);
      }
      console.log(`File paths: ${[...new Set(paths)].join(", ") || "(none found)"}`);
      
      // Check for diff content
      const diffs = op.content.filter((c: any) => c.type === "diff");
      if (diffs.length > 0) {
        console.log(`Diffs found: ${diffs.length}`);
        for (const d of diffs) {
          console.log(`  - ${d.path}: oldText=${d.oldText?.length || 0} chars, newText=${d.newText?.length || 0} chars`);
        }
      }
      
      // Check rawOutput for metadata
      if (op.rawOutput?.metadata) {
        console.log(`Metadata:`, JSON.stringify(op.rawOutput.metadata, null, 2));
      }
    }

  } catch (error) {
    log("---", "ERROR", error);
  } finally {
    log("---", "Shutting down");
    proc.kill();
    
    // Cleanup test files
    for (const f of testFiles) {
      try { rmSync(f); } catch {}
    }
  }
}

main().catch(console.error);
