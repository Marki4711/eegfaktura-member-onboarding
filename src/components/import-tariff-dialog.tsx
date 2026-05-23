"use client";

import { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { fetchTariffs, fetchNextMemberNumber, type Tariff, type MeteringPointDetail } from "@/lib/api";

interface Props {
  open: boolean;
  applicationId: string;
  rcNumber: string;
  meteringPoints: MeteringPointDetail[];
  accessToken?: string;
  // CORE_AUTH_MODE=exchange: token for outgoing Faktura-Core REST-Calls.
  // Undefined in direct mode — backend then falls back to accessToken.
  coreAccessToken?: string;
  loading: boolean;
  // Server-side error from the last import attempt (e.g. member-number
  // already used). Surfaced inside the dialog so the admin sees what went
  // wrong without having to close the dialog and scroll.
  errorMessage?: string | null;
  onCancel: () => void;
  onConfirm: (selection: {
    memberNumber: string;
    tariffId: string;
    meterTariffs: Record<string, string>;
  }) => void;
}

const NONE = "__none__";

function tariffLabel(t: Tariff): string {
  const parts = [`${t.name} — ${t.centPerKWh} ct/kWh`];
  if (t.discount > 0) parts.push(`Rabatt ${t.discount}%`);
  if (t.useVat) parts.push(`USt ${t.vatInPercent}%`);
  return parts.join(", ");
}

export function ImportTariffDialog({
  open,
  applicationId,
  rcNumber,
  meteringPoints,
  accessToken,
  coreAccessToken,
  loading,
  errorMessage,
  onCancel,
  onConfirm,
}: Props) {
  const [tariffs, setTariffs] = useState<Tariff[] | null>(null);
  const [fetching, setFetching] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [memberTariffId, setMemberTariffId] = useState<string>(NONE);
  const [meterTariffs, setMeterTariffs] = useState<Record<string, string>>({});
  // Member number — pre-filled from the core's max+1 suggestion, but the
  // admin can override before confirming. Empty string while we're still
  // loading the suggestion; "" + invalid blocks the Confirm button.
  const [memberNumber, setMemberNumber] = useState<string>("");
  const [memberNumberLoading, setMemberNumberLoading] = useState(false);
  const [memberNumberError, setMemberNumberError] = useState<string | null>(null);

  // Reload tariffs each time the dialog opens — the user explicitly asked
  // for "Tarif-Liste zum Zeitpunkt des Imports", so no caching. Same for the
  // next-member-number suggestion: the core may have grown between two
  // dialog opens and we want the freshest value.
  useEffect(() => {
    if (!open) return;
    setTariffs(null);
    setFetchError(null);
    setMemberTariffId(NONE);
    setMeterTariffs({});
    setMemberNumber("");
    setMemberNumberError(null);
    setFetching(true);
    setMemberNumberLoading(true);
    // Abort both fetches if the dialog closes (or rcNumber/application
    // changes) before the responses land.
    const ac = new AbortController();
    fetchTariffs(rcNumber, accessToken, coreAccessToken, ac.signal)
      .then((res) => setTariffs(res.tariffs))
      .catch((err) => {
        if (err instanceof DOMException && err.name === "AbortError") return;
        setFetchError(err instanceof Error ? err.message : "Fehler beim Laden der Tarife");
      })
      .finally(() => setFetching(false));
    fetchNextMemberNumber(applicationId, accessToken, coreAccessToken, ac.signal)
      .then((res) => setMemberNumber(res.next_member_number))
      .catch((err) => {
        if (err instanceof DOMException && err.name === "AbortError") return;
        // Soft-fail: dialog stays usable, admin types the number manually.
        setMemberNumberError(
          err instanceof Error
            ? err.message
            : "Nächste Mitgliedsnummer konnte nicht aus dem Core ermittelt werden",
        );
      })
      .finally(() => setMemberNumberLoading(false));
    return () => ac.abort();
  }, [open, applicationId, rcNumber, accessToken, coreAccessToken]);

  const eegTariffs = (tariffs ?? []).filter(
    (t) => t.type === "EEG" && t.inactiveSince == null,
  );
  const vzpTariffs = (tariffs ?? []).filter(
    (t) => t.type === "VZP" && t.inactiveSince == null,
  );
  const ezpTariffs = (tariffs ?? []).filter(
    (t) => t.type === "EZP" && t.inactiveSince == null,
  );

  const trimmedMemberNumber = memberNumber.trim();
  const memberNumberValid =
    trimmedMemberNumber.length > 0 && trimmedMemberNumber.length <= 50;

  function handleConfirm() {
    if (!memberNumberValid) return;
    onConfirm({
      memberNumber: trimmedMemberNumber,
      tariffId: memberTariffId === NONE ? "" : memberTariffId,
      meterTariffs: Object.fromEntries(
        Object.entries(meterTariffs).filter(([, v]) => v !== "" && v !== NONE),
      ),
    });
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onCancel(); }}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Import vorbereiten — Tarife zuweisen</DialogTitle>
        </DialogHeader>

        {fetching && (
          <p className="text-sm text-muted-foreground">Tarife werden geladen…</p>
        )}

        {fetchError && (
          <div className="rounded-md border border-amber-500/50 bg-amber-50 p-3 text-sm text-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
            <div className="font-medium break-words">{fetchError}</div>
            <div className="mt-1 text-xs">
              Der Import läuft ohne Tarif-Zuweisung — du kannst die Tarife später in eegFaktura nachpflegen.
            </div>
          </div>
        )}

        <div className="space-y-1 py-2 border-b pb-4">
          <Label htmlFor="member-number">Mitgliedsnummer</Label>
          <Input
            id="member-number"
            type="text"
            maxLength={50}
            value={memberNumber}
            onChange={(e) => setMemberNumber(e.target.value)}
            placeholder={memberNumberLoading ? "Wird ermittelt…" : "z.B. 42 oder A006"}
            disabled={memberNumberLoading || loading}
          />
          {memberNumberError && (
            <p className="text-xs text-amber-700 dark:text-amber-300">
              {memberNumberError} — bitte manuell eintragen.
            </p>
          )}
          {!memberNumberLoading && !memberNumberError && (
            <p className="text-xs text-muted-foreground">
              Vorschlag aus eegFaktura (folgt dem dominanten Schema in dieser EEG, z.B. „A006" wenn die letzte „A005" war). Anpassbar; das Backend prüft vor dem Import auf Doppelvergabe.
            </p>
          )}
        </div>

        {tariffs && !fetchError && (
          <div className="space-y-4 py-2">
            {/* Mitglieds-Tarif */}
            <div className="space-y-1">
              <Label>Mitglieds-Tarif</Label>
              <Select value={memberTariffId} onValueChange={setMemberTariffId}>
                <SelectTrigger>
                  <SelectValue placeholder="(kein Tarif)" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={NONE}>(kein Tarif)</SelectItem>
                  {eegTariffs.map((t) => (
                    <SelectItem key={t.id} value={t.id}>{tariffLabel(t)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
              {eegTariffs.length === 0 && (
                <p className="text-xs text-muted-foreground">Keine Mitglieds-Tarife (EEG) in eegFaktura definiert.</p>
              )}
            </div>

            {/* Pro Zählpunkt */}
            {meteringPoints.length > 0 && (
              <div className="space-y-3 border-t pt-3">
                <p className="text-sm font-medium">Zählpunkt-Tarife</p>
                {meteringPoints.map((mp) => {
                  const list = mp.direction === "PRODUCTION" ? ezpTariffs : vzpTariffs;
                  const directionLabel = mp.direction === "PRODUCTION" ? "Erzeuger (EZP)" : "Verbraucher (VZP)";
                  const value = meterTariffs[mp.meteringPoint] ?? NONE;
                  return (
                    <div key={mp.id} className="space-y-1">
                      <Label>
                        <span className="font-mono text-xs">{mp.meteringPoint}</span>
                        <span className="ml-2 text-muted-foreground">— {directionLabel}</span>
                      </Label>
                      <Select
                        value={value}
                        onValueChange={(v) =>
                          setMeterTariffs((prev) => ({ ...prev, [mp.meteringPoint]: v }))
                        }
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="(kein Tarif)" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value={NONE}>(kein Tarif)</SelectItem>
                          {list.map((t) => (
                            <SelectItem key={t.id} value={t.id}>{tariffLabel(t)}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      {list.length === 0 && (
                        <p className="text-xs text-muted-foreground">
                          Keine {mp.direction === "PRODUCTION" ? "Erzeuger" : "Verbraucher"}-Tarife in eegFaktura definiert.
                        </p>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        )}

        {errorMessage && (
          <div className="rounded-md border border-destructive/40 bg-destructive/5 p-3 text-sm text-destructive">
            {errorMessage}
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={onCancel} disabled={loading}>
            Abbrechen
          </Button>
          <Button
            onClick={handleConfirm}
            disabled={loading || fetching || memberNumberLoading || !memberNumberValid}
          >
            {loading ? "Import läuft…" : "In eegFaktura importieren"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
