import type { Task, TaskStatus } from "../core/types";
import { priorityRank } from "./status";

/** Board columns, left to right. Cancelled tasks are not shown. */
export const KANBAN_STATUSES = ["backlog", "todo", "doing", "blocked", "review", "done"] as const satisfies readonly TaskStatus[];
export type ColumnStatus = (typeof KANBAN_STATUSES)[number];

const COLUMN_PREFIX = "column:";

export const columnId = (status: ColumnStatus): string => `${COLUMN_PREFIX}${status}`;

/** The status a droppable id stands for, or null if it is not a column id. */
export function statusFromColumnId(id: string | number): ColumnStatus | null {
  if (typeof id !== "string" || !id.startsWith(COLUMN_PREFIX)) return null;
  const status = id.slice(COLUMN_PREFIX.length);
  return (KANBAN_STATUSES as readonly string[]).includes(status) ? (status as ColumnStatus) : null;
}

const emptyColumns = (): Record<ColumnStatus, Task[]> =>
  Object.fromEntries(KANBAN_STATUSES.map((s) => [s, [] as Task[]])) as Record<ColumnStatus, Task[]>;

const byPriorityThenNewest = (a: Task, b: Task): number => priorityRank(a.priority) - priorityRank(b.priority) || b.id - a.id;

/** Tasks per column, optionally limited to one milestone (null = all). */
export function groupByColumn(tasks: readonly Task[], milestoneId: number | null): Record<ColumnStatus, Task[]> {
  const columns = emptyColumns();
  for (const task of tasks) {
    if (milestoneId !== null && task.milestone_id !== milestoneId) continue;
    const column = columns[task.status as ColumnStatus];
    if (column) column.push(task);
  }
  for (const status of KANBAN_STATUSES) columns[status].sort(byPriorityThenNewest);
  return columns;
}

export interface Move {
  readonly id: number;
  readonly status: ColumnStatus;
}

/** What dropping task `taskId` on `overId` should do: a status change, or null for a no-op. */
export function resolveDrop(tasks: readonly Task[], taskId: number, overId: string | number | null): Move | null {
  if (overId === null) return null;
  const status = statusFromColumnId(overId);
  const task = tasks.find((t) => t.id === taskId);
  if (!status || !task || task.status === status) return null;
  return { id: taskId, status };
}

const TASK_PREFIX = "task:";

export const taskDragId = (id: number): string => `${TASK_PREFIX}${id}`;

/** The task id a draggable id stands for, or null if it is not a task id. */
export function taskIdFromDragId(id: string | number): number | null {
  if (typeof id !== "string" || !id.startsWith(TASK_PREFIX)) return null;
  const value = Number(id.slice(TASK_PREFIX.length));
  return Number.isInteger(value) && value > 0 ? value : null;
}
