import type { McpServer } from "@modelcontextprotocol/server";
import * as z from "zod";
import { upsertProject } from "../../domain/projects.ts";
import { cwdSchema } from "../schemas.ts";
import { guarded, ok, type ToolContext } from "./helpers.ts";

const input = z.object({
  cwd: cwdSchema.optional().describe("A directory inside the git repo (use this or repo_slug)"),
  repo_slug: z.string().min(1).optional().describe("GitHub owner/repo (use this or cwd)"),
  name: z.string().min(1).optional().describe("Display name; renames an existing project"),
});

export function registerUpsertProject(server: McpServer, ctx: ToolContext): void {
  server.registerTool(
    "upsert_project",
    {
      title: "Create or rename project",
      description: "Create the project for a repo (by cwd or repo_slug), or rename it if it already exists.",
      inputSchema: input,
    },
    guarded((args) => {
      const project = upsertProject(ctx.db, args);
      return ok(`Project #${project.id} ${project.name}${project.repo_slug ? ` (${project.repo_slug})` : ""}`, {
        project,
      });
    }),
  );
}
