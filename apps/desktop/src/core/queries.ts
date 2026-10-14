import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCore } from "./context";
import { withTaskStatus } from "./optimistic";
import type { NewTask, Task, TaskFilter, TaskPatch, TaskStatus } from "./types";

export const queryKeys = {
  all: ["core"] as const,
  overview: () => ["core", "overview"] as const,
  projects: () => ["core", "projects"] as const,
  tasks: (filter: TaskFilter = {}) => ["core", "tasks", filter] as const,
  task: (id: number) => ["core", "task", id] as const,
  milestones: (projectId: number) => ["core", "milestones", projectId] as const,
};

const TASKS_PREFIX = ["core", "tasks"] as const;

export function useOverview() {
  const { client } = useCore();
  return useQuery({ queryKey: queryKeys.overview(), queryFn: () => client.overview() });
}

export function useProjects() {
  const { client } = useCore();
  return useQuery({ queryKey: queryKeys.projects(), queryFn: () => client.projects() });
}

export function useTasks(filter: TaskFilter = {}, options: { enabled?: boolean } = {}) {
  const { client } = useCore();
  return useQuery({ queryKey: queryKeys.tasks(filter), queryFn: () => client.tasks(filter), ...options });
}

export function useTask(id: number | null) {
  const { client } = useCore();
  return useQuery({
    queryKey: queryKeys.task(id ?? 0),
    queryFn: () => client.task(id as number),
    enabled: id !== null,
  });
}

export function useMilestones(projectId: number | null) {
  const { client } = useCore();
  return useQuery({
    queryKey: queryKeys.milestones(projectId ?? 0),
    queryFn: () => client.milestones(projectId as number),
    enabled: projectId !== null,
  });
}

export function useCreateTask() {
  const { client } = useCore();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (input: NewTask) => client.createTask(input),
    onSettled: () => queryClient.invalidateQueries({ queryKey: queryKeys.all }),
  });
}

export function useUpdateTask() {
  const { client } = useCore();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, patch }: { id: number; patch: TaskPatch }) => client.updateTask(id, patch),
    onSettled: () => queryClient.invalidateQueries({ queryKey: queryKeys.all }),
  });
}

/** Drag-and-drop status change: updates every cached task list at once, rolls back on failure. */
export function useMoveTask() {
  const { client } = useCore();
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }: { id: number; status: TaskStatus }) => client.updateTask(id, { status }),
    onMutate: async ({ id, status }) => {
      await queryClient.cancelQueries({ queryKey: TASKS_PREFIX });
      const snapshot = queryClient.getQueriesData<Task[]>({ queryKey: TASKS_PREFIX });
      queryClient.setQueriesData<Task[]>({ queryKey: TASKS_PREFIX }, (tasks) => withTaskStatus(tasks, id, status));
      return { snapshot };
    },
    onError: (_error, _vars, context) => {
      for (const [key, data] of context?.snapshot ?? []) queryClient.setQueryData(key, data);
    },
    onSettled: () => queryClient.invalidateQueries({ queryKey: queryKeys.all }),
  });
}
