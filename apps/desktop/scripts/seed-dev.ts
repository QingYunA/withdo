// Fills the database named by ADHD_TODO_DB with a few projects, tasks and reports so the UI has
// something to show in browser mode and in integration tests. Run it once per database (it does not
// de-duplicate). A running server notices the writes and emits `changed`.
import { openDb } from "../../../packages/core/src/db/open.ts";
import { createMilestone } from "../../../packages/core/src/domain/milestones.ts";
import { upsertProject } from "../../../packages/core/src/domain/projects.ts";
import { addReport } from "../../../packages/core/src/domain/reports.ts";
import { createTask } from "../../../packages/core/src/domain/tasks.ts";
import type { Task } from "../../../packages/core/src/domain/types.ts";

const dbPath = process.env.ADHD_TODO_DB;
if (!dbPath) throw new Error("ADHD_TODO_DB must point at the database to seed");

const db = openDb(dbPath);
const hoursAgo = (h: number): string => new Date(Date.now() - h * 3_600_000).toISOString();
const inMinutes = (m: number): string => new Date(Date.now() + m * 60_000).toISOString();

const widgets = upsertProject(db, { repo_slug: "acme/widgets" });
const docs = upsertProject(db, { repo_slug: "acme/docs" });
const v1 = createMilestone(db, { project_id: widgets.id, title: "v1.0", due_at: inMinutes(60 * 24 * 14) });

function agentTask(project: number, title: string, status: Task["status"], branch: string, extra: Partial<Task> = {}): Task {
  return createTask(db, { title, source: "agent", status, project_id: project, branch, ...(project === widgets.id ? { milestone_id: v1.id } : {}), ...extra });
}

const login = agentTask(widgets.id, "Add login form", "doing", "feat/login");
addReport(db, { task_id: login.id, agent: "claude-code", summary: "Wiring the form to the session API", next_step: "Write tests", created_at: hoursAgo(0.2) });
const search = agentTask(widgets.id, "Fix search ranking", "blocked", "fix/search");
addReport(db, { task_id: search.id, agent: "claude-code", status: "blocked", summary: "Needs a decision on tokenizer", created_at: hoursAgo(5) });
const cache = agentTask(widgets.id, "Cache invalidation", "review", "feat/cache", {
  pr_url: "https://github.com/acme/widgets/pull/42",
  priority: "P1",
});
addReport(db, { task_id: cache.id, agent: "claude-code", status: "review", summary: "PR opened, waiting for review", created_at: hoursAgo(1) });
const guide = agentTask(docs.id, "Rewrite install guide", "doing", "docs/install");
addReport(db, { task_id: guide.id, agent: "claude-code", summary: "Drafted the macOS section", created_at: hoursAgo(72) });

createTask(db, { title: "Reply to design feedback", source: "user", status: "todo", priority: "P1", remind_at: inMinutes(90), project_id: widgets.id });
createTask(db, { title: "Book dentist", source: "user", status: "todo", due_at: inMinutes(60 * 30) });
createTask(db, { title: "Review the cache PR", source: "user", status: "doing", priority: "P0", project_id: widgets.id, milestone_id: v1.id });
createTask(db, { title: "Plan the Q4 roadmap", source: "user", status: "backlog", project_id: docs.id });
createTask(db, { title: "Ship the changelog", source: "user", status: "done", project_id: widgets.id });
db.close();
process.stderr.write("seeded\n");
