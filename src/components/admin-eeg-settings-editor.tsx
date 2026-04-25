"use client";

import { useEffect, useState } from "react";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { getEEGSettings, saveEEGSettings } from "@/lib/api";

interface Props {
  rcNumber: string;
}

function isComplete(
  eegName: string,
  eegStreet: string,
  eegStreetNumber: string,
  eegZip: string,
  eegCity: string,
  creditorId: string
): boolean {
  return (
    eegName.trim() !== "" &&
    eegStreet.trim() !== "" &&
    eegStreetNumber.trim() !== "" &&
    eegZip.trim() !== "" &&
    eegCity.trim() !== "" &&
    creditorId.trim() !== ""
  );
}

export function AdminEEGSettingsEditor({ rcNumber }: Props) {
  const { data: session } = useSession();
  const [loaded, setLoaded] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveResult, setSaveResult] = useState<"ok" | "error" | null>(null);

  const [eegId, setEegId] = useState("");
  const [eegName, setEegName] = useState("");
  const [eegStreet, setEegStreet] = useState("");
  const [eegStreetNumber, setEegStreetNumber] = useState("");
  const [eegZip, setEegZip] = useState("");
  const [eegCity, setEegCity] = useState("");
  const [creditorId, setCreditorId] = useState("");
  const [sepaMandateEnabled, setSepaMandateEnabled] = useState(false);
  const [useCompanySEPAMandate, setUseCompanySEPAMandate] = useState(false);
  const [registrationActive, setRegistrationActive] = useState(false);

  useEffect(() => {
    if (!rcNumber || !session?.accessToken) return;
    setLoaded(false);
    getEEGSettings(rcNumber, session.accessToken)
      .then((s) => {
        setRegistrationActive(s.registrationActive ?? false);
        setEegId(s.eegId ?? "");
        setEegName(s.eegName ?? "");
        setEegStreet(s.eegStreet ?? "");
        setEegStreetNumber(s.eegStreetNumber ?? "");
        setEegZip(s.eegZip ?? "");
        setEegCity(s.eegCity ?? "");
        setCreditorId(s.creditorId ?? "");
        setSepaMandateEnabled(s.sepaMandateEnabled);
        setUseCompanySEPAMandate(s.useCompanySEPAMandate ?? false);
        setLoaded(true);
      })
      .catch(() => setLoaded(true));
  }, [rcNumber, session?.accessToken]);

  const fieldsComplete = isComplete(eegName, eegStreet, eegStreetNumber, eegZip, eegCity, creditorId);
  const showWarning = sepaMandateEnabled && !fieldsComplete;

  const handleSave = async () => {
    setSaving(true);
    setSaveResult(null);
    try {
      await saveEEGSettings(
        rcNumber,
        {
          registrationActive,
          eegId: eegId.trim() || null,
          eegName: eegName.trim() || null,
          eegStreet: eegStreet.trim() || null,
          eegStreetNumber: eegStreetNumber.trim() || null,
          eegZip: eegZip.trim() || null,
          eegCity: eegCity.trim() || null,
          creditorId: creditorId.trim() || null,
          sepaMandateEnabled,
          useCompanySEPAMandate,
        },
        session?.accessToken
      );
      setSaveResult("ok");
    } catch {
      setSaveResult("error");
    } finally {
      setSaving(false);
    }
  };

  const fieldClass = "h-9 text-sm";

  return (
    <div className="space-y-4">
      {!loaded && (
        <p className="text-xs text-muted-foreground">Lädt…</p>
      )}

      {loaded && (
        <>
          {/* Registrierung aktivieren/deaktivieren */}
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
              onCheckedChange={(v) => { setRegistrationActive(v); setSaveResult(null); }}
            />
          </div>

          {/* EEG-ID (Gemeinschafts-ID für Excel-Export) */}
          <div className="space-y-1.5">
            <Label htmlFor="eeg-id" className="text-sm">Gemeinschafts-ID</Label>
            <Input
              id="eeg-id"
              value={eegId}
              onChange={(e) => { setEegId(e.target.value); setSaveResult(null); }}
              placeholder="AT00300000000RC101519000000912345"
              className={fieldClass}
            />
            <p className="text-xs text-muted-foreground">
              Wird im Excel-Export (Spalte B) für den eegFaktura-Import verwendet.
            </p>
          </div>

          {/* EEG-Name */}
          <div className="space-y-1.5">
            <Label htmlFor="eeg-name" className="text-sm">EEG-Name</Label>
            <Input
              id="eeg-name"
              value={eegName}
              onChange={(e) => { setEegName(e.target.value); setSaveResult(null); }}
              placeholder="Erneuerbare Energiegemeinschaft Muster"
              className={fieldClass}
            />
          </div>

          {/* Straße + Hausnummer */}
          <div className="flex gap-3">
            <div className="flex-1 space-y-1.5">
              <Label htmlFor="eeg-street" className="text-sm">Straße</Label>
              <Input
                id="eeg-street"
                value={eegStreet}
                onChange={(e) => { setEegStreet(e.target.value); setSaveResult(null); }}
                placeholder="Musterstraße"
                className={fieldClass}
              />
            </div>
            <div className="w-28 space-y-1.5">
              <Label htmlFor="eeg-street-number" className="text-sm">Hausnummer</Label>
              <Input
                id="eeg-street-number"
                value={eegStreetNumber}
                onChange={(e) => { setEegStreetNumber(e.target.value); setSaveResult(null); }}
                placeholder="1"
                className={fieldClass}
              />
            </div>
          </div>

          {/* PLZ + Ort */}
          <div className="flex gap-3">
            <div className="w-28 space-y-1.5">
              <Label htmlFor="eeg-zip" className="text-sm">PLZ</Label>
              <Input
                id="eeg-zip"
                value={eegZip}
                onChange={(e) => { setEegZip(e.target.value); setSaveResult(null); }}
                placeholder="1234"
                className={fieldClass}
              />
            </div>
            <div className="flex-1 space-y-1.5">
              <Label htmlFor="eeg-city" className="text-sm">Ort</Label>
              <Input
                id="eeg-city"
                value={eegCity}
                onChange={(e) => { setEegCity(e.target.value); setSaveResult(null); }}
                placeholder="Musterort"
                className={fieldClass}
              />
            </div>
          </div>

          {/* Creditor-ID */}
          <div className="space-y-1.5">
            <Label htmlFor="creditor-id" className="text-sm">Creditor-ID</Label>
            <Input
              id="creditor-id"
              value={creditorId}
              onChange={(e) => { setCreditorId(e.target.value); setSaveResult(null); }}
              placeholder="AT00ZZZ00000000000"
              className={fieldClass}
            />
          </div>

          {/* SEPA Toggle */}
          <div className="flex items-center gap-3 pt-1">
            <Switch
              id="sepa-mandate-enabled"
              checked={sepaMandateEnabled}
              onCheckedChange={(v) => { setSepaMandateEnabled(v); if (!v) setUseCompanySEPAMandate(false); setSaveResult(null); }}
            />
            <Label htmlFor="sepa-mandate-enabled" className="text-sm cursor-pointer">
              SEPA-Lastschriftmandat dem Willkommensmail anhängen
            </Label>
          </div>

          {/* Company SEPA Toggle — only when SEPA is enabled */}
          {sepaMandateEnabled && (
            <div className="flex items-center gap-3 pl-10">
              <Switch
                id="use-company-sepa-mandate"
                checked={useCompanySEPAMandate}
                onCheckedChange={(v) => { setUseCompanySEPAMandate(v); setSaveResult(null); }}
              />
              <Label htmlFor="use-company-sepa-mandate" className="text-sm cursor-pointer">
                Firmenlastschrift (B2B) für Unternehmen und Verbände verwenden
              </Label>
            </div>
          )}

          {showWarning && (
            <Alert variant="destructive" className="py-2">
              <AlertDescription className="text-xs">
                Bitte alle EEG-Felder ausfüllen bevor Sie die Funktion aktivieren. Solange Felder fehlen, wird kein PDF generiert.
              </AlertDescription>
            </Alert>
          )}

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
