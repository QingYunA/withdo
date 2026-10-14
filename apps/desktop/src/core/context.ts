import { createContext, useContext } from "react";
import type { CoreClient, Endpoint } from "./client";

export interface CoreContextValue {
  readonly client: CoreClient;
  readonly endpoint: Endpoint;
}

export const CoreContext = createContext<CoreContextValue | null>(null);

export function useCore(): CoreContextValue {
  const value = useContext(CoreContext);
  if (!value) throw new Error("useCore must be used inside <CoreProvider>");
  return value;
}
