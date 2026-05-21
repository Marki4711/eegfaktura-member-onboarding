"use client";

import { useState } from "react";
import { useSession } from "next-auth/react";
import { Info } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import {
  CONFIGURABLE_FIELDS,
  saveFieldConfig,
  type AdminFieldConfig,
  type AdminFieldConfigEntry,
  type FieldState,
  type ConfigurableField,
  type VisibilityTag,
} from "@/lib/api";

// PROJ-45: tag display config. Distinct colours so admins can scan the
// list quickly; dark-mode variants included so the editor stays legible
// in both themes. "+" prefix on EV signals "additional condition".
const TAG_META: Record<VisibilityTag, { label: string; className: string }> = {
  consumption: {
    label: "Verbraucher",
    className: "bg-blue-100 text-blue-800 dark:bg-blue-950/60 dark:text-blue-300",
  },
  production: {
    label: "Einspeisung",
    className: "bg-amber-100 text-amber-900 dark:bg-amber-950/60 dark:text-amber-300",
  },
  pv: {
    label: "PV",
    className: "bg-orange-100 text-orange-900 dark:bg-orange-950/60 dark:text-orange-300",
  },
  ev: {
    label: "+E-Auto",
    className: "bg-purple-100 text-purple-900 dark:bg-purple-950/60 dark:text-purple-300",
  },
  // PROJ-49 follow-up: Speicher-Gruppe (Master-Toggle „Batteriespeicher
  // vorhanden" im Mitgliedsformular). "+" zeigt zusätzliche Abhängigkeit.
  battery: {
    label: "+Speicher",
    className: "bg-emerald-100 text-emerald-900 dark:bg-emerald-950/60 dark:text-emerald-300",
  },
  // PROJ-56: Netzbetreiber-Vollmacht-abhängige Felder. "+"-Prefix signalisiert
  // die zusätzliche Abhängigkeit von der Vollmacht-Checkbox im Public-Formular.
  network_authorization: {
    label: "+Vollmacht",
    className: "bg-sky-100 text-sky-900 dark:bg-sky-950/60 dark:text-sky-300",
  },
};

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
        <div className="flex items-center gap-1.5 flex-wrap">
          <span className="text-sm">{field.label}</span>
          {field.visibilityTags?.map((tag) => {
            const meta = TAG_META[tag];
            return (
              <span
                key={tag}
                className={`inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium leading-none ${meta.className}`}
              >
                {meta.label}
              </span>
            );
          })}
          {field.visibilityHint && (
            <Popover>
              <PopoverTrigger type="button" className="cursor-help" aria-label={`Hinweis zu ${field.label}`}>
                <Info className="h-3.5 w-3.5 text-muted-foreground" />
              </PopoverTrigger>
              <PopoverContent className="max-w-80 text-sm">
                {field.visibilityHint}
              </PopoverContent>
            </Popover>
          )}
        </div>
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
