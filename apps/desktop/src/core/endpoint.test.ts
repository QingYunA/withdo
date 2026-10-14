import { describe, expect, test, vi } from "vitest";
import { discoverEndpoint } from "./endpoint";

describe("discoverEndpoint", () => {
  test("asks the Go shell for the connection when running inside mygo", async () => {
    const conn = vi.fn(async () => ({ port: 4321, token: "secret" }));
    const endpoint = await discoverEndpoint({ inShell: true, conn, env: {} });
    expect(conn).toHaveBeenCalledOnce();
    expect(endpoint).toEqual({ baseUrl: "http://127.0.0.1:4321", token: "secret" });
  });

  test("uses VITE_CORE_URL and VITE_CORE_TOKEN in a plain browser", async () => {
    const endpoint = await discoverEndpoint({
      inShell: false,
      env: { VITE_CORE_URL: "http://127.0.0.1:4317/", VITE_CORE_TOKEN: "devtoken" },
    });
    expect(endpoint).toEqual({ baseUrl: "http://127.0.0.1:4317", token: "devtoken" });
  });

  test("explains what to set when the browser variables are missing", async () => {
    await expect(discoverEndpoint({ inShell: false, env: {} })).rejects.toThrow(/VITE_CORE_URL/);
  });

  test("passes on the shell's error when the sidecar failed", async () => {
    const conn = vi.fn(async () => {
      throw new Error("adhd-core exited: exit status 1");
    });
    await expect(discoverEndpoint({ inShell: true, conn, env: {} })).rejects.toThrow("adhd-core exited");
  });
});
