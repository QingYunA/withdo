-- A task the user handed over waits for an agent: handed_to is "agent" (any agent; an earlier build
-- wrote a specific agent's id such as claude), handed_at when. Both clear when an agent reports on
-- the task, or the user takes it back.
ALTER TABLE tasks ADD COLUMN handed_to TEXT;
ALTER TABLE tasks ADD COLUMN handed_at TEXT;
