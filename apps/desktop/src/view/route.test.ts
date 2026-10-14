import { describe, expect, test } from "vitest";
import { parseRoute, taskHash } from "./route";

describe("parseRoute", () => {
  test("recognises the panel", () => {
    expect(parseRoute("#/panel")).toEqual({ name: "panel" });
  });

  test("main defaults to the board with no task", () => {
    expect(parseRoute("#/main")).toEqual({ name: "main", view: "board", taskId: null });
    expect(parseRoute("")).toEqual({ name: "main", view: "board", taskId: null });
    expect(parseRoute("#/nonsense")).toEqual({ name: "main", view: "board", taskId: null });
  });

  test("main can deep-link to a task or to settings", () => {
    expect(parseRoute("#/main/task/42")).toEqual({ name: "main", view: "board", taskId: 42 });
    expect(parseRoute("#/main/settings")).toEqual({ name: "main", view: "settings", taskId: null });
  });

  test("a malformed task id falls back to no task", () => {
    expect(parseRoute("#/main/task/abc")).toEqual({ name: "main", view: "board", taskId: null });
    expect(parseRoute("#/main/task/0")).toEqual({ name: "main", view: "board", taskId: null });
  });
});

describe("taskHash", () => {
  test("round-trips through parseRoute", () => {
    expect(parseRoute(taskHash(7))).toMatchObject({ taskId: 7 });
  });
});
