-- Seeding lets the first device of a workspace upload its snapshot across several pushes. Exactly
-- one device per workspace may be seeding; it may write new rows with any ids, while every other
-- device waits (409 workspace is seeding) until POST /v1/devices/seeded ends it.
ALTER TABLE devices ADD COLUMN seeding BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX devices_workspace_seeding_idx ON devices (workspace_id) WHERE seeding;
