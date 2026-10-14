import type { McpServer } from "@modelcontextprotocol/server";
import * as z from "zod";
import { reportProgress } from "../../domain/reports.ts";
import { formatTaskLine } from "../../domain/format.ts";
import { cwdSchema, prUrl, statusSchema, taskId } from "../schemas.ts";
import { guarded, ok, type ToolContext } from "./helpers.ts";

const input = z.object({
  summary: z.string().min(1).describe("One or two sentences: what you are doing or just did"),
  status: statusSchema.optional().describe("New task status; omit to leave unchanged"),
  next_step: z.string().min(1).optional().describe("What happens next, or what you are waiting on"),
  task_id: taskId.optional().describe("Task to report on; omit to match by repo and branch"),
  title: z.string().min(1).optional().describe("Title for a newly created task; defaults to the branch name"),
  cwd: cwdSchema.optional().describe("Your current working directory. Always pass it: it decides the repo and branch"),
  branch: z.string().min(1).optional().describe("Branch name; defaults to the current git branch of cwd"),
  pr_url: prUrl.optional(),
});

export function registerReportProgress(server: McpServer, ctx: ToolContext): void {
  server.registerTool(
    "report_progress",
    {
      title: "Report progress",
      description:
        "Report what you are working on. Finds the open task for this repo and branch (or task_id), " +
        "creating it if needed, then records a progress report. Call when you start, get blocked, " +
        "open a PR, and finish.",
      inputSchema: input,
    },
    guarded((args) => {
      const result = reportProgress(ctx.db, {
        ...args,
        agent: ctx.agentName(),
        cwd: args.task_id === undefined ? (args.cwd ?? ctx.defaultCwd) : args.cwd,
      });
      const verb = result.created ? "Created" : "Updated";
      return ok(`${verb} ${formatTaskLine(result.task)}`, {
        task_id: result.task.id,
        status: result.task.status,
        created: result.created,
        task: result.task,
      });
    }),
  );
}
