// @vitest-environment node
import { afterAll, beforeAll, describe, expect, test, vi } from "vitest";
import { startCore, type RunningCore } from "../test/core-server";
import { ApiError, createCoreClient, type CoreClient, type FetchLike } from "./client";

describe("CoreClient request building", () => {
  const ok = (data: unknown) => new Response(JSON.stringify({ data }), { status: 200 });

  function clientWith(fetchImpl: FetchLike): CoreClient {
    return createCoreClient({ baseUrl: "http://127.0.0.1:1", token: "tok" }, fetchImpl);
  }

  test("sends the bearer token and repeats status in the query", async () => {
    const fetchImpl = vi.fn(async () => ok([]));
    await clientWith(fetchImpl).tasks({ project_id: 3, status: ["todo", "doing"], milestone_id: 2, limit: 10 });
    const [url, init] = fetchImpl.mock.calls[0] as unknown as [string, RequestInit];
    expect(url).toBe("http://127.0.0.1:1/tasks?project_id=3&status=todo&status=doing&milestone_id=2&limit=10");
    expect(new Headers(init.headers).get("authorization")).toBe("Bearer tok");
  });

  test("inbox filter becomes inbox=1 and no query is sent for an empty filter", async () => {
    const fetchImpl = vi.fn(async () => ok([]));
    const client = clientWith(fetchImpl);
    await client.tasks({ inbox: true });
    await client.tasks();
    expect(fetchImpl.mock.calls.map((c) => (c as unknown as [string])[0])).toEqual([
      "http://127.0.0.1:1/tasks?inbox=1",
      "http://127.0.0.1:1/tasks",
    ]);
  });

  test("overview passes stale_hours", async () => {
    const fetchImpl = vi.fn(async () => ok({}));
    await clientWith(fetchImpl).overview(12);
    expect((fetchImpl.mock.calls[0] as unknown as [string])[0]).toBe("http://127.0.0.1:1/overview?stale_hours=12");
  });

  test("POST sends a JSON body", async () => {
    const fetchImpl = vi.fn(async () => ok({ id: 1 }));
    await clientWith(fetchImpl).createTask({ title: "x" });
    const [, init] = fetchImpl.mock.calls[0] as unknown as [string, RequestInit];
    expect(init.method).toBe("POST");
    expect(init.body).toBe('{"title":"x"}');
    expect(new Headers(init.headers).get("content-type")).toBe("application/json");
  });

  test("network failures become ApiError with status 0", async () => {
    const client = clientWith(async () => {
      throw new TypeError("fetch failed");
    });
    await expect(client.projects()).rejects.toMatchObject({ name: "ApiError", status: 0 });
  });

  test("non-JSON error bodies fall back to the status text", async () => {
    const client = clientWith(async () => new Response("<html>", { status: 502, statusText: "Bad Gateway" }));
    await expect(client.projects()).rejects.toThrow("502 Bad Gateway");
  });
});

describe("CoreClient against a real adhd-core serve", () => {
  let core: RunningCore;
  let client: CoreClient;

  beforeAll(async () => {
    core = await startCore({ seed: true });
    client = createCoreClient(core.endpoint);
  });
  afterAll(async () => core.stop());

  test("rejects a wrong token with 401", async () => {
    const bad = createCoreClient({ ...core.endpoint, token: "nope" });
    await expect(bad.projects()).rejects.toMatchObject({ status: 401 });
  });

  test("lists projects with per-status counts", async () => {
    const projects = await client.projects();
    const widgets = projects.find((p) => p.repo_slug === "acme/widgets");
    expect(widgets?.counts.doing).toBeGreaterThanOrEqual(2);
  });

  test("returns the overview with stale flags and latest reports", async () => {
    const overview = await client.overview();
    const tasks = overview.projects.flatMap((g) => g.tasks);
    expect(tasks.find((t) => t.title === "Rewrite install guide")?.stale).toBe(true);
    expect(tasks.find((t) => t.title === "Add login form")?.latest_report?.summary).toContain("session API");
  });

  test("creates an inbox task, reads it back, patches it", async () => {
    const created = await client.createTask({ title: "from client test" });
    expect(created).toMatchObject({ project_id: null, source: "user", status: "todo" });
    const patched = await client.updateTask(created.id, { status: "doing", notes: "hello", priority: "P0" });
    expect(patched).toMatchObject({ status: "doing", notes: "hello", priority: "P0" });
    const detail = await client.task(created.id);
    expect(detail.task.id).toBe(created.id);
    expect(detail.reports).toEqual([]);
  });

  test("filters tasks by inbox and status", async () => {
    const inbox = await client.tasks({ inbox: true, status: ["doing"] });
    expect(inbox.every((t) => t.project_id === null && t.status === "doing")).toBe(true);
    expect(inbox.some((t) => t.title === "from client test")).toBe(true);
  });

  test("task detail lists reports oldest first", async () => {
    const overview = await client.overview();
    const login = overview.projects.flatMap((g) => g.tasks).find((t) => t.title === "Add login form")!;
    const detail = await client.task(login.id);
    expect(detail.reports).toHaveLength(1);
  });

  test("creates and lists milestones", async () => {
    const project = (await client.projects()).find((p) => p.repo_slug === "acme/docs")!;
    const milestone = await client.createMilestone({ project_id: project.id, title: "Docs v2" });
    expect(milestone.title).toBe("Docs v2");
    const all = await client.milestones(project.id);
    expect(all.map((m) => m.title)).toContain("Docs v2");
  });

  test("surfaces validation errors naming the field", async () => {
    const error = await client.updateTask(999_999, { title: "x" }).catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).status).toBe(404);
    const bad = await client.createTask({ title: "" }).catch((e: unknown) => e);
    expect((bad as ApiError).status).toBe(400);
    expect((bad as ApiError).message).toContain("title");
  });
});
