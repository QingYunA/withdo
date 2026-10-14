import type { Database } from "bun:sqlite";
import { latestReport } from "./reports.ts";
import { ACTIVE_STATUSES, type Project, type Report, type Task } from "./types.ts";

export const DEFAULT_STALE_HOURS = 48;

export interface OverviewTask extends Task {
  readonly latest_report: Report | null;
  readonly stale: boolean;
  readonly hours_since_activity: number;
}

export interface OverviewGroup {
  /** Null for inbox tasks that belong to no project. */
  readonly project: Pick<Project, "id" | "name" | "repo_slug"> | null;
  readonly tasks: readonly OverviewTask[];
}

export interface Overview {
  readonly generated_at: string;
  readonly stale_hours: number;
  readonly projects: readonly OverviewGroup[];
}

export interface OverviewOptions {
  readonly staleHours?: number;
  readonly now?: Date;
}

const ATTENTION_ORDER: Record<string, number> = { blocked: 0, review: 1, doing: 2 };

function describeTask(db: Database, task: Task, now: Date, staleHours: number): OverviewTask {
  const report = latestReport(db, task.id);
  const since = Date.parse(report?.created_at ?? task.updated_at);
  const hours = Math.floor((now.getTime() - since) / 3_600_000);
  return { ...task, latest_report: report, hours_since_activity: hours, stale: hours >= staleHours };
}

function byAttention(a: OverviewTask, b: OverviewTask): number {
  return (ATTENTION_ORDER[a.status] ?? 9) - (ATTENTION_ORDER[b.status] ?? 9) || b.id - a.id;
}

/** Active (doing/blocked/review) tasks per project, with latest report and a stale flag. */
export function getOverview(db: Database, options: OverviewOptions = {}): Overview {
  const now = options.now ?? new Date();
  const staleHours = options.staleHours ?? DEFAULT_STALE_HOURS;
  const placeholders = ACTIVE_STATUSES.map(() => "?").join(",");
  const tasks = db
    .query<Task, string[]>(
      `SELECT t.* FROM tasks t LEFT JOIN projects p ON p.id = t.project_id
       WHERE t.status IN (${placeholders}) AND (t.project_id IS NULL OR p.archived_at IS NULL)`,
    )
    .all(...ACTIVE_STATUSES);

  const projects = new Map(
    db
      .query<Project, []>("SELECT * FROM projects WHERE archived_at IS NULL")
      .all()
      .map((p) => [p.id, p] as const),
  );
  const grouped = new Map<number | null, OverviewTask[]>();
  for (const task of tasks) {
    const list = grouped.get(task.project_id) ?? [];
    grouped.set(task.project_id, [...list, describeTask(db, task, now, staleHours)]);
  }

  const groups: OverviewGroup[] = [...grouped.entries()].map(([id, list]) => {
    const p = id === null ? undefined : projects.get(id);
    return {
      project: p ? { id: p.id, name: p.name, repo_slug: p.repo_slug } : null,
      tasks: [...list].sort(byAttention),
    };
  });
  groups.sort((a, b) => {
    if (a.project === null) return 1;
    if (b.project === null) return -1;
    return a.project.name.localeCompare(b.project.name);
  });

  return { generated_at: now.toISOString(), stale_hours: staleHours, projects: groups };
}
