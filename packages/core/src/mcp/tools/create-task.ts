import type { McpServer } from "@modelcontextprotocol/server";
import * as z from "zod";
import { DomainError } from "../../domain/errors.ts";
import { formatTaskLine } from "../../domain/format.ts";
import { resolveProjectId } from "../../domain/projects.ts";
import { createTask } from "../../domain/tasks.ts";
import { cwdSchema, milestoneId, prioritySchema, projectId, prUrl, timestamp } from "../schemas.ts";
import { guarded, ok, type ToolContext } from "./helpers.ts";

export const createTaskInput = z.object({
  title: z.string().min(1).describe("Short task title"),
  cwd: cwdSchema.optional().describe("Working directory whose repo owns the task; defaults to the server's cwd"),
  project_id: projectId.optional().describe("Project to file the task under (instead of cwd)"),
  notes: z.string().optional(),
  priority: prioritySchema.optional().describe("P0 (urgent) to P3 (low); default P2"),
  milestone_id: milestoneId.optional(),
  due_at: timestamp.optional().describe("ISO 8601 date or datetime"),
  remind_at: timestamp.optional().describe("ISO 8601 date or datetime"),
  branch: z.string().min(1).optional(),
  pr_url: prUrl.optional(),
});

export function registerCreateTask(server: McpServer, ctx: ToolContext): void {
  server.registerTool(
    "create_task",
    {
      title: "Create task",
      description: "Create a task under a project (resolved from cwd or project_id).",
      inputSchema: createTaskInput,
    },
    guarded(({ cwd, project_id, ...fields }) => {
      if (cwd !== undefined && project_id !== undefined) {
        throw new DomainError("provide either cwd or project_id, not both", "cwd");
      }
      const resolved = resolveProjectId(ctx.db, { project_id, cwd: project_id === undefined ? (cwd ?? ctx.defaultCwd) : undefined });
      const task = createTask(ctx.db, { ...fields, project_id: resolved, source: "agent" });
      return ok(`Created ${formatTaskLine(task)}`, { task });
    }),
  );
}
