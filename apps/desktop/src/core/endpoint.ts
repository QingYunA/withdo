import type { Endpoint } from "./client";

type Invoke = (command: string) => Promise<unknown>;

export interface DiscoveryDeps {
  /** True when running inside the Tauri shell. */
  readonly tauri: boolean;
  readonly invoke?: Invoke;
  readonly env: Readonly<Record<string, string | undefined>>;
}

/** The Tauri runtime injects this global into every webview it creates. */
export const isTauri = (): boolean => typeof window !== "undefined" && "__TAURI_INTERNALS__" in window;

async function tauriInvoke(command: string): Promise<unknown> {
  const { invoke } = await import("@tauri-apps/api/core");
  return invoke(command);
}

const describe = (error: unknown): string => (error instanceof Error ? error.message : String(error));

const defaultDeps = (): DiscoveryDeps => ({
  tauri: isTauri(),
  env: { VITE_CORE_URL: import.meta.env.VITE_CORE_URL, VITE_CORE_TOKEN: import.meta.env.VITE_CORE_TOKEN },
});

/**
 * Where is adhd-core? Inside Tauri, the Rust shell knows (it started the sidecar). In a plain browser
 * the URL and token come from `VITE_CORE_URL` and `VITE_CORE_TOKEN`.
 */
export async function discoverEndpoint(deps: DiscoveryDeps = defaultDeps()): Promise<Endpoint> {
  if (deps.tauri) {
    try {
      const { port, token } = (await (deps.invoke ?? tauriInvoke)("core_endpoint")) as { port: number; token: string };
      return { baseUrl: `http://127.0.0.1:${port}`, token };
    } catch (error) {
      throw new Error(describe(error));
    }
  }
  const url = deps.env.VITE_CORE_URL;
  const token = deps.env.VITE_CORE_TOKEN;
  if (!url || !token) {
    throw new Error("Set VITE_CORE_URL and VITE_CORE_TOKEN to reach adhd-core from a browser (see apps/desktop/README.md).");
  }
  return { baseUrl: url.replace(/\/+$/, ""), token };
}
