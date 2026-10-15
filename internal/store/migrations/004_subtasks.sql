-- A subtask belongs to the task of the agent that started its subagent. Deleting the parent
-- leaves the subtask standing as a top-level task.
ALTER TABLE tasks ADD COLUMN parent_id INTEGER REFERENCES tasks(id) ON DELETE SET NULL;
CREATE INDEX tasks_parent ON tasks(parent_id);
