import type { Database } from "bun:sqlite";

export interface RouteContext {
  readonly req: Request;
  readonly url: URL;
  readonly params: Readonly<Record<string, string>>;
  readonly db: Database;
}

export interface Route {
  readonly method: string;
  /** Path with `:name` segments, e.g. `/tasks/:id`. */
  readonly path: string;
  readonly handler: (ctx: RouteContext) => Response | Promise<Response>;
  /** Accept the token from `?token=` too (EventSource cannot set headers). */
  readonly queryToken?: boolean;
}

export type Match =
  | { readonly kind: "match"; readonly route: Route; readonly params: Record<string, string> }
  | { readonly kind: "method"; readonly allowed: readonly string[] }
  | { readonly kind: "none" };

function matchPath(pattern: string, pathname: string): Record<string, string> | null {
  const want = pattern.split("/");
  const have = pathname.split("/");
  if (want.length !== have.length) return null;
  const params: Record<string, string> = {};
  for (let i = 0; i < want.length; i++) {
    const w = want[i]!;
    const h = have[i]!;
    if (w.startsWith(":")) {
      if (h === "") return null;
      params[w.slice(1)] = safeDecode(h);
    } else if (w !== h) {
      return null;
    }
  }
  return params;
}

function safeDecode(segment: string): string {
  try {
    return decodeURIComponent(segment);
  } catch {
    return segment;
  }
}

export function matchRoute(routes: readonly Route[], method: string, pathname: string): Match {
  const allowed: string[] = [];
  for (const route of routes) {
    const params = matchPath(route.path, pathname);
    if (!params) continue;
    if (route.method === method) return { kind: "match", route, params };
    allowed.push(route.method);
  }
  return allowed.length > 0 ? { kind: "method", allowed } : { kind: "none" };
}
