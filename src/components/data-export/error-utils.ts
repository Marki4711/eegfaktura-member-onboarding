import { ApiResponseError } from "@/lib/api";

// formatValidationError turns an ApiResponseError (or any error) into a list
// of human-readable lines: top-level message followed by per-field messages
// when the backend returned them. Column-path keys (`columns[1].header`) are
// prettified to "Spalte 2 → Spaltenkopf" so admins don't have to mentally
// translate the JSON path.
//
// Used by every data-export UI surface (editor, trigger-dialog, job-status,
// configs-list) so error display is consistent across the feature.
export function formatValidationError(err: unknown): string[] {
  if (err instanceof ApiResponseError) {
    const fields = err.apiError.fields ?? {};
    const keys = Object.keys(fields);
    if (keys.length === 0) {
      return [err.apiError.message || "Validierung fehlgeschlagen"];
    }
    return keys.map((key) => `${prettifyFieldKey(key)}: ${fields[key]}`);
  }
  if (err instanceof Error) return [err.message];
  return ["Unbekannter Fehler"];
}

function prettifyFieldKey(key: string): string {
  const m = /^columns\[(\d+)\]\.(.+)$/.exec(key);
  if (m) {
    const idx = Number(m[1]) + 1;
    const sub = m[2];
    const subLabel =
      sub === "header" ? "Spaltenkopf" :
      sub === "field" ? "Feld" :
      sub === "format" ? "Format" : sub;
    return `Spalte ${idx} → ${subLabel}`;
  }
  if (key === "name") return "Name";
  if (key === "columns") return "Spalten";
  if (key === "format") return "Format";
  if (key === "config") return "Konfiguration";
  if (key === "pluginType") return "Plugin-Typ";
  if (key === "applicationIds") return "Antrags-IDs";
  if (key === "configId") return "Konfiguration";
  if (key === "limit") return "Limit";
  return key;
}
