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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { fetchTariffs, type Tariff, type MeteringPointDetail } from "@/lib/api";

interface Props {
  open: boolean;
  rcNumber: string;
  meteringPoints: MeteringPointDetail[];
  accessToken?: string;
  loading: boolean;
  onCancel: () => void;
  onConfirm: (selection: {
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
  rcNumber,
  meteringPoints,
  accessToken,
  loading,
  onCancel,
  onConfirm,
}: Props) {
  const [tariffs, setTariffs] = useState<Tariff[] | null>(null);
  const [fetching, setFetching] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [memberTariffId, setMemberTariffId] = useState<string>(NONE);
  const [meterTariffs, setMeterTariffs] = useState<Record<string, string>>({});

  // Reload tariffs each time the dialog opens — the user explicitly asked
  // for "Tarif-Liste zum Zeitpunkt des Imports", so no caching.
  useEffect(() => {
    if (!open) return;
    setTariffs(null);
    setFetchError(null);
    setMemberTariffId(NONE);
    setMeterTariffs({});
    setFetching(true);
    fetchTariffs(rcNumber, accessToken)
      .then((res) => setTariffs(res.tariffs))
      .catch((err) => setFetchError(err instanceof Error ? err.message : "Fehler beim Laden der Tarife"))
      .finally(() => setFetching(false));
  }, [open, rcNumber, accessToken]);

  const eegTariffs = (tariffs ?? []).filter(
    (t) => t.type === "EEG" && t.inactiveSince == null,
  );
  const vzpTariffs = (tariffs ?? []).filter(
    (t) => t.type === "VZP" && t.inactiveSince == null,
  );
  const ezpTariffs = (tariffs ?? []).filter(
    (t) => t.type === "EZP" && t.inactiveSince == null,
  );

  function handleConfirm() {
    onConfirm({
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
            Tarife konnten nicht aus dem Core geladen werden ({fetchError}). Der Import läuft ohne Tarif-Zuweisung — du kannst die Tarife später in eegFaktura nachpflegen.
          </div>
        )}

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

        <DialogFooter>
          <Button variant="outline" onClick={onCancel} disabled={loading}>
            Abbrechen
          </Button>
          <Button onClick={handleConfirm} disabled={loading || fetching}>
            {loading ? "Import läuft…" : "In eegFaktura importieren"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
