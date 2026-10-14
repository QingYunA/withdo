import type { Report } from "../core/types";
import { StatusDot } from "../ui/StatusDot";
import { formatAge } from "../view/age";

/** Agent reports, newest first. The server returns them oldest first. */
export function ReportTimeline({ reports, now }: { reports: readonly Report[]; now: Date }) {
  if (reports.length === 0) return <p className="text-xs text-muted">No reports yet. Agents add them with report_progress.</p>;
  return (
    <ol className="space-y-3">
      {[...reports].reverse().map((report) => (
        <li key={report.id} className="text-xs">
          <p className="flex items-center gap-1.5 text-muted">
            {report.status ? <StatusDot status={report.status} /> : null}
            <span>{report.agent}</span>
            <span title={new Date(report.created_at).toLocaleString()}>{formatAge(report.created_at, now)}</span>
          </p>
          <p className="mt-0.5 text-[13px]">{report.summary}</p>
          {report.next_step ? <p className="text-muted">Next: {report.next_step}</p> : null}
        </li>
      ))}
    </ol>
  );
}
