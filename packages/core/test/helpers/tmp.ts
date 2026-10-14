import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { afterAll } from "bun:test";

/** Create a temp dir that is removed after the current test file finishes. */
export function useTmpDir(prefix = "adhd-test-"): () => string {
  const dirs: string[] = [];
  afterAll(() => {
    for (const d of dirs) rmSync(d, { recursive: true, force: true });
  });
  return () => {
    const d = mkdtempSync(join(tmpdir(), prefix));
    dirs.push(d);
    return d;
  };
}
