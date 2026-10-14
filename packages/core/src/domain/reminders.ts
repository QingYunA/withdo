import type { Database } from "bun:sqlite";
import { nowIso } from "./types.ts";

export interface Reminder {
  readonly task_id: number;
  readonly title: string;
  /** Null for inbox tasks. */
  readonly project_name: string | null;
}

/**
 * Tasks whose reminder is due (`remind_at <= now`, not yet reminded, not done or cancelled).
 * Marks each one as reminded in the same transaction, so a reminder is handed out exactly once.
 */
export function claimDueReminders(db: Database, now: string = nowIso()): Reminder[] {
  const claim = db.transaction((): Reminder[] => {
    const due = db
      .query<Reminder, [string]>(
        `SELECT t.id AS task_id, t.title, p.name AS project_name
           FROM tasks t LEFT JOIN projects p ON p.id = t.project_id
          WHERE t.remind_at IS NOT NULL AND t.remind_at <= ? AND t.reminded_at IS NULL
            AND t.status NOT IN ('done','cancelled')
          ORDER BY t.remind_at, t.id`,
      )
      .all(now);
    return due.filter(
      (r) => db.run("UPDATE tasks SET reminded_at = ? WHERE id = ? AND reminded_at IS NULL", [now, r.task_id]).changes === 1,
    );
  });
  // IMMEDIATE takes the write lock up front, so a concurrent MCP writer cannot invalidate our snapshot.
  return claim.immediate();
}
