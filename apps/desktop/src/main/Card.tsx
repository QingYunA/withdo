import { useDraggable } from "@dnd-kit/core";
import type { KeyboardEvent, MouseEvent, PointerEvent } from "react";
import type { Task } from "../core/types";
import { taskDragId } from "../view/kanban";

export interface CardProps {
  readonly task: Task;
  readonly onOpen: (id: number) => void;
  readonly onOpenPr: (url: string) => void;
}

export function Card({ task, onOpen, onOpenPr }: CardProps) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id: taskDragId(task.id) });
  const style = transform ? { transform: `translate3d(${transform.x}px, ${transform.y}px, 0)` } : undefined;

  const openPr = (event: MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
    if (task.pr_url) onOpenPr(task.pr_url);
  };
  const onKeyDown = (event: KeyboardEvent) => {
    if (event.key === "Enter" && event.target === event.currentTarget) onOpen(task.id);
  };

  return (
    <div
      ref={setNodeRef}
      data-card
      style={style}
      {...attributes}
      {...listeners}
      onClick={() => onOpen(task.id)}
      onKeyDown={onKeyDown}
      className={`rounded-md border border-border bg-bg p-2 shadow-sm hover:border-muted ${isDragging ? "relative z-10 opacity-80 shadow-md" : ""}`}
    >
      <p className="leading-snug font-medium">{task.title}</p>
      <div className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-[11px] text-muted">
        {task.priority !== "P2" ? <span className={task.priority === "P0" ? "text-danger" : ""}>{task.priority}</span> : null}
        {task.branch ? <span className="max-w-full truncate font-mono">{task.branch}</span> : null}
        {task.pr_url ? (
          <a
            href={task.pr_url}
            onClick={openPr}
            onPointerDown={(event: PointerEvent) => event.stopPropagation()}
            className="text-accent hover:underline"
          >
            {task.pr_number ? `PR #${task.pr_number}` : "PR"}
          </a>
        ) : null}
      </div>
    </div>
  );
}
