-- The working directory an agent reported from; empty when it did not say. The app opens it
-- (terminal, editor, Finder) from a row an agent handed over.
ALTER TABLE reports ADD COLUMN cwd TEXT NOT NULL DEFAULT '';
