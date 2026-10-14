import type { Database } from "bun:sqlite";
import type { CallToolResult } from "@modelcontextprotocol/server";
import { DomainError } from "../../domain/errors.ts";

export interface ToolContext {
  readonly db: Database;
  /** Directory used when a tool is called without `cwd` (the server process cwd). */
  readonly defaultCwd: string;
  /** Name of the connected MCP client, e.g. "claude-code"; "agent" until known. */
  readonly agentName: () => string;
}

export function ok(text: string, data: Record<string, unknown>): CallToolResult {
  return { content: [{ type: "text", text }], structuredContent: data };
}

export function fail(message: string): CallToolResult {
  return { content: [{ type: "text", text: message }], isError: true };
}

/** Wrap a handler so domain errors become `isError` results naming the bad field. */
export function guarded<A>(handler: (args: A) => CallToolResult): (args: A) => CallToolResult {
  return (args) => {
    try {
      return handler(args);
    } catch (error) {
      if (error instanceof DomainError) return fail(error.message);
      const message = error instanceof Error ? error.message : String(error);
      process.stderr.write(`[adhd-todo] tool error: ${message}\n`);
      return fail(`internal error: ${message}`);
    }
  };
}
