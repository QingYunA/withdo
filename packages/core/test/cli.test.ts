import { describe, expect, test } from "bun:test";
import { join } from "node:path";
import { openDb } from "../src/db/open.ts";
import { runCli } from "../src/cli.ts";
import { reportProgress } from "../src/domain/reports.ts";
import { makeRepo } from "./helpers/repo.ts";
import { useTmpDir } from "./helpers/tmp.ts";

const tmp = useTmpDir();

function io(dbPath: string) {
  const out: string[] = [];
  const err: string[] = [];
  return {
    out,
    err,
    deps: {
      env: { ADHD_TODO_DB: dbPath } as Record<string, string>,
      stdout: (s: string) => out.push(s),
      stderr: (s: string) => err.push(s),
    },
  };
}

describe("runCli", () => {
  test("overview prints active tasks grouped by project", async () => {
    const dbPath = join(tmp(), "t.db");
    const repo = makeRepo(tmp(), "widgets", { remote: "git@github.com:acme/widgets.git", branch: "feat/login" });
    const db = openDb(dbPath);
    reportProgress(db, { summary: "wiring up login", cwd: repo, agent: "claude-code", next_step: "write tests" });
    db.close();

    const { out, deps } = io(dbPath);
    expect(await runCli(["overview"], deps)).toBe(0);
    const text = out.join("");
    expect(text).toContain("widgets (acme/widgets)");
    expect(text).toContain("feat/login");
    expect(text).toContain("wiring up login");
    expect(text).toContain("next: write tests");
    expect(text.endsWith("\n")).toBe(true);
  });

  test("overview on an empty database says so", async () => {
    const { out, deps } = io(join(tmp(), "empty.db"));
    expect(await runCli(["overview"], deps)).toBe(0);
    expect(out.join("")).toContain("No active tasks");
  });

  test("no command and unknown command print usage to stderr with exit 2", async () => {
    for (const args of [[], ["bogus"]]) {
      const { out, err, deps } = io(join(tmp(), "x.db"));
      expect(await runCli(args, deps)).toBe(2);
      expect(out).toEqual([]);
      expect(err.join("")).toContain("Usage: adhd-core");
    }
  });

  test("--help prints usage to stdout with exit 0", async () => {
    const { out, deps } = io(join(tmp(), "x.db"));
    expect(await runCli(["--help"], deps)).toBe(0);
    expect(out.join("")).toContain("overview");
  });
});
