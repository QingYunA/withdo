import { afterAll, afterEach, describe, expect, test } from "bun:test";
import { openDb } from "../../src/db/open.ts";
import { createTask, updateTask } from "../../src/domain/tasks.ts";
import { patchJson, postJson, startTestApi, TOKEN } from "../helpers/http.ts";
import { openSse, sleep, type SseClient } from "../helpers/sse.ts";

const api = startTestApi({ events: { pollMs: 25, heartbeatMs: 60 } });
const clients: SseClient[] = [];
const connect = async (path = `/events?token=${TOKEN}`, headers: Record<string, string> = {}) => {
  const c = await openSse(`${api.server.url}${path}`, headers);
  clients.push(c);
  return c;
};
afterEach(() => {
  for (const c of clients.splice(0)) c.close();
});
afterAll(() => api.stop());

const isChanged = (e: { event: string }) => e.event === "changed";
const isReminder = (e: { event: string }) => e.event === "reminder";

describe("/events auth", () => {
  test("rejects a missing or wrong token", async () => {
    expect((await connect("/events")).status).toBe(401);
    expect((await connect("/events?token=wrong")).status).toBe(401);
  });

  test("accepts the token in the query or the Authorization header", async () => {
    const viaQuery = await connect();
    expect(viaQuery.status).toBe(200);
    expect(viaQuery.contentType).toStartWith("text/event-stream");
    const viaHeader = await connect("/events", { authorization: `Bearer ${TOKEN}` });
    expect(viaHeader.status).toBe(200);
  });
});

describe("changed", () => {
  test("fires after the server's own writes", async () => {
    const sse = await connect();
    await api.json("/tasks", postJson({ title: "own write" }));
    await sse.waitFor(isChanged);
  });

  test("does not fire for failed writes or reads", async () => {
    const sse = await connect();
    await api.json("/tasks", postJson({ title: "" }));
    await api.json("/tasks");
    await sleep(150);
    expect(sse.events.filter(isChanged)).toHaveLength(0);
  });

  test("fires for a write from a second database connection (data_version polling)", async () => {
    const sse = await connect();
    await sleep(100);
    const other = openDb(api.dbPath);
    createTask(other, { title: "written by an MCP process", source: "agent" });
    other.close();
    await sse.waitFor(isChanged);
  });

  test("a quiet database produces no changed events", async () => {
    const sse = await connect();
    await sleep(200);
    expect(sse.events).toEqual([]);
  });

  test("sends an opening comment and periodic keep-alive comments", async () => {
    const sse = await connect();
    await sleep(250);
    expect(sse.comments).toContain("connected");
    expect(sse.comments.filter((c) => c === "keepalive").length).toBeGreaterThanOrEqual(2);
  });
});

describe("reminder", () => {
  test("fires exactly once per task, with title and project name", async () => {
    const sse = await connect();
    const created = await api.json("/tasks", postJson({ title: "stand up", remind_at: "2000-01-01T00:00:00Z" }));
    const id = created.body.data.id;
    const event = await sse.waitFor(isReminder);
    expect(event.data).toEqual({ task_id: id, title: "stand up", project_name: null });
    await sleep(200);
    expect(sse.events.filter(isReminder)).toHaveLength(1);

    const late = await connect();
    await sleep(150);
    expect(late.events.filter(isReminder)).toHaveLength(0);
    const stored = api.db.query<{ reminded_at: string | null }, [number]>("SELECT reminded_at FROM tasks WHERE id = ?").get(id);
    expect(stored?.reminded_at).not.toBeNull();
  });

  test("two connected clients share one delivery path (each sees the event once)", async () => {
    const a = await connect();
    const b = await connect();
    await api.json("/tasks", postJson({ title: "both", remind_at: "2000-01-01T00:00:00Z" }));
    await a.waitFor(isReminder);
    await b.waitFor(isReminder);
    await sleep(150);
    expect(a.events.filter(isReminder)).toHaveLength(1);
    expect(b.events.filter(isReminder)).toHaveLength(1);
  });

  test("a future reminder fires when it comes due, and PATCHing remind_at re-arms it", async () => {
    const sse = await connect();
    const soon = new Date(Date.now() + 300).toISOString();
    const { body } = await api.json("/tasks", postJson({ title: "soon", remind_at: soon }));
    await sleep(100);
    expect(sse.events.filter(isReminder)).toHaveLength(0);
    await sse.waitFor(isReminder);

    await api.json(`/tasks/${body.data.id}`, patchJson({ remind_at: "2000-01-01T00:00:00Z" }));
    await sse.waitFor((e) => isReminder(e) && sse.events.filter(isReminder).length === 2);
  });

  test("does not fire for done or cancelled tasks", async () => {
    const sse = await connect();
    await api.json("/tasks", postJson({ title: "finished", status: "done", remind_at: "2000-01-01T00:00:00Z" }));
    await sleep(150);
    expect(sse.events.filter(isReminder).some((e) => e.data.title === "finished")).toBe(false);
  });

  test("a reminder set from another connection also fires", async () => {
    const sse = await connect();
    const other = openDb(api.dbPath);
    const t = createTask(other, { title: "from mcp", source: "agent" });
    updateTask(other, t.id, { remind_at: "2000-01-01T00:00:00.000Z" });
    other.close();
    const event = await sse.waitFor((e) => isReminder(e) && e.data.title === "from mcp");
    expect(event.data.task_id).toBe(t.id);
  });
});

describe("lifecycle", () => {
  test("a disconnected client is dropped from the hub", async () => {
    const sse = await connect();
    await sleep(50);
    const before = api.server.hub.size;
    expect(before).toBeGreaterThan(0);
    sse.close();
    await sleep(200);
    expect(api.server.hub.size).toBe(before - 1);
  });
});
