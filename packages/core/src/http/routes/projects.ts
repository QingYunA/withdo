import { listActiveProjects } from "../../domain/projects.ts";
import { jsonOk } from "../respond.ts";
import type { Route } from "../router.ts";

export const projectRoutes: readonly Route[] = [
  { method: "GET", path: "/projects", handler: ({ db }) => jsonOk(listActiveProjects(db)) },
];
