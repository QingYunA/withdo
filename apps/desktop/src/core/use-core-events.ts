import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { useCore } from "./context";
import { connectCoreEvents, type EventSourceConstructor } from "./events";
import { notifyReminder } from "./notify";
import type { Reminder } from "./types";

export interface UseCoreEventsOptions {
  /**
   * Show system notifications for reminders. Enable in exactly one window: every window with an
   * open stream receives each reminder, so enabling it twice would notify twice.
   */
  readonly notifications?: boolean;
  readonly notify?: (reminder: Reminder) => Promise<void>;
  readonly EventSourceImpl?: EventSourceConstructor;
}

/** Keeps TanStack Query fresh: any `changed` event invalidates every query. */
export function useCoreEvents({ notifications = false, notify = notifyReminder, EventSourceImpl }: UseCoreEventsOptions = {}): void {
  const { endpoint } = useCore();
  const queryClient = useQueryClient();

  useEffect(
    () =>
      connectCoreEvents(
        endpoint,
        {
          onChanged: () => void queryClient.invalidateQueries(),
          onReminder: (reminder) => {
            if (notifications) void notify(reminder);
          },
        },
        EventSourceImpl,
      ),
    [endpoint, queryClient, notifications, notify, EventSourceImpl],
  );
}
