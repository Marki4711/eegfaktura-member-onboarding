"use client";

import { useCallback, useEffect, useState } from "react";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { CheckCircle2, AlertCircle, RefreshCw, Lock } from "lucide-react";
import {
  getEEGSettings,
  saveEEGSettings,
  compareEEGSettingsWithCore,
  syncEEGSettingsFromCore,
  type EEGSettings,
  type EEGSettingsComparisonResponse,
} from "@/lib/api";
import { formatDateTime } from "@/lib/datetime";

interface Props {
  rcNumber: string;
}

// SyncedField is a read-only display row for one of the seven Core-mastered
// fields. Shows the (locked) value plus a small icon — looks like an input
// but isn't.
function SyncedField({ label, value }: { label: string; value: string | null | undefined }) {
  return (
    <div className="space-y-1.5">
      <Label className="text-sm flex items-center gap-1.5 text-muted-foreground">
        {label}
        <Lock className="h-3 w-3" />
      </Label>
      <div className="h-9 px-3 py-1.5 rounded-md border bg-muted/40 text-sm flex items-center">
        {value && value.length > 0 ? value : <span className="text-muted-foreground italic">—</span>}
      </div>
    </div>
  );
}

export function AdminEEGSettingsEditor({ rcNumber }: Props) {
  const { data: session } = useSession();
  const [loaded, setLoaded] = useState(false);
  const [settings, setSettings] = useState<EEGSettings | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveResult, setSaveResult] = useState<"ok" | "error" | null>(null);

  // PROJ-32 sync state
  const [comparison, setComparison] = useState<EEGSettingsComparisonResponse | null>(null);
  const [comparisonLoaded, setComparisonLoaded] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [syncError, setSyncError] = useState<string | null>(null);
  const [diffExpanded, setDiffExpanded] = useState(false);

  // Onboarding-only editable fields
  const [eegId, setEegId] = useState("");
  const [sepaMandateEnabled, setSepaMandateEnabled] = useState(false);
  const [useCompanySEPAMandate, setUseCompanySEPAMandate] = useState(false);
  const [registrationActive, setRegistrationActive] = useState(false);
  const [requireEmailConfirmation, setRequireEmailConfirmation] = useState(false);

  const reloadSettings = useCallback(async () => {
    if (!rcNumber || !session?.accessToken) return;
    const s = await getEEGSettings(rcNumber, session.accessToken);
    setSettings(s);
    setRegistrationActive(s.registrationActive ?? false);
    setEegId(s.eegId ?? "");
    setSepaMandateEnabled(s.sepaMandateEnabled);
    setUseCompanySEPAMandate(s.useCompanySEPAMandate ?? false);
    setRequireEmailConfirmation(s.requireEmailConfirmation ?? false);
    setLoaded(true);
  }, [rcNumber, session?.accessToken]);

  const reloadComparison = useCallback(async () => {
    if (!rcNumber || !session?.accessToken) return;
    try {
      const c = await compareEEGSettingsWithCore(rcNumber, session.accessToken);
      setComparison(c);
    } catch {
      // Endpoint returns 503 when CORE_BASE_URL is empty — treat as
      // "feature disabled" and render no banner at all.
      setComparison(null);
    } finally {
      setComparisonLoaded(true);
    }
  }, [rcNumber, session?.accessToken]);

  useEffect(() => {
    reloadSettings().catch(() => setLoaded(true));
  }, [reloadSettings]);

  useEffect(() => {
    reloadComparison();
  }, [reloadComparison]);

  const handleSave = async () => {
    setSaving(true);
    setSaveResult(null);
    try {
      await saveEEGSettings(
        rcNumber,
        {
          registrationActive,
          eegId: eegId.trim() || null,
          // Synced fields are NOT sent — backend ignores them anyway.
          eegName: settings?.eegName ?? null,
          eegStreet: settings?.eegStreet ?? null,
          eegStreetNumber: settings?.eegStreetNumber ?? null,
          eegZip: settings?.eegZip ?? null,
          eegCity: settings?.eegCity ?? null,
          creditorId: settings?.creditorId ?? null,
          sepaMandateEnabled,
          useCompanySEPAMandate,
          requireEmailConfirmation,
        },
        session?.accessToken,
      );
      setSaveResult("ok");
    } catch {
      setSaveResult("error");
    } finally {
      setSaving(false);
    }
  };

  const handleSync = async () => {
    if (!session?.accessToken) return;
    setSyncing(true);
    setSyncError(null);
    try {
      const updated = await syncEEGSettingsFromCore(rcNumber, session.accessToken);
      setComparison(updated);
      // Reload the settings so the read-only fields show the new values.
      await reloadSettings();
    } catch (err) {
      setSyncError(err instanceof Error ? err.message : "Sync fehlgeschlagen");
    } finally {
      setSyncing(false);
    }
  };

  const fieldClass = "h-9 text-sm";

  return (
    <div className="space-y-4">
      {!loaded && <p className="text-xs text-muted-foreground">Lädt…</p>}

      {loaded && settings && (
        <>
          {/* Onboarding-Steuerung — editierbar */}
          <div className="flex items-center justify-between rounded-md border px-4 py-3 bg-muted/40">
            <div>
              <p className="text-sm font-medium">Mitgliederregistrierung aktiv</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                Wenn deaktiviert, erhalten Besucher des Registrierungslinks eine Fehlermeldung.
              </p>
            </div>
            <Switch
              id="registration-active"
              checked={registrationActive}
              onCheckedChange={(v) => {
                setRegistrationActive(v);
                setSaveResult(null);
              }}
            />
          </div>

          {/* Gemeinschafts-ID (Onboarding-only) */}
          <div className="space-y-1.5">
            <Label htmlFor="eeg-id" className="text-sm">
              Gemeinschafts-ID
            </Label>
            <Input
              id="eeg-id"
              value={eegId}
              onChange={(e) => {
                setEegId(e.target.value);
                setSaveResult(null);
              }}
              className={fieldClass}
            />
            <p className="text-xs text-muted-foreground">
              Wird im Excel-Export (Spalte B) für den eegFaktura-Import verwendet.
            </p>
          </div>

          {/* PROJ-32: Stammdaten aus eegFaktura — read-only */}
          <div className="rounded-md border bg-card p-4 space-y-3">
            <div className="flex items-start justify-between gap-2">
              <div>
                <p className="text-sm font-medium">Stammdaten</p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Diese Daten werden aus eegFaktura übernommen und können nur dort geändert werden.
                </p>
              </div>
              {comparisonLoaded && comparison && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleSync}
                  disabled={syncing}
                  className="shrink-0"
                >
                  <RefreshCw className={`h-3.5 w-3.5 mr-1.5 ${syncing ? "animate-spin" : ""}`} />
                  {syncing ? "Wird aktualisiert…" : "Aus eegFaktura aktualisieren"}
                </Button>
              )}
            </div>

            {/* Banner */}
            {comparisonLoaded && comparison && (
              <>
                {!comparison.coreReachable && (
                  <Alert className="py-2">
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription className="text-xs">
                      eegFaktura ist gerade nicht erreichbar
                      {comparison.lastSyncedAt && <> — Stand: {formatDateTime(comparison.lastSyncedAt)}</>}
                      .
                    </AlertDescription>
                  </Alert>
                )}
                {comparison.coreReachable && comparison.inSync && (
                  <Alert className="border-green-300 bg-green-50 text-green-900 py-2">
                    <CheckCircle2 className="h-4 w-4 text-green-600" />
                    <AlertDescription className="text-xs">
                      Synchron mit eegFaktura
                      {comparison.lastSyncedAt && <> · Stand: {formatDateTime(comparison.lastSyncedAt)}</>}
                    </AlertDescription>
                  </Alert>
                )}
                {comparison.coreReachable && !comparison.inSync && (
                  <Alert className="border-orange-300 bg-orange-50 text-orange-900 py-2">
                    <AlertCircle className="h-4 w-4 text-orange-600" />
                    <AlertDescription className="text-xs space-y-1">
                      <p>
                        <strong>Stammdaten weichen von eegFaktura ab.</strong>{" "}
                        Klicke „Aus eegFaktura aktualisieren", um die Daten zu übernehmen.
                      </p>
                      <button
                        type="button"
                        onClick={() => setDiffExpanded((v) => !v)}
                        className="underline text-orange-900 hover:text-orange-700"
                      >
                        {diffExpanded ? "Details verbergen ▴" : "Details anzeigen ▾"}
                      </button>
                      {diffExpanded && comparison.differingFields && (
                        <table className="w-full mt-2 text-xs border-collapse">
                          <thead>
                            <tr className="text-left text-orange-700">
                              <th className="pr-2 pb-1 font-medium">Feld</th>
                              <th className="pr-2 pb-1 font-medium">Im Onboarding</th>
                              <th className="pb-1 font-medium">In eegFaktura</th>
                            </tr>
                          </thead>
                          <tbody>
                            {comparison.differingFields.map((d) => (
                              <tr key={d.field} className="border-t border-orange-200">
                                <td className="pr-2 py-1">{d.label}</td>
                                <td className="pr-2 py-1">
                                  {d.localValue || <span className="italic text-orange-700">—</span>}
                                </td>
                                <td className="py-1">
                                  {d.coreValue || <span className="italic text-orange-700">—</span>}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      )}
                    </AlertDescription>
                  </Alert>
                )}
                {syncError && (
                  <Alert variant="destructive" className="py-2">
                    <AlertDescription className="text-xs">{syncError}</AlertDescription>
                  </Alert>
                )}
              </>
            )}

            {/* Bootstrap: never synced yet */}
            {comparisonLoaded && comparison?.coreReachable && !comparison.lastSyncedAt && (
              <Alert className="py-2">
                <AlertDescription className="text-xs">
                  Stammdaten wurden noch nicht aus eegFaktura geladen. Klicke
                  „Aus eegFaktura aktualisieren" oben rechts, um sie zu übernehmen.
                </AlertDescription>
              </Alert>
            )}

            {/* The seven read-only synced fields */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <SyncedField label="EEG-Name" value={settings.eegName} />
              <SyncedField label="Kontakt-E-Mail" value={settings.contactEmail} />
              <SyncedField label="Straße" value={settings.eegStreet} />
              <SyncedField label="Hausnummer" value={settings.eegStreetNumber} />
              <SyncedField label="PLZ" value={settings.eegZip} />
              <SyncedField label="Ort" value={settings.eegCity} />
              <SyncedField label="Creditor-ID" value={settings.creditorId} />
            </div>
          </div>

          {/* SEPA Toggle (Onboarding-only) */}
          <div className="flex items-center gap-3 pt-1">
            <Switch
              id="sepa-mandate-enabled"
              checked={sepaMandateEnabled}
              onCheckedChange={(v) => {
                setSepaMandateEnabled(v);
                if (!v) setUseCompanySEPAMandate(false);
                setSaveResult(null);
              }}
            />
            <Label htmlFor="sepa-mandate-enabled" className="text-sm cursor-pointer">
              SEPA-Lastschriftmandat dem Willkommensmail anhängen
            </Label>
          </div>

          {sepaMandateEnabled && (
            <div className="flex items-center gap-3 pl-10">
              <Switch
                id="use-company-sepa-mandate"
                checked={useCompanySEPAMandate}
                onCheckedChange={(v) => {
                  setUseCompanySEPAMandate(v);
                  setSaveResult(null);
                }}
              />
              <Label htmlFor="use-company-sepa-mandate" className="text-sm cursor-pointer">
                Firmenlastschrift (B2B) für Unternehmen und Verbände verwenden
              </Label>
            </div>
          )}

          {/* E-Mail-Bestätigung (PROJ-31) */}
          <div className="flex items-start gap-3 pt-1">
            <Switch
              id="require-email-confirmation"
              checked={requireEmailConfirmation}
              onCheckedChange={(v) => {
                setRequireEmailConfirmation(v);
                setSaveResult(null);
              }}
            />
            <div className="-mt-0.5">
              <Label htmlFor="require-email-confirmation" className="text-sm cursor-pointer">
                E-Mail-Adresse bestätigen
              </Label>
              <p className="text-xs text-muted-foreground mt-1 max-w-xl">
                Wenn aktiviert, erhält das neue Mitglied in der Bestätigungs-Mail einen Button
                „E-Mail-Adresse bestätigen". Erst nach dem Klick wird der Antrag für Sie zur
                Prüfung freigegeben. Empfohlen als Schutz vor Müll-Anträgen und Tippfehlern bei
                der E-Mail-Adresse.
              </p>
            </div>
          </div>

          {/* Save */}
          <div className="flex items-center gap-3">
            <Button onClick={handleSave} disabled={saving || !loaded} size="sm">
              {saving ? "Wird gespeichert…" : "Speichern"}
            </Button>
            {saveResult === "ok" && (
              <span className="text-sm text-green-600">EEG-Einstellungen gespeichert</span>
            )}
            {saveResult === "error" && (
              <span className="text-sm text-destructive">Fehler beim Speichern</span>
            )}
          </div>
        </>
      )}
    </div>
  );
}
