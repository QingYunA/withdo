import type { Database } from "bun:sqlite";
import { claimDueReminders } from "../domain/reminders.ts";

export type Send = (chunk: string) => void;

export interface EventHubOptions {
  /** How often to check for foreign writes and due reminders. Default 500 ms. */
  readonly pollMs?: number;
  /** How often to send a keep-alive comment. Default 15 s. */
  readonly heartbeatMs?: number;
  readonly log?: (message: string) => void;
}

export interface EventHub {
  /** Number of connected subscribers. */
  readonly size: number;
  /** Register `send`; returns the unsubscribe function. Timers run only while someone is subscribed. */
  subscribe(send: Send): () => void;
  /** Tell every subscriber that data changed (used after this server's own writes). */
  notifyChanged(): void;
  close(): void;
}

const KEEPALIVE = ": keepalive\n\n";

export const sseMessage = (event: string, data: unknown): string => `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;

function dataVersion(db: Database): number {
  return db.query<{ data_version: number }, []>("PRAGMA data_version").get()?.data_version ?? 0;
}

/**
 * Fans events out to SSE subscribers. `PRAGMA data_version` changes only when another connection
 * commits, which is how writes from MCP processes are noticed; our own writes call `notifyChanged`.
 */
export function createEventHub(db: Database, options: EventHubOptions = {}): EventHub {
  const pollMs = options.pollMs ?? 500;
  const heartbeatMs = options.heartbeatMs ?? 15_000;
  const log = options.log ?? ((m: string) => void process.stderr.write(`[adhd-todo] ${m}\n`));
  const subscribers = new Set<Send>();
  let timers: ReturnType<typeof setInterval>[] = [];
  let version = 0;

  const stop = (): void => {
    for (const t of timers) clearInterval(t);
    timers = [];
  };

  const broadcast = (chunk: string): void => {
    for (const send of [...subscribers]) {
      try {
        send(chunk);
      } catch {
        subscribers.delete(send);
      }
    }
    if (subscribers.size === 0) stop();
  };

  const poll = (): void => {
    try {
      const current = dataVersion(db);
      if (current !== version) {
        version = current;
        broadcast(sseMessage("changed", {}));
      }
      for (const reminder of claimDueReminders(db)) broadcast(sseMessage("reminder", reminder));
    } catch (error) {
      log(`event poll failed: ${error instanceof Error ? error.message : String(error)}`);
    }
  };

  const start = (): void => {
    if (timers.length > 0) return;
    version = dataVersion(db);
    timers = [setInterval(poll, pollMs), setInterval(() => broadcast(KEEPALIVE), heartbeatMs)];
  };

  return {
    get size() {
      return subscribers.size;
    },
    subscribe(send) {
      subscribers.add(send);
      start();
      return () => {
        subscribers.delete(send);
        if (subscribers.size === 0) stop();
      };
    },
    notifyChanged: () => broadcast(sseMessage("changed", {})),
    close() {
      subscribers.clear();
      stop();
    },
  };
}
