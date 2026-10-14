import type { Endpoint } from "./client";

/** The slice of `EventSource` we use; lets tests inject a fake or the `eventsource` package. */
export interface EventSourceLike {
  addEventListener(type: string, listener: (event: MessageEvent) => void): void;
  close(): void;
}

export type EventSourceConstructor = new (url: string) => EventSourceLike;

export interface CoreEventHandlers {
  readonly onChanged: () => void;
}

/**
 * Subscribe to `/events`. Reminders are not handled here: the Go shell turns them into notifications.
 * EventSource cannot set headers, so the token travels in the query string.
 * The browser reconnects on its own after a dropped connection. Returns the unsubscribe function.
 */
export function connectCoreEvents(
  endpoint: Endpoint,
  handlers: CoreEventHandlers,
  EventSourceImpl: EventSourceConstructor = EventSource,
): () => void {
  const source = new EventSourceImpl(`${endpoint.baseUrl}/events?token=${encodeURIComponent(endpoint.token)}`);
  source.addEventListener("changed", () => handlers.onChanged());
  return () => source.close();
}
