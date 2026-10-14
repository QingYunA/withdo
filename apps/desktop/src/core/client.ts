import type {
  Milestone,
  NewMilestone,
  NewTask,
  Overview,
  ProjectWithCounts,
  Task,
  TaskDetail,
  TaskFilter,
  TaskPatch,
} from "./types";

export interface Endpoint {
  /** For example `http://127.0.0.1:52341`, without a trailing slash. */
  readonly baseUrl: string;
  readonly token: string;
}

/** A failed request. `status` is 0 when the server could not be reached at all. */
export class ApiError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

/** The slice of `fetch` the client needs; tests pass a stub. */
export type FetchLike = (input: string, init?: RequestInit) => Promise<Response>;

export interface CoreClient {
  overview(staleHours?: number): Promise<Overview>;
  projects(): Promise<ProjectWithCounts[]>;
  tasks(filter?: TaskFilter): Promise<Task[]>;
  task(id: number): Promise<TaskDetail>;
  createTask(input: NewTask): Promise<Task>;
  updateTask(id: number, patch: TaskPatch): Promise<Task>;
  milestones(projectId: number): Promise<Milestone[]>;
  createMilestone(input: NewMilestone): Promise<Milestone>;
}

type Query = ReadonlyArray<readonly [string, string | number]>;

interface RequestOptions {
  readonly query?: Query;
  readonly body?: unknown;
}

function tasksQuery(filter: TaskFilter): Query {
  return [
    ...(filter.project_id !== undefined ? ([["project_id", filter.project_id]] as const) : []),
    ...(filter.inbox ? ([["inbox", 1]] as const) : []),
    ...(filter.status ?? []).map((s) => ["status", s] as const),
    ...(filter.milestone_id !== undefined ? ([["milestone_id", filter.milestone_id]] as const) : []),
    ...(filter.limit !== undefined ? ([["limit", filter.limit]] as const) : []),
  ];
}

function withQuery(path: string, query: Query = []): string {
  if (query.length === 0) return path;
  const params = new URLSearchParams(query.map(([k, v]) => [k, String(v)]));
  return `${path}?${params.toString()}`;
}

async function failure(res: Response): Promise<ApiError> {
  const body: unknown = await res.json().catch(() => null);
  const message = typeof (body as { error?: unknown } | null)?.error === "string" ? (body as { error: string }).error : null;
  return new ApiError(res.status, message ?? `${res.status} ${res.statusText}`.trim());
}

/** Typed client for the `adhd-core serve` HTTP API. */
export function createCoreClient(endpoint: Endpoint, fetchImpl: FetchLike = (input, init) => fetch(input, init)): CoreClient {
  async function request<T>(method: string, path: string, { query, body }: RequestOptions = {}): Promise<T> {
    const headers: Record<string, string> = { authorization: `Bearer ${endpoint.token}` };
    if (body !== undefined) headers["content-type"] = "application/json";
    let res: Response;
    try {
      res = await fetchImpl(`${endpoint.baseUrl}${withQuery(path, query)}`, {
        method,
        headers,
        ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
      });
    } catch {
      throw new ApiError(0, "Cannot reach adhd-core");
    }
    if (!res.ok) throw await failure(res);
    return ((await res.json()) as { data: T }).data;
  }

  return {
    overview: (staleHours) =>
      request("GET", "/overview", staleHours === undefined ? {} : { query: [["stale_hours", staleHours]] }),
    projects: () => request("GET", "/projects"),
    tasks: (filter = {}) => request("GET", "/tasks", { query: tasksQuery(filter) }),
    task: (id) => request("GET", `/tasks/${id}`),
    createTask: (input) => request("POST", "/tasks", { body: input }),
    updateTask: (id, patch) => request("PATCH", `/tasks/${id}`, { body: patch }),
    milestones: (projectId) => request("GET", `/projects/${projectId}/milestones`),
    createMilestone: (input) => request("POST", "/milestones", { body: input }),
  };
}
