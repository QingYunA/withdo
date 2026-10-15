-- What the GitHub poller last saw of a task's pull request. All NULL until the first check, and
-- again whenever pr_url changes.
ALTER TABLE tasks ADD COLUMN pr_state TEXT CHECK (pr_state IN ('open','draft','merged','closed'));
ALTER TABLE tasks ADD COLUMN pr_checks TEXT CHECK (pr_checks IN ('passing','failing','pending','none'));
ALTER TABLE tasks ADD COLUMN pr_review TEXT CHECK (pr_review IN ('approved','changes_requested','review_required','none'));
-- When the poller last looked, and when the user dismissed the "closed without merging" row.
ALTER TABLE tasks ADD COLUMN pr_checked_at TEXT;
ALTER TABLE tasks ADD COLUMN pr_closed_seen_at TEXT;
