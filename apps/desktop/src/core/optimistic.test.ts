import { describe, expect, test } from "vitest";
import { makeTask } from "../test/fixtures";
import { withTaskStatus } from "./optimistic";

describe("withTaskStatus", () => {
  test("returns a new list with the one task moved, leaving the input untouched", () => {
    const tasks = [makeTask({ id: 1, status: "todo" }), makeTask({ id: 2, status: "todo" })];
    const next = withTaskStatus(tasks, 2, "doing")!;
    expect(next.map((t) => t.status)).toEqual(["todo", "doing"]);
    expect(tasks[1]?.status).toBe("todo");
    expect(next[0]).toBe(tasks[0]);
  });

  test("leaves the list alone for an unknown id and passes undefined through", () => {
    const tasks = [makeTask({ id: 1 })];
    expect(withTaskStatus(tasks, 99, "done")).toEqual(tasks);
    expect(withTaskStatus(undefined, 1, "done")).toBeUndefined();
  });
});
