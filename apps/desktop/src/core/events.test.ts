// @vitest-environment node
import { EventSource } from "eventsource";
import { afterAll, beforeAll, describe, expect, test, vi } from "vitest";
import { startCore, type RunningCore } from "../test/core-server";
import { createCoreClient, type CoreClient } from "./client";
import { connectCoreEvents, type EventSourceLike } from "./events";

describe("connectCoreEvents against a real adhd-core serve", () => {
  let core: RunningCore;
  let client: CoreClient;

  beforeAll(async () => {
    core = await startCore();
    client = createCoreClient(core.endpoint);
  });
  afterAll(async () => core.stop());

  test("emits changed after a write", async () => {
    const onChanged = vi.fn();
    const close = connectCoreEvents(core.endpoint, { onChanged }, EventSource);
    try {
      await new Promise((r) => setTimeout(r, 300)); // let the subscription register
      await client.createTask({ title: "something changed" });
      await vi.waitFor(() => expect(onChanged).toHaveBeenCalled(), { timeout: 5000 });
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
    connectCoreEvents(endpoint, { onChanged: vi.fn() }, FakeSource);
    expect(FakeSource.last.url).toBe("http://127.0.0.1:9/events?token=a%20b");
  });

  test("calls onChanged for a changed event and stops after close", () => {
    const onChanged = vi.fn();
    const close = connectCoreEvents(endpoint, { onChanged }, FakeSource);
    FakeSource.last.emit("changed", "{}");
    expect(onChanged).toHaveBeenCalledOnce();
    close();
    expect(FakeSource.last.closed).toBe(true);
  });
});
