-- When the device last pushed or pulled, so the Sync pane can show how fresh each device is.
-- NULL until the device's first sync.
ALTER TABLE devices ADD COLUMN last_seen_at TIMESTAMPTZ;

CREATE INDEX devices_workspace_last_seen_idx
  ON devices (workspace_id, last_seen_at DESC NULLS LAST);
