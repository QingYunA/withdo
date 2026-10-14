import { DndContext, PointerSensor, useSensor, useSensors, type DragEndEvent } from "@dnd-kit/core";
import { useMemo } from "react";
import type { Task } from "../core/types";
import { KANBAN_STATUSES, groupByColumn, resolveDrop, taskIdFromDragId, type Move } from "../view/kanban";
import { Column } from "./Column";

export interface BoardProps {
  readonly tasks: readonly Task[];
  /** Show only this milestone's tasks; null shows everything. */
  readonly milestoneId: number | null;
  readonly onMove: (move: Move) => void;
  readonly onOpen: (id: number) => void;
  readonly onOpenPr: (url: string) => void;
}

export function Board({ tasks, milestoneId, onMove, onOpen, onOpenPr }: BoardProps) {
  // A small drag distance keeps a plain click on a card from starting a drag.
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 6 } }));
  const columns = useMemo(() => groupByColumn(tasks, milestoneId), [tasks, milestoneId]);

  const onDragEnd = ({ active, over }: Pick<DragEndEvent, "active" | "over">) => {
    const id = taskIdFromDragId(active.id);
    const move = id === null ? null : resolveDrop(tasks, id, over?.id ?? null);
    if (move) onMove(move);
  };

  return (
    <DndContext sensors={sensors} onDragEnd={onDragEnd}>
      <div className="flex min-h-0 flex-1 gap-3 overflow-x-auto p-4">
        {KANBAN_STATUSES.map((status) => (
          <Column key={status} status={status} tasks={columns[status]} onOpen={onOpen} onOpenPr={onOpenPr} />
        ))}
      </div>
    </DndContext>
  );
}
