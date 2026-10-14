import type { McpServer } from "@modelcontextprotocol/server";
import * as z from "zod";
import { formatOverview } from "../../domain/format.ts";
import { DEFAULT_STALE_HOURS, getOverview } from "../../domain/overview.ts";
import { guarded, ok, type ToolContext } from "./helpers.ts";

export const getOverviewInput = z.object({
  stale_hours: z.number().positive().optional().describe(`Hours without a report before a task is stale; default ${DEFAULT_STALE_HOURS}`),
});

export function registerGetOverview(server: McpServer, ctx: ToolContext): void {
  server.registerTool(
    "get_overview",
    {
      title: "Get overview",
      description:
        "Per project: tasks in doing, blocked or review, each with its latest report and a stale flag " +
        "when nothing was reported for stale_hours.",
      inputSchema: getOverviewInput,
    },
    guarded(({ stale_hours }) => {
      const overview = getOverview(ctx.db, { staleHours: stale_hours });
      return ok(formatOverview(overview), { ...overview });
    }),
  );
}
