import type { Endpoint } from "./client";
import type { Reminder } from "./types";

/** The slice of `EventSource` we use; lets tests inject a fake or the `eventsource` package. */
export interface EventSourceLike {
  addEventListener(type: string, listener: (event: MessageEvent) => void): void;
  close(): void;
}

export type EventSourceConstructor = new (url: string) => EventSourceLike;

export interface CoreEventHandlers {
  readonly onChanged: () => void;
  readonly onReminder: (reminder: Reminder) => void;
}

function parseReminder(data: unknown): Reminder | null {
  if (typeof data !== "string") return null;
  try {
    const value = JSON.parse(data) as Partial<Reminder>;
    if (typeof value.task_id !== "number" || typeof value.title !== "string") return null;
    return { task_id: value.task_id, title: value.title, project_name: value.project_name ?? null };
  } catch {
    return null;
  }
}

/**
 * Subscribe to `/events`. EventSource cannot set headers, so the token travels in the query string.
 * The browser reconnects on its own after a dropped connection. Returns the unsubscribe function.
 */
export function connectCoreEvents(
  endpoint: Endpoint,
  handlers: CoreEventHandlers,
  EventSourceImpl: EventSourceConstructor = EventSource,
): () => void {
  const source = new EventSourceImpl(`${endpoint.baseUrl}/events?token=${encodeURIComponent(endpoint.token)}`);
  source.addEventListener("changed", () => handlers.onChanged());
  source.addEventListener("reminder", (event) => {
    const reminder = parseReminder(event.data);
    if (reminder) handlers.onReminder(reminder);
  });
  return () => source.close();
}
