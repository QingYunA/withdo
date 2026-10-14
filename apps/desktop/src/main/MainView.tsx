import { useCallback, useEffect, useMemo, useState } from "react";
import { useCoreEvents } from "../core/use-core-events";
import { useMilestones, useMoveTask, useProjects, useTask, useTasks } from "../core/queries";
import { onMainNavigate, openExternal } from "../platform/shell";
import { EmptyState } from "../ui/EmptyState";
import { BOARD_HASH, taskHash, type Route } from "../view/route";
import { defaultSelection, selectionFilter, selectionForTask, type Selection } from "../view/sidebar";
import { Board } from "./Board";
import { Sidebar } from "./Sidebar";
import { TaskDrawer } from "./TaskDrawer";

type MainRoute = Extract<Route, { name: "main" }>;

const go = (hash: string): void => {
  window.location.hash = hash;
};

/** Project kanban with a task drawer. */
export function MainView({ route }: { route: MainRoute }) {
  useCoreEvents();

  const projects = useProjects();
  const linked = useTask(route.taskId);
  const [picked, setPicked] = useState<Selection | null>(null);
  const [milestoneId, setMilestoneId] = useState<number | null>(null);
  const move = useMoveTask();

  const linkedTask = linked.data?.task;
  useEffect(() => {
    if (linkedTask) setPicked(selectionForTask(linkedTask));
  }, [linkedTask?.id]);

  // The Go shell asks us to show a task (panel click, notification click).
  useEffect(() => onMainNavigate(go), []);

  const projectList = projects.data ?? [];
  const selection = picked ?? defaultSelection(projectList);
  const selectedProject = selection.kind === "project" ? projectList.find((p) => p.id === selection.id) : undefined;
  const tasks = useTasks(selectionFilter(selection), { enabled: projects.isSuccess });
  const milestones = useMilestones(selection.kind === "project" ? selection.id : null);

  const select = useCallback((next: Selection) => {
    setPicked(next);
    setMilestoneId(null);
    go(BOARD_HASH);
  }, []);
  const closeDrawer = useCallback(() => go(BOARD_HASH), []);
  const title = selection.kind === "inbox" ? "Inbox" : (selectedProject?.name ?? "Project");

  const taskList = useMemo(() => tasks.data ?? [], [tasks.data]);

  return (
    <div className="relative flex h-full">
      <Sidebar projects={projectList} selection={selection} onSelect={select} />
      <main className="flex min-w-0 flex-1 flex-col">
        <header className="flex items-center gap-3 border-b border-border px-4 py-2">
        <h1 className="text-sm font-semibold">{title}</h1>
        {selectedProject?.repo_slug ? <span className="text-xs text-muted">{selectedProject.repo_slug}</span> : null}
        <div className="ml-auto">
          <select
            aria-label="Milestone"
            className="field !w-auto"
            value={milestoneId ?? ""}
            disabled={(milestones.data ?? []).length === 0}
            onChange={(e) => setMilestoneId(e.target.value === "" ? null : Number(e.target.value))}
          >
            <option value="">All milestones</option>
            {(milestones.data ?? []).map((m) => (
              <option key={m.id} value={m.id}>
                {m.title}
              </option>
            ))}
          </select>
        </div>
        </header>
        {tasks.isSuccess && taskList.length === 0 ? (
        <EmptyState title={selection.kind === "inbox" ? "Your inbox is empty" : "No tasks in this project yet"}>
          {selection.kind === "inbox"
            ? "Capture a task from the quick panel and it lands here."
            : "Tasks appear when an agent reports progress from this repository, or when you add one from the quick panel."}
        </EmptyState>
        ) : null}
        {tasks.isError ? <p className="px-4 py-2 text-xs text-danger">{tasks.error.message}</p> : null}
        <Board
        tasks={taskList}
        milestoneId={milestoneId}
        onMove={(m) => move.mutate(m)}
        onOpen={(id) => go(taskHash(id))}
        onOpenPr={openExternal}
        />
      </main>
      {route.taskId !== null ? <TaskDrawer taskId={route.taskId} onClose={closeDrawer} /> : null}
    </div>
  );
}
