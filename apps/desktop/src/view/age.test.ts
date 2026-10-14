import { describe, expect, test } from "vitest";
import { formatAge, formatWhen, staleBadge } from "./age";

const now = new Date("2026-10-14T12:00:00.000Z");
const ago = (ms: number) => new Date(now.getTime() - ms).toISOString();
const MIN = 60_000;
const HOUR = 60 * MIN;
const DAY = 24 * HOUR;

describe("formatAge", () => {
  test.each([
    [10_000, "now"],
    [5 * MIN, "5m"],
    [59 * MIN, "59m"],
    [3 * HOUR, "3h"],
    [23 * HOUR, "23h"],
    [2 * DAY, "2d"],
    [45 * DAY, "6w"],
  ])("%d ms ago reads %s", (ms, expected) => {
    expect(formatAge(ago(ms), now)).toBe(expected);
  });

  test("a time slightly in the future (clock skew) reads now", () => {
    expect(formatAge(new Date(now.getTime() + 5000).toISOString(), now)).toBe("now");
  });
});

describe("staleBadge", () => {
  test("is null for fresh tasks", () => {
    expect(staleBadge({ stale: false, since: ago(HOUR) }, now)).toBeNull();
  });

  test("shows how long the task has been quiet", () => {
    expect(staleBadge({ stale: true, since: ago(3 * DAY) }, now)).toBe("Stale 3d");
  });
});

describe("formatWhen", () => {
  test("future times read 'in ...'", () => {
    expect(formatWhen(new Date(now.getTime() + 90 * MIN).toISOString(), now)).toEqual({ text: "in 1h", overdue: false });
    expect(formatWhen(new Date(now.getTime() + 20_000).toISOString(), now)).toEqual({ text: "now", overdue: false });
    expect(formatWhen(new Date(now.getTime() + 2 * DAY + HOUR).toISOString(), now).text).toBe("in 2d");
  });

  test("past times are overdue", () => {
    expect(formatWhen(ago(2 * HOUR), now)).toEqual({ text: "2h ago", overdue: true });
  });
});
