import type { Task } from "../core/types";
import { CONNECT_COMMAND, EmptyState } from "../ui/EmptyState";
import type { AgentGroup } from "../view/agents";
import { AgentRowView, MineRowView } from "./Rows";

function SectionTitle({ children }: { children: string }) {
  return <h2 className="px-4 pt-3 pb-1 text-[11px] font-medium uppercase tracking-wide text-muted">{children}</h2>;
}

interface AgentsProps {
  readonly groups: readonly AgentGroup[];
  readonly error: string | null;
  readonly activeKey: string | null;
  readonly onOpen: (id: number) => void;
}

export function AgentsSection({ groups, error, activeKey, onOpen }: AgentsProps) {
  return (
    <section aria-label="Agents">
      <SectionTitle>Agents</SectionTitle>
      {error ? <p className="px-4 py-2 text-xs text-danger">{error}</p> : null}
      {!error && groups.length === 0 ? (
        <EmptyState title="No agents are reporting yet">
          Connect Claude Code with <span className="code">{CONNECT_COMMAND}</span>. Tasks appear here when an agent calls report_progress.
        </EmptyState>
      ) : null}
      {groups.map((group) => (
        <div key={group.key} role="group" aria-label={group.name} className="px-2 pb-1">
          <p className="px-2 pt-1 text-xs text-muted">{group.name}</p>
          {group.rows.map((row) => (
            <AgentRowView key={row.id} row={row} active={activeKey === `agent-${row.id}`} onOpen={() => onOpen(row.id)} />
          ))}
        </div>
      ))}
    </section>
  );
}

interface MineProps {
  readonly tasks: readonly Task[];
  readonly now: Date;
  readonly error: string | null;
  readonly activeKey: string | null;
  readonly onOpen: (id: number) => void;
}

export function MineSection({ tasks, now, error, activeKey, onOpen }: MineProps) {
  return (
    <section aria-label="Mine">
      <SectionTitle>Mine</SectionTitle>
      {error ? <p className="px-4 py-2 text-xs text-danger">{error}</p> : null}
      {!error && tasks.length === 0 ? (
        <EmptyState title="Nothing on your list">Type above and press Enter to add a task to your inbox.</EmptyState>
      ) : null}
      <div className="px-2 pb-2">
        {tasks.map((task) => (
          <MineRowView key={task.id} task={task} now={now} active={activeKey === `mine-${task.id}`} onOpen={() => onOpen(task.id)} />
        ))}
      </div>
    </section>
  );
}
