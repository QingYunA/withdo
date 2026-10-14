import type { Task, TaskStatus } from "./types";

/** A copy of `tasks` with one task's status changed; used to move a kanban card before the server answers. */
export function withTaskStatus(tasks: Task[] | undefined, id: number, status: TaskStatus): Task[] | undefined {
  return tasks?.map((task) => (task.id === id ? { ...task, status } : task));
}
