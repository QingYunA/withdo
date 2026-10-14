import { createHash, timingSafeEqual } from "node:crypto";

const digest = (value: string): Buffer => createHash("sha256").update(value).digest();

/** Constant-time token comparison. Hashing first makes the compared buffers equal length. */
export function tokenMatches(provided: string | null | undefined, expected: string): boolean {
  const equal = timingSafeEqual(digest(provided ?? ""), digest(expected));
  return equal && !!provided;
}

/** The token from `Authorization: Bearer <token>`, or null. */
export function bearerToken(req: Request): string | null {
  const match = /^Bearer (.+)$/i.exec(req.headers.get("authorization") ?? "");
  return match?.[1] ?? null;
}
