import { mkdirSync, realpathSync, writeFileSync } from "node:fs";
import { join } from "node:path";

export function git(cwd: string, ...args: string[]): string {
  const p = Bun.spawnSync(["git", "-c", "user.name=t", "-c", "user.email=t@t", ...args], {
    cwd,
    stdout: "pipe",
    stderr: "pipe",
    env: { ...process.env, GIT_CONFIG_GLOBAL: "/dev/null", GIT_CONFIG_SYSTEM: "/dev/null" },
  });
  if (p.exitCode !== 0) throw new Error(`git ${args.join(" ")} failed: ${p.stderr.toString()}`);
  return p.stdout.toString().trim();
}

/** Create a repo in `<root>/<name>` on branch `branch`, optionally with an origin remote and one commit. */
export function makeRepo(
  root: string,
  name: string,
  opts: { remote?: string; branch?: string; commit?: boolean } = {},
): string {
  const dir = join(root, name);
  mkdirSync(dir, { recursive: true });
  git(dir, "init", "-q", "-b", opts.branch ?? "main");
  if (opts.remote) git(dir, "remote", "add", "origin", opts.remote);
  if (opts.commit) {
    writeFileSync(join(dir, "README.md"), "x\n");
    git(dir, "add", ".");
    git(dir, "commit", "-q", "-m", "init");
  }
  return realpathSync(dir);
}
