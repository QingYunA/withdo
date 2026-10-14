import { describe, expect, test } from "bun:test";
import { openMemoryDb } from "../src/db/open.ts";
import { createMilestone } from "../src/domain/milestones.ts";
import { upsertProject } from "../src/domain/projects.ts";
import { createTask, findOpenTask, getTask, listTasks, parsePrNumber, updateTask } from "../src/domain/tasks.ts";

function setup() {
  const db = openMemoryDb();
  const project = upsertProject(db, { repo_slug: "acme/widgets" });
  return { db, project };
}

describe("parsePrNumber", () => {
  test.each([
    ["https://github.com/acme/widgets/pull/42", 42],
    ["https://github.com/acme/widgets/pull/42/files", 42],
    ["https://github.com/acme/widgets/issues/42", null],
    ["https://gitlab.com/acme/widgets/pull/42", null],
    ["not a url", null],
  ])("%s", (url, n) => {
    expect(parsePrNumber(url)).toBe(n);
  });
});

describe("createTask", () => {
  test("defaults", () => {
    const { db, project } = setup();
    const t = createTask(db, { title: "Write docs", project_id: project.id, source: "user" });
    expect(t).toMatchObject({
      title: "Write docs",
      project_id: project.id,
      status: "todo",
      priority: "P2",
      source: "user",
      completed_at: null,
      pr_number: null,
    });
    expect(t.created_at).toMatch(/^\d{4}-\d\d-\d\dT.*Z$/);
    expect(getTask(db, t.id)).toEqual(t);
  });

  test("null project is inbox", () => {
    const { db } = setup();
    expect(createTask(db, { title: "x", source: "user" }).project_id).toBeNull();
  });

  test("pr_url sets pr_number", () => {
    const { db } = setup();
    const t = createTask(db, { title: "x", source: "agent", pr_url: "https://github.com/a/b/pull/7" });
    expect(t.pr_number).toBe(7);
  });

  test("milestone must belong to the project; project inferred from it", () => {
    const { db, project } = setup();
    const other = upsertProject(db, { repo_slug: "acme/other" });
    const m = createMilestone(db, { project_id: project.id, title: "v1" });
    expect(() => createTask(db, { title: "x", source: "user", project_id: other.id, milestone_id: m.id })).toThrow(
      /milestone_id/,
    );
    expect(createTask(db, { title: "x", source: "user", milestone_id: m.id }).project_id).toBe(project.id);
    expect(() => createTask(db, { title: "x", source: "user", milestone_id: 99 })).toThrow(/milestone_id/);
  });

  test("unknown project errors", () => {
    const { db } = setup();
    expect(() => createTask(db, { title: "x", source: "user", project_id: 99 })).toThrow(/project_id/);
  });

  test("creating as done sets completed_at", () => {
    const { db } = setup();
    expect(createTask(db, { title: "x", source: "user", status: "done" }).completed_at).not.toBeNull();
  });
});

describe("updateTask", () => {
  test("patches fields without mutating the input", () => {
    const { db } = setup();
    const t = createTask(db, { title: "a", source: "user" });
    const patch = Object.freeze({ title: "b", priority: "P0" as const, notes: "n" });
    const u = updateTask(db, t.id, patch);
    expect(u).toMatchObject({ title: "b", priority: "P0", notes: "n", status: "todo" });
    expect(u.updated_at >= t.updated_at).toBe(true);
    expect(t.title).toBe("a");
  });

  test("done and cancelled set completed_at; moving back clears it", () => {
    const { db } = setup();
    const t = createTask(db, { title: "a", source: "user" });
    const done = updateTask(db, t.id, { status: "done" });
    expect(done.completed_at).not.toBeNull();
    const cancelled = updateTask(db, t.id, { status: "cancelled" });
    expect(cancelled.completed_at).toBe(done.completed_at);
    expect(updateTask(db, t.id, { status: "doing" }).completed_at).toBeNull();
  });

  test("pr_url updates and clears pr_number; null clears fields", () => {
    const { db } = setup();
    const t = createTask(db, { title: "a", source: "user", notes: "keep", due_at: "2026-01-01T00:00:00.000Z" });
    const withPr = updateTask(db, t.id, { pr_url: "https://github.com/a/b/pull/9" });
    expect(withPr.pr_number).toBe(9);
    const cleared = updateTask(db, t.id, { pr_url: null, notes: null, due_at: null });
    expect(cleared).toMatchObject({ pr_url: null, pr_number: null, notes: null, due_at: null });
  });

  test("errors name the field", () => {
    const { db } = setup();
    const t = createTask(db, { title: "a", source: "user" });
    expect(() => updateTask(db, 999, { title: "x" })).toThrow(/task_id/);
    expect(() => updateTask(db, t.id, { project_id: 999 })).toThrow(/project_id/);
    expect(() => updateTask(db, t.id, { milestone_id: 999 })).toThrow(/milestone_id/);
  });

  test("can move task to a project and attach a milestone", () => {
    const { db, project } = setup();
    const t = createTask(db, { title: "a", source: "user" });
    const m = createMilestone(db, { project_id: project.id, title: "v1" });
    const u = updateTask(db, t.id, { project_id: project.id, milestone_id: m.id });
    expect(u).toMatchObject({ project_id: project.id, milestone_id: m.id });
  });
});

describe("listTasks / findOpenTask", () => {
  test("filters, newest first, limit", () => {
    const { db, project } = setup();
    const a = createTask(db, { title: "a", source: "user", project_id: project.id, status: "doing" });
    const b = createTask(db, { title: "b", source: "user", project_id: project.id, status: "todo" });
    const c = createTask(db, { title: "c", source: "user", status: "doing" });
    expect(listTasks(db, {}).map((t) => t.id)).toEqual([c.id, b.id, a.id]);
    expect(listTasks(db, { project_id: project.id }).map((t) => t.id)).toEqual([b.id, a.id]);
    expect(listTasks(db, { status: ["doing"] }).map((t) => t.id)).toEqual([c.id, a.id]);
    expect(listTasks(db, { limit: 1 })).toHaveLength(1);
    expect(listTasks(db, { status: [] })).toHaveLength(3);
  });

  test("milestone filter", () => {
    const { db, project } = setup();
    const m = createMilestone(db, { project_id: project.id, title: "v1" });
    const a = createTask(db, { title: "a", source: "user", milestone_id: m.id });
    createTask(db, { title: "b", source: "user", project_id: project.id });
    expect(listTasks(db, { milestone_id: m.id }).map((t) => t.id)).toEqual([a.id]);
  });

  test("findOpenTask matches project+branch and ignores closed tasks", () => {
    const { db, project } = setup();
    const t = createTask(db, { title: "a", source: "agent", project_id: project.id, branch: "feat/x" });
    expect(findOpenTask(db, project.id, "feat/x")?.id).toBe(t.id);
    expect(findOpenTask(db, project.id, "feat/y")).toBeNull();
    expect(findOpenTask(db, null, "feat/x")).toBeNull();
    updateTask(db, t.id, { status: "done" });
    expect(findOpenTask(db, project.id, "feat/x")).toBeNull();
  });
});
