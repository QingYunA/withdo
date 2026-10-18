-- The pictures of tasks. A picture is kept once, whoever links it: its file is named by the hash
-- of its content (images/<hash>.<ext>, a 256 px thumbnail beside it as <hash>.thumb.png).
CREATE TABLE images (
  hash TEXT PRIMARY KEY,
  ext TEXT NOT NULL,
  mime TEXT NOT NULL,
  bytes INTEGER NOT NULL,
  created_at TEXT NOT NULL
);

-- Which task has which picture, under what name and when it was added. Deleting a task drops its
-- links; a picture that no task links any more is collected (its row and its files).
CREATE TABLE task_images (
  task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  image_hash TEXT NOT NULL REFERENCES images(hash),
  name TEXT NOT NULL,
  width INTEGER NOT NULL,
  height INTEGER NOT NULL,
  added_at TEXT NOT NULL,
  PRIMARY KEY (task_id, image_hash)
);
CREATE INDEX task_images_hash ON task_images(image_hash);
