import type { Database } from "bun:sqlite";
import { DomainError } from "./errors.ts";
import { resolveProjectFromCwd } from "./projects.ts";
import { createTask, findOpenTask, getTask, updateTask, type TaskPatch } from "./tasks.ts";
import { nowIso, type Report, type Task, type TaskStatus } from "./types.ts";

export interface AddReportInput {
  readonly task_id: number;
  readonly agent: string;
  readonly summary: string;
  readonly status?: TaskStatus | null;
  readonly next_step?: string | null;
  readonly created_at?: string;
}

export interface ReportProgressInput {
  readonly summary: string;
  readonly agent: string;
  readonly status?: TaskStatus;
  readonly next_step?: string;
  readonly task_id?: number;
  readonly title?: string;
  readonly cwd?: string;
  readonly branch?: string;
  readonly pr_url?: string;
}

export interface ReportProgressResult {
  readonly task: Task;
  readonly report: Report;
  readonly created: boolean;
}

const MAX_TITLE = 80;

export function addReport(db: Database, input: AddReportInput): Report {
  const result = db.run(
    "INSERT INTO reports (task_id, agent, status, summary, next_step, created_at) VALUES (?, ?, ?, ?, ?, ?)",
    [
      input.task_id,
      input.agent,
      input.status ?? null,
      input.summary,
      input.next_step ?? null,
      input.created_at ?? nowIso(),
    ],
  );
  return db.query<Report, [number]>("SELECT * FROM reports WHERE id = ?").get(Number(result.lastInsertRowid))!;
}

export function listReports(db: Database, taskId: number): Report[] {
  return db
    .query<Report, [number]>("SELECT * FROM reports WHERE task_id = ? ORDER BY created_at, id")
    .all(taskId);
}

export function latestReport(db: Database, taskId: number): Report | null {
  return (
    db
      .query<Report, [number]>("SELECT * FROM reports WHERE task_id = ? ORDER BY created_at DESC, id DESC LIMIT 1")
      .get(taskId) ?? null
  );
}

function truncate(text: string): string {
  const line = text.trim().split("\n")[0]!;
  return line.length <= MAX_TITLE ? line : `${line.slice(0, MAX_TITLE - 1)}…`;
}

function patchFrom(input: ReportProgressInput, branch?: string): TaskPatch {
  return {
    ...(input.status ? { status: input.status } : {}),
    ...(input.pr_url ? { pr_url: input.pr_url } : {}),
    ...(branch ? { branch } : {}),
  };
}

function locateOrCreate(db: Database, input: ReportProgressInput): { task: Task; created: boolean } {
  if (input.task_id !== undefined) {
    const task = getTask(db, input.task_id);
    if (!task) throw new DomainError(`task ${input.task_id} not found`, "task_id");
    return { task, created: false };
  }
  const resolved = input.cwd !== undefined ? resolveProjectFromCwd(db, input.cwd) : null;
  const projectId = resolved?.project.id ?? null;
  const branch = input.branch ?? resolved?.git.branch ?? null;
  const existing = branch ? findOpenTask(db, projectId, branch) : null;
  if (existing) return { task: existing, created: false };
  const task = createTask(db, {
    title: input.title ?? branch ?? truncate(input.summary),
    source: "agent",
    status: input.status ?? "doing",
    project_id: projectId,
    branch,
    pr_url: input.pr_url ?? null,
  });
  return { task, created: true };
}

/** Record an agent's progress: find or create the task, apply status/PR, append a report. */
export function reportProgress(db: Database, input: ReportProgressInput): ReportProgressResult {
  return db.transaction((): ReportProgressResult => {
    const { task: located, created } = locateOrCreate(db, input);
    const patch = patchFrom(input, input.task_id !== undefined ? input.branch : undefined);
    // Always touch the task so updated_at reflects the report; creation already applied the patch.
    const task = updateTask(db, located.id, created ? {} : patch);
    const report = addReport(db, {
      task_id: task.id,
      agent: input.agent,
      summary: input.summary,
      status: input.status ?? null,
      next_step: input.next_step ?? null,
    });
    return { task, report, created };
  })();
}
