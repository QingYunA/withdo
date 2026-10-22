-- Extra subjects that sign in as an existing user, so a device still holding a token of the old
-- identity (a Zitadel id after the account moved to PocketBase) keeps reaching the same workspace.
-- A subject is never both users.subject and an alias; MoveSubject and AddAlias enforce that.
CREATE TABLE subject_aliases (
  subject    TEXT PRIMARY KEY,
  user_id    BIGINT NOT NULL REFERENCES users (id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX subject_aliases_user_idx ON subject_aliases (user_id);
