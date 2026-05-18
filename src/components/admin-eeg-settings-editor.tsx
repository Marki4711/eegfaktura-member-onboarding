"use client";

import { useCallback, useEffect, useState } from "react";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { CheckCircle2, AlertCircle, RefreshCw, Lock, ChevronUp, ChevronDown } from "lucide-react";
import {
  getEEGSettings,
  saveEEGSettings,
  compareEEGSettingsWithCore,
  syncEEGSettingsFromCore,
  fetchEEGLogoBlob,
  ApiResponseError,
  type EEGSettings,
  type EEGSettingsComparisonResponse,
} from "@/lib/api";
import { formatDateTime } from "@/lib/datetime";

interface Props {
  rcNumber: string;
}

// SyncedField is a read-only display row for one of the Core-mastered
// fields. Uses a disabled <Input> so screen readers see it as a form
// control (not just decorative text). The Lock-Icon next to the label
// signals visually + semantically (via aria-label on the icon's parent)
// that the value is managed by the Core.
// PrefixPreview (PROJ-52) renders a compact visualisation of how the
// configured prefix appears in the member-facing Zählpunkt-Mask. Empty
// prefix ⇒ "nur AT fix, 31 Stellen frei". Prefix mit < 2 oder ohne
// "AT"-Start ⇒ Inline-Hinweis. Sonst: Prefix in Mono + Anzahl freier
// Stellen. Kein Format-Check über DB-Constraint hinaus — der Save
// validiert serverseitig.
function PrefixPreview({ prefix }: { prefix: string }) {
  if (prefix === "") {
    return (
      <p className="text-xs text-muted-foreground font-mono">
        AT + 31 Stellen frei
      </p>
    );
  }
  if (!prefix.startsWith("AT") || prefix.length < 2) {
    return (
      <p className="text-xs text-amber-700">
        Prefix muss mit „AT" beginnen.
      </p>
    );
  }
  if (prefix.length > 33) {
    return (
      <p className="text-xs text-amber-700">
        Prefix darf maximal 33 Stellen lang sein.
      </p>
    );
  }
  const remaining = 33 - prefix.length;
  return (
    <p className="text-xs text-muted-foreground">
      <span className="font-mono font-medium text-foreground">{prefix}</span>
      {" "}+ {remaining} Stelle{remaining === 1 ? "" : "n"} vom Mitglied
    </p>
  );
}

function SyncedField({ label, value }: { label: string; value: string | null | undefined }) {
  const display = value && value.length > 0 ? value : "—";
  return (
    <div className="space-y-1.5">
      <Label className="text-sm flex items-center gap-1.5 text-muted-foreground">
        {label}
        <span aria-label="Wird aus eegFaktura-Core synchronisiert (schreibgeschützt)">
          <Lock className="h-3 w-3" />
        </span>
      </Label>
      <Input
        value={display}
        readOnly
        disabled
        aria-readonly="true"
        className="bg-muted/40"
      />
    </div>
  );
}

