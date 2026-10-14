// Compiles packages/core into a standalone binary that mygo ships as a resource.
// mygo copies resources/<os>-<arch>/ into Contents/Resources, so the sidecar ends up at
// Contents/Resources/bin/adhd-core and the Go side finds it with App.Path(PathResources).
import { spawnSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import { join, resolve } from "node:path";

const desktopDir = resolve(import.meta.dirname, "..");
const entry = resolve(desktopDir, "../../packages/core/src/cli.ts");
const platform = `${process.platform}-${process.arch}`;
const outDir = join(desktopDir, "resources", platform, "bin");
const outfile = join(outDir, "adhd-core");

mkdirSync(outDir, { recursive: true });
const result = spawnSync("bun", ["build", "--compile", entry, "--outfile", outfile], { stdio: "inherit" });
if (result.status !== 0) throw new Error(`bun build --compile failed with status ${result.status}`);
process.stderr.write(`sidecar: ${outfile}\n`);
