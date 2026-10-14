import type { Database } from "bun:sqlite";
import { join } from "node:path";
import { openDb } from "../../src/db/open.ts";
import { startHttpServer, type HttpServerOptions, type RunningServer } from "../../src/http/server.ts";
import { useTmpDir } from "./tmp.ts";

export const TOKEN = "test-token-0123456789";

export interface TestApi {
  readonly db: Database;
  readonly dbPath: string;
  readonly server: RunningServer;
  /** fetch with the bearer token; pass `auth: false` to omit it. */
  call(path: string, init?: RequestInit & { auth?: boolean }): Promise<Response>;
  json(path: string, init?: RequestInit & { auth?: boolean }): Promise<{ status: number; body: any }>;
  stop(): void;
}

const tmp = useTmpDir("adhd-http-");

/** Start a server on port 0 against a fresh temp database file. */
export function startTestApi(options: Partial<HttpServerOptions> = {}): TestApi {
  const dbPath = join(tmp(), "http.db");
  const db = openDb(dbPath);
  const server = startHttpServer(db, { token: TOKEN, port: 0, log: () => {}, ...options });
  const fetchWith: TestApi["call"] = (path, { auth = true, ...init } = {}) => {
    const headers = new Headers(init.headers);
    if (auth) headers.set("authorization", `Bearer ${TOKEN}`);
    return fetch(`${server.url}${path}`, { ...init, headers });
  };
  return {
    db,
    dbPath,
    server,
    call: fetchWith,
    json: async (path, init) => {
      const res = await fetchWith(path, init);
      return { status: res.status, body: await res.json() };
    },
    stop: () => {
      server.stop();
      db.close();
    },
  };
}

export const postJson = (body: unknown): RequestInit => ({
  method: "POST",
  headers: { "content-type": "application/json" },
  body: JSON.stringify(body),
});

export const patchJson = (body: unknown): RequestInit => ({
  method: "PATCH",
  headers: { "content-type": "application/json" },
  body: JSON.stringify(body),
});
