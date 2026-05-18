"use client";

import { useState } from "react";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { AlertTriangle, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { markImportedManually, clearImportLock, ApiResponseError } from "@/lib/api";
import { formatDateTime } from "@/lib/datetime";

interface Props {
  applicationId: string;
  importStartedAt: string | null;
  targetParticipantId: string | null;
  onRefresh: () => void;
}

type Mode = "mark" | "clear" | null;

/**
 * PROJ-34: Banner shown on an application detail page when the import is
 * stuck — `status='approved'`, `import_started_at` set > 2 min ago,
 * `import_finished_at` null. Offers two recovery actions:
 *
 * - "Als importiert markieren" — admin enters core UUID + member-number,
 *   transitions to `imported`. The clean path when the core call did
 *   succeed but the bookkeeping failed.
 *
 * - "Import-Lock räumen (Retry)" — releases the in-flight slot, status
 *   stays `approved`. Lets the admin retry but risks a duplicate in the
 *   core if the original attempt had already inserted there. Reason
 *   required.
 */
export function AdminImportUnstuckBanner({
  applicationId,
  importStartedAt,
  targetParticipantId,
  onRefresh,
}: Props) {
  const { data: session } = useSession();
  const [mode, setMode] = useState<Mode>(null);
  const [participantID, setParticipantID] = useState(targetParticipantId ?? "");
  const [memberNumber, setMemberNumber] = useState("");
  const [reason, setReason] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const closeDialog = () => {
    setMode(null);
    setError(null);
    setLoading(false);
  };

  const handleMarkImported = async () => {
    if (!session?.accessToken) return;
    if (!participantID.trim() || !memberNumber.trim()) {
      setError("Teilnehmer-UUID und Mitgliedsnummer sind Pflichtfelder.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await markImportedManually(
        applicationId,
        {
          targetParticipantId: participantID.trim(),
          memberNumber: memberNumber.trim(),
          reason: reason.trim(),
        },
        session.accessToken,
      );
      toast.success("Antrag wurde als importiert markiert.");
      closeDialog();
      onRefresh();
    } catch (e) {
      const msg = e instanceof ApiResponseError ? e.message : "Unbekannter Fehler";
      setError(msg);
      setLoading(false);
    }
  };

  const handleClearLock = async () => {
    if (!session?.accessToken) return;
    if (reason.trim().length < 5) {
      setError("Bitte mindestens 5 Zeichen Begründung angeben.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      await clearImportLock(
        applicationId,
        { reason: reason.trim() },
        session.accessToken,
      );
      toast.success("Import-Lock wurde zurückgesetzt.");
      closeDialog();
      onRefresh();
    } catch (e) {
      const msg = e instanceof ApiResponseError ? e.message : "Unbekannter Fehler";
      setError(msg);
      setLoading(false);
    }
  };

  return (
    <>
      <Alert className="border-orange-300 bg-orange-50 text-orange-900">
        <AlertTriangle className="h-4 w-4 text-orange-600" />
        <AlertDescription className="space-y-2">
          <p>
            <strong>Import-Vorgang hängt fest.</strong>{" "}
            Der letzte Import-Versuch
            {importStartedAt && <> wurde um <strong>{formatDateTime(importStartedAt)}</strong> gestartet</>}
            {" "}und nicht sauber abgeschlossen. Bitte wähle eine
            der folgenden Aktionen, um den Antrag zu reparieren:
          </p>
          <div className="flex flex-wrap gap-2 pt-1">
            <Button
              size="sm"
              variant="default"
              onClick={() => {
                setMode("mark");
                setReason("");
                setError(null);
              }}
            >
              Als importiert markieren
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                setMode("clear");
                setReason("");
                setError(null);
              }}
            >
              Import-Lock räumen (Retry)
            </Button>
          </div>
          {targetParticipantId && (
            <p className="text-xs text-orange-800">
              Im letzten Versuch protokollierte Teilnehmer-UUID:{" "}
              <code className="text-xs">{targetParticipantId}</code>
            </p>
          )}
        </AlertDescription>
      </Alert>

      {/* Dialog: mark imported manually */}
      <Dialog open={mode === "mark"} onOpenChange={(o) => !o && closeDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Als importiert markieren</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Trage die Teilnehmer-UUID und die Mitgliedsnummer ein, wie sie
            in eegFaktura beim importierten Mitglied angezeigt werden. Der
            Antrag wechselt anschließend auf „Importiert".
          </p>
          <div className="space-y-3 pt-2">
            <div className="space-y-1.5">
              <Label htmlFor="participant-uuid">Teilnehmer-UUID aus eegFaktura</Label>
              <Input
                id="participant-uuid"
                value={participantID}
                onChange={(e) => setParticipantID(e.target.value)}
                disabled={loading}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="member-number">Mitgliedsnummer</Label>
              <Input
                id="member-number"
                value={memberNumber}
                onChange={(e) => setMemberNumber(e.target.value)}
                disabled={loading}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="mark-reason">Begründung (optional)</Label>
              <Textarea
                id="mark-reason"
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                rows={2}
                disabled={loading}
              />
            </div>
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <DialogFooter>
            <Button variant="outline" onClick={closeDialog} disabled={loading}>
              Abbrechen
            </Button>
            <Button onClick={handleMarkImported} disabled={loading || !participantID.trim() || !memberNumber.trim()}>
              {loading && <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />}
              Als importiert markieren
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Dialog: clear import lock */}
      <Dialog open={mode === "clear"} onOpenChange={(o) => !o && closeDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Import-Lock räumen</DialogTitle>
          </DialogHeader>
          <Alert variant="destructive">
            <AlertTriangle className="h-4 w-4" />
            <AlertDescription className="text-xs">
              <strong>Achtung:</strong> Wenn der vorige Import-Versuch im eegFaktura-Core
              bereits einen Teilnehmer angelegt hat, entsteht beim nächsten Import-Klick
              ein <strong>Duplikat</strong>. Verwende diese Aktion nur, wenn du
              sicher bist, dass im Core kein Teilnehmer existiert (oder du diesen vorher
              manuell gelöscht hast).
            </AlertDescription>
          </Alert>
          <div className="space-y-1.5 pt-2">
            <Label htmlFor="clear-reason">Begründung (mind. 5 Zeichen)</Label>
            <Textarea
              id="clear-reason"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={3}
              disabled={loading}
            />
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
          <DialogFooter>
            <Button variant="outline" onClick={closeDialog} disabled={loading}>
              Abbrechen
            </Button>
            <Button variant="destructive" onClick={handleClearLock} disabled={loading || reason.trim().length < 5}>
              {loading && <Loader2 className="h-3.5 w-3.5 mr-1.5 animate-spin" />}
              Lock räumen
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
