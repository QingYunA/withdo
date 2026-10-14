import { isMyGo } from "mygo-runtime";
import { Core } from "../mygo";
import type { Endpoint } from "./client";

export interface DiscoveryDeps {
  /** True when running inside the mygo shell. */
  readonly inShell: boolean;
  /** Asks the Go shell where adhd-core listens; it waits until the sidecar is ready. */
  readonly conn?: () => Promise<{ readonly port: number; readonly token: string }>;
  readonly env: Readonly<Record<string, string | undefined>>;
}

const defaultDeps = (): DiscoveryDeps => ({
  inShell: isMyGo(),
  conn: () => Core.conn(),
  env: { VITE_CORE_URL: import.meta.env.VITE_CORE_URL, VITE_CORE_TOKEN: import.meta.env.VITE_CORE_TOKEN },
});

/**
 * Where is adhd-core? Inside the mygo shell, the Go side knows (it started the sidecar). In a plain
 * browser the URL and token come from `VITE_CORE_URL` and `VITE_CORE_TOKEN`.
 */
export async function discoverEndpoint(deps: DiscoveryDeps = defaultDeps()): Promise<Endpoint> {
  if (deps.inShell && deps.conn) {
    const { port, token } = await deps.conn();
    return { baseUrl: `http://127.0.0.1:${port}`, token };
  }
  const url = deps.env.VITE_CORE_URL;
  const token = deps.env.VITE_CORE_TOKEN;
  if (!url || !token) {
    throw new Error("Set VITE_CORE_URL and VITE_CORE_TOKEN to reach adhd-core from a browser (see apps/desktop/README.md).");
  }
  return { baseUrl: url.replace(/\/+$/, ""), token };
}
