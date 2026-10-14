import { expect, test } from "bun:test";
import pkg from "../package.json" with { type: "json" };

test("workspace package is wired up", () => {
  expect(pkg.name).toBe("@adhd-todo/core");
});
