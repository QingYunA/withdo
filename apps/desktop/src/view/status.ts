import type { Priority, TaskStatus } from "../core/types";

export const STATUS_LABEL: Readonly<Record<TaskStatus, string>> = {
  backlog: "Backlog",
  todo: "Todo",
  doing: "Doing",
  blocked: "Blocked",
  review: "Review",
  done: "Done",
  cancelled: "Cancelled",
};

/** Full class names so Tailwind's scanner can see them. Colours are defined in index.css. */
export const STATUS_DOT: Readonly<Record<TaskStatus, string>> = {
  backlog: "bg-status-backlog",
  todo: "bg-status-todo",
  doing: "bg-status-doing",
  blocked: "bg-status-blocked",
  review: "bg-status-review",
  done: "bg-status-done",
  cancelled: "bg-status-cancelled",
};

export const PRIORITY_LABEL: Readonly<Record<Priority, string>> = {
  P0: "Urgent",
  P1: "High",
  P2: "Normal",
  P3: "Low",
};

const RANK: Readonly<Record<Priority, number>> = { P0: 0, P1: 1, P2: 2, P3: 3 };

/** Lower sorts first. */
export const priorityRank = (priority: Priority): number => RANK[priority];
