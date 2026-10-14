import type { Database } from "bun:sqlite";
import { Client, InMemoryTransport } from "@modelcontextprotocol/client";
import { openMemoryDb } from "../../src/db/open.ts";
import { createServer } from "../../src/mcp/server.ts";

export interface TestRig {
  readonly client: Client;
  readonly db: Database;
  call(name: string, args?: Record<string, unknown>): Promise<ToolResult>;
  close(): Promise<void>;
}

export interface ToolResult {
  readonly isError?: boolean;
  readonly content: ReadonlyArray<{ type: string; text?: string }>;
  readonly structuredContent?: Record<string, any>;
}

export async function connect(defaultCwd: string): Promise<TestRig> {
  const db = openMemoryDb();
  const server = createServer(db, { defaultCwd });
  const client = new Client({ name: "test-agent", version: "1.0.0" });
  const [a, b] = InMemoryTransport.createLinkedPair();
  await Promise.all([server.connect(a), client.connect(b)]);
  return {
    client,
    db,
    call: async (name, args = {}) => (await client.callTool({ name, arguments: args })) as ToolResult,
    close: async () => {
      await client.close();
      await server.close();
    },
  };
}

export const textOf = (r: ToolResult): string => r.content.map((c) => c.text ?? "").join("\n");
