import type { ReactNode } from "react";
import type { Task } from "../core/types";
import { StatusDot } from "../ui/StatusDot";
import type { AgentRow } from "../view/agents";
import { taskWhen } from "../view/mine";

function RowButton({ active, onOpen, children }: { active: boolean; onOpen: () => void; children: ReactNode }) {
  return (
    <button
      type="button"
      data-active={active}
      aria-current={active ? "true" : undefined}
      onClick={onOpen}
      ref={(el) => {
        if (active) el?.scrollIntoView?.({ block: "nearest" });
      }}
      className="flex w-full items-start gap-2 rounded-md px-2 py-1.5 text-left hover:bg-hover data-[active=true]:bg-hover"
    >
      {children}
    </button>
  );
}

export function AgentRowView({ row, active, onOpen }: { row: AgentRow; active: boolean; onOpen: () => void }) {
  return (
    <RowButton active={active} onOpen={onOpen}>
      <StatusDot status={row.status} className="mt-1.5" />
      <span className="min-w-0 flex-1">
        <span className="flex items-baseline gap-2">
          <span className="truncate font-medium">{row.title}</span>
          {row.branch ? <span className="truncate font-mono text-[11px] text-muted">{row.branch}</span> : null}
        </span>
        {row.summary ? <span className="block truncate text-xs text-muted">{row.summary}</span> : null}
      </span>
      <span className="flex shrink-0 items-center gap-1.5 text-xs text-muted">
        {row.staleLabel ? <span className="rounded border border-border px-1 text-[11px] text-danger">{row.staleLabel}</span> : null}
        <span>{row.age}</span>
      </span>
    </RowButton>
  );
}

export function MineRowView({ task, now, active, onOpen }: { task: Task; now: Date; active: boolean; onOpen: () => void }) {
  const when = taskWhen(task, now);
  return (
    <RowButton active={active} onOpen={onOpen}>
      <StatusDot status={task.status} className="mt-1.5" />
      <span className="min-w-0 flex-1 truncate font-medium">{task.title}</span>
      <span className="flex shrink-0 items-center gap-1.5 text-xs text-muted">
        {task.priority === "P0" || task.priority === "P1" ? <span>{task.priority}</span> : null}
        {when ? <span className={when.overdue ? "text-danger" : ""}>{when.text}</span> : null}
      </span>
    </RowButton>
  );
}
