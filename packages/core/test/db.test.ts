import { describe, expect, test } from "bun:test";
import { join } from "node:path";
import { existsSync } from "node:fs";
import { openDb, openMemoryDb } from "../src/db/open.ts";
import { MIGRATIONS, migrate } from "../src/db/migrations.ts";
import { useTmpDir } from "./helpers/tmp.ts";

const tmp = useTmpDir();

describe("openDb", () => {
  test("creates parent dirs, applies pragmas and migrations", () => {
    const path = join(tmp(), "nested", "dir", "t.db");
    const db = openDb(path);
    expect(existsSync(path)).toBe(true);
    expect(db.query("PRAGMA journal_mode").get()).toEqual({ journal_mode: "wal" });
    expect(db.query("PRAGMA busy_timeout").get()).toEqual({ timeout: 5000 });
    expect(db.query("PRAGMA foreign_keys").get()).toEqual({ foreign_keys: 1 });
    expect(db.query("PRAGMA user_version").get()).toEqual({ user_version: MIGRATIONS.length });
    db.close();
  });

  test("creates all schema tables", () => {
    const db = openMemoryDb();
    const names = db
      .query<{ name: string }, []>("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
      .all()
      .map((r) => r.name);
    expect(names).toEqual(expect.arrayContaining(["projects", "milestones", "tasks", "reports"]));
  });

  test("re-running migrate is a no-op", () => {
    const db = openMemoryDb();
    db.run("INSERT INTO projects (name, created_at, updated_at) VALUES ('a','t','t')");
    migrate(db);
    migrate(db);
    expect(db.query("SELECT COUNT(*) AS c FROM projects").get()).toEqual({ c: 1 });
    expect(db.query("PRAGMA user_version").get()).toEqual({ user_version: MIGRATIONS.length });
  });

  test("reopening an existing file keeps data", () => {
    const path = join(tmp(), "re.db");
    const a = openDb(path);
    a.run("INSERT INTO projects (name, created_at, updated_at) VALUES ('keep','t','t')");
    a.close();
    const b = openDb(path);
    expect(b.query("SELECT name FROM projects").get()).toEqual({ name: "keep" });
    b.close();
  });

  test("foreign keys are enforced", () => {
    const db = openMemoryDb();
    expect(() =>
      db.run(
        "INSERT INTO tasks (project_id, title, source, created_at, updated_at) VALUES (999,'x','user','t','t')",
      ),
    ).toThrow();
  });

  test("CHECK constraints reject bad status", () => {
    const db = openMemoryDb();
    expect(() =>
      db.run("INSERT INTO tasks (title, status, source, created_at, updated_at) VALUES ('x','nope','user','t','t')"),
    ).toThrow();
  });

  test("refuses a database from a newer version", () => {
    const db = openMemoryDb();
    db.run(`PRAGMA user_version = ${MIGRATIONS.length + 5}`);
    expect(() => migrate(db)).toThrow(/newer/);
  });

  test("two processes write concurrently without SQLITE_BUSY", async () => {
    const path = join(tmp(), "concurrent.db");
    openDb(path).close();
    const helper = join(import.meta.dir, "helpers", "writer.ts");
    const run = (tag: string) =>
      Bun.spawn([process.execPath, helper, path, tag, "150"], { stdout: "pipe", stderr: "pipe" });
    const procs = [run("a"), run("b")];
    const codes = await Promise.all(procs.map((p) => p.exited));
    expect(codes).toEqual([0, 0]);
    const db = openDb(path);
    expect(db.query("SELECT COUNT(*) AS c FROM projects").get()).toEqual({ c: 300 });
    db.close();
  });
});
