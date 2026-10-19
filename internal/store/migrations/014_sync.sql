-- Local change capture for multi-device sync (M11 S1). Nothing here talks to a server: a linked
-- database (sync_state.enabled = '1') records every change to a synced row in an outbox, and takes
-- new ids from blocks the server leases to it. An unlinked database behaves as before.
--
-- Device-local columns never sync: projects.repo_path, reports.cwd and tasks.reminded_at are not in
-- the column lists below, so changing one of them alone records nothing.

-- enabled is '1' once the device is linked; applying is '1' while rows from the server are written.
CREATE TABLE sync_state (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

-- One local change waiting for upload. columns is a JSON array naming the synced columns an update
-- changed; it is NULL for an insert (every column) and a delete. seq is the cursor the uploader acks;
-- AUTOINCREMENT keeps it rising after the outbox empties, so it can serve as the mutation id.
CREATE TABLE sync_outbox (
  seq        INTEGER PRIMARY KEY AUTOINCREMENT,
  entity     TEXT NOT NULL,
  row_id     INTEGER NOT NULL,
  op         TEXT NOT NULL CHECK (op IN ('put','delete')),
  columns    TEXT,
  created_at TEXT NOT NULL
);

-- The half-open range [next, end) a linked device may still use for ids of entity. Several blocks
-- per entity are allowed; the one with the smallest next is used first. An id block is spent when
-- next reaches end, and its row is then deleted.
CREATE TABLE sync_id_blocks (
  entity TEXT NOT NULL,
  next   INTEGER NOT NULL,
  end    INTEGER NOT NULL
);

CREATE TRIGGER sync_projects_insert AFTER INSERT ON projects
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, created_at)
  VALUES ('projects', NEW.id, 'put', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_projects_update AFTER UPDATE ON projects
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
  AND (OLD.name IS NOT NEW.name
    OR OLD.repo_slug IS NOT NEW.repo_slug
    OR OLD.created_at IS NOT NEW.created_at
    OR OLD.updated_at IS NOT NEW.updated_at
    OR OLD.archived_at IS NOT NEW.archived_at
    OR OLD.parent_id IS NOT NEW.parent_id)
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, columns, created_at)
  VALUES ('projects', NEW.id, 'put',
    (SELECT json_group_array(name) FROM (
      SELECT 'name' AS name, 1 AS n WHERE OLD.name IS NOT NEW.name
      UNION ALL SELECT 'repo_slug' AS name, 2 AS n WHERE OLD.repo_slug IS NOT NEW.repo_slug
      UNION ALL SELECT 'created_at' AS name, 3 AS n WHERE OLD.created_at IS NOT NEW.created_at
      UNION ALL SELECT 'updated_at' AS name, 4 AS n WHERE OLD.updated_at IS NOT NEW.updated_at
      UNION ALL SELECT 'archived_at' AS name, 5 AS n WHERE OLD.archived_at IS NOT NEW.archived_at
      UNION ALL SELECT 'parent_id' AS name, 6 AS n WHERE OLD.parent_id IS NOT NEW.parent_id
    ORDER BY n)),
    strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_projects_delete AFTER DELETE ON projects
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, created_at)
  VALUES ('projects', OLD.id, 'delete', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_milestones_insert AFTER INSERT ON milestones
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, created_at)
  VALUES ('milestones', NEW.id, 'put', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_milestones_update AFTER UPDATE ON milestones
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
  AND (OLD.project_id IS NOT NEW.project_id
    OR OLD.title IS NOT NEW.title
    OR OLD.due_at IS NOT NEW.due_at
    OR OLD.status IS NOT NEW.status
    OR OLD.created_at IS NOT NEW.created_at)
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, columns, created_at)
  VALUES ('milestones', NEW.id, 'put',
    (SELECT json_group_array(name) FROM (
      SELECT 'project_id' AS name, 1 AS n WHERE OLD.project_id IS NOT NEW.project_id
      UNION ALL SELECT 'title' AS name, 2 AS n WHERE OLD.title IS NOT NEW.title
      UNION ALL SELECT 'due_at' AS name, 3 AS n WHERE OLD.due_at IS NOT NEW.due_at
      UNION ALL SELECT 'status' AS name, 4 AS n WHERE OLD.status IS NOT NEW.status
      UNION ALL SELECT 'created_at' AS name, 5 AS n WHERE OLD.created_at IS NOT NEW.created_at
    ORDER BY n)),
    strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_milestones_delete AFTER DELETE ON milestones
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, created_at)
  VALUES ('milestones', OLD.id, 'delete', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_tasks_insert AFTER INSERT ON tasks
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, created_at)
  VALUES ('tasks', NEW.id, 'put', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_tasks_update AFTER UPDATE ON tasks
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
  AND (OLD.project_id IS NOT NEW.project_id
    OR OLD.milestone_id IS NOT NEW.milestone_id
    OR OLD.title IS NOT NEW.title
    OR OLD.notes IS NOT NEW.notes
    OR OLD.status IS NOT NEW.status
    OR OLD.priority IS NOT NEW.priority
    OR OLD.due_at IS NOT NEW.due_at
    OR OLD.remind_at IS NOT NEW.remind_at
    OR OLD.branch IS NOT NEW.branch
    OR OLD.pr_url IS NOT NEW.pr_url
    OR OLD.pr_number IS NOT NEW.pr_number
    OR OLD.source IS NOT NEW.source
    OR OLD.created_at IS NOT NEW.created_at
    OR OLD.updated_at IS NOT NEW.updated_at
    OR OLD.completed_at IS NOT NEW.completed_at
    OR OLD.pr_state IS NOT NEW.pr_state
    OR OLD.pr_checks IS NOT NEW.pr_checks
    OR OLD.pr_review IS NOT NEW.pr_review
    OR OLD.pr_checked_at IS NOT NEW.pr_checked_at
    OR OLD.pr_closed_seen_at IS NOT NEW.pr_closed_seen_at
    OR OLD.parent_id IS NOT NEW.parent_id
    OR OLD.handed_to IS NOT NEW.handed_to
    OR OLD.handed_at IS NOT NEW.handed_at
    OR OLD.category_id IS NOT NEW.category_id)
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, columns, created_at)
  VALUES ('tasks', NEW.id, 'put',
    (SELECT json_group_array(name) FROM (
      SELECT 'project_id' AS name, 1 AS n WHERE OLD.project_id IS NOT NEW.project_id
      UNION ALL SELECT 'milestone_id' AS name, 2 AS n WHERE OLD.milestone_id IS NOT NEW.milestone_id
      UNION ALL SELECT 'title' AS name, 3 AS n WHERE OLD.title IS NOT NEW.title
      UNION ALL SELECT 'notes' AS name, 4 AS n WHERE OLD.notes IS NOT NEW.notes
      UNION ALL SELECT 'status' AS name, 5 AS n WHERE OLD.status IS NOT NEW.status
      UNION ALL SELECT 'priority' AS name, 6 AS n WHERE OLD.priority IS NOT NEW.priority
      UNION ALL SELECT 'due_at' AS name, 7 AS n WHERE OLD.due_at IS NOT NEW.due_at
      UNION ALL SELECT 'remind_at' AS name, 8 AS n WHERE OLD.remind_at IS NOT NEW.remind_at
      UNION ALL SELECT 'branch' AS name, 9 AS n WHERE OLD.branch IS NOT NEW.branch
      UNION ALL SELECT 'pr_url' AS name, 10 AS n WHERE OLD.pr_url IS NOT NEW.pr_url
      UNION ALL SELECT 'pr_number' AS name, 11 AS n WHERE OLD.pr_number IS NOT NEW.pr_number
      UNION ALL SELECT 'source' AS name, 12 AS n WHERE OLD.source IS NOT NEW.source
      UNION ALL SELECT 'created_at' AS name, 13 AS n WHERE OLD.created_at IS NOT NEW.created_at
      UNION ALL SELECT 'updated_at' AS name, 14 AS n WHERE OLD.updated_at IS NOT NEW.updated_at
      UNION ALL SELECT 'completed_at' AS name, 15 AS n WHERE OLD.completed_at IS NOT NEW.completed_at
      UNION ALL SELECT 'pr_state' AS name, 16 AS n WHERE OLD.pr_state IS NOT NEW.pr_state
      UNION ALL SELECT 'pr_checks' AS name, 17 AS n WHERE OLD.pr_checks IS NOT NEW.pr_checks
      UNION ALL SELECT 'pr_review' AS name, 18 AS n WHERE OLD.pr_review IS NOT NEW.pr_review
      UNION ALL SELECT 'pr_checked_at' AS name, 19 AS n WHERE OLD.pr_checked_at IS NOT NEW.pr_checked_at
      UNION ALL SELECT 'pr_closed_seen_at' AS name, 20 AS n WHERE OLD.pr_closed_seen_at IS NOT NEW.pr_closed_seen_at
      UNION ALL SELECT 'parent_id' AS name, 21 AS n WHERE OLD.parent_id IS NOT NEW.parent_id
      UNION ALL SELECT 'handed_to' AS name, 22 AS n WHERE OLD.handed_to IS NOT NEW.handed_to
      UNION ALL SELECT 'handed_at' AS name, 23 AS n WHERE OLD.handed_at IS NOT NEW.handed_at
      UNION ALL SELECT 'category_id' AS name, 24 AS n WHERE OLD.category_id IS NOT NEW.category_id
    ORDER BY n)),
    strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_tasks_delete AFTER DELETE ON tasks
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, created_at)
  VALUES ('tasks', OLD.id, 'delete', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_reports_insert AFTER INSERT ON reports
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, created_at)
  VALUES ('reports', NEW.id, 'put', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_reports_update AFTER UPDATE ON reports
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
  AND (OLD.task_id IS NOT NEW.task_id
    OR OLD.agent IS NOT NEW.agent
    OR OLD.status IS NOT NEW.status
    OR OLD.summary IS NOT NEW.summary
    OR OLD.next_step IS NOT NEW.next_step
    OR OLD.created_at IS NOT NEW.created_at
    OR OLD.kind IS NOT NEW.kind
    OR OLD.links IS NOT NEW.links)
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, columns, created_at)
  VALUES ('reports', NEW.id, 'put',
    (SELECT json_group_array(name) FROM (
      SELECT 'task_id' AS name, 1 AS n WHERE OLD.task_id IS NOT NEW.task_id
      UNION ALL SELECT 'agent' AS name, 2 AS n WHERE OLD.agent IS NOT NEW.agent
      UNION ALL SELECT 'status' AS name, 3 AS n WHERE OLD.status IS NOT NEW.status
      UNION ALL SELECT 'summary' AS name, 4 AS n WHERE OLD.summary IS NOT NEW.summary
      UNION ALL SELECT 'next_step' AS name, 5 AS n WHERE OLD.next_step IS NOT NEW.next_step
      UNION ALL SELECT 'created_at' AS name, 6 AS n WHERE OLD.created_at IS NOT NEW.created_at
      UNION ALL SELECT 'kind' AS name, 7 AS n WHERE OLD.kind IS NOT NEW.kind
      UNION ALL SELECT 'links' AS name, 8 AS n WHERE OLD.links IS NOT NEW.links
    ORDER BY n)),
    strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_reports_delete AFTER DELETE ON reports
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, created_at)
  VALUES ('reports', OLD.id, 'delete', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_categories_insert AFTER INSERT ON categories
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, created_at)
  VALUES ('categories', NEW.id, 'put', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_categories_update AFTER UPDATE ON categories
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
  AND (OLD.builtin IS NOT NEW.builtin
    OR OLD.name IS NOT NEW.name
    OR OLD.icon IS NOT NEW.icon
    OR OLD.color IS NOT NEW.color
    OR OLD.position IS NOT NEW.position
    OR OLD.created_at IS NOT NEW.created_at)
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, columns, created_at)
  VALUES ('categories', NEW.id, 'put',
    (SELECT json_group_array(name) FROM (
      SELECT 'builtin' AS name, 1 AS n WHERE OLD.builtin IS NOT NEW.builtin
      UNION ALL SELECT 'name' AS name, 2 AS n WHERE OLD.name IS NOT NEW.name
      UNION ALL SELECT 'icon' AS name, 3 AS n WHERE OLD.icon IS NOT NEW.icon
      UNION ALL SELECT 'color' AS name, 4 AS n WHERE OLD.color IS NOT NEW.color
      UNION ALL SELECT 'position' AS name, 5 AS n WHERE OLD.position IS NOT NEW.position
      UNION ALL SELECT 'created_at' AS name, 6 AS n WHERE OLD.created_at IS NOT NEW.created_at
    ORDER BY n)),
    strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;

CREATE TRIGGER sync_categories_delete AFTER DELETE ON categories
WHEN (SELECT value FROM sync_state WHERE key = 'enabled') = '1'
  AND COALESCE((SELECT value FROM sync_state WHERE key = 'applying'), '0') <> '1'
BEGIN
  INSERT INTO sync_outbox (entity, row_id, op, created_at)
  VALUES ('categories', OLD.id, 'delete', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
END;
