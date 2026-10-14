import type { ProjectWithCounts } from "../core/types";
import { selectionKey, sidebarItems, type Selection } from "../view/sidebar";

export interface SidebarProps {
  readonly projects: readonly ProjectWithCounts[];
  readonly selection: Selection | null;
  readonly settingsActive: boolean;
  readonly onSelect: (selection: Selection) => void;
  readonly onOpenSettings: () => void;
}

const itemClass = (active: boolean): string =>
  `flex w-full items-center justify-between rounded-md px-2 py-1 text-left hover:bg-hover ${active ? "bg-hover font-medium" : ""}`;

export function Sidebar({ projects, selection, settingsActive, onSelect, onOpenSettings }: SidebarProps) {
  const current = selection ? selectionKey(selection) : null;
  return (
    <div className="flex h-full w-48 shrink-0 flex-col border-r border-border bg-surface">
      <nav aria-label="Projects" className="flex-1 space-y-0.5 overflow-y-auto p-2">
        {sidebarItems(projects).map((item) => (
          <button
            key={item.key}
            type="button"
            className={itemClass(!settingsActive && current === item.key)}
            onClick={() => onSelect(item.selection)}
          >
            <span className="truncate">{item.label}</span>
            {item.open ? <span className="text-xs text-muted">{item.open}</span> : null}
          </button>
        ))}
      </nav>
      <div className="border-t border-border p-2">
        <button type="button" className={itemClass(settingsActive)} onClick={onOpenSettings}>
          Settings
        </button>
      </div>
    </div>
  );
}
