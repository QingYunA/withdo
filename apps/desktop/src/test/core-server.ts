// Starts a real `adhd-core serve` (from source, via bun) on a temp database for integration tests.
import { spawn, spawnSync, type ChildProcessWithoutNullStreams } from "node:child_process";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import type { Endpoint } from "../core/client";

const CLI = resolve(import.meta.dirname, "../../../../packages/core/src/cli.ts");
const SEED = resolve(import.meta.dirname, "../../scripts/seed-dev.ts");

export interface RunningCore {
  readonly endpoint: Endpoint;
  readonly dbPath: string;
  stop(): Promise<void>;
}

function readFirstLine(child: ChildProcessWithoutNullStreams): Promise<string> {
  return new Promise((resolveLine, reject) => {
    let buffer = "";
    const fail = (message: string) => reject(new Error(message));
    child.once("exit", (code) => fail(`adhd-core exited early (${code})`));
    child.stdout.on("data", (chunk: Buffer) => {
      buffer += chunk.toString();
      const end = buffer.indexOf("\n");
      if (end >= 0) resolveLine(buffer.slice(0, end));
    });
  });
}

export async function startCore(options: { seed?: boolean } = {}): Promise<RunningCore> {
  const dir = mkdtempSync(join(tmpdir(), "adhd-desktop-test-"));
  const dbPath = join(dir, "test.db");
  if (options.seed) {
    const seeded = spawnSync("bun", [SEED], { env: { ...process.env, ADHD_TODO_DB: dbPath } });
    if (seeded.status !== 0) throw new Error(`seeding failed: ${seeded.stderr.toString()}`);
  }
  const child = spawn("bun", [CLI, "serve"], {
    env: { ...process.env, ADHD_TODO_DB: dbPath, ADHD_CORE_PORT: "", ADHD_CORE_TOKEN: "" },
    stdio: ["pipe", "pipe", "pipe"],
  });
  const line = await readFirstLine(child);
  const { port, token } = JSON.parse(line) as { port: number; token: string };
  return {
    endpoint: { baseUrl: `http://127.0.0.1:${port}`, token },
    dbPath,
    stop: async () => {
      child.stdin.end();
      await new Promise<void>((done) => {
        if (child.exitCode !== null) return done();
        child.once("exit", () => done());
        setTimeout(() => child.kill("SIGKILL"), 3000).unref();
      });
      rmSync(dir, { recursive: true, force: true });
    },
  };
}
