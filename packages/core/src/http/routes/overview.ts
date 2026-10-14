import { getOverviewInput } from "../../mcp/tools/get-overview.ts";
import { getOverview } from "../../domain/overview.ts";
import { jsonOk, parseInput } from "../respond.ts";
import type { Route } from "../router.ts";

export const overviewRoutes: readonly Route[] = [
  {
    method: "GET",
    path: "/overview",
    handler: ({ url, db }) => {
      const raw = url.searchParams.get("stale_hours");
      const { stale_hours } = parseInput(getOverviewInput.strict(), raw === null ? {} : { stale_hours: Number(raw) });
      return jsonOk(getOverview(db, { staleHours: stale_hours }));
    },
  },
];
