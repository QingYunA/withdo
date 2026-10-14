import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { useUpdateTask } from "../core/queries";
import { PRIORITIES, TASK_STATUSES, type Milestone, type Task } from "../core/types";
import { buildPatch, formError, toFormValues, type TaskFormValues } from "../view/task-form";
import { PRIORITY_LABEL, STATUS_LABEL } from "../view/status";

function Field({ label, htmlFor, children }: { label: string; htmlFor: string; children: ReactNode }) {
  return (
    <div>
      <label className="label" htmlFor={htmlFor}>
        {label}
      </label>
      {children}
    </div>
  );
}

export interface TaskFormProps {
  readonly task: Task;
  readonly milestones: readonly Milestone[];
}

/** Edit form for one task. Saves only what changed, and keeps unsaved edits when the task refreshes. */
export function TaskForm({ task, milestones }: TaskFormProps) {
  const id = useId();
  const update = useUpdateTask();
  const [values, setValues] = useState<TaskFormValues>(() => toFormValues(task));
  const [error, setError] = useState<string | null>(null);
  const touched = useRef(false);

  // Follow changes made elsewhere (an agent's report, the panel) unless the user is mid-edit.
  useEffect(() => {
    if (!touched.current) setValues(toFormValues(task));
  }, [task]);
  useEffect(() => {
    touched.current = false;
    setError(null);
  }, [task.id]);

  const patch = buildPatch(task, values);
  const dirty = Object.keys(patch).length > 0;

  const set = <K extends keyof TaskFormValues>(key: K, value: TaskFormValues[K]) => {
    touched.current = true;
    setValues((current) => ({ ...current, [key]: value }));
  };

  const revert = () => {
    touched.current = false;
    setError(null);
    setValues(toFormValues(task));
  };

  const save = async () => {
    const problem = formError(values);
    if (problem) return setError(problem);
    setError(null);
    try {
      const saved = await update.mutateAsync({ id: task.id, patch });
      touched.current = false;
      setValues(toFormValues(saved));
    } catch (failure) {
      setError(failure instanceof Error ? failure.message : String(failure));
    }
  };

  return (
    <form
      className="space-y-3"
      onSubmit={(event) => {
        event.preventDefault();
        void save();
      }}
    >
      <Field label="Title" htmlFor={`${id}-title`}>
        <input id={`${id}-title`} className="field" value={values.title} onChange={(e) => set("title", e.target.value)} />
      </Field>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Status" htmlFor={`${id}-status`}>
          <select id={`${id}-status`} className="field" value={values.status} onChange={(e) => set("status", e.target.value as typeof values.status)}>
            {TASK_STATUSES.map((s) => (
              <option key={s} value={s}>
                {STATUS_LABEL[s]}
              </option>
            ))}
          </select>
        </Field>
        <Field label="Priority" htmlFor={`${id}-priority`}>
          <select id={`${id}-priority`} className="field" value={values.priority} onChange={(e) => set("priority", e.target.value as typeof values.priority)}>
            {PRIORITIES.map((p) => (
              <option key={p} value={p}>
                {p} {PRIORITY_LABEL[p]}
              </option>
            ))}
          </select>
        </Field>
      </div>
      <Field label="Milestone" htmlFor={`${id}-milestone`}>
        <select
          id={`${id}-milestone`}
          className="field"
          value={values.milestone_id}
          disabled={task.project_id === null}
          onChange={(e) => set("milestone_id", e.target.value)}
        >
          <option value="">{task.project_id === null ? "Inbox tasks have no milestone" : "None"}</option>
          {milestones.map((m) => (
            <option key={m.id} value={String(m.id)}>
              {m.title}
            </option>
          ))}
        </select>
      </Field>
      <div className="grid grid-cols-2 gap-3">
        <Field label="Due" htmlFor={`${id}-due`}>
          <input id={`${id}-due`} type="datetime-local" className="field" value={values.due_at} onChange={(e) => set("due_at", e.target.value)} />
        </Field>
        <Field label="Remind me" htmlFor={`${id}-remind`}>
          <input id={`${id}-remind`} type="datetime-local" className="field" value={values.remind_at} onChange={(e) => set("remind_at", e.target.value)} />
        </Field>
      </div>
      <Field label="Notes" htmlFor={`${id}-notes`}>
        <textarea id={`${id}-notes`} rows={4} className="field resize-y" value={values.notes} onChange={(e) => set("notes", e.target.value)} />
      </Field>
      {error ? <p className="text-xs text-danger">{error}</p> : null}
      <div className="flex gap-2">
        <button type="submit" className="btn btn-primary" disabled={!dirty || update.isPending}>
          Save
        </button>
        <button type="button" className="btn" disabled={!dirty} onClick={revert}>
          Revert
        </button>
      </div>
    </form>
  );
}
