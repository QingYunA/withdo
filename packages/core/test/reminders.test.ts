import { describe, expect, test } from "bun:test";
import { openMemoryDb } from "../src/db/open.ts";
import { upsertProject } from "../src/domain/projects.ts";
import { claimDueReminders } from "../src/domain/reminders.ts";
import { createTask, updateTask } from "../src/domain/tasks.ts";

const NOW = "2026-06-01T12:00:00.000Z";
const PAST = "2026-06-01T11:00:00.000Z";
const FUTURE = "2026-06-01T13:00:00.000Z";

function setup() {
  const db = openMemoryDb();
  const project = upsertProject(db, { repo_slug: "acme/widgets" });
  return { db, project };
}

describe("claimDueReminders", () => {
  test("returns due tasks with their project name and marks them reminded", () => {
    const { db, project } = setup();
    const a = createTask(db, { title: "in project", source: "user", project_id: project.id, remind_at: PAST });
    const b = createTask(db, { title: "in inbox", source: "user", remind_at: NOW });
    const due = claimDueReminders(db, NOW);
    expect(due).toEqual([
      { task_id: a.id, title: "in project", project_name: "widgets" },
      { task_id: b.id, title: "in inbox", project_name: null },
    ]);
    const stored = db.query<{ reminded_at: string }, [number]>("SELECT reminded_at FROM tasks WHERE id = ?").get(a.id);
    expect(stored?.reminded_at).toBe(NOW);
  });

  test("fires once per task", () => {
    const { db } = setup();
    createTask(db, { title: "x", source: "user", remind_at: PAST });
    expect(claimDueReminders(db, NOW)).toHaveLength(1);
    expect(claimDueReminders(db, NOW)).toHaveLength(0);
    expect(claimDueReminders(db, "2027-01-01T00:00:00.000Z")).toHaveLength(0);
  });

  test("skips future, unset, done and cancelled tasks", () => {
    const { db } = setup();
    createTask(db, { title: "future", source: "user", remind_at: FUTURE });
    createTask(db, { title: "none", source: "user" });
    createTask(db, { title: "done", source: "user", remind_at: PAST, status: "done" });
    createTask(db, { title: "cancelled", source: "user", remind_at: PAST, status: "cancelled" });
    expect(claimDueReminders(db, NOW)).toEqual([]);
  });

  test("setting remind_at again re-arms the reminder", () => {
    const { db } = setup();
    const t = createTask(db, { title: "x", source: "user", remind_at: PAST });
    expect(claimDueReminders(db, NOW)).toHaveLength(1);
    updateTask(db, t.id, { remind_at: PAST });
    expect(claimDueReminders(db, NOW)).toHaveLength(1);
    expect(claimDueReminders(db, NOW)).toHaveLength(0);
  });

  test("defaults to the current time", () => {
    const { db } = setup();
    createTask(db, { title: "x", source: "user", remind_at: "2000-01-01T00:00:00.000Z" });
    expect(claimDueReminders(db)).toHaveLength(1);
  });
});
