// Mirrors packages/core/src/domain/types.ts and the HTTP API payloads (M2a). Wire format, so snake_case.

export const TASK_STATUSES = ["backlog", "todo", "doing", "blocked", "review", "done", "cancelled"] as const;
export type TaskStatus = (typeof TASK_STATUSES)[number];

export const PRIORITIES = ["P0", "P1", "P2", "P3"] as const;
export type Priority = (typeof PRIORITIES)[number];

export interface Project {
  readonly id: number;
  readonly name: string;
  readonly repo_slug: string | null;
  readonly repo_path: string | null;
  readonly created_at: string;
  readonly updated_at: string;
  readonly archived_at: string | null;
}

export interface ProjectWithCounts extends Project {
  readonly counts: Readonly<Record<TaskStatus, number>>;
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
  readonly reminded_at: string | null;
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

export interface OverviewTask extends Task {
  readonly latest_report: Report | null;
  readonly stale: boolean;
  readonly hours_since_activity: number;
}

export interface OverviewGroup {
  /** Null for tasks that belong to no project (the inbox). */
  readonly project: Pick<Project, "id" | "name" | "repo_slug"> | null;
  readonly tasks: readonly OverviewTask[];
}

export interface Overview {
  readonly generated_at: string;
  readonly stale_hours: number;
  readonly projects: readonly OverviewGroup[];
}

export interface TaskDetail {
  readonly task: Task;
  /** Oldest first, as the server returns them. */
  readonly reports: readonly Report[];
}

export interface TaskFilter {
  readonly project_id?: number;
  /** Only tasks without a project. */
  readonly inbox?: boolean;
  readonly status?: readonly TaskStatus[];
  readonly milestone_id?: number;
  readonly limit?: number;
}

/** Body of POST /tasks. The server rejects unknown fields (no `cwd`, no `source`). */
export interface NewTask {
  readonly title: string;
  readonly project_id?: number;
  readonly notes?: string;
  readonly priority?: Priority;
  readonly milestone_id?: number;
  readonly due_at?: string;
  readonly remind_at?: string;
  readonly branch?: string;
  readonly pr_url?: string;
}

/** Body of PATCH /tasks/:id. `null` clears a nullable field; omitted fields are unchanged. */
export interface TaskPatch {
  readonly title?: string;
  readonly notes?: string | null;
  readonly status?: TaskStatus;
  readonly priority?: Priority;
  readonly project_id?: number | null;
  readonly milestone_id?: number | null;
  readonly due_at?: string | null;
  readonly remind_at?: string | null;
  readonly branch?: string | null;
  readonly pr_url?: string | null;
}

export interface NewMilestone {
  readonly project_id: number;
  readonly title: string;
  readonly due_at?: string;
}
