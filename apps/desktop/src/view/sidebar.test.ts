import { describe, expect, test } from "vitest";
import type { ProjectWithCounts } from "../core/types";
import { defaultSelection, selectionFilter, selectionForTask, selectionKey, sidebarItems } from "./sidebar";
import { makeTask } from "../test/fixtures";

const counts = { backlog: 1, todo: 2, doing: 1, blocked: 0, review: 1, done: 5, cancelled: 2 };
const project = (id: number, name: string): ProjectWithCounts => ({
  id,
  name,
  repo_slug: null,
  repo_path: null,
  created_at: "",
  updated_at: "",
  archived_at: null,
  counts,
});

describe("sidebarItems", () => {
  test("lists Inbox first, then projects with their open task counts", () => {
    const items = sidebarItems([project(2, "widgets"), project(5, "docs")]);
    expect(items.map((i) => i.label)).toEqual(["Inbox", "widgets", "docs"]);
    expect(items[1]?.open).toBe(5); // backlog+todo+doing+blocked+review, not done/cancelled
    expect(items[0]?.selection).toEqual({ kind: "inbox" });
    expect(items[1]?.selection).toEqual({ kind: "project", id: 2 });
  });
});

describe("selection", () => {
  test("defaults to the first project, or the inbox when there are none", () => {
    expect(defaultSelection([project(2, "widgets")])).toEqual({ kind: "project", id: 2 });
    expect(defaultSelection([])).toEqual({ kind: "inbox" });
  });

  test("maps to the task filter the API understands", () => {
    expect(selectionFilter({ kind: "inbox" })).toEqual({ inbox: true });
    expect(selectionFilter({ kind: "project", id: 4 })).toEqual({ project_id: 4 });
  });

  test("a task belongs to its project or the inbox", () => {
    expect(selectionForTask(makeTask({ project_id: 3 }))).toEqual({ kind: "project", id: 3 });
    expect(selectionForTask(makeTask({ project_id: null }))).toEqual({ kind: "inbox" });
  });

  test("keys are stable and distinct", () => {
    expect(selectionKey({ kind: "inbox" })).toBe("inbox");
    expect(selectionKey({ kind: "project", id: 4 })).toBe("project-4");
  });
});
