-- M11 sync server schema (S2a). Postgres keeps the authoritative rows, one version clock per
-- workspace and one id counter per workspace and entity.
CREATE TABLE users (
  id         BIGSERIAL PRIMARY KEY,
  subject    TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One workspace per user for now; the unique owner makes the first-request insert idempotent.
CREATE TABLE workspaces (
  id         BIGSERIAL PRIMARY KEY,
  owner_id   BIGINT UNIQUE NOT NULL REFERENCES users (id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE devices (
  id               TEXT PRIMARY KEY,
  workspace_id     BIGINT NOT NULL REFERENCES workspaces (id),
  user_id          BIGINT NOT NULL REFERENCES users (id),
  name             TEXT NOT NULL,
  last_mutation_id BIGINT NOT NULL DEFAULT 0,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- A row is the merged synced columns of one entity row; a delete leaves a tombstone with no data.
CREATE TABLE rows (
  workspace_id BIGINT NOT NULL REFERENCES workspaces (id),
  entity       TEXT NOT NULL,
  row_id       BIGINT NOT NULL,
  data         JSONB,
  deleted      BOOLEAN NOT NULL DEFAULT false,
  version      BIGINT NOT NULL,
  PRIMARY KEY (workspace_id, entity, row_id)
);

CREATE INDEX rows_workspace_version_idx ON rows (workspace_id, version);

-- The workspace's clock: a push takes the next value for every row it changes.
CREATE TABLE workspace_versions (
  workspace_id BIGINT PRIMARY KEY REFERENCES workspaces (id),
  version      BIGINT NOT NULL
);

-- next is the first id the workspace has not handed out for entity.
CREATE TABLE id_counters (
  workspace_id BIGINT NOT NULL REFERENCES workspaces (id),
  entity       TEXT NOT NULL,
  next         BIGINT NOT NULL,
  PRIMARY KEY (workspace_id, entity)
);
