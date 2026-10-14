-- Remember that a reminder has fired so it fires once per remind_at.
ALTER TABLE tasks ADD COLUMN reminded_at TEXT;
