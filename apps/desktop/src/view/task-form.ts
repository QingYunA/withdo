import type { Priority, Task, TaskPatch, TaskStatus } from "../core/types";

/** Editable fields of a task as strings/enums the form controls use. */
export interface TaskFormValues {
  readonly title: string;
  readonly notes: string;
  readonly status: TaskStatus;
  readonly priority: Priority;
  /** Milestone id as a string; empty means none. */
  readonly milestone_id: string;
  /** `YYYY-MM-DDTHH:mm` in local time (the `datetime-local` format); empty means none. */
  readonly due_at: string;
  readonly remind_at: string;
}

const pad = (n: number): string => String(n).padStart(2, "0");

/** ISO timestamp to the local wall-clock string a `datetime-local` input expects. */
export function toLocalInput(iso: string | null): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** The reverse of `toLocalInput`; null for an empty or unparseable value. */
export function fromLocalInput(value: string): string | null {
  if (!value) return null;
  const d = new Date(value);
  return Number.isNaN(d.getTime()) ? null : d.toISOString();
}

export function toFormValues(task: Task): TaskFormValues {
  return {
    title: task.title,
    notes: task.notes ?? "",
    status: task.status,
    priority: task.priority,
    milestone_id: task.milestone_id === null ? "" : String(task.milestone_id),
    due_at: toLocalInput(task.due_at),
    remind_at: toLocalInput(task.remind_at),
  };
}

/** The PATCH body for what the user changed; `{}` when nothing differs from `task`. */
export function buildPatch(task: Task, form: TaskFormValues): TaskPatch {
  const original = toFormValues(task);
  const patch: { -readonly [K in keyof TaskPatch]: TaskPatch[K] } = {};
  if (form.title.trim() !== original.title) patch.title = form.title.trim();
  if (form.notes !== original.notes) patch.notes = form.notes.trim() === "" ? null : form.notes;
  if (form.status !== original.status) patch.status = form.status;
  if (form.priority !== original.priority) patch.priority = form.priority;
  if (form.milestone_id !== original.milestone_id) patch.milestone_id = form.milestone_id === "" ? null : Number(form.milestone_id);
  if (form.due_at !== original.due_at) patch.due_at = fromLocalInput(form.due_at);
  if (form.remind_at !== original.remind_at) patch.remind_at = fromLocalInput(form.remind_at);
  return patch;
}

export function formError(form: TaskFormValues): string | null {
  return form.title.trim() === "" ? "Title is required" : null;
}
