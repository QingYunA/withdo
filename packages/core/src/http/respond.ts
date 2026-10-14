import * as z from "zod";
import { DomainError, NotFoundError } from "../domain/errors.ts";

/** A failure that maps straight to a status code and a client-visible message. */
export class HttpError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "HttpError";
    this.status = status;
  }
}

export const jsonOk = (data: unknown, status = 200): Response => Response.json({ data }, { status });

export const jsonError = (message: string, status: number): Response => Response.json({ error: message }, { status });

function describeIssues(error: z.ZodError, field: string | undefined): string {
  return error.issues
    .map((issue) => {
      const path = issue.path.map(String).join(".") || field;
      return path ? `${path}: ${issue.message}` : issue.message;
    })
    .join("; ");
}

/** Validate `value`; failures become a 400 whose message names the field. */
export function parseInput<S extends z.ZodType>(schema: S, value: unknown, field?: string): z.output<S> {
  const result = schema.safeParse(value);
  if (!result.success) throw new HttpError(400, describeIssues(result.error, field));
  return result.data;
}

export async function readJsonBody(req: Request): Promise<unknown> {
  try {
    return await req.json();
  } catch {
    throw new HttpError(400, "body: must be valid JSON");
  }
}

/** Map any thrown value to a response. Only messages meant for clients are exposed; no stacks. */
export function toErrorResponse(error: unknown, log: (message: string) => void): Response {
  if (error instanceof HttpError) return jsonError(error.message, error.status);
  if (error instanceof NotFoundError) return jsonError(error.message, 404);
  if (error instanceof DomainError) return jsonError(error.message, 400);
  log(`request failed: ${error instanceof Error ? error.message : String(error)}`);
  return jsonError("internal error", 500);
}
