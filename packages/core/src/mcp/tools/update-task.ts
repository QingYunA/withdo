import type { McpServer } from "@modelcontextprotocol/server";
import * as z from "zod";
import { formatTaskLine } from "../../domain/format.ts";
import { updateTask } from "../../domain/tasks.ts";
import { milestoneId, prioritySchema, projectId, prUrl, statusSchema, taskId, timestamp } from "../schemas.ts";
import { guarded, ok, type ToolContext } from "./helpers.ts";

export const updateTaskInput = z.object({
  task_id: taskId,
  title: z.string().min(1).optional(),
  notes: z.string().nullable().optional().describe("null clears the notes"),
  status: statusSchema.optional(),
  priority: prioritySchema.optional(),
  project_id: projectId.nullable().optional().describe("null moves the task to the inbox"),
  milestone_id: milestoneId.nullable().optional().describe("null detaches the milestone"),
  due_at: timestamp.nullable().optional().describe("null clears the due date"),
  remind_at: timestamp.nullable().optional().describe("null clears the reminder"),
  branch: z.string().min(1).nullable().optional(),
  pr_url: prUrl.nullable().optional().describe("null clears the PR; pr_number is derived from the URL"),
});

export function registerUpdateTask(server: McpServer, ctx: ToolContext): void {
  server.registerTool(
    "update_task",
    {
      title: "Update task",
      description: "Patch fields of a task. Omitted fields are unchanged; null clears nullable fields.",
      inputSchema: updateTaskInput,
    },
    guarded(({ task_id, ...patch }) => {
      const task = updateTask(ctx.db, task_id, patch);
      return ok(`Updated ${formatTaskLine(task)}`, { task });
    }),
  );
}
