"use client";

import { useEffect, useMemo, useState } from "react";
import { useSession } from "next-auth/react";
import { ArrowDown, ArrowUp, Info, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import {
  createDataExportConfig,
  previewDataExportConfig,
  updateDataExportConfig,
  type DataExportConfigResponse,
  type DataExportPreviewResponse,
  type DataExportStandardConfigInfo,
} from "@/lib/api";
import { formatValidationError } from "./error-utils";
import {
  defaultFormatForType,
  EXCEL_FIELD_CATALOG,
  EXCEL_FIELD_CATEGORIES,
  EXCEL_MAX_COLUMNS,
  findExcelField,
  formatOptionsForType,
  type ExcelColumnConfig,
  type ExcelConfig,
} from "@/lib/data-export-fields";

interface Props {
  rcNumber: string;
  // Either an existing config to edit OR a standard template to clone OR null
  // for a blank new config.
  initial?: DataExportConfigResponse | null;
  template?: DataExportStandardConfigInfo | null;
  onSaved: () => void;
  onCancel: () => void;
}

function emptyConfig(): ExcelConfig {
  return { format: "xlsx", columns: [] };
}

// isConfigComplete returns true when every column has both a non-empty
// header and a selected field. Used to suppress noisy preview API calls
// while the admin is mid-edit on a new column.
function isConfigComplete(cfg: ExcelConfig): boolean {
  if (cfg.columns.length === 0) return false;
  return cfg.columns.every((c) => c.header.trim() !== "" && c.field !== "");
}

function parseConfig(raw: Record<string, unknown> | undefined): ExcelConfig {
  if (!raw) return emptyConfig();
  const format = raw.format === "csv" ? "csv" : "xlsx";
  const cols = Array.isArray(raw.columns) ? raw.columns : [];
  const columns: ExcelColumnConfig[] = cols.map((c) => {
    const obj = c as Record<string, unknown>;
    return {
      header: String(obj.header ?? ""),
      field: String(obj.field ?? ""),
      format: String(obj.format ?? ""),
    };
  });
  return { format, columns };
}

export function DataExportExcelEditor({ rcNumber, initial, template, onSaved, onCancel }: Props) {
  const { data: session } = useSession();
  const [name, setName] = useState(initial?.name ?? template?.name ?? "");
  const [config, setConfig] = useState<ExcelConfig>(() => {
    if (initial) return parseConfig(initial.config);
    if (template) return parseConfig(template.config);
    return emptyConfig();
  });
  const [saving, setSaving] = useState(false);
  const [errors, setErrors] = useState<string[]>([]);

  const [preview, setPreview] = useState<DataExportPreviewResponse | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewErrors, setPreviewErrors] = useState<string[]>([]);

  // Debounced live-preview: rebuild whenever the config changes. Skipped
  // while the admin is still filling in a new column (no header / no field
  // selected yet) — otherwise the backend returns a noisy validation error
  // for every keystroke before the column is usable.
  useEffect(() => {
    if (!isConfigComplete(config)) {
      setPreview(null);
      setPreviewErrors([]);
      return;
    }
    const handle = window.setTimeout(() => {
      setPreviewLoading(true);
      setPreviewErrors([]);
      previewDataExportConfig(
        rcNumber,
        { pluginType: "excel", rcNumber, config: config as unknown as Record<string, unknown> },
        session?.accessToken,
      )
        .then((res) => setPreview(res))
        .catch((err: unknown) => setPreviewErrors(formatValidationError(err)))
        .finally(() => setPreviewLoading(false));
    }, 400);
    return () => window.clearTimeout(handle);
  }, [config, rcNumber, session?.accessToken]);

  const hasSensitive = useMemo(() => {
    return config.columns.some((c) => findExcelField(c.field)?.sensitive);
  }, [config.columns]);

  function addColumn() {
    if (config.columns.length >= EXCEL_MAX_COLUMNS) return;
    setConfig((c) => ({
      ...c,
      columns: [...c.columns, { header: "Neue Spalte", field: "", format: "" }],
    }));
  }

  function removeColumn(idx: number) {
    setConfig((c) => ({ ...c, columns: c.columns.filter((_, i) => i !== idx) }));
  }

  function moveColumn(idx: number, delta: -1 | 1) {
    setConfig((c) => {
      const cols = [...c.columns];
      const target = idx + delta;
      if (target < 0 || target >= cols.length) return c;
      [cols[idx], cols[target]] = [cols[target], cols[idx]];
      return { ...c, columns: cols };
    });
  }

  function updateColumn(idx: number, patch: Partial<ExcelColumnConfig>) {
    setConfig((c) => {
      const cols = [...c.columns];
      const merged = { ...cols[idx], ...patch };
      // When the field changes, reset format to a valid default for the new type.
      if (patch.field && patch.field !== cols[idx].field) {
        const def = findExcelField(patch.field);
        merged.format = def ? defaultFormatForType(def.type) : "";
      }
      cols[idx] = merged;
      return { ...c, columns: cols };
    });
  }

  async function handleSave() {
    if (!name.trim()) {
      setErrors(["Name ist erforderlich."]);
      return;
    }
    if (config.columns.length === 0) {
      setErrors(["Mindestens eine Spalte ist erforderlich."]);
      return;
    }
    setErrors([]);
    setSaving(true);
    try {
      const body = {
        pluginType: "excel",
        name: name.trim(),
        config: config as unknown as Record<string, unknown>,
      };
      if (initial) {
        await updateDataExportConfig(rcNumber, initial.id, body, session?.accessToken);
      } else {
        await createDataExportConfig(rcNumber, body, session?.accessToken);
      }
      onSaved();
    } catch (err) {
      setErrors(formatValidationError(err));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="excel-name">Name</Label>
          <Input
            id="excel-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={100}
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="excel-format">Format</Label>
          <Select
            value={config.format}
            onValueChange={(v) => setConfig((c) => ({ ...c, format: v as "xlsx" | "csv" }))}
          >
            <SelectTrigger id="excel-format">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="xlsx">XLSX (Excel)</SelectItem>
              <SelectItem value="csv">CSV (UTF-8, Semikolon)</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Label>Spalten ({config.columns.length} / {EXCEL_MAX_COLUMNS})</Label>
            {hasSensitive && (
              <Popover>
                <PopoverTrigger type="button" className="cursor-help">
                  <Info className="h-3.5 w-3.5 text-amber-600" />
                </PopoverTrigger>
                <PopoverContent className="max-w-80 text-sm">
                  Diese Konfiguration enthält sensible personenbezogene Daten (z.&nbsp;B. IBAN
                  oder Geburtsdatum). Sie tragen die Verantwortung für die DSGVO-konforme
                  Weiterverarbeitung im Zielsystem (Art.&nbsp;32 DSGVO).
                </PopoverContent>
              </Popover>
            )}
          </div>
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={addColumn}
            disabled={config.columns.length >= EXCEL_MAX_COLUMNS}
          >
            Spalte hinzufügen
          </Button>
        </div>

        {config.columns.length === 0 && (
          <div className="rounded border border-dashed py-6 text-center text-sm text-muted-foreground">
            Noch keine Spalten konfiguriert. Klicke auf <em>Spalte hinzufügen</em>.
          </div>
        )}

        {config.columns.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">#</TableHead>
                <TableHead>Spaltenkopf</TableHead>
                <TableHead>Feld</TableHead>
                <TableHead>Format</TableHead>
                <TableHead className="w-32 text-right">Aktionen</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {config.columns.map((col, idx) => {
                const def = findExcelField(col.field);
                const formats = def ? formatOptionsForType(def.type) : [];
                return (
                  <TableRow key={idx}>
                    <TableCell className="text-muted-foreground">{idx + 1}</TableCell>
                    <TableCell>
                      <Input
                        value={col.header}
                        onChange={(e) => updateColumn(idx, { header: e.target.value })}
                        maxLength={100}
                      />
                    </TableCell>
                    <TableCell>
                      <Select
                        value={col.field}
                        onValueChange={(v) => updateColumn(idx, { field: v })}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Feld wählen…" />
                        </SelectTrigger>
                        <SelectContent>
                          {EXCEL_FIELD_CATEGORIES.map((cat) => {
                            const items = EXCEL_FIELD_CATALOG.filter((f) => f.category === cat);
                            if (items.length === 0) return null;
                            return (
                              <SelectGroup key={cat}>
                                <SelectLabel>{cat}</SelectLabel>
                                {items.map((f) => (
                                  <SelectItem key={f.key} value={f.key}>
                                    {f.label}
                                  </SelectItem>
                                ))}
                              </SelectGroup>
                            );
                          })}
                        </SelectContent>
                      </Select>
                    </TableCell>
                    <TableCell>
                      <Select
                        value={col.format || (def ? defaultFormatForType(def.type) : "")}
                        onValueChange={(v) => updateColumn(idx, { format: v })}
                        disabled={!def}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {formats.map((f) => (
                            <SelectItem key={f.value} value={f.value}>
                              {f.label}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1">
                        <Button
                          type="button"
                          size="icon"
                          variant="ghost"
                          onClick={() => moveColumn(idx, -1)}
                          disabled={idx === 0}
                          aria-label="Spalte nach oben"
                        >
                          <ArrowUp className="h-4 w-4" />
                        </Button>
                        <Button
                          type="button"
                          size="icon"
                          variant="ghost"
                          onClick={() => moveColumn(idx, 1)}
                          disabled={idx === config.columns.length - 1}
                          aria-label="Spalte nach unten"
                        >
                          <ArrowDown className="h-4 w-4" />
                        </Button>
                        <Button
                          type="button"
                          size="icon"
                          variant="ghost"
                          onClick={() => removeColumn(idx)}
                          aria-label="Spalte entfernen"
                        >
                          <Trash2 className="h-4 w-4 text-destructive" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </div>

      {/* Live preview */}
      <div className="space-y-2">
        <Label>Vorschau (letzte 5 importierte Mitglieder)</Label>
        {previewLoading && <Skeleton className="h-24 w-full" />}
        {previewErrors.length > 0 && (
          <div className="rounded border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
            <p className="font-medium">Vorschau nicht möglich:</p>
            <ul className="mt-1 list-disc pl-5 space-y-0.5">
              {previewErrors.map((msg, i) => (
                <li key={i}>{msg}</li>
              ))}
            </ul>
          </div>
        )}
        {preview && !previewLoading && (
          <div className="rounded border overflow-auto">
            {preview.note && (
              <p className="border-b bg-muted/40 px-3 py-2 text-xs text-muted-foreground">
                {preview.note}
              </p>
            )}
            {preview.rows.length === 0 ? (
              <p className="px-3 py-4 text-sm text-muted-foreground">
                Keine Vorschau-Daten verfügbar.
              </p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    {preview.headers.map((h) => (
                      <TableHead key={h}>{h}</TableHead>
                    ))}
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {preview.rows.map((row, i) => (
                    <TableRow key={i}>
                      {preview.headers.map((h) => (
                        <TableCell key={h} className="text-xs">
                          {String(row[h] ?? "")}
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </div>
        )}
      </div>

      {errors.length > 0 && (
        <div className="rounded border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
          <p className="font-medium">Speichern nicht möglich:</p>
          <ul className="mt-1 list-disc pl-5 space-y-0.5">
            {errors.map((msg, i) => (
              <li key={i}>{msg}</li>
            ))}
          </ul>
        </div>
      )}

      <div className="flex justify-end gap-2 pt-2">
        <Button type="button" variant="outline" onClick={onCancel} disabled={saving}>
          Abbrechen
        </Button>
        <Button type="button" onClick={handleSave} disabled={saving}>
          {saving ? "Wird gespeichert…" : initial ? "Speichern" : "Anlegen"}
        </Button>
      </div>
    </div>
  );
}
