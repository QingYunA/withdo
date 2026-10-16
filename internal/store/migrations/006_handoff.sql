-- A task the user handed to an agent waits for it: handed_to is the agent's id (claude, codex...),
-- handed_at when. Both clear when the agent reports on the task, or the user takes it back.
ALTER TABLE tasks ADD COLUMN handed_to TEXT;
ALTER TABLE tasks ADD COLUMN handed_at TEXT;