export function AdminEEGSettingsEditor({ rcNumber }: Props) {
  const { data: session } = useSession();
  const [loaded, setLoaded] = useState(false);
  const [settings, setSettings] = useState<EEGSettings | null>(null);
  const [saving, setSaving] = useState(false);
  const [saveResult, setSaveResult] = useState<"ok" | "error" | null>(null);
  const [saveErrorMsg, setSaveErrorMsg] = useState<string | null>(null);

  // PROJ-32 sync state
  const [comparison, setComparison] = useState<EEGSettingsComparisonResponse | null>(null);
  const [comparisonLoaded, setComparisonLoaded] = useState(false);

  // PROJ-33 logo preview state. logoURL is an Object URL backed by a Blob;
  // we revoke it on change/unmount via the returned dispose() callback.
  const [logoURL, setLogoURL] = useState<string | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [syncError, setSyncError] = useState<string | null>(null);
  const [diffExpanded, setDiffExpanded] = useState(false);

  // Onboarding-only editable fields
  const [sepaMandateEnabled, setSepaMandateEnabled] = useState(false);
  const [useCompanySEPAMandate, setUseCompanySEPAMandate] = useState(false);
  // PROJ-48: Mandat-Timing — wenn TRUE, wird das SEPA-Mandat erst beim Import
  // mit eingedruckter Mandatsreferenz = Mitgliedsnummer versendet.
  const [sepaMandateAtImport, setSepaMandateAtImport] = useState(false);
  const [registrationActive, setRegistrationActive] = useState(false);
  const [requireEmailConfirmation, setRequireEmailConfirmation] = useState(false);
  // PROJ-37 Genossenschaftsanteile. shareAmountInput is a string so the
  // admin can comfortably type "100,00" or "100.50" without React
  // fighting the input cursor. Converted to cents on save.
  const [cooperativeSharesEnabled, setCooperativeSharesEnabled] = useState(false);
  const [cooperativeRequiredShares, setCooperativeRequiredShares] = useState<number>(1);
  const [shareAmountInput, setShareAmountInput] = useState("");
  // PROJ-52: pro-Richtung Zählpunkt-Prefix. Leerer String = nicht
  // konfiguriert. Input zwingt automatisch uppercase und entfernt
  // Whitespace, damit das Eintippformat zur Backend-Normalisierung passt.
  const [meteringPointPrefixConsumption, setMeteringPointPrefixConsumption] = useState("");
  const [meteringPointPrefixProduction, setMeteringPointPrefixProduction] = useState("");

  const reloadSettings = useCallback(async () => {
    if (!rcNumber || !session?.accessToken) return;
    const s = await getEEGSettings(rcNumber, session.accessToken);
    setSettings(s);
    setRegistrationActive(s.registrationActive ?? false);
    setSepaMandateEnabled(s.sepaMandateEnabled);
    setUseCompanySEPAMandate(s.useCompanySEPAMandate ?? false);
    setSepaMandateAtImport(s.sepaMandateAtImport ?? false);
    setRequireEmailConfirmation(s.requireEmailConfirmation ?? false);
    setMeteringPointPrefixConsumption(s.meteringPointPrefixConsumption ?? "");
    setMeteringPointPrefixProduction(s.meteringPointPrefixProduction ?? "");
    setCooperativeSharesEnabled(s.cooperativeSharesEnabled ?? false);
    setCooperativeRequiredShares(s.cooperativeRequiredShares ?? 1);
    setShareAmountInput(
      s.cooperativeShareAmountCents != null
        ? (s.cooperativeShareAmountCents / 100).toFixed(2).replace(".", ",")
        : "",
    );
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

  // PROJ-33: load the cached logo preview. Re-runs whenever the EEGLogoSyncedAt
  // changes (i.e. after a sync click). Object URL is revoked on cleanup so the
  // blob doesn't leak across re-renders.
  useEffect(() => {
    if (!rcNumber || !session?.accessToken) return;
    let disposed = false;
    let cleanup: (() => void) | null = null;
    fetchEEGLogoBlob(rcNumber, session.accessToken)
      .then((res) => {
        if (disposed) {
          res?.dispose();
          return;
        }
        if (res) {
          cleanup = res.dispose;
          setLogoURL(res.objectURL);
        } else {
          setLogoURL(null);
        }
      })
      .catch(() => setLogoURL(null));
    return () => {
      disposed = true;
      if (cleanup) cleanup();
    };
  }, [rcNumber, session?.accessToken, settings?.eegLogoSyncedAt]);

  const handleSave = async () => {
    setSaving(true);
    setSaveResult(null);
    try {
      // PROJ-37: parse decimal-comma-or-dot Euro string into integer cents.
      // Empty string when feature disabled — backend clears the field.
      let amountCents: number | undefined = undefined;
      if (cooperativeSharesEnabled) {
        const normalised = shareAmountInput.replace(",", ".").trim();
        const parsed = parseFloat(normalised);
        if (!isNaN(parsed) && parsed > 0) {
          amountCents = Math.round(parsed * 100);
        }
      }
      await saveEEGSettings(
        rcNumber,
        {
          registrationActive,
          sepaMandateEnabled,
          useCompanySEPAMandate,
          sepaMandateAtImport,
          requireEmailConfirmation,
          // PROJ-52: Patch-Semantik aktivieren — sonst lässt der Handler
          // die Prefix-Spalten unberührt. Leerer Input ⇒ Backend cleart.
          meteringPointPrefixesPresent: true,
          meteringPointPrefixConsumption: meteringPointPrefixConsumption || null,
          meteringPointPrefixProduction: meteringPointPrefixProduction || null,
          cooperativeSharesEnabled,
          cooperativeRequiredShares: cooperativeSharesEnabled ? cooperativeRequiredShares : undefined,
          cooperativeShareAmountCents: amountCents,
        },
        session?.accessToken,
      );
      setSaveResult("ok");
      setSaveErrorMsg(null);
    } catch (err) {
      setSaveResult("error");
      // Surface the server-side message when available — especially for
      // validation_error responses that carry field-level reasons like
      // "Anteilswert ist erforderlich und muss größer 0 sein". The
      // generic "Fehler beim Speichern" stays as a fallback.
      if (err instanceof ApiResponseError) {
        const fieldMsgs = err.apiError.fields
          ? Object.values(err.apiError.fields).filter((v): v is string => !!v)
          : [];
        if (fieldMsgs.length > 0) {
          setSaveErrorMsg(fieldMsgs.join(" · "));
        } else if (err.apiError.message) {
          setSaveErrorMsg(err.apiError.message);
        } else {
          setSaveErrorMsg(null);
        }
      } else {
        setSaveErrorMsg(null);
      }
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
                        Klicken Sie „Aus eegFaktura aktualisieren", um die Daten zu übernehmen.
                      </p>
                      <button
                        type="button"
                        onClick={() => setDiffExpanded((v) => !v)}
                        className="underline text-orange-900 hover:text-orange-700"
                      >
                        {diffExpanded ? (
                          <span className="inline-flex items-center gap-1">Details verbergen <ChevronUp className="h-3.5 w-3.5" /></span>
                        ) : (
                          <span className="inline-flex items-center gap-1">Details anzeigen <ChevronDown className="h-3.5 w-3.5" /></span>
                        )}
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
                  Stammdaten wurden noch nicht aus eegFaktura geladen. Klicken Sie
                  „Aus eegFaktura aktualisieren" oben rechts, um sie zu übernehmen.
                </AlertDescription>
              </Alert>
            )}

            {/* The eight read-only synced fields */}
            <div className="space-y-1.5">
              <SyncedField label="Gemeinschafts-ID" value={settings.eegId} />
              <p className="text-xs text-muted-foreground">
                Wird im Excel-Export (Spalte B) für den eegFaktura-Import verwendet.
              </p>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              <SyncedField label="EEG-Name" value={settings.eegName} />
              <SyncedField label="Kontakt-E-Mail" value={settings.contactEmail} />
              <SyncedField label="Straße" value={settings.eegStreet} />
              <SyncedField label="Hausnummer" value={settings.eegStreetNumber} />
              <SyncedField label="PLZ" value={settings.eegZip} />
              <SyncedField label="Ort" value={settings.eegCity} />
              <SyncedField label="Creditor-ID" value={settings.creditorId} />
            </div>

            {/* PROJ-33: Logo-Vorschau */}
            <div className="space-y-1.5">
              <Label className="text-sm flex items-center gap-1.5 text-muted-foreground">
                Logo
                <Lock className="h-3 w-3" />
              </Label>
              <div className="min-h-[80px] px-3 py-2 rounded-md border bg-muted/40 flex items-center">
                {logoURL ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={logoURL}
                    alt="EEG-Logo"
                    className="max-h-[60px] max-w-[200px] object-contain"
                  />
                ) : (
                  <span className="text-muted-foreground italic text-sm">
                    Noch kein Logo aus eegFaktura geladen
                  </span>
                )}
              </div>
              {comparison?.logoSyncWarning && (
                <p className="text-xs text-orange-700">
                  {comparison.logoSyncWarning}
                </p>
              )}
              {settings.eegLogoSyncedAt && (
                <p className="text-xs text-muted-foreground">
                  Logo zuletzt aus eegFaktura geladen: {formatDateTime(settings.eegLogoSyncedAt)}.
                  Der „Synchron"-Status oben prüft nur die Text-Felder — das Logo wird bei jedem
                  „Aus eegFaktura aktualisieren" mitgeholt.
                </p>
              )}
              {!settings.eegLogoSyncedAt && (
                <p className="text-xs text-muted-foreground">
                  Erscheint oben rechts auf Beitrittsbestätigung + SEPA-Mandat. Wird beim
                  nächsten „Aus eegFaktura aktualisieren" mitsynchronisiert.
                </p>
              )}
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
              SEPA-Mandat von der EEG bereitstellen
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
                Firmenlastschrift (B2B) für Unternehmen und Gemeinden zulassen
              </Label>
            </div>
          )}

          {/* PROJ-48: Mandat-Timing — Submit vs. Import */}
          {sepaMandateEnabled && (
            <div className="flex items-start gap-3 pl-10">
              <Switch
                id="sepa-mandate-at-import"
                checked={sepaMandateAtImport}
                onCheckedChange={(v) => {
                  setSepaMandateAtImport(v);
                  setSaveResult(null);
                }}
              />
              <div className="space-y-1">
                <Label htmlFor="sepa-mandate-at-import" className="text-sm cursor-pointer">
                  SEPA-Mandat erst beim Import senden (mit Mitgliedsnummer als Mandatsreferenz)
                </Label>
                <p className="text-xs text-muted-foreground">
                  Standard (aus): das Mandat wird der Eingangsbestätigung als PDF-Anhang
                  beigelegt — ohne Mandatsreferenz, der Member trägt sie händisch ein.
                  Aktiv: das Mandat kommt erst beim Import in eegFaktura mit eingedruckter
                  Mandatsreferenz (= Mitgliedsnummer). Notwendig, wenn das Mandat digital
                  signiert werden soll (eine spätere Modifikation würde die Signatur ungültig
                  machen). Greift gleichermaßen für Basis- und Firmenlastschrift.
                </p>
              </div>
            </div>
          )}

          {/* PROJ-37: Genossenschaftsanteile (Onboarding-only) */}
          <div className="flex items-center gap-3 pt-1">
            <Switch
              id="cooperative-shares-enabled"
              checked={cooperativeSharesEnabled}
              onCheckedChange={(v) => {
                setCooperativeSharesEnabled(v);
                setSaveResult(null);
              }}
            />
            <Label htmlFor="cooperative-shares-enabled" className="text-sm cursor-pointer">
              Genossenschaftsanteile erfassen
            </Label>
          </div>
          {cooperativeSharesEnabled && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 pl-10">
              <div className="space-y-1.5">
                <Label htmlFor="coop-required-shares" className="text-sm">
                  Pflichtanteile je Standort *
                </Label>
                <Input
                  id="coop-required-shares"
                  type="number"
                  inputMode="numeric"
                  min={1}
                  value={cooperativeRequiredShares}
                  onChange={(e) => {
                    const v = parseInt(e.target.value, 10);
                    setCooperativeRequiredShares(isNaN(v) ? 1 : Math.max(1, v));
                    setSaveResult(null);
                  }}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="coop-share-amount" className="text-sm">
                  Genossenschaftsanteilswert (€) *
                </Label>
                <Input
                  id="coop-share-amount"
                  type="text"
                  inputMode="decimal"
                  value={shareAmountInput}
                  onChange={(e) => {
                    setShareAmountInput(e.target.value);
                    setSaveResult(null);
                  }}
                />
              </div>
              <p className="text-xs text-muted-foreground sm:col-span-2">
                Wird auf der Beitrittsbestätigung als Anzahl × Anteilswert =
                Gesamtbetrag ausgewiesen. Wird nicht an eegFaktura übermittelt —
                reine Onboarding-Erfassung.
              </p>
            </div>
          )}

          {/* PROJ-52: Zählpunkt-Prefixes pro Richtung */}
          <div className="rounded-md border bg-card p-4 space-y-3 mt-2">
            <div>
              <p className="text-sm font-medium">Zählpunkt-Prefixes</p>
              <p className="text-xs text-muted-foreground mt-0.5 max-w-2xl">
                Je mehr Stellen Sie hier festlegen, desto weniger müssen Mitglieder
                selbst eintippen. Die sinnvolle Länge hängt davon ab, ab welcher
                Stelle die Zählpunkte Ihres Netzbetreibers individuell werden.
                Beide Felder sind optional; leer lassen heißt „nur AT als fixer
                Bestandteil".
              </p>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label htmlFor="prefix-consumption" className="text-sm">
                  Verbraucher-Prefix
                </Label>
                <Input
                  id="prefix-consumption"
                  type="text"
                  inputMode="text"
                  autoComplete="off"
                  value={meteringPointPrefixConsumption}
                  onChange={(e) => {
                    const cleaned = e.target.value
                      .replace(/[\s.\-]/g, "")
                      .toUpperCase();
                    setMeteringPointPrefixConsumption(cleaned);
                    setSaveResult(null);
                  }}
                  maxLength={33}
                  className="font-mono"
                />
                <PrefixPreview prefix={meteringPointPrefixConsumption} />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="prefix-production" className="text-sm">
                  Einspeisungs-Prefix
                </Label>
                <Input
                  id="prefix-production"
                  type="text"
                  inputMode="text"
                  autoComplete="off"
                  value={meteringPointPrefixProduction}
                  onChange={(e) => {
                    const cleaned = e.target.value
                      .replace(/[\s.\-]/g, "")
                      .toUpperCase();
                    setMeteringPointPrefixProduction(cleaned);
                    setSaveResult(null);
                  }}
                  maxLength={33}
                  className="font-mono"
                />
                <PrefixPreview prefix={meteringPointPrefixProduction} />
              </div>
            </div>
          </div>

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
                Wenn aktiviert, erhält das neue Mitglied in der Eingangsbestätigung einen Button
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
              <span className="text-sm text-destructive">
                {saveErrorMsg ?? "Fehler beim Speichern"}
              </span>
            )}
          </div>
        </>
      )}
    </div>
  );
}
