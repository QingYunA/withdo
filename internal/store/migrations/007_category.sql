-- What a task is for the user: work or life. NULL until someone classifies it; a task in a
-- repository's project counts as work without a value here.
ALTER TABLE tasks ADD COLUMN category TEXT CHECK (category IN ('work','life'));
