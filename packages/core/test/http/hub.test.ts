import { afterEach, describe, expect, test } from "bun:test";
import { openMemoryDb } from "../../src/db/open.ts";
import { createTask } from "../../src/domain/tasks.ts";
import { createEventHub, sseMessage, type EventHub } from "../../src/http/events.ts";
import { sleep } from "../helpers/sse.ts";

let hub: EventHub | undefined;
afterEach(() => hub?.close());

describe("sseMessage", () => {
  test("formats an event with JSON data", () => {
    expect(sseMessage("changed", {})).toBe("event: changed\ndata: {}\n\n");
  });
});

describe("EventHub", () => {
  test("notifyChanged reaches every subscriber until it unsubscribes", () => {
    hub = createEventHub(openMemoryDb(), { pollMs: 1000, heartbeatMs: 1000 });
    const a: string[] = [];
    const b: string[] = [];
    const offA = hub.subscribe((c) => a.push(c));
    hub.subscribe((c) => b.push(c));
    expect(hub.size).toBe(2);
    hub.notifyChanged();
    offA();
    hub.notifyChanged();
    expect(a).toEqual([sseMessage("changed", {})]);
    expect(b).toHaveLength(2);
    expect(hub.size).toBe(1);
  });

  test("a subscriber whose send throws is dropped", () => {
    hub = createEventHub(openMemoryDb(), { pollMs: 1000, heartbeatMs: 1000 });
    hub.subscribe(() => {
      throw new Error("closed");
    });
    hub.notifyChanged();
    expect(hub.size).toBe(0);
  });

  test("sends heartbeat comments while subscribed", async () => {
    hub = createEventHub(openMemoryDb(), { pollMs: 1000, heartbeatMs: 20 });
    const got: string[] = [];
    hub.subscribe((c) => got.push(c));
    await sleep(100);
    expect(got.filter((c) => c === ": keepalive\n\n").length).toBeGreaterThanOrEqual(2);
  });

  test("polls reminders only while someone is subscribed, and delivers each once", async () => {
    const db = openMemoryDb();
    hub = createEventHub(db, { pollMs: 10, heartbeatMs: 1000 });
    const t = createTask(db, { title: "call mum", source: "user", remind_at: "2000-01-01T00:00:00.000Z" });
    await sleep(60);
    expect(db.query("SELECT reminded_at FROM tasks WHERE id = ?").get(t.id)).toEqual({ reminded_at: null });

    const got: string[] = [];
    hub.subscribe((c) => got.push(c));
    await sleep(100);
    expect(got).toEqual([sseMessage("reminder", { task_id: t.id, title: "call mum", project_name: null })]);
  });

  test("an error while polling is logged and does not stop the hub", async () => {
    const db = openMemoryDb();
    const logs: string[] = [];
    hub = createEventHub(db, { pollMs: 10, heartbeatMs: 1000, log: (m) => logs.push(m) });
    hub.subscribe(() => {});
    db.close();
    await sleep(50);
    expect(logs.length).toBeGreaterThan(0);
    expect(hub.size).toBe(1);
  });
});
