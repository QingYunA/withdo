import { describe, expect, test } from "vitest";
import { makeTask } from "../test/fixtures";
import { mineTasks, moveSelection, taskWhen } from "./mine";

const now = new Date("2026-10-14T12:00:00.000Z");
const at = (h: number) => new Date(now.getTime() + h * 3_600_000).toISOString();

describe("mineTasks", () => {
  test("keeps only the user's todo and doing tasks", () => {
    const tasks = [
      makeTask({ id: 1, source: "user", status: "todo" }),
      makeTask({ id: 2, source: "agent", status: "doing" }),
      makeTask({ id: 3, source: "user", status: "done" }),
      makeTask({ id: 4, source: "user", status: "backlog" }),
      makeTask({ id: 5, source: "user", status: "doing" }),
    ];
    expect(mineTasks(tasks).map((t) => t.id).sort()).toEqual([1, 5]);
  });

  test("sorts by the earlier of remind and due time, undated tasks last", () => {
    const tasks = [
      makeTask({ id: 1 }),
      makeTask({ id: 2, due_at: at(10) }),
      makeTask({ id: 3, due_at: at(48), remind_at: at(2) }),
      makeTask({ id: 4, remind_at: at(5) }),
    ];
    expect(mineTasks(tasks).map((t) => t.id)).toEqual([3, 4, 2, 1]);
  });

  test("breaks ties by priority, then newest first", () => {
    const tasks = [
      makeTask({ id: 1, priority: "P2" }),
      makeTask({ id: 2, priority: "P0" }),
      makeTask({ id: 3, priority: "P2" }),
    ];
    expect(mineTasks(tasks).map((t) => t.id)).toEqual([2, 3, 1]);
  });

  test("does not mutate its input", () => {
    const tasks = [makeTask({ id: 1, due_at: at(5) }), makeTask({ id: 2, due_at: at(1) })];
    mineTasks(tasks);
    expect(tasks.map((t) => t.id)).toEqual([1, 2]);
  });
});

describe("taskWhen", () => {
  test("is null when nothing is scheduled", () => {
    expect(taskWhen(makeTask(), now)).toBeNull();
  });

  test("describes the earlier of reminder and due date", () => {
    expect(taskWhen(makeTask({ remind_at: at(1), due_at: at(30) }), now)).toEqual({ text: "Remind in 1h", overdue: false });
    expect(taskWhen(makeTask({ due_at: at(30) }), now)).toEqual({ text: "Due in 1d", overdue: false });
    expect(taskWhen(makeTask({ due_at: at(-3) }), now)).toEqual({ text: "Due 3h ago", overdue: true });
  });
});

describe("moveSelection", () => {
  test("starts at the first row going down and the last going up", () => {
    expect(moveSelection(null, 1, 3)).toBe(0);
    expect(moveSelection(null, -1, 3)).toBe(2);
  });

  test("wraps at both ends", () => {
    expect(moveSelection(2, 1, 3)).toBe(0);
    expect(moveSelection(0, -1, 3)).toBe(2);
    expect(moveSelection(1, 1, 3)).toBe(2);
  });

  test("is null when there is nothing to select", () => {
    expect(moveSelection(null, 1, 0)).toBeNull();
    expect(moveSelection(5, 1, 0)).toBeNull();
  });
});
