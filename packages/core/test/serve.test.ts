import { describe, expect, test } from "bun:test";
import { join } from "node:path";
import { runCli, type CliDeps } from "../src/cli.ts";
import { openDb } from "../src/db/open.ts";
import { runServe } from "../src/http/serve.ts";
import { useTmpDir } from "./helpers/tmp.ts";

const tmp = useTmpDir();

function harness(env: Record<string, string>) {
  const out: string[] = [];
  const err: string[] = [];
  let close!: () => void;
  const stdinClosed = new Promise<void>((resolve) => (close = resolve));
  const deps = {
    env: { ADHD_TODO_DB: join(tmp(), "serve.db"), ...env },
    stdout: (s: string) => out.push(s),
    stderr: (s: string) => err.push(s),
    stdinClosed,
  };
  return { out, err, deps, close };
}

const waitForReady = async (out: string[]) => {
  for (let i = 0; i < 200 && out.length === 0; i++) await new Promise((r) => setTimeout(r, 10));
  return JSON.parse(out[0]!) as { port: number; token: string };
};

describe("runServe", () => {
  test("prints one ready line, serves requests, and stops when stdin closes", async () => {
    const h = harness({});
    const done = runCli(["serve"], h.deps as CliDeps);
    const ready = await waitForReady(h.out);
    expect(Object.keys(ready).sort()).toEqual(["port", "token"]);
    expect(ready.port).toBeGreaterThan(0);
    expect(ready.token).toMatch(/^[0-9a-f]{64}$/);
    expect(h.out).toEqual([`${JSON.stringify(ready)}\n`]);

    const res = await fetch(`http://127.0.0.1:${ready.port}/overview`, {
      headers: { authorization: `Bearer ${ready.token}` },
    });
    expect(res.status).toBe(200);

    h.close();
    expect(await done).toBe(0);
    expect(h.out).toHaveLength(1);
    await expect(fetch(`http://127.0.0.1:${ready.port}/overview`)).rejects.toThrow();
    expect(h.err.join("")).toContain("listening");
    expect(h.err.join("")).not.toContain(ready.token);
  });

  test("ADHD_CORE_TOKEN and ADHD_CORE_PORT override the random values", async () => {
    const probe = Bun.serve({ port: 0, fetch: () => new Response("x") });
    const port = probe.port!;
    await probe.stop(true);
    const h = harness({ ADHD_CORE_TOKEN: "fixed-token", ADHD_CORE_PORT: String(port) });
    const done = runCli(["serve"], h.deps as CliDeps);
    expect(await waitForReady(h.out)).toEqual({ port, token: "fixed-token" });
    h.close();
    await done;
  });

  test("two runs get different random tokens", async () => {
    const tokens: string[] = [];
    for (let i = 0; i < 2; i++) {
      const h = harness({});
      const done = runCli(["serve"], h.deps as CliDeps);
      tokens.push((await waitForReady(h.out)).token);
      h.close();
      await done;
    }
    expect(tokens[0]).not.toBe(tokens[1]);
  });

  test.each(["abc", "-1", "70000", "1.5"])("rejects ADHD_CORE_PORT=%s", async (port) => {
    const db = openDb(join(tmp(), "bad.db"));
    const h = harness({ ADHD_CORE_PORT: port });
    await expect(runServe(db, h.deps)).rejects.toThrow(/ADHD_CORE_PORT/);
    expect(h.out).toEqual([]);
    db.close();
  });

  test("usage lists the serve command", async () => {
    const h = harness({});
    await runCli(["--help"], h.deps as CliDeps);
    expect(h.out.join("")).toContain("serve");
  });
});
