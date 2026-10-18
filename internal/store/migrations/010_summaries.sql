-- The assistant's paragraph for a day or a week, written once and kept. period is 'day' or 'week';
-- first_date is the local date the period starts on (YYYY-MM-DD); kinds is a JSON object from task id
-- to the kind of good the task became in the scene. Rewriting a summary replaces its row.
CREATE TABLE summaries (
  period TEXT NOT NULL CHECK (period IN ('day','week')),
  first_date TEXT NOT NULL,
  paragraph TEXT NOT NULL,
  model TEXT NOT NULL,
  kinds TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  PRIMARY KEY (period, first_date)
);
