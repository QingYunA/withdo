import * as z from "zod";
import { parseInput } from "./respond.ts";

/** Parse a decimal id from a path segment, validating it with one of the shared id schemas. */
export function idParam(raw: string | undefined, schema: z.ZodType<number, number>, field: string): number {
  const digits = z.string({ error: "is required" }).regex(/^\d+$/, "must be a positive integer");
  return parseInput(digits.transform(Number).pipe(schema), raw, field);
}

/** A query value for a numeric field: absent -> undefined, numeric text -> number, else the raw text (zod rejects it). */
export function numberQuery(params: URLSearchParams, name: string): number | string | undefined {
  const raw = params.get(name);
  if (raw === null) return undefined;
  return /^-?\d+(\.\d+)?$/.test(raw) ? Number(raw) : raw;
}
