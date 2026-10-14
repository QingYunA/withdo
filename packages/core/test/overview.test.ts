import { describe, expect, test } from "bun:test";
import { openMemoryDb } from "../src/db/open.ts";
import { getOverview } from "../src/domain/overview.ts";
import { upsertProject } from "../src/domain/projects.ts";
import { addReport } from "../src/domain/reports.ts";
import { createTask } from "../src/domain/tasks.ts";

const T0 = new Date("2026-10-01T00:00:00.000Z");
const hoursAfter = (h: number) => new Date(T0.getTime() + h * 3_600_000);

function seed() {
  const db = openMemoryDb();
  const p = upsertProject(db, { repo_slug: "acme/widgets" });
  db.run("UPDATE projects SET created_at = ?, updated_at = ?", [T0.toISOString(), T0.toISOString()]);
  return { db, p };
}

function task(db: ReturnType<typeof openMemoryDb>, projectId: number | null, status: string, title: string) {
  const t = createTask(db, { title, project_id: projectId ?? undefined, status: status as never, source: "agent" });
  db.run("UPDATE tasks SET created_at = ?, updated_at = ? WHERE id = ?", [T0.toISOString(), T0.toISOString(), t.id]);
  return t;
}

describe("getOverview", () => {
  test("only doing/blocked/review tasks, grouped by project", () => {
    const { db, p } = seed();
    task(db, p.id, "doing", "a");
    task(db, p.id, "blocked", "b");
    task(db, p.id, "review", "c");
    task(db, p.id, "todo", "d");
    task(db, p.id, "done", "e");
    const o = getOverview(db, { now: hoursAfter(1) });
    expect(o.projects).toHaveLength(1);
    expect(o.projects[0]!.project?.repo_slug).toBe("acme/widgets");
    expect(o.projects[0]!.tasks.map((t) => t.title)).toEqual(["b", "c", "a"]);
  });

  test("latest report is attached and stale is based on it", () => {
    const { db, p } = seed();
    const t = task(db, p.id, "doing", "a");
    addReport(db, { task_id: t.id, agent: "x", summary: "old", created_at: hoursAfter(1).toISOString() });
    addReport(db, { task_id: t.id, agent: "x", summary: "newer", created_at: hoursAfter(40).toISOString() });
    const fresh = getOverview(db, { now: hoursAfter(60) }).projects[0]!.tasks[0]!;
    expect(fresh.latest_report?.summary).toBe("newer");
    expect(fresh.stale).toBe(false);
    const stale = getOverview(db, { now: hoursAfter(40 + 49) }).projects[0]!.tasks[0]!;
    expect(stale.stale).toBe(true);
    expect(stale.hours_since_activity).toBe(49);
  });

  test("task with no report falls back to updated_at; stale_hours is configurable", () => {
    const { db, p } = seed();
    task(db, p.id, "doing", "a");
    expect(getOverview(db, { now: hoursAfter(47) }).projects[0]!.tasks[0]!.stale).toBe(false);
    expect(getOverview(db, { now: hoursAfter(49) }).projects[0]!.tasks[0]!.stale).toBe(true);
    expect(getOverview(db, { now: hoursAfter(10), staleHours: 5 }).projects[0]!.tasks[0]!.stale).toBe(true);
    expect(getOverview(db, { now: hoursAfter(10), staleHours: 5 }).stale_hours).toBe(5);
  });

  test("inbox tasks appear under a null project; archived projects and empty projects are hidden", () => {
    const { db, p } = seed();
    upsertProject(db, { repo_slug: "acme/empty" });
    const arch = upsertProject(db, { repo_slug: "acme/arch" });
    task(db, arch.id, "doing", "hidden");
    db.run("UPDATE projects SET archived_at = ? WHERE id = ?", [T0.toISOString(), arch.id]);
    task(db, null, "blocked", "inbox-one");
    task(db, p.id, "doing", "a");
    const o = getOverview(db, { now: hoursAfter(1) });
    expect(o.projects.map((g) => g.project?.name ?? null)).toEqual(["widgets", null]);
    expect(o.projects[1]!.tasks[0]!.title).toBe("inbox-one");
  });

  test("empty database", () => {
    expect(getOverview(openMemoryDb(), { now: T0 }).projects).toEqual([]);
  });
});
