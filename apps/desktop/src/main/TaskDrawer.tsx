import { useEffect } from "react";
import { useMilestones, useTask } from "../core/queries";
import { openExternal } from "../platform/shell";
import { StatusDot } from "../ui/StatusDot";
import { useNow } from "../ui/use-now";
import { ReportTimeline } from "./ReportTimeline";
import { TaskForm } from "./TaskForm";

export interface TaskDrawerProps {
  readonly taskId: number;
  readonly onClose: () => void;
}

export function TaskDrawer({ taskId, onClose }: TaskDrawerProps) {
  const now = useNow();
  const detail = useTask(taskId);
  const task = detail.data?.task ?? null;
  const milestones = useMilestones(task?.project_id ?? null);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      // Esc inside a form field should not throw away what the user is typing.
      const editing = event.target instanceof HTMLElement && ["INPUT", "TEXTAREA", "SELECT"].includes(event.target.tagName);
      if (event.key === "Escape" && !editing) onClose();
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <aside aria-label="Task details" className="absolute inset-y-0 right-0 z-20 flex w-[400px] flex-col border-l border-border bg-bg shadow-lg">
      <header className="flex items-center justify-between border-b border-border px-4 py-2">
        <p className="flex items-center gap-2 text-xs text-muted">
          {task ? <StatusDot status={task.status} /> : null}
          <span>Task #{taskId}</span>
        </p>
        <button type="button" className="btn" onClick={onClose}>
          Close
        </button>
      </header>
      <div className="flex-1 space-y-5 overflow-y-auto p-4">
        {detail.isError ? <p className="text-xs text-danger">{detail.error.message}</p> : null}
        {task ? (
          <>
            <TaskForm task={task} milestones={milestones.data ?? []} />
            {task.pr_url ? (
              <button type="button" className="btn" onClick={() => openExternal(task.pr_url!)}>
                Open PR {task.pr_number ? `#${task.pr_number}` : ""}
              </button>
            ) : null}
            <section aria-label="Reports">
              <h3 className="label">Reports</h3>
              <ReportTimeline reports={detail.data?.reports ?? []} now={now} />
            </section>
          </>
        ) : null}
      </div>
    </aside>
  );
}
