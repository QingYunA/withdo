# M9: sub-projects

User request (2026-10-19, adhd-todo task #130): 支持项目下的子项目。Its note: 需求：项目可以再挂子项目（层级结构），
入口要简单，别加多余配置。

One rule shapes it: a project that hangs under another is still a project. It has its own page, its
own tasks, milestones and repository field; the parent only says where its row sits in the sidebar.

## P1: a project can have a parent

- Migration `012_project_parent.sql`: `projects.parent_id INTEGER REFERENCES projects(id) ON DELETE
  SET NULL`, indexed. Deleting a parent promotes its sub-projects to top-level, the way a deleted
  task promotes its subtasks.
- `domain.Project.ParentID`; `CreateProjectInput.ParentID`. `CreateProject` checks the parent exists
  and is not archived (a sub-project under an archived project would be invisible), and inserts
  `parent_id`. There is no depth limit and no reparent method: a project is nested when it is made,
  so the tree cannot grow a cycle.
- `ArchiveProject` archives the whole branch (a recursive CTE). `RestoreProject` brings back the
  project and the sub-projects archived by that same action (one archive stamps the branch with one
  time), not what was archived earlier, and it leaves the project above alone: a sub-project whose
  parent is still hidden reads as a top-level project in the sidebar. Both are what `unarchiveByID`
  (an agent reporting in an archived project) uses, and both keep the first archive time of a
  project archived twice.
- `app/data`: `Project.ParentID`, `NewProject.ParentID`, and the fake mirrors the branch rules
  (archive, restore, promote on delete) so the views are tested against the contract the real store
  keeps. `ListProjects` stays one flat list in name order; the tree is built where it is drawn.
- MCP and the assistant keep their shape: `upsert_project` keys a project by its repository and the
  assistant files by project id, both of which a sub-project answers like any other project. A
  sub-project is made by hand.

## P2: the sidebar nests them, and one menu item makes one

- The sidebar draws the top-level projects in name order and each project's sub-projects under it,
  name order at every level, one 28 pt indent step per level (a sub-project's folder sits where its
  parent's label is). The row's count is the project's own active tasks, as before.
- A project's context menu gains `New sub-project…` after `New milestone…`. It opens the same
  **New project** sheet with the title `New sub-project` and one quiet line under it, `Under
  adhd-todo`. No parent picker, no new field, no setting: the entry point is the parent.
- New locale messages: `manageNewSubproject`, `manageNewSubprojectTitle`, `manageSubprojectUnder`.

## P3: tests and the spec

- `internal/domain/subprojects_test.go`: nesting, a missing or archived parent, archiving a branch,
  restoring a sub-project (parents and children come along), deleting a parent.
- `app/data/subprojects_test.go`: the same contract on both stores.
- `app/views/subprojects_test.go`: the sidebar indent and the child's own page, the menu item and
  the sheet, and the branch leaving the sidebar when its parent is archived.
- `docs/design/ui-spec.md`: the **Sub-projects** paragraph, the menu item, the sheet and the archive
  note.

## Not in M9

- Reparenting an existing project (move under…), drag-to-nest, and folding a branch in the sidebar.
- Making a repository resolve to a sub-project on its own: a sub-project is made by hand, though the
  sheet can give it a repository like any project.
- Showing a parent's sub-project tasks on the parent's page: its page is its own tasks, as Todoist
  keeps a parent project's page.
