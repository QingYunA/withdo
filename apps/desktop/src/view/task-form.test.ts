import { describe, expect, test } from "vitest";
import { makeTask } from "../test/fixtures";
import { buildPatch, formError, fromLocalInput, toFormValues, toLocalInput } from "./task-form";

describe("local datetime conversion", () => {
  test("round-trips an ISO timestamp at minute precision", () => {
    const iso = new Date(2026, 9, 14, 9, 30).toISOString();
    expect(fromLocalInput(toLocalInput(iso))).toBe(iso);
  });

  test("renders the local wall-clock time", () => {
    expect(toLocalInput(new Date(2026, 9, 14, 9, 5).toISOString())).toBe("2026-10-14T09:05");
  });

  test("empty means no value", () => {
    expect(toLocalInput(null)).toBe("");
    expect(fromLocalInput("")).toBeNull();
    expect(fromLocalInput("garbage")).toBeNull();
  });
});

describe("buildPatch", () => {
  const task = makeTask({
    title: "Write docs",
    notes: "outline first",
    status: "doing",
    priority: "P1",
    milestone_id: 3,
    due_at: new Date(2026, 9, 20, 17, 0).toISOString(),
  });

  test("is empty when nothing changed", () => {
    expect(buildPatch(task, toFormValues(task))).toEqual({});
  });

  test("contains only the fields that changed", () => {
    const form = { ...toFormValues(task), title: "  Write the docs ", status: "review" as const, priority: "P0" as const };
    expect(buildPatch(task, form)).toEqual({ title: "Write the docs", status: "review", priority: "P0" });
  });

  test("clears nullable fields with null", () => {
    const form = { ...toFormValues(task), notes: "", milestone_id: "", due_at: "" };
    expect(buildPatch(task, form)).toEqual({ notes: null, milestone_id: null, due_at: null });
  });

  test("sets a milestone, a due date and a reminder", () => {
    const bare = makeTask();
    const form = { ...toFormValues(bare), milestone_id: "9", remind_at: "2026-11-01T08:00", notes: "hi" };
    expect(buildPatch(bare, form)).toEqual({
      milestone_id: 9,
      remind_at: new Date(2026, 10, 1, 8, 0).toISOString(),
      notes: "hi",
    });
  });
});

describe("formError", () => {
  test("requires a title", () => {
    expect(formError({ ...toFormValues(makeTask()), title: "   " })).toBe("Title is required");
    expect(formError(toFormValues(makeTask()))).toBeNull();
  });
});
