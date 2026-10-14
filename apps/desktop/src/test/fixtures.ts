import type { OverviewTask, Report, Task } from "../core/types";

const NOW = "2026-10-14T10:00:00.000Z";

export function makeTask(overrides: Partial<Task> = {}): Task {
  return {
    id: 1,
    project_id: null,
    milestone_id: null,
    title: "A task",
    notes: null,
    status: "todo",
    priority: "P2",
    due_at: null,
    remind_at: null,
    reminded_at: null,
    branch: null,
    pr_url: null,
    pr_number: null,
    source: "user",
    created_at: NOW,
    updated_at: NOW,
    completed_at: null,
    ...overrides,
  };
}

export function makeReport(overrides: Partial<Report> = {}): Report {
  return { id: 1, task_id: 1, agent: "claude-code", status: null, summary: "did a thing", next_step: null, created_at: NOW, ...overrides };
}

export function makeOverviewTask(overrides: Partial<OverviewTask> = {}): OverviewTask {
  return {
    ...makeTask({ source: "agent", status: "doing" }),
    latest_report: null,
    stale: false,
    hours_since_activity: 0,
    ...overrides,
  };
}

export { NOW };
