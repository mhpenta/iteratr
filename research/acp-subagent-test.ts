#!/usr/bin/env bun
/**
 * Throwaway test script to observe ACP subagent behavior
 * Usage: bun research/acp-subagent-test.ts
 */

import { spawn } from "bun";

const DEBUG = true;

// JSON-RPC message ID counter
let messageId = 0;

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

async function main() {
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
      // Timeout after 180s for long-running subagent tasks
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

      // Process complete lines
      let newlineIdx;
      while ((newlineIdx = buffer.indexOf("\n")) !== -1) {
        const line = buffer.slice(0, newlineIdx).trim();
        buffer = buffer.slice(newlineIdx + 1);

        if (!line) continue;

        try {
          const msg = JSON.parse(line);

          if (msg.id !== undefined && pending.has(msg.id)) {
            // Response to a request
            log("<<<", `RESPONSE (id=${msg.id})`, msg);
            const { resolve, reject } = pending.get(msg.id)!;
            pending.delete(msg.id);
            if (msg.error) {
              reject(new Error(`RPC Error: ${msg.error.message}`));
            } else {
              resolve(msg.result);
            }
          } else if (msg.method) {
            // Notification or request from agent
            const update = msg.params?.update;
            const isToolCall =
              update?.sessionUpdate === "tool_call" ||
              update?.sessionUpdate === "tool_call_update";

            const toolName = update?.title || msg.params?.toolCall?.title;
            const isTaskTool = toolName === "task" || 
              update?.rawInput?.subagent_type !== undefined;

            // Check for completed subagent with metadata
            const isCompletedSubagent = isTaskTool && update?.status === "completed";

            if (isCompletedSubagent) {
              log("<<<", `*** SUBAGENT COMPLETED ***`, {
                toolCallId: update.toolCallId,
                status: update.status,
                subagent_type: update.rawInput?.subagent_type,
                metadata: update.metadata,
                rawOutput: update.rawOutput,
                content: update.content,
              });
            } else if (isTaskTool) {
              log("<<<", `*** SUBAGENT ${update?.status?.toUpperCase() || "DETECTED"} ***`, {
                toolCallId: update?.toolCallId,
                status: update?.status,
                subagent_type: update?.rawInput?.subagent_type,
                description: update?.rawInput?.description,
              });
            } else if (isToolCall) {
              log("<<<", `TOOL [${update?.status}]: ${toolName}`, {
                toolCallId: update?.toolCallId,
                rawInput: update?.rawInput,
              });
            } else if (msg.method === "session/update") {
              // Summarize session updates
              const update = msg.params?.update;
              if (update?.sessionUpdate === "agent_message_chunk") {
                const text = update.content?.text?.slice(0, 100);
                log("<<<", `AGENT CHUNK: "${text}${text?.length >= 100 ? "..." : ""}"`);
              } else if (update?.sessionUpdate === "agent_thought_chunk") {
                const text = update.content?.text?.slice(0, 80);
                log("<<<", `THOUGHT: "${text}${text?.length >= 80 ? "..." : ""}"`);
              } else if (update?.sessionUpdate === "plan") {
                log("<<<", `PLAN UPDATE`, update.entries);
              } else {
                log("<<<", `SESSION UPDATE: ${update?.sessionUpdate}`, update);
              }
            } else if (msg.method === "session/request_permission") {
              log("<<<", `PERMISSION REQUEST`, msg.params);
              // Auto-allow for testing
              const response = {
                jsonrpc: "2.0",
                id: msg.id,
                result: { outcome: { outcome: "selected", optionId: "once" } },
              };
              log(">>>", "AUTO-ALLOW PERMISSION", response);
              writer.write(JSON.stringify(response) + "\n");
              writer.flush();
            } else {
              log("<<<", `NOTIFICATION: ${msg.method}`, msg.params);
            }
          } else {
            log("<<<", "UNKNOWN MESSAGE", msg);
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
    const initResult = await sendRequest("initialize", {
      protocolVersion: 1,
      clientCapabilities: {
        fs: { readTextFile: true, writeTextFile: true },
        terminal: true,
      },
      clientInfo: {
        name: "acp-subagent-test",
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

    // Step 3: Send prompt that should trigger subagents
    log("---", "Step 3: Sending prompt to trigger subagents");
    const promptResult = await sendRequest("session/prompt", {
      sessionId,
      prompt: [
        {
          type: "text",
          text: `Use the Task tool with subagent_type "general" to read the file "go.mod" and tell me the module name. Keep it brief.`,
        },
      ],
    });

    log("---", "Prompt completed", promptResult);

    // Wait a bit for any trailing notifications
    await Bun.sleep(2000);

  } catch (error) {
    log("---", "ERROR", error);
  } finally {
    log("---", "Shutting down");
    proc.kill();
  }
}

main().catch(console.error);
