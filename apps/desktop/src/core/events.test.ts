// @vitest-environment node
import { EventSource } from "eventsource";
import { afterAll, beforeAll, describe, expect, test, vi } from "vitest";
import { startCore, type RunningCore } from "../test/core-server";
import { createCoreClient, type CoreClient } from "./client";
import { connectCoreEvents, type EventSourceLike } from "./events";
import type { Reminder } from "./types";

describe("connectCoreEvents against a real adhd-core serve", () => {
  let core: RunningCore;
  let client: CoreClient;

  beforeAll(async () => {
    core = await startCore();
    client = createCoreClient(core.endpoint);
  });
  afterAll(async () => core.stop());

  test("emits changed after a write and reminder once for a due task", async () => {
    const onChanged = vi.fn();
    const reminders: Reminder[] = [];
    const close = connectCoreEvents(core.endpoint, { onChanged, onReminder: (r) => reminders.push(r) }, EventSource);
    try {
      await vi.waitFor(() => expect(client).toBeDefined());
      await new Promise((r) => setTimeout(r, 300)); // let the subscription register
      const task = await client.createTask({ title: "remind me", remind_at: new Date(Date.now() - 1000).toISOString() });
      await vi.waitFor(() => expect(onChanged).toHaveBeenCalled(), { timeout: 5000 });
      await vi.waitFor(() => expect(reminders).toHaveLength(1), { timeout: 5000 });
      expect(reminders[0]).toEqual({ task_id: task.id, title: "remind me", project_name: null });
      await new Promise((r) => setTimeout(r, 1200));
      expect(reminders).toHaveLength(1);
    } finally {
      close();
    }
  });
});

describe("connectCoreEvents with a fake EventSource", () => {
  class FakeSource implements EventSourceLike {
    static last: FakeSource;
    readonly listeners = new Map<string, (e: MessageEvent) => void>();
    closed = false;
    constructor(readonly url: string) {
      FakeSource.last = this;
    }
    addEventListener(type: string, listener: (e: MessageEvent) => void): void {
      this.listeners.set(type, listener);
    }
    close(): void {
      this.closed = true;
    }
    emit(type: string, data: string): void {
      this.listeners.get(type)?.(new MessageEvent(type, { data }));
    }
  }

  const endpoint = { baseUrl: "http://127.0.0.1:9", token: "a b" };

  test("puts the encoded token in the query string", () => {
    connectCoreEvents(endpoint, { onChanged: vi.fn(), onReminder: vi.fn() }, FakeSource);
    expect(FakeSource.last.url).toBe("http://127.0.0.1:9/events?token=a%20b");
  });

  test("ignores malformed reminder payloads and stops after close", () => {
    const onReminder = vi.fn();
    const close = connectCoreEvents(endpoint, { onChanged: vi.fn(), onReminder }, FakeSource);
    FakeSource.last.emit("reminder", "not json");
    FakeSource.last.emit("reminder", JSON.stringify({ nope: true }));
    expect(onReminder).not.toHaveBeenCalled();
    FakeSource.last.emit("reminder", JSON.stringify({ task_id: 1, title: "t", project_name: "p" }));
    expect(onReminder).toHaveBeenCalledWith({ task_id: 1, title: "t", project_name: "p" });
    close();
    expect(FakeSource.last.closed).toBe(true);
  });
});
