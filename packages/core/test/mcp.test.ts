import { afterEach, describe, expect, test } from "bun:test";
import { connect, textOf, type TestRig } from "./helpers/mcp.ts";
import { makeRepo } from "./helpers/repo.ts";
import { useTmpDir } from "./helpers/tmp.ts";

const tmp = useTmpDir();
const rigs: TestRig[] = [];

async function rig(branch = "feat/login"): Promise<TestRig & { dir: string }> {
  const dir = makeRepo(tmp(), "widgets", { remote: "git@github.com:acme/widgets.git", branch });
  const r = await connect(dir);
  rigs.push(r);
  return Object.assign(r, { dir });
}

afterEach(async () => {
  await Promise.all(rigs.splice(0).map((r) => r.close()));
});

const TOOLS = [
  "report_progress",
  "create_task",
  "update_task",
  "get_task",
  "list_tasks",
  "get_overview",
  "upsert_project",
  "create_milestone",
];

describe("server surface", () => {
  test("lists all 8 tools with object input schemas", async () => {
    const r = await rig();
    const { tools } = await r.client.listTools();
    expect(tools.map((t) => t.name).sort()).toEqual([...TOOLS].sort());
    for (const t of tools) {
      expect(t.description?.length).toBeGreaterThan(10);
      expect(t.inputSchema.type).toBe("object");
    }
  });

  test("instructions tell agents when to report_progress", async () => {
    const r = await rig();
    const text = r.client.getInstructions() ?? "";
    expect(text).toContain("report_progress");
    for (const w of ["start", "blocked", "PR", "finish"]) expect(text).toContain(w);
  });
});

describe("report_progress", () => {
  test("creates a task from the default cwd, then reuses it on the same branch", async () => {
    const r = await rig();
    const a = await r.call("report_progress", { summary: "starting login" });
    expect(a.isError).toBeFalsy();
    expect(a.structuredContent).toMatchObject({ created: true, status: "doing" });
    expect(a.structuredContent?.task.branch).toBe("feat/login");
    expect(textOf(a)).toContain(`#${a.structuredContent?.task_id}`);

    const b = await r.call("report_progress", {
      summary: "PR up",
      status: "review",
      pr_url: "https://github.com/acme/widgets/pull/3",
      next_step: "wait for review",
    });
    expect(b.structuredContent).toMatchObject({ created: false, status: "review", task_id: a.structuredContent?.task_id });
    expect(b.structuredContent?.task.pr_number).toBe(3);
  });

  test("records the MCP client name as the report agent", async () => {
    const r = await rig();
    const a = await r.call("report_progress", { summary: "hi" });
    const got = await r.call("get_task", { task_id: a.structuredContent?.task_id });
    expect(got.structuredContent?.reports[0].agent).toBe("test-agent");
  });

  test("explicit cwd, branch and title", async () => {
    const r = await rig();
    const other = makeRepo(tmp(), "other", { remote: "git@github.com:acme/other.git" });
    const a = await r.call("report_progress", { summary: "s", cwd: other, branch: "x", title: "Custom" });
    expect(a.structuredContent?.task).toMatchObject({ title: "Custom", branch: "x" });
  });

  test.each([
    [{}, "summary"],
    [{ summary: "" }, "summary"],
    [{ summary: "s", status: "weird" }, "status"],
    [{ summary: "s", pr_url: "not-a-url" }, "pr_url"],
    [{ summary: "s", task_id: 0 }, "task_id"],
  ])("rejects %j naming %s", async (args, field) => {
    const r = await rig();
    const res = await r.call("report_progress", args);
    expect(res.isError).toBe(true);
    expect(textOf(res)).toContain(field);
  });

  test("unknown task_id is an error naming task_id", async () => {
    const r = await rig();
    const res = await r.call("report_progress", { summary: "s", task_id: 999 });
    expect(res.isError).toBe(true);
    expect(textOf(res)).toContain("task_id");
  });
});

