import type { TaskStatus } from "../core/types";
import { STATUS_DOT, STATUS_LABEL } from "../view/status";

export function StatusDot({ status, className = "" }: { status: TaskStatus; className?: string }) {
  return (
    <span
      role="img"
      aria-label={STATUS_LABEL[status]}
      title={STATUS_LABEL[status]}
      className={`inline-block size-2 shrink-0 rounded-full ${STATUS_DOT[status]} ${className}`}
    />
  );
}
