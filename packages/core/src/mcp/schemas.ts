import * as z from "zod";
import { PRIORITIES, TASK_STATUSES } from "../domain/types.ts";

export const statusSchema = z.enum(TASK_STATUSES);
export const prioritySchema = z.enum(PRIORITIES);

const id = (what: string) => z.number().int().positive().describe(what);
export const taskId = id("Task id, as returned by report_progress or create_task");
export const projectId = id("Project id");
export const milestoneId = id("Milestone id");

export const cwdSchema = z
  .string()
  .min(1)
  .describe("Absolute path of the working directory; used to find the git repo, project and branch");

/** ISO 8601 date or datetime (offsets allowed); normalised to a UTC timestamp. */
export const timestamp = z
  .union([z.iso.datetime({ offset: true }), z.iso.date()])
  .transform((s) => new Date(s).toISOString());

export const prUrl = z.url().describe("URL of the pull request, e.g. https://github.com/owner/repo/pull/12");
