import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

afterEach(() => cleanup());

// jsdom has no EventSource. Components subscribe to /events on mount; an inert stand-in keeps them
// mountable. Tests that care about events inject their own implementation.
class InertEventSource {
  addEventListener(): void {}
  close(): void {}
}
if (typeof window !== "undefined" && !("EventSource" in window)) {
  Object.assign(globalThis, { EventSource: InertEventSource });
}
