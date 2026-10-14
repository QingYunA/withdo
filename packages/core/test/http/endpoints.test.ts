import { afterAll, describe, expect, test } from "bun:test";
import { createMilestone } from "../../src/domain/milestones.ts";
import { upsertProject } from "../../src/domain/projects.ts";
import { addReport } from "../../src/domain/reports.ts";
import { createTask } from "../../src/domain/tasks.ts";
import { patchJson, postJson, startTestApi } from "../helpers/http.ts";

const api = startTestApi();
afterAll(() => api.stop());

const project = upsertProject(api.db, { repo_slug: "acme/widgets" });
const other = upsertProject(api.db, { repo_slug: "acme/other" });
const milestone = createMilestone(api.db, { project_id: project.id, title: "v1" });

describe("GET /overview", () => {
  test("returns the get_overview shape wrapped in data", async () => {
    const t = createTask(api.db, { title: "agent work", source: "agent", project_id: project.id, status: "doing" });
    addReport(api.db, { task_id: t.id, agent: "claude-code", summary: "working" });
    const { status, body } = await api.json("/overview");
    expect(status).toBe(200);
    expect(body.data.stale_hours).toBe(48);
    const group = body.data.projects.find((g: any) => g.project?.id === project.id);
    expect(group.tasks[0]).toMatchObject({ id: t.id, stale: false, latest_report: { summary: "working" } });
  });

  test("stale_hours is honoured and validated", async () => {
    expect((await api.json("/overview?stale_hours=1.5")).body.data.stale_hours).toBe(1.5);
    for (const bad of ["abc", "0", "-3"]) {
      const { status, body } = await api.json(`/overview?stale_hours=${bad}`);
      expect(status).toBe(400);
      expect(body.error).toContain("stale_hours");
    }
  });
});

describe("GET /projects", () => {
  test("lists non-archived projects with counts per status", async () => {
    const archived = upsertProject(api.db, { repo_slug: "acme/old" });
    api.db.run("UPDATE projects SET archived_at = 'x' WHERE id = ?", [archived.id]);
    const { status, body } = await api.json("/projects");
    expect(status).toBe(200);
    const ids = body.data.map((p: any) => p.id);
    expect(ids).toContain(project.id);
    expect(ids).not.toContain(archived.id);
    const mine = body.data.find((p: any) => p.id === project.id);
    expect(mine.counts.doing).toBe(1);
    expect(mine.counts.todo).toBe(0);
  });
});

