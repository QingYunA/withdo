import type { Route } from "../router.ts";
import { overviewRoutes } from "./overview.ts";

export const ALL_ROUTES: readonly Route[] = [...overviewRoutes];
