import { expect, test } from "bun:test";
import { DomainError } from "../src/domain/errors.ts";
import { guarded } from "../src/mcp/tools/helpers.ts";

test("guarded maps DomainError to a field-naming error result", () => {
  const res = guarded(() => {
    throw new DomainError("must be positive", "limit");
  })({});
  expect(res.isError).toBe(true);
  expect(res.content[0]).toMatchObject({ text: "limit: must be positive" });
});

test("guarded hides unexpected errors behind an internal-error result", () => {
  const res = guarded(() => {
    throw new Error("disk on fire");
  })({});
  expect(res.isError).toBe(true);
  expect(res.content[0]).toMatchObject({ text: "internal error: disk on fire" });
});
