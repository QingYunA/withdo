// Smoke test: spawn an MCP server binary over stdio, list tools, make one report_progress call.
// Usage: bun run scripts/smoke.ts [path-to-adhd-core]   (default: dist/adhd-core)
import { Client } from "@modelcontextprotocol/client";
import { StdioClientTransport } from "@modelcontextprotocol/client/stdio";
import { resolve } from "node:path";

export const EXPECTED_TOOLS = [
  "create_milestone",
  "create_task",
  "get_overview",
  "get_task",
  "list_tasks",
  "report_progress",
  "update_task",
  "upsert_project",
] as const;

export interface SmokeResult {
  readonly tools: string[];
  readonly taskId: number;
}

export async function runSmoke(
  command: string,
  args: string[],
  env: Record<string, string>,
  cwd: string,
): Promise<SmokeResult> {
  const transport = new StdioClientTransport({ command, args, env, cwd, stderr: "inherit" });
  const client = new Client({ name: "adhd-smoke", version: "0.0.0" });
  await client.connect(transport);
  try {
    const tools = (await client.listTools()).tools.map((t) => t.name).sort();
    const missing = EXPECTED_TOOLS.filter((t) => !tools.includes(t));
    if (missing.length > 0) throw new Error(`missing tools: ${missing.join(", ")}`);

    const res = await client.callTool({
      name: "report_progress",
      arguments: { summary: "smoke test report", title: "Smoke test task", cwd },
    });
    if (res.isError) throw new Error(`report_progress failed: ${JSON.stringify(res.content)}`);
    const taskId = (res.structuredContent as { task_id?: number } | undefined)?.task_id;
    if (typeof taskId !== "number") throw new Error("report_progress returned no task_id");
    return { tools, taskId };
  } finally {
    await client.close();
  }
}

if (import.meta.main) {
  const bin = resolve(process.argv[2] ?? "dist/adhd-core");
  const db = process.env.ADHD_TODO_DB;
  if (!db) throw new Error("set ADHD_TODO_DB to a temp file so the smoke test never touches real data");
  const result = await runSmoke(bin, ["mcp"], { ADHD_TODO_DB: db, PATH: process.env.PATH ?? "" }, process.cwd());
  process.stdout.write(`smoke ok: ${result.tools.length} tools, report_progress -> task #${result.taskId}\n`);
}
