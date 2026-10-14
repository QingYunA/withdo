import { describe, expect, test, vi } from "vitest";
import { discoverEndpoint } from "./endpoint";

describe("discoverEndpoint", () => {
  test("asks the Tauri shell for core_endpoint when running inside Tauri", async () => {
    const invoke = vi.fn(async () => ({ port: 4321, token: "secret" }));
    const endpoint = await discoverEndpoint({ tauri: true, invoke, env: {} });
    expect(invoke).toHaveBeenCalledWith("core_endpoint");
    expect(endpoint).toEqual({ baseUrl: "http://127.0.0.1:4321", token: "secret" });
  });

  test("uses VITE_CORE_URL and VITE_CORE_TOKEN in a plain browser", async () => {
    const endpoint = await discoverEndpoint({
      tauri: false,
      env: { VITE_CORE_URL: "http://127.0.0.1:4317/", VITE_CORE_TOKEN: "devtoken" },
    });
    expect(endpoint).toEqual({ baseUrl: "http://127.0.0.1:4317", token: "devtoken" });
  });

  test("explains what to set when the browser variables are missing", async () => {
    await expect(discoverEndpoint({ tauri: false, env: {} })).rejects.toThrow(/VITE_CORE_URL/);
  });

  test("turns a failed Tauri command into a readable error", async () => {
    const invoke = vi.fn(async () => {
      throw "adhd-core exited (code Some(1))";
    });
    await expect(discoverEndpoint({ tauri: true, invoke, env: {} })).rejects.toThrow("adhd-core exited");
  });
});
