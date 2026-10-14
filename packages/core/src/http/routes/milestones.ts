import { NotFoundError } from "../../domain/errors.ts";
import { createMilestone, listMilestones } from "../../domain/milestones.ts";
import { getProject } from "../../domain/projects.ts";
import { createMilestoneInput } from "../../mcp/tools/create-milestone.ts";
import { projectId } from "../../mcp/schemas.ts";
import { idParam } from "../params.ts";
import { jsonOk, parseInput, readJsonBody } from "../respond.ts";
import type { Route } from "../router.ts";

/** HTTP clients name the project by id; `cwd` resolution is for agents. */
const body = createMilestoneInput.omit({ cwd: true }).required({ project_id: true }).strict();

export const milestoneRoutes: readonly Route[] = [
  {
    method: "GET",
    path: "/projects/:id/milestones",
    handler: ({ db, params }) => {
      const id = idParam(params.id, projectId, "project_id");
      if (!getProject(db, id)) throw new NotFoundError(`project ${id} not found`, "project_id");
      return jsonOk(listMilestones(db, id));
    },
  },
  {
    method: "POST",
    path: "/milestones",
    handler: async ({ db, req }) => {
      const input = parseInput(body, await readJsonBody(req));
      return jsonOk(createMilestone(db, input), 201);
    },
  },
];
