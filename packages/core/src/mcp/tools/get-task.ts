import type { McpServer } from "@modelcontextprotocol/server";
import * as z from "zod";
import { DomainError } from "../../domain/errors.ts";
import { formatTaskLine } from "../../domain/format.ts";
import { listReports } from "../../domain/reports.ts";
import { getTask } from "../../domain/tasks.ts";
import { taskId } from "../schemas.ts";
import { guarded, ok, type ToolContext } from "./helpers.ts";

export function registerGetTask(server: McpServer, ctx: ToolContext): void {
  server.registerTool(
    "get_task",
    {
      title: "Get task",
      description: "Get a task together with its progress report timeline (oldest first).",
      inputSchema: z.object({ task_id: taskId }),
    },
    guarded(({ task_id }) => {
      const task = getTask(ctx.db, task_id);
      if (!task) throw new DomainError(`task ${task_id} not found`, "task_id");
      const reports = listReports(ctx.db, task_id);
      const lines = reports.map((r) => `- ${r.created_at} ${r.agent}: ${r.summary}`);
      return ok([formatTaskLine(task), ...lines].join("\n"), { task, reports });
    }),
  );
}
