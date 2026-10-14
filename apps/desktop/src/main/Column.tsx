import { useDroppable } from "@dnd-kit/core";
import type { Task } from "../core/types";
import { StatusDot } from "../ui/StatusDot";
import { columnId, type ColumnStatus } from "../view/kanban";
import { STATUS_LABEL } from "../view/status";
import { Card } from "./Card";

export interface ColumnProps {
  readonly status: ColumnStatus;
  readonly tasks: readonly Task[];
  readonly onOpen: (id: number) => void;
  readonly onOpenPr: (url: string) => void;
}

export function Column({ status, tasks, onOpen, onOpenPr }: ColumnProps) {
  const { setNodeRef, isOver } = useDroppable({ id: columnId(status) });
  return (
    <section
      ref={setNodeRef}
      aria-label={STATUS_LABEL[status]}
      className={`flex min-h-0 w-60 shrink-0 flex-col rounded-lg border border-border ${isOver ? "bg-hover" : "bg-surface"}`}
    >
      <header className="flex items-center gap-2 px-3 py-2 text-xs font-medium">
        <StatusDot status={status} />
        <span>{STATUS_LABEL[status]}</span>
        <span className="text-muted">{tasks.length}</span>
      </header>
      <div className="flex min-h-12 flex-1 flex-col gap-2 overflow-y-auto px-2 pb-2">
        {tasks.map((task) => (
          <Card key={task.id} task={task} onOpen={onOpen} onOpenPr={onOpenPr} />
        ))}
      </div>
    </section>
  );
}
