import type { Database } from "bun:sqlite";
import { bearerToken, tokenMatches } from "./auth.ts";
import { createEventHub, type EventHub, type EventHubOptions } from "./events.ts";
import { preflight, withCors } from "./cors.ts";
import { HttpError, jsonError, toErrorResponse } from "./respond.ts";
import { matchRoute, type Match, type Route } from "./router.ts";
import { ALL_ROUTES } from "./routes/index.ts";

const HOST = "127.0.0.1";
const MAX_BODY_BYTES = 1024 * 1024;

export interface HttpServerOptions {
  readonly token: string;
  /** 0 picks a random free port. */
  readonly port?: number;
  /** Receives diagnostics; defaults to stderr. Never stdout: that carries the ready line. */
  readonly log?: (message: string) => void;
  /** Replace the route table (tests). */
  readonly routes?: readonly Route[];
  /** Poll and keep-alive intervals for /events (tests shorten them). */
  readonly events?: EventHubOptions;
}

export interface RunningServer {
  readonly hostname: string;
  readonly port: number;
  readonly url: string;
  readonly token: string;
  readonly hub: EventHub;
  stop(): void;
}

const stderrLog = (message: string): void => {
  process.stderr.write(`[adhd-todo] ${message}\n`);
};

function authorize(req: Request, url: URL, match: Match, token: string): void {
  const queryAllowed = match.kind === "match" && match.route.queryToken === true;
  const provided = bearerToken(req) ?? (queryAllowed ? url.searchParams.get("token") : null);
  if (!tokenMatches(provided, token)) throw new HttpError(401, "unauthorized");
}

const READ_METHODS: ReadonlySet<string> = new Set(["GET", "HEAD"]);

interface HandlerDeps {
  readonly db: Database;
  readonly hub: EventHub;
  readonly token: string;
  readonly routes: readonly Route[];
  readonly log: (message: string) => void;
}

function createHandler({ db, hub, token, routes, log }: HandlerDeps) {
  return async (req: Request): Promise<Response> => {
    if (req.method === "OPTIONS") return preflight(req);
    try {
      const url = new URL(req.url);
      const match = matchRoute(routes, req.method, url.pathname);
      authorize(req, url, match, token);
      if (match.kind === "none") throw new HttpError(404, "not found");
      if (match.kind === "method") {
        const res = jsonError("method not allowed", 405);
        res.headers.set("allow", [...new Set(match.allowed)].join(", "));
        return withCors(req, res);
      }
      const res = await match.route.handler({ req, url, params: match.params, db, hub });
      if (!READ_METHODS.has(req.method) && res.status < 400) hub.notifyChanged();
      return withCors(req, res);
    } catch (error) {
      return withCors(req, toErrorResponse(error, log));
    }
  };
}

/** Start the HTTP API on 127.0.0.1. */
export function startHttpServer(db: Database, options: HttpServerOptions): RunningServer {
  const log = options.log ?? stderrLog;
  const hub = createEventHub(db, { log, ...options.events });
  const routes = options.routes ?? ALL_ROUTES;
  const server = Bun.serve({
    hostname: HOST,
    port: options.port ?? 0,
    idleTimeout: 0,
    maxRequestBodySize: MAX_BODY_BYTES,
    fetch: createHandler({ db, hub, token: options.token, routes, log }),
  });
  return {
    hostname: HOST,
    port: server.port!,
    url: `http://${HOST}:${server.port}`,
    token: options.token,
    hub,
    stop: () => {
      hub.close();
      void server.stop(true);
    },
  };
}
