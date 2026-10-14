import { describe, expect, test } from "bun:test";
import { openMemoryDb } from "../src/db/open.ts";
import { listReports, reportProgress } from "../src/domain/reports.ts";
import { getTask, updateTask } from "../src/domain/tasks.ts";
import { listProjects } from "../src/domain/projects.ts";
import { makeRepo } from "./helpers/repo.ts";
import { useTmpDir } from "./helpers/tmp.ts";

const tmp = useTmpDir();

function repo(branch = "feat/login") {
  return makeRepo(tmp(), "widgets", { remote: "git@github.com:acme/widgets.git", branch });
}

describe("reportProgress", () => {
  test("creates a doing task from cwd, titled by branch, with a report", () => {
    const db = openMemoryDb();
    const dir = repo();
    const r = reportProgress(db, { summary: "started", cwd: dir, agent: "claude-code" });
    expect(r.created).toBe(true);
    expect(r.task).toMatchObject({ title: "feat/login", branch: "feat/login", status: "doing", source: "agent" });
    expect(r.report).toMatchObject({ agent: "claude-code", summary: "started", status: null });
    expect(listProjects(db)).toHaveLength(1);
    expect(r.task.project_id).toBe(listProjects(db)[0]!.id);
  });

  test("reuses the task on the same branch and applies status, pr_url, next_step", () => {
    const db = openMemoryDb();
    const dir = repo();
    const a = reportProgress(db, { summary: "one", cwd: dir, agent: "x" });
    const b = reportProgress(db, {
      summary: "two",
      cwd: dir,
      agent: "x",
      status: "review",
      pr_url: "https://github.com/acme/widgets/pull/12",
      next_step: "wait for CI",
    });
    expect(b.created).toBe(false);
    expect(b.task.id).toBe(a.task.id);
    expect(b.task).toMatchObject({ status: "review", pr_number: 12 });
    expect(b.report).toMatchObject({ status: "review", next_step: "wait for CI" });
    expect(listReports(db, a.task.id).map((r) => r.summary)).toEqual(["one", "two"]);
  });

  test("a different branch gets a different task; closed task is not reused", () => {
    const db = openMemoryDb();
    const dir = repo();
    const a = reportProgress(db, { summary: "one", cwd: dir, agent: "x" });
    updateTask(db, a.task.id, { status: "done" });
    const b = reportProgress(db, { summary: "again", cwd: dir, agent: "x" });
    expect(b.task.id).not.toBe(a.task.id);
    const c = reportProgress(db, { summary: "other", cwd: dir, agent: "x", branch: "feat/other" });
    expect(c.task.branch).toBe("feat/other");
    expect(c.task.id).not.toBe(b.task.id);
  });

  test("title overrides branch for new tasks", () => {
    const db = openMemoryDb();
    const r = reportProgress(db, { summary: "s", cwd: repo(), agent: "x", title: "Add login" });
    expect(r.task.title).toBe("Add login");
  });

  test("task_id targets an existing task, even from another cwd, and can reopen it", () => {
    const db = openMemoryDb();
    const a = reportProgress(db, { summary: "one", cwd: repo(), agent: "x" });
    updateTask(db, a.task.id, { status: "done" });
    const r = reportProgress(db, { summary: "more", task_id: a.task.id, cwd: tmp(), agent: "x", status: "doing" });
    expect(r.task.id).toBe(a.task.id);
    expect(r.task.status).toBe("doing");
    expect(r.task.completed_at).toBeNull();
    expect(listProjects(db)).toHaveLength(1);
  });

  test("unknown task_id is an error naming the field", () => {
    expect(() => reportProgress(openMemoryDb(), { summary: "s", task_id: 99, agent: "x" })).toThrow(/task_id/);
  });

  test("non-git cwd without branch goes to inbox with summary as title", () => {
    const db = openMemoryDb();
    const r = reportProgress(db, { summary: "did a thing", cwd: tmp(), agent: "x" });
    expect(r.task).toMatchObject({ project_id: null, branch: null, title: "did a thing", status: "doing" });
    expect(listProjects(db)).toHaveLength(0);
  });

  test("long summary is truncated for the title", () => {
    const r = reportProgress(openMemoryDb(), { summary: "x".repeat(300), cwd: tmp(), agent: "x" });
    expect(r.task.title.length).toBeLessThanOrEqual(80);
  });

  test("report bumps task.updated_at", () => {
    const db = openMemoryDb();
    const a = reportProgress(db, { summary: "one", cwd: repo(), agent: "x" });
    const b = reportProgress(db, { summary: "two", task_id: a.task.id, agent: "x" });
    expect(getTask(db, a.task.id)?.updated_at).toBe(b.task.updated_at);
  });
});
