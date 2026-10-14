import { afterEach, describe, expect, test, vi } from "vitest";

const mygo = vi.hoisted(() => ({
  isMyGo: vi.fn(() => false),
  close: vi.fn(async () => {}),
  openMain: vi.fn(async () => {}),
  on: vi.fn(() => () => {}),
}));
vi.mock("mygo-runtime", () => ({ isMyGo: mygo.isMyGo, currentWindow: { close: mygo.close } }));
vi.mock("../mygo", () => ({ Shell: { openMain: mygo.openMain }, events: { mainNavigate: { on: mygo.on } } }));

import { hidePanel, onMainNavigate, openExternal, openTaskInMain } from "./shell";

afterEach(() => {
  vi.clearAllMocks();
  mygo.isMyGo.mockReturnValue(false);
  window.location.hash = "";
});

describe("platform/shell in a browser", () => {
  test("hidePanel does nothing and a task opens by navigating", async () => {
    await hidePanel();
    await openTaskInMain(5);
    expect(mygo.close).not.toHaveBeenCalled();
    expect(window.location.hash).toBe("#/main/task/5");
  });

  test("onMainNavigate subscribes to nothing", () => {
    onMainNavigate(vi.fn())();
    expect(mygo.on).not.toHaveBeenCalled();
  });

  test("openExternal opens a new tab without the opener", () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    openExternal("https://github.com/acme/widgets/pull/42");
    expect(open).toHaveBeenCalledWith("https://github.com/acme/widgets/pull/42", "_blank", "noopener,noreferrer");
  });
});

describe("platform/shell inside mygo", () => {
  test("calls the Go side", async () => {
    mygo.isMyGo.mockReturnValue(true);
    await hidePanel();
    await openTaskInMain(9);
    const handler = vi.fn();
    onMainNavigate(handler);
    expect(mygo.close).toHaveBeenCalledOnce();
    expect(mygo.openMain).toHaveBeenCalledWith("#/main/task/9");
    expect(mygo.on).toHaveBeenCalledWith(handler);
  });
});
