"use client";

import { useState } from "react";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  CONFIGURABLE_FIELDS,
  saveFieldConfig,
  type AdminFieldConfig,
  type AdminFieldConfigEntry,
  type FieldState,
  type ConfigurableField,
} from "@/lib/api";

const STATE_OPTIONS: { value: FieldState; label: string }[] = [
  { value: "hidden",     label: "Ausblenden" },
  { value: "optional",   label: "Optional" },
  { value: "required",   label: "Pflichtfeld" },
  { value: "admin_only", label: "Admin-Vorgabe" },
];

function FieldRow({
  field,
  entry,
  onChange,
}: {
  field: ConfigurableField;
  entry: AdminFieldConfigEntry;
  onChange: (entry: AdminFieldConfigEntry) => void;
}) {
  return (
    <div className="py-2 space-y-1.5">
      <div className="flex items-center justify-between gap-4">
        <span className="text-sm">{field.label}</span>
        <div className="flex rounded-md border border-border overflow-hidden shrink-0">
          {STATE_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              type="button"
              onClick={() => onChange({ ...entry, state: opt.value })}
              className={[
                "px-3 py-1.5 text-xs font-medium transition-colors",
                entry.state === opt.value
                  ? "bg-primary text-primary-foreground"
                  : "bg-background text-muted-foreground hover:bg-muted",
                "border-r border-border last:border-r-0",
              ].join(" ")}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>
      {entry.state === "admin_only" && (
        <div className="pl-0 pr-0">
          <Input
            value={entry.adminValue ?? ""}
            onChange={(e) => onChange({ ...entry, adminValue: e.target.value || undefined })}
            placeholder="Standardwert (wird automatisch auf neue Anträge angewendet)"
            className="h-8 text-xs"
          />
        </div>
      )}
    </div>
  );
}

interface Props {
  rcNumber: string;
  initialConfig: AdminFieldConfig;
}

export function AdminFieldConfigEditor({ rcNumber, initialConfig }: Props) {
  const { data: session } = useSession();
  const [config, setConfig] = useState<AdminFieldConfig>(() => {
    const merged: AdminFieldConfig = {};
    for (const f of [...CONFIGURABLE_FIELDS.application, ...CONFIGURABLE_FIELDS.meteringPoint]) {
      merged[f.name] = initialConfig[f.name] ?? { state: f.defaultState };
    }
    return merged;
  });
  const [saving, setSaving] = useState(false);
  const [saveResult, setSaveResult] = useState<"ok" | "error" | null>(null);

  const setField = (name: string, entry: AdminFieldConfigEntry) => {
    setSaveResult(null);
    setConfig((prev) => ({ ...prev, [name]: entry }));
  };

  const handleSave = async () => {
    setSaving(true);
    setSaveResult(null);
    try {
      await saveFieldConfig(rcNumber, config, session?.accessToken);
      setSaveResult("ok");
    } catch {
      setSaveResult("error");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Antragsteller-Felder</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="divide-y divide-border">
            {CONFIGURABLE_FIELDS.application.map((field) => (
              <FieldRow
                key={field.name}
                field={field}
                entry={config[field.name] ?? { state: field.defaultState }}
                onChange={(e) => setField(field.name, e)}
              />
            ))}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Zählpunkt-Felder</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="divide-y divide-border">
            {CONFIGURABLE_FIELDS.meteringPoint.map((field) => (
              <FieldRow
                key={field.name}
                field={field}
                entry={config[field.name] ?? { state: field.defaultState }}
                onChange={(e) => setField(field.name, e)}
              />
            ))}
          </div>
        </CardContent>
      </Card>

      <div className="flex items-center gap-3">
        <Button onClick={handleSave} disabled={saving}>
          {saving ? "Wird gespeichert…" : "Speichern"}
        </Button>
        {saveResult === "ok" && (
          <span className="text-sm text-green-600">Konfiguration gespeichert</span>
        )}
        {saveResult === "error" && (
          <span className="text-sm text-destructive">Fehler beim Speichern</span>
        )}
      </div>
    </div>
  );
}
