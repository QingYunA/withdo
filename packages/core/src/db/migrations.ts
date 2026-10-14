import type { Database } from "bun:sqlite";

/** Ordered migrations. Index + 1 is the schema version stored in PRAGMA user_version. */
export const MIGRATIONS: readonly string[] = [
  `
CREATE TABLE projects (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  repo_slug TEXT UNIQUE,
  repo_path TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  archived_at TEXT
);

CREATE TABLE milestones (
  id INTEGER PRIMARY KEY,
  project_id INTEGER NOT NULL REFERENCES projects(id),
  title TEXT NOT NULL,
  due_at TEXT,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','closed')),
  created_at TEXT NOT NULL
);

CREATE TABLE tasks (
  id INTEGER PRIMARY KEY,
  project_id INTEGER REFERENCES projects(id),
  milestone_id INTEGER REFERENCES milestones(id),
  title TEXT NOT NULL,
  notes TEXT,
  status TEXT NOT NULL DEFAULT 'todo'
    CHECK (status IN ('backlog','todo','doing','blocked','review','done','cancelled')),
  priority TEXT NOT NULL DEFAULT 'P2' CHECK (priority IN ('P0','P1','P2','P3')),
  due_at TEXT,
  remind_at TEXT,
  branch TEXT,
  pr_url TEXT,
  pr_number INTEGER,
  source TEXT NOT NULL CHECK (source IN ('user','agent')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  completed_at TEXT
);

CREATE TABLE reports (
  id INTEGER PRIMARY KEY,
  task_id INTEGER NOT NULL REFERENCES tasks(id),
  agent TEXT NOT NULL,
  status TEXT,
  summary TEXT NOT NULL,
  next_step TEXT,
  created_at TEXT NOT NULL
);

CREATE INDEX tasks_project_branch ON tasks(project_id, branch);
CREATE INDEX reports_task_created ON reports(task_id, created_at);
`,
];

/** Apply any pending migrations. Idempotent; each migration runs in its own transaction. */
export function migrate(db: Database): void {
  const row = db.query<{ user_version: number }, []>("PRAGMA user_version").get();
  const current = row?.user_version ?? 0;
  if (current > MIGRATIONS.length) {
    throw new Error(
      `database schema version ${current} is newer than this build supports (${MIGRATIONS.length}); upgrade adhd-todo`,
    );
  }
  for (let i = current; i < MIGRATIONS.length; i++) {
    const sql = MIGRATIONS[i]!;
    db.transaction(() => {
      db.exec(sql);
      db.exec(`PRAGMA user_version = ${i + 1}`);
    })();
  }
}
