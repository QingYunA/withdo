// Compiles packages/core into a standalone binary that Tauri bundles as the `adhd-core` sidecar.
// Tauri expects the file name `<name>-<rustc host triple>`.
import { spawnSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import { join, resolve } from "node:path";

const desktopDir = resolve(import.meta.dirname, "..");
const entry = resolve(desktopDir, "../../packages/core/src/cli.ts");
const outDir = join(desktopDir, "src-tauri", "binaries");

function run(command: string, args: readonly string[]): string {
  const result = spawnSync(command, args, { encoding: "utf8", stdio: ["ignore", "pipe", "inherit"] });
  if (result.status !== 0) throw new Error(`${command} ${args.join(" ")} failed with status ${result.status}`);
  return result.stdout.trim();
}

const triple = run("rustc", ["--print", "host-tuple"]);
const outfile = join(outDir, `adhd-core-${triple}`);
mkdirSync(outDir, { recursive: true });
run("bun", ["build", "--compile", entry, "--outfile", outfile]);
process.stderr.write(`sidecar: ${outfile}\n`);
