import type { McpServer } from "@modelcontextprotocol/server";
import { registerCreateMilestone } from "./create-milestone.ts";
import { registerCreateTask } from "./create-task.ts";
import { registerGetOverview } from "./get-overview.ts";
import { registerGetTask } from "./get-task.ts";
import type { ToolContext } from "./helpers.ts";
import { registerListTasks } from "./list-tasks.ts";
import { registerReportProgress } from "./report-progress.ts";
import { registerUpdateTask } from "./update-task.ts";
import { registerUpsertProject } from "./upsert-project.ts";

export function registerAllTools(server: McpServer, ctx: ToolContext): void {
  registerReportProgress(server, ctx);
  registerCreateTask(server, ctx);
  registerUpdateTask(server, ctx);
  registerGetTask(server, ctx);
  registerListTasks(server, ctx);
  registerGetOverview(server, ctx);
  registerUpsertProject(server, ctx);
  registerCreateMilestone(server, ctx);
}
