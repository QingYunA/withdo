import { Database } from "bun:sqlite";
import { mkdirSync } from "node:fs";
import { dirname } from "node:path";
import { migrate } from "./migrations.ts";

function applyPragmas(db: Database): void {
  db.exec("PRAGMA journal_mode = WAL");
  db.exec("PRAGMA busy_timeout = 5000");
  db.exec("PRAGMA foreign_keys = ON");
}

/** Open (creating if needed) the on-disk database, apply pragmas and migrations. */
export function openDb(path: string): Database {
  mkdirSync(dirname(path), { recursive: true });
  const db = new Database(path, { create: true });
  applyPragmas(db);
  migrate(db);
  return db;
}

/** In-memory database with the full schema, for tests. */
export function openMemoryDb(): Database {
  const db = new Database(":memory:");
  db.exec("PRAGMA foreign_keys = ON");
  migrate(db);
  return db;
}
