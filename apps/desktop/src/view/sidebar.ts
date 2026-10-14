import type { ProjectWithCounts, Task, TaskFilter } from "../core/types";

export type Selection = { readonly kind: "inbox" } | { readonly kind: "project"; readonly id: number };

export interface SidebarItem {
  readonly key: string;
  readonly label: string;
  readonly selection: Selection;
  /** Tasks that are not done or cancelled; undefined for the inbox (the API does not count it). */
  readonly open?: number;
}

const OPEN_STATUSES = ["backlog", "todo", "doing", "blocked", "review"] as const;

export const selectionKey = (s: Selection): string => (s.kind === "inbox" ? "inbox" : `project-${s.id}`);

export const selectionFilter = (s: Selection): TaskFilter => (s.kind === "inbox" ? { inbox: true } : { project_id: s.id });

export const selectionForTask = (task: Task): Selection =>
  task.project_id === null ? { kind: "inbox" } : { kind: "project", id: task.project_id };

/** The first project, or the inbox when there are no projects yet. */
export function defaultSelection(projects: readonly ProjectWithCounts[]): Selection {
  const first = projects[0];
  return first ? { kind: "project", id: first.id } : { kind: "inbox" };
}

export function sidebarItems(projects: readonly ProjectWithCounts[]): SidebarItem[] {
  const inbox: SidebarItem = { key: "inbox", label: "Inbox", selection: { kind: "inbox" } };
  return [
    inbox,
    ...projects.map((p) => ({
      key: `project-${p.id}`,
      label: p.name,
      selection: { kind: "project", id: p.id } as const,
      open: OPEN_STATUSES.reduce((sum, status) => sum + p.counts[status], 0),
    })),
  ];
}
