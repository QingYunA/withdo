import { openDb } from "./db/open.ts";
import { formatOverview } from "./domain/format.ts";
import { getOverview } from "./domain/overview.ts";
import { runServe } from "./http/serve.ts";
import { runStdioServer } from "./mcp/server.ts";
import { resolveDbPath } from "./paths.ts";

export interface CliDeps {
  readonly env: Readonly<Record<string, string | undefined>>;
  readonly stdout: (text: string) => void;
  readonly stderr: (text: string) => void;
  /** `serve` stops when this resolves; defaults to stdin closing or SIGINT/SIGTERM. */
  readonly stdinClosed?: Promise<void>;
}

const USAGE = `Usage: adhd-core <command>

Commands:
  mcp        Run the MCP server on stdio
  overview   Print active tasks per project, with stale flags
  serve      Run the local HTTP API (prints {"port","token"} on stdout; exits when stdin closes)
`;

/** Run a CLI command. For \`mcp\` this resolves once the server is connected; stdio keeps the process alive. */
export async function runCli(args: readonly string[], deps: CliDeps): Promise<number> {
  const command = args[0];
  if (command === "--help" || command === "-h") {
    deps.stdout(USAGE);
    return 0;
  }
  if (command !== "mcp" && command !== "overview" && command !== "serve") {
    deps.stderr(command ? `Unknown command: ${command}\n${USAGE}` : USAGE);
    return 2;
  }

  const db = openDb(resolveDbPath({ env: deps.env }));
  if (command === "mcp") {
    await runStdioServer(db);
    return 0;
  }
  if (command === "serve") {
    try {
      return await runServe(db, { ...deps, stdinClosed: deps.stdinClosed ?? processShutdown() });
    } finally {
      db.close();
    }
  }
  try {
    deps.stdout(`${formatOverview(getOverview(db))}\n`);
    return 0;
  } finally {
    db.close();
  }
}

/** Resolves when stdin reaches EOF (the parent app died) or the process gets SIGINT/SIGTERM. */
function processShutdown(): Promise<void> {
  const eof = (async () => {
    const reader = Bun.stdin.stream().getReader();
    while (!(await reader.read()).done);
  })();
  const signal = new Promise<void>((resolve) => {
    process.once("SIGINT", () => resolve());
    process.once("SIGTERM", () => resolve());
  });
  return Promise.race([eof, signal]);
}

if (import.meta.main) {
  try {
    process.exitCode = await runCli(process.argv.slice(2), {
      env: process.env,
      stdout: (text) => process.stdout.write(text),
      stderr: (text) => process.stderr.write(text),
    });
  } catch (error) {
    process.stderr.write(`adhd-core: ${error instanceof Error ? error.message : String(error)}\n`);
    process.exitCode = 1;
  }
}
