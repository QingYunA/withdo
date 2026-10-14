import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { useCore } from "./context";
import { connectCoreEvents, type EventSourceConstructor } from "./events";

export interface UseCoreEventsOptions {
  readonly EventSourceImpl?: EventSourceConstructor;
}

/** Keeps TanStack Query fresh: any `changed` event invalidates every query. */
export function useCoreEvents({ EventSourceImpl }: UseCoreEventsOptions = {}): void {
  const { endpoint } = useCore();
  const queryClient = useQueryClient();

  useEffect(
    () => connectCoreEvents(endpoint, { onChanged: () => void queryClient.invalidateQueries() }, EventSourceImpl),
    [endpoint, queryClient, EventSourceImpl],
  );
}
