import type { Database } from "bun:sqlite";
import { randomBytes } from "node:crypto";
import { startHttpServer } from "./server.ts";

export interface ServeDeps {
  readonly env: Readonly<Record<string, string | undefined>>;
  /** Only the ready line may be written here. */
  readonly stdout: (text: string) => void;
  readonly stderr: (text: string) => void;
  /** Resolves when the server should shut down (stdin closed or a signal arrived). */
  readonly stdinClosed: Promise<void>;
}

function parsePort(raw: string | undefined): number {
  if (raw === undefined || raw === "") return 0;
  const port = /^\d+$/.test(raw) ? Number(raw) : Number.NaN;
  if (!(port >= 0 && port <= 65535)) throw new Error("ADHD_CORE_PORT must be an integer between 0 and 65535");
  return port;
}

/**
 * Run the HTTP API until `stdinClosed` resolves. Prints `{"port":n,"token":"..."}` as the single
 * stdout line once the socket is listening; everything else goes to stderr.
 */
export async function runServe(db: Database, deps: ServeDeps): Promise<number> {
  const token = deps.env.ADHD_CORE_TOKEN || randomBytes(32).toString("hex");
  const port = parsePort(deps.env.ADHD_CORE_PORT);
  const server = startHttpServer(db, { token, port, log: (message) => deps.stderr(`[adhd-todo] ${message}\n`) });
  try {
    deps.stderr(`[adhd-todo] HTTP API listening on ${server.hostname}:${server.port}\n`);
    deps.stdout(`${JSON.stringify({ port: server.port, token })}\n`);
    await deps.stdinClosed;
    return 0;
  } finally {
    server.stop();
  }
}
