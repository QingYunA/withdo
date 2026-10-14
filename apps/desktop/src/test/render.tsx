import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, type RenderResult } from "@testing-library/react";
import type { ReactElement } from "react";
import type { CoreClient, Endpoint } from "../core/client";
import { CoreContext } from "../core/context";

export const TEST_ENDPOINT: Endpoint = { baseUrl: "http://127.0.0.1:1", token: "t" };

/** A client whose every method rejects, so a test fails loudly if it uses something it did not stub. */
export function fakeClient(overrides: Partial<CoreClient> = {}): CoreClient {
  const unexpected = (name: string) => async () => {
    throw new Error(`unexpected client.${name}() call`);
  };
  return {
    overview: unexpected("overview"),
    projects: unexpected("projects"),
    tasks: unexpected("tasks"),
    task: unexpected("task"),
    createTask: unexpected("createTask"),
    updateTask: unexpected("updateTask"),
    milestones: unexpected("milestones"),
    createMilestone: unexpected("createMilestone"),
    ...overrides,
  };
}

export function newQueryClient(): QueryClient {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } } });
}

export function Providers({ client, queryClient, children }: { client: CoreClient; queryClient: QueryClient; children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <CoreContext.Provider value={{ client, endpoint: TEST_ENDPOINT }}>{children}</CoreContext.Provider>
    </QueryClientProvider>
  );
}

export function renderWithCore(
  ui: ReactElement,
  options: { client?: CoreClient; queryClient?: QueryClient } = {},
): RenderResult & { queryClient: QueryClient; client: CoreClient } {
  const client = options.client ?? fakeClient();
  const queryClient = options.queryClient ?? newQueryClient();
  const result = render(ui, {
    wrapper: ({ children }) => (
      <Providers client={client} queryClient={queryClient}>
        {children}
      </Providers>
    ),
  });
  return Object.assign(result, { queryClient, client });
}
