import type { Route } from "../router.ts";
import { milestoneRoutes } from "./milestones.ts";
import { overviewRoutes } from "./overview.ts";
import { projectRoutes } from "./projects.ts";
import { taskRoutes } from "./tasks.ts";

export const ALL_ROUTES: readonly Route[] = [
  ...overviewRoutes,
  ...projectRoutes,
  ...taskRoutes,
  ...milestoneRoutes,
];
