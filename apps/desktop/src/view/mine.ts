import type { Task } from "../core/types";
import { formatWhen } from "./age";
import { priorityRank } from "./status";

const WANTED = ["todo", "doing"];

/** The earlier of the reminder and the due date, in ms; undated tasks sort last. */
function attentionTime(task: Task): number {
  const times = [task.remind_at, task.due_at].filter((t): t is string => t !== null).map(Date.parse);
  return times.length > 0 ? Math.min(...times) : Number.POSITIVE_INFINITY;
}

/** Panel "Mine" section: the user's own todo/doing tasks, soonest first. */
export function mineTasks(tasks: readonly Task[]): Task[] {
  return tasks
    .filter((t) => t.source === "user" && WANTED.includes(t.status))
    .sort((a, b) => {
      const byTime = attentionTime(a) - attentionTime(b);
      if (byTime !== 0 && !Number.isNaN(byTime)) return byTime;
      return priorityRank(a.priority) - priorityRank(b.priority) || b.id - a.id;
    });
}

/** "Remind in 1h" / "Due 3h ago" for the task's next scheduled moment, if any. */
export function taskWhen(task: Task, now: Date): { text: string; overdue: boolean } | null {
  const remind = task.remind_at ? { label: "Remind", iso: task.remind_at } : null;
  const due = task.due_at ? { label: "Due", iso: task.due_at } : null;
  const next = [remind, due].filter((x) => x !== null).sort((a, b) => Date.parse(a.iso) - Date.parse(b.iso))[0];
  if (!next) return null;
  const when = formatWhen(next.iso, now);
  return { text: `${next.label} ${when.text}`, overdue: when.overdue };
}

/** Next row index for ArrowDown (+1) or ArrowUp (-1), wrapping; null when the list is empty. */
export function moveSelection(current: number | null, delta: 1 | -1, length: number): number | null {
  if (length === 0) return null;
  if (current === null || current >= length) return delta === 1 ? 0 : length - 1;
  return (current + delta + length) % length;
}
