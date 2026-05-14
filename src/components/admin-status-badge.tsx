import { Badge } from "@/components/ui/badge";
import type { ApplicationStatus } from "@/lib/api";

const STATUS_CONFIG: Record<
  ApplicationStatus,
  { label: string; className: string }
> = {
  draft:           { label: "Entwurf",                className: "bg-gray-100 text-gray-700 hover:bg-gray-100" },
  submitted:       { label: "Eingereicht",            className: "bg-blue-100 text-blue-700 hover:bg-blue-100" },
  email_confirmed: { label: "E-Mail bestätigt",       className: "bg-teal-100 text-teal-700 hover:bg-teal-100" },
  under_review:    { label: "In Prüfung",             className: "bg-yellow-100 text-yellow-700 hover:bg-yellow-100" },
  needs_info:      { label: "Info benötigt",          className: "bg-orange-100 text-orange-700 hover:bg-orange-100" },
  approved:        { label: "Genehmigt",              className: "bg-green-100 text-green-700 hover:bg-green-100" },
  rejected:        { label: "Abgelehnt",              className: "bg-red-100 text-red-700 hover:bg-red-100" },
  imported:        { label: "Importiert",             className: "bg-emerald-100 text-emerald-700 hover:bg-emerald-100" },
  import_failed:   { label: "Import fehlgeschlagen",  className: "bg-red-100 text-red-800 hover:bg-red-100" },
};

export function AdminStatusBadge({ status }: { status: string }) {
  const config = STATUS_CONFIG[status as ApplicationStatus] ?? {
    label: status,
    className: "bg-gray-100 text-gray-600",
  };
  return (
    <Badge variant="secondary" className={config.className}>
      {config.label}
    </Badge>
  );
}
