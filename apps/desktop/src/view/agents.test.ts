import { describe, expect, test } from "vitest";
import { makeOverviewTask, makeReport } from "../test/fixtures";
import type { Overview } from "../core/types";
import { agentGroups } from "./agents";

const now = new Date("2026-10-14T12:00:00.000Z");
const hoursAgo = (h: number) => new Date(now.getTime() - h * 3_600_000).toISOString();

function overview(groups: Overview["projects"]): Overview {
  return { generated_at: now.toISOString(), stale_hours: 48, projects: groups };
}

describe("agentGroups", () => {
  test("keeps the server's grouping and names inbox tasks", () => {
    const groups = agentGroups(
      overview([
        { project: { id: 1, name: "widgets", repo_slug: "acme/widgets" }, tasks: [makeOverviewTask({ id: 1, status: "blocked" })] },
        { project: null, tasks: [makeOverviewTask({ id: 2 })] },
      ]),
      now,
    );
    expect(groups.map((g) => g.name)).toEqual(["widgets", "Inbox"]);
    expect(groups[0]?.key).toBe("project-1");
    expect(groups[1]?.key).toBe("inbox");
  });

  test("builds a row from the latest report and its age", () => {
    const [group] = agentGroups(
      overview([
        {
          project: { id: 1, name: "widgets", repo_slug: null },
          tasks: [
            makeOverviewTask({
              id: 5,
              title: "Login",
              status: "review",
              branch: "feat/login",
              pr_url: "https://github.com/a/b/pull/3",
              latest_report: makeReport({ summary: "PR is up", created_at: hoursAgo(3) }),
            }),
          ],
        },
      ]),
      now,
    );
    expect(group?.rows[0]).toMatchObject({
      id: 5,
      title: "Login",
      status: "review",
      branch: "feat/login",
      summary: "PR is up",
      age: "3h",
      staleLabel: null,
    });
  });

  test("falls back to updated_at when there is no report and marks stale tasks", () => {
    const [group] = agentGroups(
      overview([
        {
          project: { id: 1, name: "widgets", repo_slug: null },
          tasks: [makeOverviewTask({ id: 6, updated_at: hoursAgo(72), stale: true, latest_report: null })],
        },
      ]),
      now,
    );
    expect(group?.rows[0]).toMatchObject({ summary: null, age: "3d", staleLabel: "Stale 3d" });
  });

  test("drops tasks outside doing/blocked/review and groups left empty", () => {
    const groups = agentGroups(
      overview([
        { project: { id: 1, name: "a", repo_slug: null }, tasks: [makeOverviewTask({ status: "done" })] },
        { project: { id: 2, name: "b", repo_slug: null }, tasks: [makeOverviewTask({ id: 9, status: "doing" })] },
      ]),
      now,
    );
    expect(groups.map((g) => g.name)).toEqual(["b"]);
  });
});
