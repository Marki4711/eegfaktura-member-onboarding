"use client";

import { useEffect, useState } from "react";
import { useSession } from "next-auth/react";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { AdminFieldConfigEditor } from "@/components/admin-field-config-editor";
import { AdminIntroTextEditor } from "@/components/admin-intro-text-editor";
import { Separator } from "@/components/ui/separator";
import { getFieldConfig, type FieldConfig } from "@/lib/api";

export default function SettingsPage() {
  const { data: session } = useSession();

  const rcNumbers: string[] = (session as unknown as { tenant?: string[] })?.tenant ?? [];
  const isSuperuser = ((session as unknown as { roles?: string[] })?.roles ?? []).includes("superuser");

  const [selectedRc, setSelectedRc] = useState<string>("");
  const [fieldConfig, setFieldConfig] = useState<FieldConfig | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (rcNumbers.length > 0 && !selectedRc) {
      setSelectedRc(rcNumbers[0]);
    }
  }, [rcNumbers, selectedRc]);

  useEffect(() => {
    if (!selectedRc) return;
    setLoading(true);
    setError(null);
    setFieldConfig(null);
    getFieldConfig(selectedRc, session?.accessToken)
      .then(setFieldConfig)
      .catch(() => setError("Konfiguration konnte nicht geladen werden. Bitte später erneut versuchen."))
      .finally(() => setLoading(false));
  }, [selectedRc, session?.accessToken]);

  if (isSuperuser && rcNumbers.length === 0) {
    return (
      <div className="space-y-6">
        <h1 className="text-xl font-semibold">Einstellungen</h1>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground text-sm">
            Als Superuser bitte RC-Nummer direkt in der URL angeben:<br />
            <code className="text-xs mt-2 block">/admin/settings?rc=RC123456</code>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!isSuperuser && rcNumbers.length === 0) {
    return (
      <div className="space-y-6">
        <h1 className="text-xl font-semibold">Einstellungen</h1>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground text-sm">
            Ihrem Account sind keine EEGs zugewiesen.
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-4">
        <h1 className="text-xl font-semibold">Einstellungen</h1>
        {rcNumbers.length > 1 && (
          <Select value={selectedRc} onValueChange={setSelectedRc}>
            <SelectTrigger className="w-48">
              <SelectValue placeholder="EEG auswählen" />
            </SelectTrigger>
            <SelectContent>
              {rcNumbers.map((rc) => (
                <SelectItem key={rc} value={rc}>{rc}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        {rcNumbers.length === 1 && (
          <span className="text-sm text-muted-foreground">{selectedRc}</span>
        )}
      </div>

      {/* Einleitungstext */}
      {selectedRc && (
        <div>
          <h2 className="text-base font-medium mb-1">Einleitungstext</h2>
          <p className="text-sm text-muted-foreground mb-4">
            Wird oberhalb des Registrierungsformulars angezeigt. Unterstützt Fett, Kursiv, Listen und Links.
            Leer lassen für den Standardtext.
          </p>
          <AdminIntroTextEditor rcNumber={selectedRc} />
        </div>
      )}

      <Separator />

      <div>
        <h2 className="text-base font-medium mb-1">Formular-Felder</h2>
        <p className="text-sm text-muted-foreground mb-4">
          Legen Sie fest, welche optionalen Felder im Registrierungsformular für Ihre EEG angezeigt werden.
        </p>

        {loading && (
          <div className="space-y-4">
            <Skeleton className="h-48 w-full" />
            <Skeleton className="h-32 w-full" />
          </div>
        )}

        {error && (
          <Card>
            <CardContent className="py-8 text-center text-sm text-destructive">
              {error}
            </CardContent>
          </Card>
        )}

        {!loading && !error && fieldConfig && selectedRc && (
          <AdminFieldConfigEditor rcNumber={selectedRc} initialConfig={fieldConfig} />
        )}
      </div>
    </div>
  );
}
