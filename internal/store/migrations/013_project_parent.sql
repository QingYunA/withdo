-- A project can sit under another project: a repository or area the user split into sub-projects.
-- Deleting the parent leaves the sub-project standing as a top-level project.
ALTER TABLE projects ADD COLUMN parent_id INTEGER REFERENCES projects(id) ON DELETE SET NULL;
CREATE INDEX projects_parent ON projects(parent_id);
