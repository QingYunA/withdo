export const ALLOWED_ORIGINS: ReadonlySet<string> = new Set([
  "http://localhost:1420", // Vite dev server
  "tauri://localhost", // Tauri on macOS and Linux
  "http://tauri.localhost", // Tauri on Windows
]);

const ALLOW_METHODS = "GET, POST, PATCH, OPTIONS";
const ALLOW_HEADERS = "Authorization, Content-Type";

/** The request's Origin when it is on the allow-list, else null. */
export function allowedOrigin(req: Request): string | null {
  const origin = req.headers.get("origin");
  return origin !== null && ALLOWED_ORIGINS.has(origin) ? origin : null;
}

/** Answer an `OPTIONS` request. It needs no token; disallowed browser origins get 403 and no CORS headers. */
export function preflight(req: Request): Response {
  if (req.headers.get("origin") === null) return new Response(null, { status: 204 });
  const origin = allowedOrigin(req);
  if (origin === null) return Response.json({ error: "origin not allowed" }, { status: 403 });
  return new Response(null, {
    status: 204,
    headers: {
      "access-control-allow-origin": origin,
      "access-control-allow-methods": ALLOW_METHODS,
      "access-control-allow-headers": ALLOW_HEADERS,
      "access-control-max-age": "600",
      vary: "Origin",
    },
  });
}

/** A copy of `res` with CORS headers when the request came from an allowed origin. */
export function withCors(req: Request, res: Response): Response {
  const origin = allowedOrigin(req);
  if (origin === null) return res;
  const headers = new Headers(res.headers);
  headers.set("access-control-allow-origin", origin);
  headers.append("vary", "Origin");
  return new Response(res.body, { status: res.status, statusText: res.statusText, headers });
}
