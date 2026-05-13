// Display timezone for every user-visible date/datetime in the admin web.
// Mirrors the backend (Go: shared.DisplayLocation = Europe/Vienna) so that
// PDF, email and UI all render the same wall-clock time, independent of the
// admin's local browser timezone.
const DISPLAY_TZ = "Europe/Vienna";
const LOCALE = "de-AT";

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleString(LOCALE, {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    timeZone: DISPLAY_TZ,
  });
}

// Use for backend `timestamp` columns that should be rendered as a date only.
export function formatDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString(LOCALE, {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    timeZone: DISPLAY_TZ,
  });
}

// Use for backend DATE columns (no time component). The ISO string is parsed
// as plain Y-M-D so no timezone shift ever happens.
export function formatPlainDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  const [year, month, day] = iso.slice(0, 10).split("-").map(Number);
  return new Date(year, month - 1, day).toLocaleDateString(LOCALE, {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });
}
