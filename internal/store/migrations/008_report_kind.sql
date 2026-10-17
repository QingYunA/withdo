-- What a report from the GitHub poller is about ('pr_merged', 'checks_failing', ...), so that the
-- app can tell the events apart without reading their text. '' for every other report.
ALTER TABLE reports ADD COLUMN kind TEXT NOT NULL DEFAULT '';
-- The app reads the reports filed since its watermark on every change of the database.
CREATE INDEX reports_created ON reports(created_at);
