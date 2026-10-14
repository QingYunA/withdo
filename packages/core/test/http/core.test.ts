import { afterAll, describe, expect, test } from "bun:test";
import { tokenMatches } from "../../src/http/auth.ts";
import { ALLOWED_ORIGINS } from "../../src/http/cors.ts";
import { startTestApi, TOKEN } from "../helpers/http.ts";

const api = startTestApi();
afterAll(() => api.stop());

describe("auth", () => {
  test("tokenMatches is exact and length-safe", () => {
    expect(tokenMatches("abc", "abc")).toBe(true);
    expect(tokenMatches("abd", "abc")).toBe(false);
    expect(tokenMatches("ab", "abc")).toBe(false);
    expect(tokenMatches("abcd", "abc")).toBe(false);
    expect(tokenMatches("", "abc")).toBe(false);
    expect(tokenMatches(null, "abc")).toBe(false);
  });

  test("missing, wrong, near-miss and malformed credentials all get the same 401", async () => {
    const attempts: Array<Record<string, string>> = [
      {},
      { authorization: "Bearer nope" },
      { authorization: `Bearer ${TOKEN.slice(0, -1)}x` },
      { authorization: `Bearer ${TOKEN}extra` },
      { authorization: TOKEN },
      { authorization: `Basic ${TOKEN}` },
    ];
    const bodies = new Set<string>();
    for (const headers of attempts) {
      const res = await api.call("/overview", { auth: false, headers });
      expect(res.status).toBe(401);
      bodies.add(await res.text());
    }
    expect(bodies.size).toBe(1);
    expect([...bodies][0]).toBe(JSON.stringify({ error: "unauthorized" }));
  });

  test("a valid token is accepted", async () => {
    expect((await api.call("/overview")).status).toBe(200);
  });

  test("the query token is only honoured on /events", async () => {
    const res = await api.call(`/overview?token=${TOKEN}`, { auth: false });
    expect(res.status).toBe(401);
  });

  test("the server binds to loopback only", () => {
    expect(api.server.hostname).toBe("127.0.0.1");
    expect(api.server.url).toStartWith("http://127.0.0.1:");
  });
});

describe("CORS", () => {
  test("the allow-list is exactly the three documented origins", () => {
    expect([...ALLOWED_ORIGINS].sort()).toEqual(
      ["http://localhost:1420", "http://tauri.localhost", "tauri://localhost"].sort(),
    );
  });

  test.each([...ALLOWED_ORIGINS])("preflight from %s needs no token and gets CORS headers", async (origin) => {
    const res = await api.call("/tasks", {
      auth: false,
      method: "OPTIONS",
      headers: {
        origin,
        "access-control-request-method": "PATCH",
        "access-control-request-headers": "authorization,content-type",
      },
    });
    expect(res.status).toBe(204);
    expect(res.headers.get("access-control-allow-origin")).toBe(origin);
    expect(res.headers.get("access-control-allow-methods")).toContain("PATCH");
    expect(res.headers.get("access-control-allow-headers")?.toLowerCase()).toContain("authorization");
    expect(res.headers.get("vary")).toContain("Origin");
  });

  test.each(["http://evil.example", "http://localhost:3000", "http://localhost:1420.evil.example", "null"])(
    "preflight from %s is refused without CORS headers",
    async (origin) => {
      const res = await api.call("/tasks", { auth: false, method: "OPTIONS", headers: { origin } });
      expect(res.status).toBe(403);
      expect(res.headers.get("access-control-allow-origin")).toBeNull();
    },
  );

  test("responses (including 401) carry ACAO only for allowed origins", async () => {
    const ok = await api.call("/overview", { headers: { origin: "http://localhost:1420" } });
    expect(ok.headers.get("access-control-allow-origin")).toBe("http://localhost:1420");
    const denied = await api.call("/overview", { auth: false, headers: { origin: "tauri://localhost" } });
    expect(denied.status).toBe(401);
    expect(denied.headers.get("access-control-allow-origin")).toBe("tauri://localhost");
    const evil = await api.call("/overview", { headers: { origin: "http://evil.example" } });
    expect(evil.headers.get("access-control-allow-origin")).toBeNull();
  });
});

describe("routing and errors", () => {
  test("unknown path is 404 with an error body", async () => {
    expect(await api.json("/nope")).toEqual({ status: 404, body: { error: "not found" } });
  });

  test("known path with the wrong method is 405 with Allow", async () => {
    const res = await api.call("/overview", { method: "DELETE" });
    expect(res.status).toBe(405);
    expect(res.headers.get("allow")).toContain("GET");
  });

  test("unauthenticated unknown paths are still 401, not 404", async () => {
    expect((await api.call("/nope", { auth: false })).status).toBe(401);
  });

  test("unexpected failures become a generic 500 without a stack trace", async () => {
    const logs: string[] = [];
    const boom = startTestApi({
      log: (m) => logs.push(m),
      routes: [
        {
          method: "GET",
          path: "/boom",
          handler: () => {
            throw new Error("secret detail");
          },
        },
      ],
    });
    try {
      const res = await boom.call("/boom");
      expect(res.status).toBe(500);
      const text = await res.text();
      expect(JSON.parse(text)).toEqual({ error: "internal error" });
      expect(text).not.toContain("secret detail");
      expect(text).not.toContain(" at ");
      expect(logs.join("\n")).toContain("secret detail");
      expect(logs.join("\n")).not.toMatch(/\n\s+at /);
    } finally {
      boom.stop();
    }
  });
});
