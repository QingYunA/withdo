import { describe, expect, test } from "bun:test";
import { join } from "node:path";
import { openMemoryDb } from "../src/db/open.ts";
import { DomainError } from "../src/domain/errors.ts";
import { createMilestone, listMilestones } from "../src/domain/milestones.ts";
import { getProject, listProjects, resolveProjectFromCwd, resolveProjectId, upsertProject } from "../src/domain/projects.ts";
import { git, makeRepo } from "./helpers/repo.ts";
import { useTmpDir } from "./helpers/tmp.ts";

const tmp = useTmpDir();

describe("resolveProjectFromCwd", () => {
  test("creates a project from a GitHub repo and reuses it", () => {
    const db = openMemoryDb();
    const dir = makeRepo(tmp(), "widgets", { remote: "git@github.com:acme/widgets.git" });
    const a = resolveProjectFromCwd(db, dir);
    const b = resolveProjectFromCwd(db, dir);
    expect(a?.project).toMatchObject({ name: "widgets", repo_slug: "acme/widgets", repo_path: dir });
    expect(b?.project.id).toBe(a?.project.id);
    expect(listProjects(db)).toHaveLength(1);
  });

  test("a worktree maps to the same project and updates repo_path", () => {
    const db = openMemoryDb();
    const root = tmp();
    const main = makeRepo(root, "widgets", { remote: "git@github.com:acme/widgets.git", commit: true });
    const wt = join(root, "wt");
    git(main, "worktree", "add", "-q", "-b", "feat/a", wt);
    const a = resolveProjectFromCwd(db, main);
    const b = resolveProjectFromCwd(db, wt);
    expect(b?.project.id).toBe(a?.project.id);
    expect(b?.project.repo_path).toBe(b?.git.toplevel);
    expect(b?.git.branch).toBe("feat/a");
  });

  test("repo without remote is keyed by path", () => {
    const db = openMemoryDb();
    const root = tmp();
    const one = makeRepo(root, "one");
    const two = makeRepo(root, "two");
    const a = resolveProjectFromCwd(db, one);
    const b = resolveProjectFromCwd(db, two);
    expect(a?.project.repo_slug).toBeNull();
    expect(a?.project.id).not.toBe(b?.project.id);
    expect(resolveProjectFromCwd(db, one)?.project.id).toBe(a?.project.id);
  });

  test("non-git cwd resolves to null", () => {
    expect(resolveProjectFromCwd(openMemoryDb(), tmp())).toBeNull();
  });
});

describe("upsertProject", () => {
  test("by cwd, with rename", () => {
    const db = openMemoryDb();
    const dir = makeRepo(tmp(), "widgets", { remote: "https://github.com/acme/widgets.git" });
    const a = upsertProject(db, { cwd: dir });
    const b = upsertProject(db, { cwd: dir, name: "Widgets App" });
    expect(b.id).toBe(a.id);
    expect(b.name).toBe("Widgets App");
  });

  test("by repo_slug creates then finds", () => {
    const db = openMemoryDb();
    const a = upsertProject(db, { repo_slug: "acme/api" });
    expect(a).toMatchObject({ name: "api", repo_slug: "acme/api", repo_path: null });
    expect(upsertProject(db, { repo_slug: "acme/api", name: "API" }).name).toBe("API");
    expect(listProjects(db)).toHaveLength(1);
  });

  test("rejects bad input naming the field", () => {
    const db = openMemoryDb();
    expect(() => upsertProject(db, {})).toThrow(DomainError);
    expect(() => upsertProject(db, { repo_slug: "nope" })).toThrow(/repo_slug/);
    expect(() => upsertProject(db, { cwd: "/x", repo_slug: "a/b" })).toThrow(/either/);
    expect(() => upsertProject(db, { cwd: tmp() })).toThrow(/cwd/);
  });
});

describe("resolveProjectId", () => {
  test("project_id wins and must exist", () => {
    const db = openMemoryDb();
    const p = upsertProject(db, { repo_slug: "a/b" });
    expect(resolveProjectId(db, { project_id: p.id })).toBe(p.id);
    expect(() => resolveProjectId(db, { project_id: 999 })).toThrow(/project_id/);
  });

  test("cwd resolves, non-git gives null", () => {
    const db = openMemoryDb();
    const dir = makeRepo(tmp(), "r", { remote: "git@github.com:a/r.git" });
    expect(resolveProjectId(db, { cwd: dir })).toBe(getProject(db, 1)?.id ?? -1);
    expect(resolveProjectId(db, { cwd: tmp() })).toBeNull();
    expect(resolveProjectId(db, {})).toBeNull();
  });
});

describe("milestones", () => {
  test("create and list for a project", () => {
    const db = openMemoryDb();
    const p = upsertProject(db, { repo_slug: "a/b" });
    const m = createMilestone(db, { project_id: p.id, title: "v1", due_at: "2026-12-01T00:00:00.000Z" });
    expect(m).toMatchObject({ project_id: p.id, title: "v1", status: "open", due_at: "2026-12-01T00:00:00.000Z" });
    expect(listMilestones(db, p.id)).toEqual([m]);
  });

  test("unknown project is an error naming project_id", () => {
    expect(() => createMilestone(openMemoryDb(), { project_id: 5, title: "x" })).toThrow(/project_id/);
  });
});
