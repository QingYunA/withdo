import * as z from "zod";
import { NotFoundError } from "../../domain/errors.ts";
import { listReports } from "../../domain/reports.ts";
import { createTask, getTask, listTasks, updateTask } from "../../domain/tasks.ts";
import { taskId } from "../../mcp/schemas.ts";
import { createTaskInput } from "../../mcp/tools/create-task.ts";
import { listTasksInput } from "../../mcp/tools/list-tasks.ts";
import { updateTaskInput } from "../../mcp/tools/update-task.ts";
import { idParam, numberQuery } from "../params.ts";
import { HttpError, jsonOk, parseInput, readJsonBody } from "../respond.ts";
import type { Route } from "../router.ts";

/** A kanban board needs more than the MCP default of 50 rows. */
const DEFAULT_LIMIT = 500;

// HTTP clients use project ids; `cwd` resolution and `source` are not theirs to set.
const createBody = createTaskInput.omit({ cwd: true }).strict();
const patchBody = updateTaskInput.omit({ task_id: true }).strict();
const listQuery = listTasksInput
  .omit({ cwd: true })
  .extend({ limit: z.number().int().min(1).max(1000).optional() })
  .strict();

const KNOWN_QUERY: ReadonlySet<string> = new Set(["project_id", "milestone_id", "limit", "status", "inbox"]);

function parseListQuery(params: URLSearchParams) {
  const inbox = params.get("inbox");
  if (inbox !== null && inbox !== "1") throw new HttpError(400, "inbox: must be 1");
  const unknown = [...params.keys()].filter((k) => !KNOWN_QUERY.has(k));
  const filter = parseInput(listQuery, {
    ...Object.fromEntries(unknown.map((k) => [k, params.get(k)])),
    project_id: numberQuery(params, "project_id"),
    milestone_id: numberQuery(params, "milestone_id"),
    limit: numberQuery(params, "limit"),
    ...(params.has("status") ? { status: params.getAll("status") } : {}),
  });
  if (inbox !== null && filter.project_id !== undefined) {
    throw new HttpError(400, "inbox: cannot be combined with project_id");
  }
  return { ...filter, inbox: inbox !== null, limit: filter.limit ?? DEFAULT_LIMIT };
}

export const taskRoutes: readonly Route[] = [
  { method: "GET", path: "/tasks", handler: ({ db, url }) => jsonOk(listTasks(db, parseListQuery(url.searchParams))) },
  {
    method: "GET",
    path: "/tasks/:id",
    handler: ({ db, params }) => {
      const id = idParam(params.id, taskId, "task_id");
      const task = getTask(db, id);
      if (!task) throw new NotFoundError(`task ${id} not found`, "task_id");
      return jsonOk({ task, reports: listReports(db, id) });
    },
  },
  {
    method: "POST",
    path: "/tasks",
    handler: async ({ db, req }) => {
      const input = parseInput(createBody, await readJsonBody(req));
      return jsonOk(createTask(db, { ...input, source: "user" }), 201);
    },
  },
  {
    method: "PATCH",
    path: "/tasks/:id",
    handler: async ({ db, req, params }) => {
      const id = idParam(params.id, taskId, "task_id");
      const patch = parseInput(patchBody, await readJsonBody(req));
      return jsonOk(updateTask(db, id, patch));
    },
  },
];