describe("task tools", () => {
  test("create_task by cwd, update_task, get_task", async () => {
    const r = await rig();
    const c = await r.call("create_task", {
      title: "Write docs",
      priority: "P1",
      notes: "n",
      due_at: "2026-12-01",
      remind_at: "2026-11-30T09:00:00+08:00",
    });
    expect(c.isError).toBeFalsy();
    const task = c.structuredContent?.task;
    expect(task).toMatchObject({ title: "Write docs", priority: "P1", source: "agent", status: "todo" });
    expect(task.project_id).not.toBeNull();
    expect(task.due_at).toBe("2026-12-01T00:00:00.000Z");
    expect(task.remind_at).toBe("2026-11-30T01:00:00.000Z");

    const u = await r.call("update_task", { task_id: task.id, status: "done", notes: null });
    expect(u.structuredContent?.task).toMatchObject({ status: "done", notes: null });
    expect(u.structuredContent?.task.completed_at).not.toBeNull();

    const g = await r.call("get_task", { task_id: task.id });
    expect(g.structuredContent?.task.id).toBe(task.id);
    expect(g.structuredContent?.reports).toEqual([]);
  });

  test("create_task with project_id, milestone, and conflicting scope", async () => {
    const r = await rig();
    const p = await r.call("upsert_project", { repo_slug: "acme/api" });
    const pid = p.structuredContent?.project.id;
    const m = await r.call("create_milestone", { title: "v1", project_id: pid });
    const t = await r.call("create_task", { title: "x", project_id: pid, milestone_id: m.structuredContent?.milestone.id });
    expect(t.structuredContent?.task).toMatchObject({ project_id: pid });
    const both = await r.call("create_task", { title: "x", project_id: pid, cwd: r.dir });
    expect(both.isError).toBe(true);
    expect(textOf(both)).toContain("cwd");
  });

  test("create_task validation", async () => {
    const r = await rig();
    for (const [args, field] of [
      [{}, "title"],
      [{ title: "x", priority: "P9" }, "priority"],
      [{ title: "x", due_at: "tomorrow" }, "due_at"],
      [{ title: "x", project_id: 999 }, "project_id"],
    ] as const) {
      const res = await r.call("create_task", args);
      expect(res.isError).toBe(true);
      expect(textOf(res)).toContain(field);
    }
  });

  test("update_task and get_task errors", async () => {
    const r = await rig();
    for (const tool of ["update_task", "get_task"]) {
      const res = await r.call(tool, { task_id: 999 });
      expect(res.isError).toBe(true);
      expect(textOf(res)).toContain("task_id");
    }
    const bad = await r.call("update_task", { task_id: 1, status: "nope" });
    expect(textOf(bad)).toContain("status");
  });

  test("get_task returns the report timeline", async () => {
    const r = await rig();
    await r.call("report_progress", { summary: "one" });
    const b = await r.call("report_progress", { summary: "two" });
    const g = await r.call("get_task", { task_id: b.structuredContent?.task_id });
    expect(g.structuredContent?.reports.map((x: { summary: string }) => x.summary)).toEqual(["one", "two"]);
    expect(textOf(g)).toContain("two");
  });

  test("list_tasks filters by status, project and limit", async () => {
    const r = await rig();
    await r.call("report_progress", { summary: "a" });
    await r.call("create_task", { title: "b" });
    const all = await r.call("list_tasks", {});
    expect(all.structuredContent?.tasks).toHaveLength(2);
    const doing = await r.call("list_tasks", { status: ["doing"] });
    expect(doing.structuredContent?.tasks).toHaveLength(1);
    const byCwd = await r.call("list_tasks", { cwd: r.dir, limit: 1 });
    expect(byCwd.structuredContent?.tasks).toHaveLength(1);
    const both = await r.call("list_tasks", { cwd: r.dir, project_id: 1 });
    expect(both.isError).toBe(true);
    expect((await r.call("list_tasks", { limit: 0 })).isError).toBe(true);
  });

  test("list_tasks with a non-git cwd returns inbox-scoped nothing, not everything", async () => {
    const r = await rig();
    await r.call("report_progress", { summary: "a" });
    const res = await r.call("list_tasks", { cwd: tmp() });
    expect(res.structuredContent?.tasks).toEqual([]);
  });
});

describe("project and milestone tools", () => {
  test("upsert_project by cwd renames", async () => {
    const r = await rig();
    const a = await r.call("upsert_project", { cwd: r.dir });
    const b = await r.call("upsert_project", { cwd: r.dir, name: "Widgets" });
    expect(b.structuredContent?.project).toMatchObject({ id: a.structuredContent?.project.id, name: "Widgets" });
  });

  test("upsert_project errors", async () => {
    const r = await rig();
    expect((await r.call("upsert_project", {})).isError).toBe(true);
    const bad = await r.call("upsert_project", { repo_slug: "nope" });
    expect(textOf(bad)).toContain("repo_slug");
  });

  test("create_milestone by cwd and by project_id; errors", async () => {
    const r = await rig();
    const a = await r.call("create_milestone", { title: "v1", cwd: r.dir, due_at: "2026-12-31T00:00:00Z" });
    expect(a.structuredContent?.milestone).toMatchObject({ title: "v1", status: "open" });
    const nonGit = await r.call("create_milestone", { title: "v1", cwd: tmp() });
    expect(nonGit.isError).toBe(true);
    expect(textOf(nonGit)).toContain("cwd");
    expect((await r.call("create_milestone", { title: "v1", project_id: 99 })).isError).toBe(true);
  });
});

describe("get_overview", () => {
  test("shows active tasks with stale flag honouring stale_hours", async () => {
    const r = await rig();
    await r.call("report_progress", { summary: "working" });
    const o = await r.call("get_overview", {});
    expect(o.structuredContent?.projects).toHaveLength(1);
    expect(o.structuredContent?.projects[0].tasks[0]).toMatchObject({ stale: false, status: "doing" });
    expect(textOf(o)).toContain("widgets");
    const none = await r.call("get_overview", { stale_hours: 0.0001 });
    expect(none.structuredContent?.stale_hours).toBe(0.0001);
    expect((await r.call("get_overview", { stale_hours: -1 })).isError).toBe(true);
  });

  test("empty overview says so", async () => {
    const r = await rig();
    expect(textOf(await r.call("get_overview", {}))).toContain("No active tasks");
  });
});
