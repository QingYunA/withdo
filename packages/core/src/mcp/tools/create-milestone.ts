import type { McpServer } from "@modelcontextprotocol/server";
import * as z from "zod";
import { DomainError } from "../../domain/errors.ts";
import { createMilestone } from "../../domain/milestones.ts";
import { resolveProjectId } from "../../domain/projects.ts";
import { cwdSchema, projectId, timestamp } from "../schemas.ts";
import { guarded, ok, type ToolContext } from "./helpers.ts";

export const createMilestoneInput = z.object({
  title: z.string().min(1),
  cwd: cwdSchema.optional().describe("Directory whose repo owns the milestone; defaults to the server's cwd"),
  project_id: projectId.optional(),
  due_at: timestamp.optional().describe("ISO 8601 date or datetime"),
});

export function registerCreateMilestone(server: McpServer, ctx: ToolContext): void {
  server.registerTool(
    "create_milestone",
    {
      title: "Create milestone",
      description: "Create a milestone under a project (resolved from cwd or project_id).",
      inputSchema: createMilestoneInput,
    },
    guarded(({ title, cwd, project_id, due_at }) => {
      if (cwd !== undefined && project_id !== undefined) {
        throw new DomainError("provide either cwd or project_id, not both", "cwd");
      }
      const resolved = resolveProjectId(ctx.db, {
        project_id,
        cwd: project_id === undefined ? (cwd ?? ctx.defaultCwd) : undefined,
      });
      if (resolved === null) throw new DomainError("not inside a git repository; pass project_id", "cwd");
      const milestone = createMilestone(ctx.db, { project_id: resolved, title, due_at });
      return ok(`Milestone #${milestone.id} ${milestone.title}`, { milestone });
    }),
  );
}
