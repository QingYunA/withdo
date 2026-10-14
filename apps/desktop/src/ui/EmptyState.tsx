import type { ReactNode } from "react";

export function EmptyState({ title, children }: { title: string; children?: ReactNode }) {
  return (
    <div className="px-4 py-5 text-xs text-muted">
      <p className="mb-1 font-medium text-fg">{title}</p>
      {children ? <p className="leading-relaxed">{children}</p> : null}
    </div>
  );
}

export const CONNECT_COMMAND = "claude mcp add --scope user adhd-todo -- /path/to/adhd-core mcp";
