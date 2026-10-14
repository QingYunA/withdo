import type { Database } from "bun:sqlite";
import { DomainError } from "./errors.ts";
import { getProject } from "./projects.ts";
import { nowIso, type Milestone } from "./types.ts";

export interface CreateMilestoneInput {
  readonly project_id: number;
  readonly title: string;
  readonly due_at?: string | null;
}

export function getMilestone(db: Database, id: number): Milestone | null {
  return db.query<Milestone, [number]>("SELECT * FROM milestones WHERE id = ?").get(id) ?? null;
}

export function createMilestone(db: Database, input: CreateMilestoneInput): Milestone {
  if (!getProject(db, input.project_id)) throw new DomainError(`project ${input.project_id} not found`, "project_id");
  const result = db.run("INSERT INTO milestones (project_id, title, due_at, created_at) VALUES (?, ?, ?, ?)", [
    input.project_id,
    input.title,
    input.due_at ?? null,
    nowIso(),
  ]);
  return getMilestone(db, Number(result.lastInsertRowid))!;
}

export function listMilestones(db: Database, projectId: number): Milestone[] {
  return db
    .query<Milestone, [number]>("SELECT * FROM milestones WHERE project_id = ? ORDER BY id")
    .all(projectId);
}
