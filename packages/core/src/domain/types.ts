export const TASK_STATUSES = ["backlog", "todo", "doing", "blocked", "review", "done", "cancelled"] as const;
export type TaskStatus = (typeof TASK_STATUSES)[number];

export const PRIORITIES = ["P0", "P1", "P2", "P3"] as const;
export type Priority = (typeof PRIORITIES)[number];

export const TERMINAL_STATUSES: readonly TaskStatus[] = ["done", "cancelled"];
export const ACTIVE_STATUSES: readonly TaskStatus[] = ["doing", "blocked", "review"];

export interface Project {
  readonly id: number;
  readonly name: string;
  readonly repo_slug: string | null;
  readonly repo_path: string | null;
  readonly created_at: string;
  readonly updated_at: string;
  readonly archived_at: string | null;
}

export interface Milestone {
  readonly id: number;
  readonly project_id: number;
  readonly title: string;
  readonly due_at: string | null;
  readonly status: "open" | "closed";
  readonly created_at: string;
}

export interface Task {
  readonly id: number;
  readonly project_id: number | null;
  readonly milestone_id: number | null;
  readonly title: string;
  readonly notes: string | null;
  readonly status: TaskStatus;
  readonly priority: Priority;
  readonly due_at: string | null;
  readonly remind_at: string | null;
  readonly branch: string | null;
  readonly pr_url: string | null;
  readonly pr_number: number | null;
  readonly source: "user" | "agent";
  readonly created_at: string;
  readonly updated_at: string;
  readonly completed_at: string | null;
}

export interface Report {
  readonly id: number;
  readonly task_id: number;
  readonly agent: string;
  readonly status: TaskStatus | null;
  readonly summary: string;
  readonly next_step: string | null;
  readonly created_at: string;
}

export const nowIso = (): string => new Date().toISOString();
