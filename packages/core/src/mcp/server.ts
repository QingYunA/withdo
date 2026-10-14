import type { Database } from "bun:sqlite";
import { McpServer } from "@modelcontextprotocol/server";
import { StdioServerTransport } from "@modelcontextprotocol/server/stdio";
import { registerAllTools } from "./tools/index.ts";

export const INSTRUCTIONS = [
  "adhd-todo tracks what coding agents are doing across repos, branches and pull requests, so the user can see everything in one place.",
  "Call report_progress:",
  "- when you start a task (summary = the goal; status \"doing\"),",
  "- when you get blocked (status \"blocked\"; say in next_step what you need),",
  "- when you open a PR (pr_url, status \"review\"),",
  "- when you finish (status \"done\").",
  "Calls from the same repo and branch update one task, so keep each report short. Pass task_id if you have it.",
  "Do not report trivial questions or read-only exploration.",
].join("\n");

export interface ServerOptions {
  /** Directory used when a tool call has no `cwd`; defaults to the process cwd. */
  readonly defaultCwd?: string;
}

export function createServer(db: Database, options: ServerOptions = {}): McpServer {
  const server = new McpServer({ name: "adhd-todo", version: "0.1.0" }, { instructions: INSTRUCTIONS });
  registerAllTools(server, {
    db,
    defaultCwd: options.defaultCwd ?? process.cwd(),
    agentName: () => server.server.getClientVersion()?.name ?? "agent",
  });
  return server;
}

/** Serve MCP over stdio. stdout belongs to the transport; diagnostics go to stderr. */
export async function runStdioServer(db: Database): Promise<void> {
  const server = createServer(db);
  const transport = new StdioServerTransport();
  server.server.onclose = () => {
    db.close();
    process.exit(0);
  };
  await server.connect(transport);
  process.stderr.write("[adhd-todo] MCP server ready on stdio\n");
}