describe("tasks", () => {
  test("POST /tasks creates a user task in the inbox by default", async () => {
    const { status, body } = await api.json("/tasks", postJson({ title: "buy milk" }));
    expect(status).toBe(201);
    expect(body.data).toMatchObject({
      title: "buy milk", source: "user", project_id: null, status: "todo", priority: "P2", reminded_at: null,
    });
    const fetched = await api.json(`/tasks/${body.data.id}`);
    expect(fetched.body.data.task.title).toBe("buy milk");
  });

  test("POST /tasks accepts the create_task fields", async () => {
    const { status, body } = await api.json(
      "/tasks",
      postJson({
        title: "ship it",
        project_id: project.id,
        milestone_id: milestone.id,
        priority: "P0",
        notes: "n",
        due_at: "2026-03-01",
        remind_at: "2026-03-01T09:00:00Z",
        branch: "feat/x",
        pr_url: "https://github.com/acme/widgets/pull/7",
      }),
    );
    expect(status).toBe(201);
    expect(body.data).toMatchObject({
      project_id: project.id, milestone_id: milestone.id, priority: "P0", pr_number: 7,
      due_at: "2026-03-01T00:00:00.000Z", remind_at: "2026-03-01T09:00:00.000Z",
    });
  });

  test("POST /tasks rejects bad input naming the field", async () => {
    const cases: Array<[unknown, string]> = [
      [{}, "title"],
      [{ title: "" }, "title"],
      [{ title: "x", priority: "P9" }, "priority"],
      [{ title: "x", remind_at: "tomorrow" }, "remind_at"],
      [{ title: "x", source: "agent" }, "source"],
      [{ title: "x", cwd: "/tmp" }, "cwd"],
      [{ title: "x", project_id: 999 }, "project_id"],
      [[], "expected object"],
    ];
    for (const [payload, field] of cases) {
      const { status, body } = await api.json("/tasks", postJson(payload));
      expect(status).toBe(payload && (payload as any).project_id === 999 ? 404 : 400);
      expect(body.error).toContain(field);
      expect(body.error).not.toMatch(/\n\s+at /);
    }
  });

  test("POST /tasks with a malformed body is 400", async () => {
    const { status, body } = await api.json("/tasks", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: "{nope",
    });
    expect(status).toBe(400);
    expect(body.error).toContain("JSON");
  });

  test("GET /tasks filters by project, inbox, repeatable status and milestone", async () => {
    const a = createTask(api.db, { title: "a", source: "user", project_id: other.id, status: "doing" });
    const b = createTask(api.db, { title: "b", source: "user", project_id: other.id, status: "blocked" });
    createTask(api.db, { title: "c", source: "user", project_id: other.id, status: "done" });
    const ids = async (qs: string) => (await api.json(`/tasks?${qs}`)).body.data.map((t: any) => t.id);

    expect((await ids(`project_id=${other.id}`)).length).toBe(3);
    expect((await ids(`project_id=${other.id}&status=doing&status=blocked`)).sort()).toEqual([a.id, b.id].sort());
    expect(await ids(`project_id=${other.id}&status=done`)).toHaveLength(1);
    const inbox = (await api.json("/tasks?inbox=1")).body.data;
    expect(inbox.length).toBeGreaterThan(0);
    expect(inbox.every((t: any) => t.project_id === null)).toBe(true);
    const inMilestone = createTask(api.db, { title: "m", source: "user", milestone_id: milestone.id });
    expect(await ids(`milestone_id=${milestone.id}`)).toContain(inMilestone.id);
    expect((await api.json("/tasks?limit=1")).body.data).toHaveLength(1);
  });

  test("GET /tasks rejects invalid filters naming the field", async () => {
    for (const [qs, field] of [
      ["project_id=abc", "project_id"],
      ["project_id=1&inbox=1", "inbox"],
      ["inbox=yes", "inbox"],
      ["status=nope", "status"],
      ["milestone_id=-1", "milestone_id"],
      ["limit=0", "limit"],
      ["limit=5000", "limit"],
      ["cwd=/tmp", "cwd"],
    ] as const) {
      const { status, body } = await api.json(`/tasks?${qs}`);
      expect(status).toBe(400);
      expect(body.error).toContain(field);
    }
  });

  test("GET /tasks/:id returns the task and its reports oldest first", async () => {
    const t = createTask(api.db, { title: "with reports", source: "agent", project_id: project.id });
    addReport(api.db, { task_id: t.id, agent: "a", summary: "one", created_at: "2026-01-01T00:00:00.000Z" });
    addReport(api.db, { task_id: t.id, agent: "a", summary: "two", created_at: "2026-01-02T00:00:00.000Z" });
    const { status, body } = await api.json(`/tasks/${t.id}`);
    expect(status).toBe(200);
    expect(body.data.task.id).toBe(t.id);
    expect(body.data.reports.map((r: any) => r.summary)).toEqual(["one", "two"]);
  });

  test("GET /tasks/:id: unknown is 404, malformed is 400", async () => {
    const missing = await api.json("/tasks/99999");
    expect(missing.status).toBe(404);
    expect(missing.body.error).toContain("task");
    for (const bad of ["abc", "0", "1.5", "-1"]) {
      const res = await api.json(`/tasks/${bad}`);
      expect(res.status).toBe(400);
      expect(res.body.error).toContain("task_id");
    }
  });

  test("PATCH /tasks/:id updates fields and applies the domain rules", async () => {
    const created = (await api.json("/tasks", postJson({ title: "p", remind_at: "2026-01-01T09:00:00Z" }))).body.data;
    api.db.run("UPDATE tasks SET reminded_at = 'x' WHERE id = ?", [created.id]);

    const done = await api.json(`/tasks/${created.id}`, patchJson({ status: "done", notes: "fin" }));
    expect(done.status).toBe(200);
    expect(done.body.data).toMatchObject({ status: "done", notes: "fin", reminded_at: "x" });
    expect(done.body.data.completed_at).not.toBeNull();

    const rearmed = await api.json(`/tasks/${created.id}`, patchJson({ status: "todo", remind_at: "2026-05-05T09:00:00Z" }));
    expect(rearmed.body.data).toMatchObject({ completed_at: null, reminded_at: null });

    const cleared = await api.json(`/tasks/${created.id}`, patchJson({ remind_at: null, notes: null }));
    expect(cleared.body.data).toMatchObject({ remind_at: null, notes: null });
  });

  test("PATCH /tasks/:id errors", async () => {
    const t = createTask(api.db, { title: "e", source: "user" });
    expect((await api.json("/tasks/99999", patchJson({ title: "x" }))).status).toBe(404);
    expect((await api.json("/tasks/abc", patchJson({ title: "x" }))).status).toBe(400);
    for (const [payload, field] of [
      [{ title: "" }, "title"],
      [{ status: "nope" }, "status"],
      [{ task_id: 5 }, "task_id"],
      [{ pr_url: "nope" }, "pr_url"],
    ] as const) {
      const { status, body } = await api.json(`/tasks/${t.id}`, patchJson(payload));
      expect(status).toBe(400);
      expect(body.error).toContain(field);
    }
    const missingRef = await api.json(`/tasks/${t.id}`, patchJson({ milestone_id: 99999 }));
    expect(missingRef.status).toBe(404);
    expect(missingRef.body.error).toContain("milestone_id");
  });
});

describe("milestones", () => {
  test("POST /milestones then GET /projects/:id/milestones", async () => {
    const created = await api.json("/milestones", postJson({ project_id: other.id, title: "beta", due_at: "2026-06-01" }));
    expect(created.status).toBe(201);
    expect(created.body.data).toMatchObject({ project_id: other.id, title: "beta", due_at: "2026-06-01T00:00:00.000Z" });
    const list = await api.json(`/projects/${other.id}/milestones`);
    expect(list.status).toBe(200);
    expect(list.body.data.map((m: any) => m.title)).toEqual(["beta"]);
  });

  test("errors name the field", async () => {
    const cases: Array<[unknown, number, string]> = [
      [{ title: "x" }, 400, "project_id"],
      [{ project_id: other.id }, 400, "title"],
      [{ project_id: other.id, title: "x", cwd: "/tmp" }, 400, "cwd"],
      [{ project_id: 99999, title: "x" }, 404, "project_id"],
    ];
    for (const [payload, expected, field] of cases) {
      const { status, body } = await api.json("/milestones", postJson(payload));
      expect(status).toBe(expected);
      expect(body.error).toContain(field);
    }
    const missing = await api.json("/projects/99999/milestones");
    expect(missing.status).toBe(404);
    expect(missing.body.error).toContain("project_id");
    expect((await api.json("/projects/abc/milestones")).status).toBe(400);
  });
});
