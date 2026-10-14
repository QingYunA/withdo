import type { McpServer } from "@modelcontextprotocol/server";
import * as z from "zod";
import { DomainError } from "../../domain/errors.ts";
import { formatTaskLine } from "../../domain/format.ts";
import { resolveProjectFromCwd } from "../../domain/projects.ts";
import { listTasks } from "../../domain/tasks.ts";
import { cwdSchema, milestoneId, projectId, statusSchema } from "../schemas.ts";
import { guarded, ok, type ToolContext } from "./helpers.ts";

export const listTasksInput = z.object({
  project_id: projectId.optional(),
  cwd: cwdSchema.optional().describe("List tasks of the project that owns this directory"),
  status: z.array(statusSchema).optional().describe("Only tasks in these statuses"),
  milestone_id: milestoneId.optional(),
  limit: z.number().int().min(1).max(200).optional().describe("Default 50"),
});

export function registerListTasks(server: McpServer, ctx: ToolContext): void {
  server.registerTool(
    "list_tasks",
    {
      title: "List tasks",
      description: "List tasks, newest first, optionally filtered by project, status or milestone.",
      inputSchema: listTasksInput,
    },
    guarded(({ cwd, project_id, ...rest }) => {
      if (cwd !== undefined && project_id !== undefined) {
        throw new DomainError("provide either cwd or project_id, not both", "cwd");
      }
      let scope = project_id;
      if (cwd !== undefined) {
        const resolved = resolveProjectFromCwd(ctx.db, cwd);
        if (!resolved) return ok("No tasks (cwd is not in a git repository).", { tasks: [] });
        scope = resolved.project.id;
      }
      const tasks = listTasks(ctx.db, { ...rest, project_id: scope });
      const text = tasks.length ? tasks.map(formatTaskLine).join("\n") : "No tasks.";
      return ok(text, { tasks });
    }),
  );
}
