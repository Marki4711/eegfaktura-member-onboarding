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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { AdminFieldConfigEditor } from "@/components/admin-field-config-editor";
import { AdminIntroTextEditor } from "@/components/admin-intro-text-editor";
import { AdminEEGSettingsEditor } from "@/components/admin-eeg-settings-editor";
import { AdminApiKeyEditor } from "@/components/admin-api-key-editor";
import { AdminLegalDocumentsEditor } from "@/components/admin-legal-documents-editor";
import { DataExportSection } from "@/components/data-export/section";
import { getFieldConfig, type AdminFieldConfig } from "@/lib/api";

// SettingsPage groups the per-EEG configuration into Tabs (statt einer
// langen vertikalen Liste). Tabs persistieren über Re-Renders innerhalb
// der Page; bei EEG-Wechsel bleibt der Tab erhalten, der geladene
// fieldConfig wird automatisch neu geholt.
export default function SettingsPage() {
  const { data: session } = useSession();

  const rcNumbers: string[] = (session as unknown as { tenant?: string[] })?.tenant ?? [];
  const isSuperuser = ((session as unknown as { roles?: string[] })?.roles ?? []).includes("superuser");

  const [selectedRc, setSelectedRc] = useState<string>("");
  const [fieldConfig, setFieldConfig] = useState<AdminFieldConfig | null>(null);
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
      .then((res) => setFieldConfig(res.fieldConfig ?? {}))
      .catch(() => setError("Konfiguration konnte nicht geladen werden. Bitte später erneut versuchen."))
      .finally(() => setLoading(false));
  }, [selectedRc, session?.accessToken]);

  if (isSuperuser && rcNumbers.length === 0) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Einstellungen</h1>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground text-sm">
            Als Superuser ohne zugewiesene EEGs: Bitte zuerst eine EEG in der
            Antragsliste auswählen, dann kehre hierher zurück.
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!isSuperuser && rcNumbers.length === 0) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Einstellungen</h1>
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground text-sm">
            Deinem Account sind keine EEGs zugewiesen.
          </CardContent>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-4">
        <h1 className="text-2xl font-bold">Einstellungen</h1>
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

      {selectedRc && (
        <Tabs defaultValue="stammdaten" className="w-full">
          <TabsList className="flex flex-wrap h-auto">
            <TabsTrigger value="stammdaten">Stammdaten &amp; SEPA</TabsTrigger>
            <TabsTrigger value="einleitung">Einleitungstext</TabsTrigger>
            <TabsTrigger value="formular">Formular-Felder</TabsTrigger>
            <TabsTrigger value="dokumente">Rechtsdokumente</TabsTrigger>
            <TabsTrigger value="api">Externe API</TabsTrigger>
            <TabsTrigger value="export">Datenweiterleitung</TabsTrigger>
          </TabsList>

          <TabsContent value="stammdaten" className="mt-6 space-y-3">
            <div>
              <h2 className="text-xl font-semibold mb-1">EEG-Stammdaten &amp; SEPA-Mandat</h2>
              <p className="text-sm text-muted-foreground">
                Aktiviere oder deaktiviere die öffentliche Registrierung und konfiguriere
                das SEPA-Lastschriftmandat. Das Mandat wird je nach Einstellung als PDF-Anhang in
                der Eingangsbestätigung oder erst nach erfolgreichem Import (mit Mitgliedsnummer
                als Mandatsreferenz) versendet.
              </p>
            </div>
            <AdminEEGSettingsEditor rcNumber={selectedRc} />
          </TabsContent>

          <TabsContent value="einleitung" className="mt-6 space-y-3">
            <div>
              <h2 className="text-xl font-semibold mb-1">Einleitungstext</h2>
              <p className="text-sm text-muted-foreground">
                Wird oberhalb des Registrierungsformulars angezeigt. Unterstützt Fett, Kursiv, Listen und Links.
                Leer lassen für den Standardtext.
              </p>
            </div>
            <AdminIntroTextEditor rcNumber={selectedRc} />
          </TabsContent>

          <TabsContent value="formular" className="mt-6 space-y-3">
            <div>
              <h2 className="text-xl font-semibold mb-1">Formular-Felder &amp; Zählpunktfelder</h2>
              <p className="text-sm text-muted-foreground">
                Lege fest, welche optionalen Felder im Registrierungsformular für deine EEG angezeigt werden.
                Felder können ausgeblendet, optional, verpflichtend oder als Admin-Vorbefüllung konfiguriert werden.
              </p>
            </div>
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
            {!loading && !error && fieldConfig && (
              <AdminFieldConfigEditor rcNumber={selectedRc} initialConfig={fieldConfig} />
            )}
          </TabsContent>

          <TabsContent value="dokumente" className="mt-6 space-y-3">
            <div>
              <h2 className="text-xl font-semibold mb-1">Rechtsdokumente</h2>
              <p className="text-sm text-muted-foreground">
                EEG-spezifische Dokumente (z. B. Satzung, Nutzungsbedingungen), denen Mitglieder bei
                der Registrierung zustimmen müssen. Die zentrale Datenschutzerklärung wird global
                vom Betreiber konfiguriert; ob sie zusätzlich im Formular angezeigt wird, steuerst
                du unten per Toggle.
              </p>
            </div>
            <AdminLegalDocumentsEditor rcNumber={selectedRc} />
          </TabsContent>

          <TabsContent value="api" className="mt-6 space-y-3">
            <div>
              <h2 className="text-xl font-semibold mb-1">Externe API</h2>
              <p className="text-sm text-muted-foreground">
                API-Key für die externe Registrierungs-API. Der Key ermöglicht das Einreichen von Mitgliedsanträgen
                über eine eigene Integration (z.&nbsp;B. eigenes Formular auf deiner Website).
                Der Key darf ausschließlich server-seitig verwendet werden — niemals in Browser-seitigem Code.
              </p>
            </div>
            <AdminApiKeyEditor rcNumber={selectedRc} />
          </TabsContent>

          <TabsContent value="export" className="mt-6 space-y-3">
            <div>
              <h2 className="text-xl font-semibold mb-1">Datenweiterleitung</h2>
              <p className="text-sm text-muted-foreground">
                Konfiguriere Spalten-Mappings, um importierte Mitglieder als Excel- oder CSV-Datei zu
                exportieren. Aus der Antragsliste kannst du beliebige Anträge selektieren und an eine
                Konfiguration weitergeben — Datei wird im Hintergrund erzeugt und steht 24 Stunden zum
                Download bereit.
              </p>
            </div>
            <DataExportSection rcNumber={selectedRc} />
          </TabsContent>
        </Tabs>
      )}
    </div>
  );
}
