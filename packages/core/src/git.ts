import { basename, dirname } from "node:path";

export interface GitContext {
  readonly toplevel: string;
  /** Root of the main working tree; differs from toplevel inside a linked worktree. */
  readonly repoRoot: string;
  /** Current branch, or null when HEAD is detached. */
  readonly branch: string | null;
  /** "owner/repo" when origin is a GitHub remote, otherwise null. */
  readonly repoSlug: string | null;
  readonly repoName: string;
}

const GITHUB_HOSTS = new Set(["github.com", "www.github.com", "ssh.github.com"]);

/** Parse a git remote URL into "owner/repo". Only GitHub remotes are recognised. */
export function parseRemoteUrl(url: string): string | null {
  const trimmed = url.trim();
  let host: string;
  let path: string;

  const scp = /^(?:[^@/\s]+@)?([^:/\s]+):(?!\/\/)(.+)$/.exec(trimmed);
  if (scp && !trimmed.includes("://")) {
    host = scp[1]!;
    path = scp[2]!;
  } else {
    try {
      const u = new URL(trimmed);
      host = u.hostname;
      path = u.pathname;
    } catch {
      return null;
    }
  }

  if (!GITHUB_HOSTS.has(host.toLowerCase())) return null;
  const parts = path
    .replace(/^\/+/, "")
    .replace(/\/+$/, "")
    .replace(/\.git$/, "")
    .split("/");
  if (parts.length !== 2 || !parts[0] || !parts[1]) return null;
  return `${parts[0]}/${parts[1]}`;
}

function runGit(cwd: string, args: readonly string[]): string | null {
  try {
    const p = Bun.spawnSync(["git", ...args], { cwd, stdout: "pipe", stderr: "pipe" });
    if (p.exitCode !== 0) return null;
    return p.stdout.toString().trim();
  } catch {
    return null; // git missing or cwd does not exist
  }
}

/** The main working tree for a linked worktree, otherwise `toplevel`. */
function resolveRepoRoot(toplevel: string): string {
  const dirs = runGit(toplevel, ["rev-parse", "--path-format=absolute", "--git-dir", "--git-common-dir"]);
  const [gitDir, commonDir] = dirs?.split("\n") ?? [];
  if (!gitDir || !commonDir || gitDir === commonDir) return toplevel;
  // A bare repository's common dir is the repository itself.
  return basename(commonDir) === ".git" ? dirname(commonDir) : commonDir;
}

/** Describe the git repository containing `cwd`, or null when it is not inside one. */
export function resolveGitContext(cwd: string): GitContext | null {
  const toplevel = runGit(cwd, ["rev-parse", "--show-toplevel"]);
  if (!toplevel) return null;

  // symbolic-ref also works on an unborn branch (no commits yet) and fails when detached.
  const branch = runGit(toplevel, ["symbolic-ref", "--short", "-q", "HEAD"]) || null;
  const remote = runGit(toplevel, ["remote", "get-url", "origin"]);
  const repoSlug = remote ? parseRemoteUrl(remote) : null;
  const repoRoot = resolveRepoRoot(toplevel);
  const repoName = repoSlug ? repoSlug.split("/")[1]! : basename(repoRoot);

  return { toplevel, repoRoot, branch, repoSlug, repoName };
}
