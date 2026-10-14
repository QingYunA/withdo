import { describe, expect, test } from "bun:test";
import { mkdirSync, realpathSync } from "node:fs";
import { join } from "node:path";
import { parseRemoteUrl, resolveGitContext } from "../src/git.ts";
import { git, makeRepo } from "./helpers/repo.ts";
import { useTmpDir } from "./helpers/tmp.ts";

const tmp = useTmpDir();

describe("parseRemoteUrl", () => {
  const cases: Array<[string, string | null]> = [
    ["git@github.com:acme/widgets.git", "acme/widgets"],
    ["git@github.com:acme/widgets", "acme/widgets"],
    ["https://github.com/acme/widgets.git", "acme/widgets"],
    ["https://github.com/acme/widgets", "acme/widgets"],
    ["https://github.com/acme/widgets/", "acme/widgets"],
    ["https://user:tok@github.com/acme/widgets.git", "acme/widgets"],
    ["ssh://git@github.com/acme/widgets.git", "acme/widgets"],
    ["ssh://git@github.com:22/acme/widgets.git", "acme/widgets"],
    ["git://github.com/acme/widgets.git", "acme/widgets"],
    ["https://github.com/acme/wid.gets.git", "acme/wid.gets"],
    ["https://GitHub.com/Acme/Widgets.git", "Acme/Widgets"],
    ["https://gitlab.com/acme/widgets.git", null],
    ["git@example.com:acme/widgets.git", null],
    ["https://github.com/acme", null],
    ["/local/path/repo.git", null],
    ["", null],
  ];
  test.each(cases)("%s -> %s", (url, slug) => {
    expect(parseRemoteUrl(url)).toBe(slug);
  });
});

describe("resolveGitContext", () => {
  test("repo with github remote", () => {
    const dir = makeRepo(tmp(), "widgets", { remote: "git@github.com:acme/widgets.git", branch: "feat/x" });
    expect(resolveGitContext(dir)).toEqual({
      toplevel: dir,
      repoRoot: dir,
      branch: "feat/x",
      repoSlug: "acme/widgets",
      repoName: "widgets",
    });
  });

  test("subdirectory resolves to toplevel", () => {
    const dir = makeRepo(tmp(), "widgets", { remote: "https://github.com/acme/widgets.git" });
    const sub = join(dir, "a", "b");
    mkdirSync(sub, { recursive: true });
    expect(resolveGitContext(sub)?.toplevel).toBe(dir);
  });

  test("repo without remote uses directory name and null slug", () => {
    const dir = makeRepo(tmp(), "localonly", { commit: true });
    expect(resolveGitContext(dir)).toEqual({
      toplevel: dir,
      repoRoot: dir,
      branch: "main",
      repoSlug: null,
      repoName: "localonly",
    });
  });

  test("non-GitHub remote falls back to directory name", () => {
    const dir = makeRepo(tmp(), "gl", { remote: "https://gitlab.com/acme/gl.git" });
    const ctx = resolveGitContext(dir);
    expect(ctx?.repoSlug).toBeNull();
    expect(ctx?.repoName).toBe("gl");
  });

  test("worktree shares the remote slug but has its own branch", () => {
    const root = tmp();
    const main = makeRepo(root, "widgets", { remote: "git@github.com:acme/widgets.git", commit: true });
    const wt = join(root, "widgets-wt");
    git(main, "worktree", "add", "-q", "-b", "feat/wt", wt);
    const ctx = resolveGitContext(wt);
    expect(ctx?.repoSlug).toBe("acme/widgets");
    expect(ctx?.branch).toBe("feat/wt");
    expect(ctx?.toplevel).toBe(realpathSync(wt));
    expect(ctx?.repoRoot).toBe(main);
  });

  test("worktree of a repo without remote resolves to the main repo", () => {
    const root = tmp();
    const main = makeRepo(root, "localonly", { commit: true });
    const wt = join(root, "localonly-wt");
    git(main, "worktree", "add", "-q", "-b", "feat/wt", wt);
    expect(resolveGitContext(wt)).toEqual({
      toplevel: realpathSync(wt),
      repoRoot: main,
      branch: "feat/wt",
      repoSlug: null,
      repoName: "localonly",
    });
  });

  test("worktree of a bare repo resolves to the bare repo", () => {
    const root = tmp();
    const src = makeRepo(root, "src", { commit: true });
    const bare = join(root, "widgets.git");
    git(root, "clone", "-q", "--bare", src, bare);
    const wt = join(root, "widgets-wt");
    git(bare, "worktree", "add", "-q", "-b", "feat/wt", wt);
    const ctx = resolveGitContext(wt);
    expect(ctx?.repoRoot).toBe(realpathSync(bare));
    expect(ctx?.repoName).toBe("widgets.git");
  });

  test("detached HEAD yields null branch", () => {
    const dir = makeRepo(tmp(), "det", { commit: true });
    git(dir, "checkout", "-q", "--detach");
    expect(resolveGitContext(dir)?.branch).toBeNull();
  });

  test("non-git directory resolves to null", () => {
    expect(resolveGitContext(tmp())).toBeNull();
  });

  test("missing directory resolves to null", () => {
    expect(resolveGitContext(join(tmp(), "nope"))).toBeNull();
  });
});
