import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useCoreEvents } from "../core/use-core-events";
import { useCreateTask, useOverview, useTasks } from "../core/queries";
import { hidePanel, onShellEvent, openTaskInMain } from "../platform/shell";
import { useNow } from "../ui/use-now";
import { agentGroups } from "../view/agents";
import { mineTasks, moveSelection } from "../view/mine";
import { QuickAdd } from "./QuickAdd";
import { AgentsSection, MineSection } from "./Sections";
import { useQueryClient } from "@tanstack/react-query";

const MINE_FILTER = { status: ["todo", "doing"] } as const;

const errorText = (error: unknown): string | null => (error ? (error instanceof Error ? error.message : String(error)) : null);

/** Spotlight-style panel: quick capture on top, what agents are doing, then the user's own tasks. */
export function PanelView() {
  useCoreEvents();
  const queryClient = useQueryClient();
  const now = useNow();
  const overview = useOverview();
  const tasks = useTasks(MINE_FILTER);
  const create = useCreateTask();

  const inputRef = useRef<HTMLInputElement>(null);
  const [value, setValue] = useState("");
  const [notice, setNotice] = useState<string | null>(null);
  const [active, setActive] = useState<number | null>(null);

  const groups = useMemo(() => (overview.data ? agentGroups(overview.data, now) : []), [overview.data, now]);
  const mine = useMemo(() => mineTasks(tasks.data ?? []), [tasks.data]);
  const entries = useMemo(
    () => [
      ...groups.flatMap((g) => g.rows.map((r) => ({ key: `agent-${r.id}`, id: r.id }))),
      ...mine.map((t) => ({ key: `mine-${t.id}`, id: t.id })),
    ],
    [groups, mine],
  );

  const open = useCallback((id: number) => {
    void openTaskInMain(id).then(hidePanel);
  }, []);

  const submit = () => {
    const title = value.trim();
    if (!title) return;
    setNotice(null);
    create.mutate(
      { title },
      {
        onSuccess: () => {
          setValue("");
          setNotice("Added to Inbox");
          inputRef.current?.focus();
        },
      },
    );
  };

  const onEnter = () => {
    const entry = active === null ? undefined : entries[active];
    if (value.trim() || !entry) submit();
    else open(entry.id);
  };

  useEffect(() => {
    const refocus = () => {
      inputRef.current?.focus();
      void queryClient.invalidateQueries();
    };
    window.addEventListener("focus", refocus);
    const stopShown = onShellEvent("panel-shown", refocus);
    return () => {
      window.removeEventListener("focus", refocus);
      stopShown();
    };
  }, [queryClient]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setValue("");
        setNotice(null);
        void hidePanel();
      } else if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        event.preventDefault();
        setActive((current) => moveSelection(current, event.key === "ArrowDown" ? 1 : -1, entries.length));
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [entries.length]);

  const activeKey = active === null ? null : (entries[active]?.key ?? null);

  return (
    <div className="mx-auto flex h-full max-w-[640px] flex-col border border-border bg-bg">
      <QuickAdd
        value={value}
        onChange={(next) => {
          setValue(next);
          setNotice(null);
        }}
        onEnter={onEnter}
        inputRef={inputRef}
        notice={notice}
        error={errorText(create.error)}
        busy={create.isPending}
      />
      <div className="min-h-0 flex-1 overflow-y-auto">
        <AgentsSection groups={groups} error={errorText(overview.error)} activeKey={activeKey} onOpen={open} />
        <MineSection tasks={mine} now={now} error={errorText(tasks.error)} activeKey={activeKey} onOpen={open} />
      </div>
      <footer className="flex gap-4 border-t border-border px-4 py-1.5 text-[11px] text-muted">
        <span>Enter to add</span>
        <span>Up/Down to browse</span>
        <span>Esc to close</span>
      </footer>
    </div>
  );
}
