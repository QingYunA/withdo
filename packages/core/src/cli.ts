import { openDb } from "./db/open.ts";
import { formatOverview } from "./domain/format.ts";
import { getOverview } from "./domain/overview.ts";
import { runStdioServer } from "./mcp/server.ts";
import { resolveDbPath } from "./paths.ts";

export interface CliDeps {
  readonly env: Readonly<Record<string, string | undefined>>;
  readonly stdout: (text: string) => void;
  readonly stderr: (text: string) => void;
}

const USAGE = `Usage: adhd-core <command>

Commands:
  mcp        Run the MCP server on stdio
  overview   Print active tasks per project, with stale flags
`;

/** Run a CLI command. For \`mcp\` this resolves once the server is connected; stdio keeps the process alive. */
export async function runCli(args: readonly string[], deps: CliDeps): Promise<number> {
  const command = args[0];
  if (command === "--help" || command === "-h") {
    deps.stdout(USAGE);
    return 0;
  }
  if (command !== "mcp" && command !== "overview") {
    deps.stderr(command ? `Unknown command: ${command}\n${USAGE}` : USAGE);
    return 2;
  }

  const db = openDb(resolveDbPath({ env: deps.env }));
  if (command === "mcp") {
    await runStdioServer(db);
    return 0;
  }
  try {
    deps.stdout(`${formatOverview(getOverview(db))}\n`);
    return 0;
  } finally {
    db.close();
  }
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
