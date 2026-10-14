import { describe, expect, test } from "vitest";
import { makeTask } from "../test/fixtures";
import { KANBAN_STATUSES, columnId, groupByColumn, resolveDrop, statusFromColumnId, taskDragId, taskIdFromDragId } from "./kanban";

describe("column ids", () => {
  test("round-trip every status", () => {
    for (const status of KANBAN_STATUSES) expect(statusFromColumnId(columnId(status))).toBe(status);
  });

  test("reject ids that are not columns", () => {
    expect(statusFromColumnId("task:3")).toBeNull();
    expect(statusFromColumnId("column:cancelled")).toBeNull();
    expect(statusFromColumnId(5)).toBeNull();
  });
});

describe("groupByColumn", () => {
  test("puts each task in its status column and hides cancelled ones", () => {
    const tasks = [
      makeTask({ id: 1, status: "todo" }),
      makeTask({ id: 2, status: "doing" }),
      makeTask({ id: 3, status: "cancelled" }),
      makeTask({ id: 4, status: "todo" }),
    ];
    const columns = groupByColumn(tasks, null);
    expect(Object.keys(columns)).toEqual([...KANBAN_STATUSES]);
    expect(columns.todo.map((t) => t.id).sort()).toEqual([1, 4]);
    expect(columns.doing.map((t) => t.id)).toEqual([2]);
    expect(Object.values(columns).flat().map((t) => t.id)).not.toContain(3);
    expect(columns.done).toEqual([]);
  });

  test("orders a column by priority, then newest first", () => {
    const tasks = [
      makeTask({ id: 1, priority: "P2" }),
      makeTask({ id: 2, priority: "P0" }),
      makeTask({ id: 3, priority: "P2" }),
      makeTask({ id: 4, priority: "P3" }),
    ];
    expect(groupByColumn(tasks, null).todo.map((t) => t.id)).toEqual([2, 3, 1, 4]);
  });

  test("filters by milestone", () => {
    const tasks = [makeTask({ id: 1, milestone_id: 7 }), makeTask({ id: 2, milestone_id: null }), makeTask({ id: 3, milestone_id: 8 })];
    expect(groupByColumn(tasks, 7).todo.map((t) => t.id)).toEqual([1]);
    expect(groupByColumn(tasks, null).todo).toHaveLength(3);
  });
});

describe("resolveDrop", () => {
  const tasks = [makeTask({ id: 1, status: "todo" }), makeTask({ id: 2, status: "doing" })];

  test("moving to another column yields the status change", () => {
    expect(resolveDrop(tasks, 1, "column:review")).toEqual({ id: 1, status: "review" });
  });

  test("dropping on the same column or on nothing changes nothing", () => {
    expect(resolveDrop(tasks, 1, "column:todo")).toBeNull();
    expect(resolveDrop(tasks, 1, null)).toBeNull();
  });

  test("ignores drops on non-column targets and unknown tasks", () => {
    expect(resolveDrop(tasks, 1, "task:2")).toBeNull();
    expect(resolveDrop(tasks, 99, "column:done")).toBeNull();
  });
});

describe("task drag ids", () => {
  test("round-trip and reject other ids", () => {
    expect(taskIdFromDragId(taskDragId(12))).toBe(12);
    expect(taskIdFromDragId("column:todo")).toBeNull();
    expect(taskIdFromDragId("task:abc")).toBeNull();
    expect(taskIdFromDragId(3)).toBeNull();
  });
});
