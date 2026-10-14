import type { Overview, OverviewTask, TaskStatus } from "../core/types";
import { formatAge, staleBadge } from "./age";

export interface AgentRow {
  readonly id: number;
  readonly title: string;
  readonly status: TaskStatus;
  readonly branch: string | null;
  /** Summary of the latest report, if the agent has reported. */
  readonly summary: string | null;
  readonly age: string;
  readonly staleLabel: string | null;
}

export interface AgentGroup {
  readonly key: string;
  readonly name: string;
  readonly rows: readonly AgentRow[];
}

const SHOWN: readonly TaskStatus[] = ["doing", "blocked", "review"];

function toRow(task: OverviewTask, now: Date): AgentRow {
  const since = task.latest_report?.created_at ?? task.updated_at;
  return {
    id: task.id,
    title: task.title,
    status: task.status,
    branch: task.branch,
    summary: task.latest_report?.summary ?? null,
    age: formatAge(since, now),
    staleLabel: staleBadge({ stale: task.stale, since }, now),
  };
}

/**
 * Panel "Agents" section: active agent-created tasks per project, in the server's order
 * (blocked first). Tasks the user created are shown under "Mine" instead.
 */
export function agentGroups(overview: Overview, now: Date): AgentGroup[] {
  return overview.projects
    .map((group) => ({
      key: group.project ? `project-${group.project.id}` : "inbox",
      name: group.project?.name ?? "Inbox",
      rows: group.tasks
        .filter((t) => t.source === "agent" && SHOWN.includes(t.status))
        .map((t) => toRow(t, now)),
    }))
    .filter((group) => group.rows.length > 0);
}
