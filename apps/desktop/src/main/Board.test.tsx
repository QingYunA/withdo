import { render, screen } from "@testing-library/react";
import type { DragEndEvent } from "@dnd-kit/core";
import { describe, expect, test, vi } from "vitest";
import { makeTask } from "../test/fixtures";

const dnd = vi.hoisted(() => ({ onDragEnd: null as null | ((e: Pick<DragEndEvent, "active" | "over">) => void) }));
vi.mock("@dnd-kit/core", async (importOriginal) => {
  const original = await importOriginal<typeof import("@dnd-kit/core")>();
  return {
    ...original,
    DndContext: ({ children, onDragEnd }: { children: React.ReactNode; onDragEnd: typeof dnd.onDragEnd }) => {
      dnd.onDragEnd = onDragEnd;
      return <>{children}</>;
    },
  };
});

import { Board } from "./Board";

const tasks = [makeTask({ id: 1, status: "todo", title: "Card one" }), makeTask({ id: 2, status: "doing", title: "Card two" })];

function drop(active: string, over: string | null) {
  dnd.onDragEnd?.({ active: { id: active }, over: over === null ? null : { id: over } } as never);
}

describe("Board drag and drop", () => {
  test("dropping a card on another column reports the status change", () => {
    const onMove = vi.fn();
    render(<Board tasks={tasks} milestoneId={null} onMove={onMove} onOpen={vi.fn()} onOpenPr={vi.fn()} />);
    drop("task:1", "column:review");
    expect(onMove).toHaveBeenCalledWith({ id: 1, status: "review" });
  });

  test("dropping on its own column, on nothing, or on a non-column does nothing", () => {
    const onMove = vi.fn();
    render(<Board tasks={tasks} milestoneId={null} onMove={onMove} onOpen={vi.fn()} onOpenPr={vi.fn()} />);
    drop("task:1", "column:todo");
    drop("task:1", null);
    drop("task:1", "task:2");
    drop("bogus", "column:done");
    expect(onMove).not.toHaveBeenCalled();
  });

  test("renders all six columns", () => {
    render(<Board tasks={tasks} milestoneId={null} onMove={vi.fn()} onOpen={vi.fn()} onOpenPr={vi.fn()} />);
    for (const name of ["Backlog", "Todo", "Doing", "Blocked", "Review", "Done"]) {
      expect(screen.getByRole("region", { name })).toBeInTheDocument();
    }
  });
});
