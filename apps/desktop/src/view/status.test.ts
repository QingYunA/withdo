import { describe, expect, test } from "vitest";
import { TASK_STATUSES } from "../core/types";
import { PRIORITY_LABEL, STATUS_DOT, STATUS_LABEL, priorityRank } from "./status";

describe("status presentation", () => {
  test("every status has a label and a dot colour class", () => {
    for (const status of TASK_STATUSES) {
      expect(STATUS_LABEL[status]).toMatch(/^[A-Z]/);
      expect(STATUS_DOT[status]).toMatch(/^bg-status-/);
    }
  });

  test("priorities rank P0 first", () => {
    expect(["P2", "P0", "P3", "P1"].sort((a, b) => priorityRank(a as "P0") - priorityRank(b as "P0"))).toEqual(["P0", "P1", "P2", "P3"]);
    expect(PRIORITY_LABEL.P0).toBe("Urgent");
  });
});
