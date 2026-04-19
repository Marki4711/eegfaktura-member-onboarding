import { Separator } from "@/components/ui/separator";
import { AdminStatusBadge } from "@/components/admin-status-badge";
import type { StatusLogEntry } from "@/lib/api";

interface Props {
  entries: StatusLogEntry[];
}

function formatDateTime(iso: string) {
  return new Date(iso).toLocaleString("de-AT", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function AdminStatusLog({ entries }: Props) {
  if (entries.length === 0) {
    return (
      <p className="text-sm text-muted-foreground py-4">
        Noch keine Statuseinträge vorhanden.
      </p>
    );
  }

  const sorted = [...entries].sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

  return (
    <div className="space-y-3">
      {sorted.map((entry, i) => (
        <div key={i}>
          <div className="flex flex-wrap items-center gap-2 text-sm">
            <span className="text-muted-foreground">{formatDateTime(entry.createdAt)}</span>
            {entry.fromStatus && (
              <>
                <AdminStatusBadge status={entry.fromStatus} />
                <span className="text-muted-foreground">→</span>
              </>
            )}
            <AdminStatusBadge status={entry.toStatus} />
          </div>
          {entry.reason && (
            <p className="text-sm text-muted-foreground mt-1 ml-0 italic">
              {entry.reason}
            </p>
          )}
          {i < sorted.length - 1 && <Separator className="mt-3" />}
        </div>
      ))}
    </div>
  );
}
