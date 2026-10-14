import { expect, test } from "bun:test";
import { join } from "node:path";
import { EXPECTED_TOOLS, runSmoke } from "../scripts/smoke.ts";
import { openDb } from "../src/db/open.ts";
import { getOverview } from "../src/domain/overview.ts";
import { makeRepo } from "./helpers/repo.ts";
import { useTmpDir } from "./helpers/tmp.ts";

const tmp = useTmpDir();

test("`cli.ts mcp` serves all tools over real stdio and persists a report", async () => {
  const dbPath = join(tmp(), "stdio.db");
  const repo = makeRepo(tmp(), "widgets", { remote: "git@github.com:acme/widgets.git", branch: "feat/stdio" });
  const cli = join(import.meta.dir, "..", "src", "cli.ts");

  const result = await runSmoke(
    process.execPath,
    [cli, "mcp"],
    { ADHD_TODO_DB: dbPath, PATH: process.env.PATH ?? "" },
    repo,
  );
  expect(result.tools).toEqual([...EXPECTED_TOOLS]);

  const db = openDb(dbPath);
  const overview = getOverview(db);
  db.close();
  expect(overview.projects[0]?.project?.repo_slug).toBe("acme/widgets");
  expect(overview.projects[0]?.tasks[0]).toMatchObject({ id: result.taskId, branch: "feat/stdio", title: "Smoke test task" });
});
