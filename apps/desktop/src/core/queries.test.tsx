import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { Providers, fakeClient, newQueryClient } from "../test/render";
import { makeTask } from "../test/fixtures";
import { queryKeys, useCreateTask, useMoveTask, useTasks, useUpdateTask } from "./queries";

function wrap(client = fakeClient(), queryClient = newQueryClient()) {
  return {
    queryClient,
    wrapper: ({ children }: { children: React.ReactNode }) => (
      <Providers client={client} queryClient={queryClient}>
        {children}
      </Providers>
    ),
  };
}

describe("useTasks", () => {
  test("fetches with the filter and caches under a filter-specific key", async () => {
    const tasks = vi.fn(async () => [makeTask({ id: 5 })]);
    const { wrapper, queryClient } = wrap(fakeClient({ tasks }));
    const { result } = renderHook(() => useTasks({ project_id: 2 }), { wrapper });
    await waitFor(() => expect(result.current.data).toHaveLength(1));
    expect(tasks).toHaveBeenCalledWith({ project_id: 2 });
    expect(queryClient.getQueryData(queryKeys.tasks({ project_id: 2 }))).toHaveLength(1);
  });
});

describe("useMoveTask", () => {
  const filter = { project_id: 1 };

  test("moves the card in the cache before the server answers, then settles", async () => {
    let release: () => void = () => {};
    const updateTask = vi.fn(
      () => new Promise<ReturnType<typeof makeTask>>((resolve) => (release = () => resolve(makeTask({ id: 1, status: "doing" })))),
    );
    const tasks = vi.fn(async () => [makeTask({ id: 1, status: "doing" })]);
    const { wrapper, queryClient } = wrap(fakeClient({ updateTask, tasks }));
    queryClient.setQueryData(queryKeys.tasks(filter), [makeTask({ id: 1, status: "todo" })]);
    const { result } = renderHook(() => useMoveTask(), { wrapper });

    act(() => result.current.mutate({ id: 1, status: "doing" }));
    await waitFor(() => expect(queryClient.getQueryData<{ status: string }[]>(queryKeys.tasks(filter))?.[0]?.status).toBe("doing"));
    expect(updateTask).toHaveBeenCalledWith(1, { status: "doing" });
    release();
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
  });

  test("rolls back when the server rejects the change", async () => {
    const updateTask = vi.fn(async () => {
      throw new Error("nope");
    });
    const { wrapper, queryClient } = wrap(fakeClient({ updateTask, tasks: async () => [makeTask({ id: 1, status: "todo" })] }));
    queryClient.setQueryData(queryKeys.tasks(filter), [makeTask({ id: 1, status: "todo" })]);
    const { result } = renderHook(() => useMoveTask(), { wrapper });
    act(() => result.current.mutate({ id: 1, status: "done" }));
    await waitFor(() => expect(result.current.isError).toBe(true));
    await waitFor(() => expect(queryClient.getQueryData<{ status: string }[]>(queryKeys.tasks(filter))?.[0]?.status).toBe("todo"));
  });
});

describe("mutations", () => {
  test("useCreateTask posts and refreshes every query", async () => {
    const createTask = vi.fn(async () => makeTask({ id: 9 }));
    const { wrapper, queryClient } = wrap(fakeClient({ createTask }));
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useCreateTask(), { wrapper });
    await act(async () => {
      await result.current.mutateAsync({ title: "x" });
    });
    expect(createTask).toHaveBeenCalledWith({ title: "x" });
    expect(invalidate).toHaveBeenCalled();
  });

  test("useUpdateTask patches and refreshes every query", async () => {
    const updateTask = vi.fn(async () => makeTask({ id: 4, title: "new" }));
    const { wrapper, queryClient } = wrap(fakeClient({ updateTask }));
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");
    const { result } = renderHook(() => useUpdateTask(), { wrapper });
    await act(async () => {
      await result.current.mutateAsync({ id: 4, patch: { title: "new" } });
    });
    expect(updateTask).toHaveBeenCalledWith(4, { title: "new" });
    expect(invalidate).toHaveBeenCalled();
  });
});
