import type { Database } from "bun:sqlite";
import { resolveGitContext, type GitContext } from "../git.ts";
import { DomainError, NotFoundError } from "./errors.ts";
import { nowIso, type Project } from "./types.ts";

export interface ResolvedProject {
  readonly project: Project;
  readonly git: GitContext;
}

export interface UpsertProjectInput {
  readonly cwd?: string;
  readonly repo_slug?: string;
  readonly name?: string;
}

const SLUG_RE = /^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/;

export function getProject(db: Database, id: number): Project | null {
  return db.query<Project, [number]>("SELECT * FROM projects WHERE id = ?").get(id) ?? null;
}

export function listProjects(db: Database): Project[] {
  return db.query<Project, []>("SELECT * FROM projects ORDER BY id").all();
}

function insertProject(db: Database, name: string, repoSlug: string | null, repoPath: string | null): Project {
  const ts = nowIso();
  const result = db.run(
    "INSERT INTO projects (name, repo_slug, repo_path, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
    [name, repoSlug, repoPath, ts, ts],
  );
  return getProject(db, Number(result.lastInsertRowid))!;
}

function findBySlug(db: Database, slug: string): Project | null {
  return db.query<Project, [string]>("SELECT * FROM projects WHERE repo_slug = ?").get(slug) ?? null;
}

function findByPath(db: Database, path: string): Project | null {
  return (
    db
      .query<Project, [string]>("SELECT * FROM projects WHERE repo_slug IS NULL AND repo_path = ?")
      .get(path) ?? null
  );
}

function touchPath(db: Database, project: Project, path: string): Project {
  if (project.repo_path === path) return project;
  db.run("UPDATE projects SET repo_path = ?, updated_at = ? WHERE id = ?", [path, nowIso(), project.id]);
  return getProject(db, project.id)!;
}

/** Find or create the project that owns `cwd`. Null when `cwd` is not inside a git repository. */
export function resolveProjectFromCwd(db: Database, cwd: string): ResolvedProject | null {
  const git = resolveGitContext(cwd);
  if (!git) return null;
  return db.transaction((): ResolvedProject => {
    const existing = git.repoSlug ? findBySlug(db, git.repoSlug) : findByPath(db, git.toplevel);
    const project = existing
      ? touchPath(db, existing, git.toplevel)
      : insertProject(db, git.repoName, git.repoSlug, git.toplevel);
    return { project, git };
  })();
}

function rename(db: Database, project: Project, name: string | undefined): Project {
  if (name === undefined || name === project.name) return project;
  db.run("UPDATE projects SET name = ?, updated_at = ? WHERE id = ?", [name, nowIso(), project.id]);
  return getProject(db, project.id)!;
}

/** Explicit project creation or rename, keyed by cwd (git) or repo_slug. */
export function upsertProject(db: Database, input: UpsertProjectInput): Project {
  if (input.cwd !== undefined && input.repo_slug !== undefined) {
    throw new DomainError("provide either cwd or repo_slug, not both", "cwd");
  }
  if (input.cwd !== undefined) {
    const resolved = resolveProjectFromCwd(db, input.cwd);
    if (!resolved) throw new DomainError(`${input.cwd} is not inside a git repository`, "cwd");
    return rename(db, resolved.project, input.name);
  }
  if (input.repo_slug !== undefined) {
    if (!SLUG_RE.test(input.repo_slug)) throw new DomainError("must look like owner/repo", "repo_slug");
    const existing = findBySlug(db, input.repo_slug);
    const project = existing ?? insertProject(db, input.repo_slug.split("/")[1]!, input.repo_slug, null);
    return rename(db, project, input.name);
  }
  throw new DomainError("provide cwd or repo_slug", "cwd");
}

/** Resolve a project id from an explicit id or a cwd. Null means "no project" (inbox). */
export function resolveProjectId(db: Database, input: { project_id?: number; cwd?: string }): number | null {
  if (input.project_id !== undefined) {
    if (!getProject(db, input.project_id)) throw new NotFoundError(`project ${input.project_id} not found`, "project_id");
    return input.project_id;
  }
  if (input.cwd !== undefined) return resolveProjectFromCwd(db, input.cwd)?.project.id ?? null;
  return null;
}
