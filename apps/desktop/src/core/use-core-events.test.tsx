import { renderHook } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { Providers, fakeClient, newQueryClient } from "../test/render";
import type { EventSourceLike } from "./events";
import { useCoreEvents } from "./use-core-events";

class FakeSource implements EventSourceLike {
  static instances: FakeSource[] = [];
  readonly listeners = new Map<string, (e: MessageEvent) => void>();
  closed = false;
  constructor(readonly url: string) {
    FakeSource.instances.push(this);
  }
  addEventListener(type: string, listener: (e: MessageEvent) => void): void {
    this.listeners.set(type, listener);
  }
  close(): void {
    this.closed = true;
  }
  emit(type: string, data = "{}"): void {
    this.listeners.get(type)?.(new MessageEvent(type, { data }));
  }
}

function setup(options: Parameters<typeof useCoreEvents>[0]) {
  FakeSource.instances = [];
  const queryClient = newQueryClient();
  const invalidate = vi.spyOn(queryClient, "invalidateQueries");
  const hook = renderHook(() => useCoreEvents(options), {
    wrapper: ({ children }) => (
      <Providers client={fakeClient()} queryClient={queryClient}>
        {children}
      </Providers>
    ),
  });
  return { ...hook, invalidate, source: () => FakeSource.instances.at(-1)! };
}

describe("useCoreEvents", () => {
  test("invalidates every query when data changed", () => {
    const { source, invalidate } = setup({ EventSourceImpl: FakeSource });
    source().emit("changed");
    expect(invalidate).toHaveBeenCalledTimes(1);
  });

  test("closes the stream on unmount", () => {
    const { source, unmount } = setup({ EventSourceImpl: FakeSource });
    unmount();
    expect(source().closed).toBe(true);
  });
});
