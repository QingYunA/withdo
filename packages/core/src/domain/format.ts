import type { Overview, OverviewTask } from "./overview.ts";
import type { Task } from "./types.ts";

/** One-line task summary, e.g. `#12 [doing] P2 feat/login — Add login (PR #3)`. */
export function formatTaskLine(task: Task): string {
  const branch = task.branch ? ` ${task.branch}` : "";
  const pr = task.pr_number ? ` (PR #${task.pr_number})` : task.pr_url ? ` (${task.pr_url})` : "";
  return `#${task.id} [${task.status}] ${task.priority}${branch} — ${task.title}${pr}`;
}

function formatOverviewTask(t: OverviewTask): string[] {
  const flag = t.stale ? `  STALE (${t.hours_since_activity}h)` : "";
  const lines = [`  ${formatTaskLine(t)}${flag}`];
  if (t.latest_report) {
    lines.push(`      last: ${t.latest_report.summary} (${t.latest_report.agent}, ${t.latest_report.created_at})`);
    if (t.latest_report.next_step) lines.push(`      next: ${t.latest_report.next_step}`);
  }
  return lines;
}

/** Human-readable overview used by both the CLI and the get_overview tool. */
export function formatOverview(overview: Overview): string {
  if (overview.projects.length === 0) return "No active tasks (nothing in doing, blocked or review).";
  const out: string[] = [];
  for (const group of overview.projects) {
    const title = group.project
      ? `${group.project.name}${group.project.repo_slug ? ` (${group.project.repo_slug})` : ""}`
      : "Inbox";
    out.push(title);
    for (const task of group.tasks) out.push(...formatOverviewTask(task));
    out.push("");
  }
  return out.join("\n").trimEnd();
}
