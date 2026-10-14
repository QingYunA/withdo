import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { createCoreClient, type Endpoint } from "./client";
import { CoreContext } from "./context";
import { isMyGo } from "mygo-runtime";
import { events } from "../mygo";
import { discoverEndpoint } from "./endpoint";

type Unsubscribe = () => void;

/** The Go shell sends `core-error` when adhd-core has died and could not be restarted. */
function watchShellErrors(onError: (message: string) => void): Unsubscribe {
  return isMyGo() ? events.coreError.on(onError) : () => {};
}

type State =
  | { readonly kind: "loading" }
  | { readonly kind: "ready"; readonly endpoint: Endpoint }
  | { readonly kind: "error"; readonly message: string };

export interface CoreProviderProps {
  readonly children: ReactNode;
  readonly discover?: () => Promise<Endpoint>;
  readonly watchErrors?: (onError: (message: string) => void) => Unsubscribe;
}

function Notice({ title, detail, onRetry }: { title: string; detail?: string; onRetry?: () => void }) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2 p-6 text-center">
      <p className="text-sm font-medium">{title}</p>
      {detail ? <p className="max-w-md text-xs text-muted">{detail}</p> : null}
      {onRetry ? (
        <button type="button" className="btn" onClick={onRetry}>
          Try again
        </button>
      ) : null}
    </div>
  );
}

/** Finds adhd-core, then renders children with a typed client in context. */
export function CoreProvider({ children, discover = discoverEndpoint, watchErrors = watchShellErrors }: CoreProviderProps) {
  const [state, setState] = useState<State>({ kind: "loading" });

  const connect = useCallback(() => {
    setState({ kind: "loading" });
    discover().then(
      (endpoint) => setState({ kind: "ready", endpoint }),
      (error: unknown) => setState({ kind: "error", message: error instanceof Error ? error.message : String(error) }),
    );
  }, [discover]);

  useEffect(connect, [connect]);
  useEffect(() => watchErrors((message) => setState({ kind: "error", message })), [watchErrors]);

  const value = useMemo(
    () => (state.kind === "ready" ? { endpoint: state.endpoint, client: createCoreClient(state.endpoint) } : null),
    [state],
  );

  if (state.kind === "loading") return <Notice title="Starting adhd-core..." />;
  if (state.kind === "error") return <Notice title="Cannot reach adhd-core" detail={state.message} onRetry={connect} />;
  return <CoreContext.Provider value={value}>{children}</CoreContext.Provider>;
}
