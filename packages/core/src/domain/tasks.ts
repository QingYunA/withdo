import type { Database, SQLQueryBindings } from "bun:sqlite";
import { DomainError, NotFoundError } from "./errors.ts";
import { getMilestone } from "./milestones.ts";
import { getProject } from "./projects.ts";
import { nowIso, TERMINAL_STATUSES, type Priority, type Task, type TaskStatus } from "./types.ts";

export interface CreateTaskInput {
  readonly title: string;
  readonly source: "user" | "agent";
  readonly project_id?: number | null;
  readonly milestone_id?: number | null;
  readonly notes?: string | null;
  readonly status?: TaskStatus;
  readonly priority?: Priority;
  readonly due_at?: string | null;
  readonly remind_at?: string | null;
  readonly branch?: string | null;
  readonly pr_url?: string | null;
}

export type TaskPatch = Partial<Omit<CreateTaskInput, "source">>;

export interface TaskFilter {
  readonly project_id?: number;
  readonly status?: readonly TaskStatus[];
  readonly milestone_id?: number;
  readonly limit?: number;
}

const PR_URL_RE = /^https?:\/\/github\.com\/[^/\s]+\/[^/\s]+\/pull\/(\d+)(?:[/?#].*)?$/;
const DEFAULT_LIMIT = 50;

/** Extract the PR number from a GitHub pull request URL, else null. */
export function parsePrNumber(url: string): number | null {
  const m = PR_URL_RE.exec(url.trim());
  return m ? Number(m[1]) : null;
}

const isTerminal = (s: TaskStatus): boolean => TERMINAL_STATUSES.includes(s);

export function getTask(db: Database, id: number): Task | null {
  return db.query<Task, [number]>("SELECT * FROM tasks WHERE id = ?").get(id) ?? null;
}

function requireTask(db: Database, id: number): Task {
  const task = getTask(db, id);
  if (!task) throw new NotFoundError(`task ${id} not found`, "task_id");
  return task;
}

/** Validate project/milestone references; infer the project from a milestone when absent. */
function checkRefs(
  db: Database,
  projectId: number | null,
  milestoneId: number | null,
): { projectId: number | null; milestoneId: number | null } {
  if (projectId !== null && !getProject(db, projectId)) {
    throw new NotFoundError(`project ${projectId} not found`, "project_id");
  }
  if (milestoneId === null) return { projectId, milestoneId };
  const milestone = getMilestone(db, milestoneId);
  if (!milestone) throw new NotFoundError(`milestone ${milestoneId} not found`, "milestone_id");
  if (projectId !== null && milestone.project_id !== projectId) {
    throw new DomainError(`milestone ${milestoneId} belongs to another project`, "milestone_id");
  }
  return { projectId: projectId ?? milestone.project_id, milestoneId };
}

export function createTask(db: Database, input: CreateTaskInput): Task {
  const refs = checkRefs(db, input.project_id ?? null, input.milestone_id ?? null);
  const ts = nowIso();
  const status = input.status ?? "todo";
  const prUrl = input.pr_url ?? null;
  const result = db.run(
    `INSERT INTO tasks (project_id, milestone_id, title, notes, status, priority, due_at, remind_at,
       branch, pr_url, pr_number, source, created_at, updated_at, completed_at)
     VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
    [
      refs.projectId,
      refs.milestoneId,
      input.title,
      input.notes ?? null,
      status,
      input.priority ?? "P2",
      input.due_at ?? null,
      input.remind_at ?? null,
      input.branch ?? null,
      prUrl,
      prUrl ? parsePrNumber(prUrl) : null,
      input.source,
      ts,
      ts,
      isTerminal(status) ? ts : null,
    ],
  );
  return getTask(db, Number(result.lastInsertRowid))!;
}

/** Patch a task. `undefined` leaves a field alone; `null` clears a nullable one. */
export function updateTask(db: Database, id: number, patch: TaskPatch): Task {
  const current = requireTask(db, id);
  const merged = { ...current, ...definedOnly(patch) };
  const refs = checkRefs(db, merged.project_id, merged.milestone_id);
  const ts = nowIso();
  const status = merged.status;
  const completedAt = isTerminal(status) ? (isTerminal(current.status) ? current.completed_at : ts) : null;
  const prUrl = merged.pr_url;
  // Any write to remind_at re-arms the reminder.
  const remindedAt = "remind_at" in definedOnly(patch) ? null : current.reminded_at;
  db.run(
    `UPDATE tasks SET project_id = ?, milestone_id = ?, title = ?, notes = ?, status = ?, priority = ?,
       due_at = ?, remind_at = ?, reminded_at = ?, branch = ?, pr_url = ?, pr_number = ?, updated_at = ?,
       completed_at = ?
     WHERE id = ?`,
    [
      refs.projectId,
      refs.milestoneId,
      merged.title,
      merged.notes,
      status,
      merged.priority,
      merged.due_at,
      merged.remind_at,
      remindedAt,
      merged.branch,
      prUrl,
      prUrl ? parsePrNumber(prUrl) : null,
      ts,
      completedAt,
      id,
    ],
  );
  return getTask(db, id)!;
}

function definedOnly<T extends object>(obj: T): Partial<T> {
  return Object.fromEntries(Object.entries(obj).filter(([, v]) => v !== undefined)) as Partial<T>;
}

export function listTasks(db: Database, filter: TaskFilter): Task[] {
  const where: string[] = [];
  const params: SQLQueryBindings[] = [];
  if (filter.project_id !== undefined) {
    where.push("project_id = ?");
    params.push(filter.project_id);
  }
  if (filter.milestone_id !== undefined) {
    where.push("milestone_id = ?");
    params.push(filter.milestone_id);
  }
  if (filter.status && filter.status.length > 0) {
    where.push(`status IN (${filter.status.map(() => "?").join(",")})`);
    params.push(...filter.status);
  }
  const sql = `SELECT * FROM tasks ${where.length ? `WHERE ${where.join(" AND ")}` : ""} ORDER BY id DESC LIMIT ?`;
  params.push(filter.limit ?? DEFAULT_LIMIT);
  return db.query<Task, SQLQueryBindings[]>(sql).all(...params);
}

/** The newest not-yet-finished task on `branch` within a project (null = inbox). */
export function findOpenTask(db: Database, projectId: number | null, branch: string): Task | null {
  return (
    db
      .query<Task, [number | null, string]>(
        `SELECT * FROM tasks WHERE project_id IS ? AND branch = ?
           AND status NOT IN ('done','cancelled') ORDER BY id DESC LIMIT 1`,
      )
      .get(projectId, branch) ?? null
  );
}
