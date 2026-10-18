-- Pages an agent attached to a report (an HTML report, a design draft): a JSON array of
-- {"url": ..., "title": ...}, '[]' for a report with none. They hang on the report, not the task,
-- so the timeline says which step made them.
ALTER TABLE reports ADD COLUMN links TEXT NOT NULL DEFAULT '[]';
