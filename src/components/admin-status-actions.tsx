"use client";

import { useState } from "react";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { toast } from "sonner";
import { changeApplicationStatus, importApplication, resetImportApplication, ApiResponseError } from "@/lib/api";
import type { ApplicationStatus, MeteringPointDetail } from "@/lib/api";
import { ImportTariffDialog } from "@/components/import-tariff-dialog";

interface Props {
  applicationId: string;
  rcNumber: string;
  status: ApplicationStatus;
  targetParticipantId?: string | null;
  importErrorMessage?: string | null;
  meteringPoints: MeteringPointDetail[];
  onRefresh: () => void;
  // PROJ-31: when true, only "Ablehnen" is available — the EEG requires
  // e-mail confirmation and the member hasn't clicked yet, so review-style
  // transitions are blocked server-side too.
  emailConfirmationPending?: boolean;
}

type DialogTarget = "rejected" | "needs_info" | "reset_import";

const STATIC_NOTES: Partial<Record<ApplicationStatus, string>> = {
  draft:    "Antrag noch nicht eingereicht. Keine Admin-Aktionen verfügbar.",
  rejected: "Antrag abgelehnt. Keine weiteren Aktionen verfügbar.",
};

const DIALOG_LABELS: Record<DialogTarget, { title: string; placeholder: string; confirm: string; warning?: string }> = {
  rejected:   { title: "Antrag ablehnen", placeholder: "Begründung der Ablehnung...", confirm: "Ablehnen" },
  needs_info: { title: "Informationen anfordern", placeholder: "Welche Informationen werden benötigt?", confirm: "Anforderung senden" },
  reset_import: {
    title: "Import zurücksetzen",
    placeholder: "Warum wird der Import zurückgesetzt? (mind. 5 Zeichen)",
    confirm: "Zurücksetzen",
    warning:
      "Diese Aktion setzt den Antrag zurück auf „Genehmigt\" und löscht die Verknüpfung zum Core-Teilnehmer. " +
      "Verwende dies nur, wenn du den Teilnehmer vorher im eegFaktura-Core gelöscht hast — sonst werden beim Re-Import Dubletten erzeugt.",
  },
};

export function AdminStatusActions({ applicationId, rcNumber, status, targetParticipantId, importErrorMessage, meteringPoints, onRefresh, emailConfirmationPending }: Props) {
  const { data: session } = useSession();
  const [dialogTarget, setDialogTarget] = useState<DialogTarget | null>(null);
  const [reason, setReason] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [isConflict, setIsConflict] = useState(false);
  const [importDialogOpen, setImportDialogOpen] = useState(false);

  const staticNote = STATIC_NOTES[status];
  if (staticNote) {
    return (
      <p className="text-sm text-muted-foreground italic">{staticNote}</p>
    );
  }

  function handleActionError(err: unknown) {
    if (err instanceof ApiResponseError && err.apiError.code === "conflict") {
      setIsConflict(true);
      setError("Diese Aktion ist nicht mehr gültig. Bitte laden Sie die Seite neu, um den aktuellen Status zu sehen.");
    } else {
      setIsConflict(false);
      setError(err instanceof Error ? err.message : "Fehler bei der Statusänderung");
    }
  }

  async function directAction(toStatus: string) {
    setLoading(true);
    setError(null);
    setIsConflict(false);
    try {
      await changeApplicationStatus(applicationId, { toStatus }, session?.accessToken);
      toast.success("Status erfolgreich geändert");
      onRefresh();
    } catch (err: unknown) {
      handleActionError(err);
    } finally {
      setLoading(false);
    }
  }

  function openImportDialog() {
    setError(null);
    setIsConflict(false);
    setImportDialogOpen(true);
  }

  async function runImport(selection: {
    memberNumber: string;
    tariffId: string;
    meterTariffs: Record<string, string>;
  }) {
    setLoading(true);
    setError(null);
    setIsConflict(false);
    try {
      const res = await importApplication(applicationId, selection, session?.accessToken);
      if (res.memberTariffWarning) {
        toast.warning(
          `Import erfolgreich (Participant-ID: ${res.targetParticipantId ?? "—"}). ` +
            `Mitglieds-Tarif konnte nicht gesetzt werden: ${res.memberTariffWarning}`,
        );
      } else {
        toast.success(`Import erfolgreich (Participant-ID: ${res.targetParticipantId ?? "—"})`);
      }
      setImportDialogOpen(false);
      onRefresh();
    } catch (err: unknown) {
      if (err instanceof ApiResponseError) {
        setError(err.apiError.message || "Import fehlgeschlagen.");
      } else {
        setError(err instanceof Error ? err.message : "Import fehlgeschlagen.");
      }
      onRefresh();
    } finally {
      setLoading(false);
    }
  }

  function openDialog(target: DialogTarget) {
    setReason("");
    setError(null);
    setDialogTarget(target);
  }

  function closeDialog() {
    setDialogTarget(null);
    setReason("");
    setError(null);
  }

  async function confirmDialog() {
    if (!dialogTarget || !reason.trim()) return;
    setLoading(true);
    setError(null);
    setIsConflict(false);
    try {
      if (dialogTarget === "reset_import") {
        await resetImportApplication(applicationId, reason.trim(), session?.accessToken);
        toast.success("Import zurückgesetzt — Antrag ist wieder auf „Genehmigt\".");
      } else {
        await changeApplicationStatus(applicationId, {
          toStatus: dialogTarget,
          reason: reason.trim(),
        }, session?.accessToken);
        toast.success("Status erfolgreich geändert");
      }
      closeDialog();
      onRefresh();
    } catch (err: unknown) {
      handleActionError(err);
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <div className="flex flex-wrap gap-2">
        {status === "submitted" && (
          <>
            <Button
              onClick={() => directAction("under_review")}
              disabled={loading || emailConfirmationPending}
              title={emailConfirmationPending ? "E-Mail-Adresse muss zuerst bestätigt werden" : undefined}
            >
              {loading ? "Bitte warten..." : "In Prüfung nehmen"}
            </Button>
            <Button
              variant="destructive"
              onClick={() => openDialog("rejected")}
              disabled={loading}
            >
              Ablehnen
            </Button>
          </>
        )}

        {status === "email_confirmed" && (
          <Button
            onClick={() => directAction("under_review")}
            disabled={loading}
          >
            {loading ? "Bitte warten..." : "In Prüfung nehmen"}
          </Button>
        )}

        {status === "under_review" && (
          <>
            <Button
              variant="default"
              className="bg-green-600 hover:bg-green-700"
              onClick={() => directAction("approved")}
              disabled={loading}
            >
              {loading ? "Bitte warten..." : "Genehmigen"}
            </Button>
            <Button
              variant="destructive"
              onClick={() => openDialog("rejected")}
              disabled={loading}
            >
              Ablehnen
            </Button>
            <Button
              variant="outline"
              onClick={() => openDialog("needs_info")}
              disabled={loading}
            >
              Informationen anfordern
            </Button>
          </>
        )}

        {status === "needs_info" && (
          <Button
            variant="outline"
            onClick={() => directAction("submitted")}
            disabled={loading}
          >
            {loading ? "Bitte warten..." : "Erneut einreichen"}
          </Button>
        )}

        {status === "approved" && (
          <Button onClick={openImportDialog} disabled={loading}>
            {loading ? "Import läuft..." : "In eegFaktura importieren"}
          </Button>
        )}

        {status === "import_failed" && (
          <>
            <Button onClick={openImportDialog} disabled={loading}>
              {loading ? "Import läuft..." : "Import erneut versuchen"}
            </Button>
            <Button
              variant="outline"
              onClick={() => directAction("approved")}
              disabled={loading}
            >
              Auf "Genehmigt" zurücksetzen
            </Button>
          </>
        )}
      </div>

      {status === "imported" && (
        <div className="space-y-3">
          <div className="text-sm space-y-1">
            <p className="text-muted-foreground italic">Antrag wurde erfolgreich importiert.</p>
            {targetParticipantId && (
              <p>
                <span className="text-muted-foreground">Participant-ID im Core: </span>
                <code className="font-mono">{targetParticipantId}</code>
              </p>
            )}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={() => openDialog("reset_import")}
            disabled={loading}
          >
            Import zurücksetzen
          </Button>
        </div>
      )}

      {status === "import_failed" && importErrorMessage && (
        <div className="mt-2 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm">
          <p className="font-medium text-destructive">Letzter Import fehlgeschlagen:</p>
          <p className="mt-1 break-words text-destructive/90">{importErrorMessage}</p>
        </div>
      )}

      {error && (
        <div className="mt-2 space-y-1">
          <p className="text-sm text-destructive">{error}</p>
          {isConflict && (
            <Button
              variant="outline"
              size="sm"
              onClick={() => window.location.reload()}
            >
              Seite neu laden
            </Button>
          )}
        </div>
      )}

      <Dialog open={dialogTarget !== null} onOpenChange={(open) => { if (!open) closeDialog(); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {dialogTarget ? DIALOG_LABELS[dialogTarget].title : ""}
            </DialogTitle>
          </DialogHeader>
          {dialogTarget && DIALOG_LABELS[dialogTarget].warning && (
            <div className="rounded-md border border-amber-500/50 bg-amber-50 p-3 text-sm text-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
              {DIALOG_LABELS[dialogTarget].warning}
            </div>
          )}
          <div className="space-y-2 py-2">
            <Label htmlFor="reason-input">Begründung</Label>
            <Textarea
              id="reason-input"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder={dialogTarget ? DIALOG_LABELS[dialogTarget].placeholder : ""}
              rows={4}
              className="resize-none"
            />
            {error && (
              <div className="space-y-1">
                <p className="text-sm text-destructive">{error}</p>
                {isConflict && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => window.location.reload()}
                  >
                    Seite neu laden
                  </Button>
                )}
              </div>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeDialog} disabled={loading}>
              Abbrechen
            </Button>
            <Button
              onClick={confirmDialog}
              disabled={loading || reason.trim().length < (dialogTarget === "reset_import" ? 5 : 1)}
              variant={dialogTarget === "rejected" ? "destructive" : "default"}
            >
              {loading
                ? "Bitte warten..."
                : dialogTarget
                ? DIALOG_LABELS[dialogTarget].confirm
                : "Bestätigen"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ImportTariffDialog
        open={importDialogOpen}
        applicationId={applicationId}
        rcNumber={rcNumber}
        meteringPoints={meteringPoints}
        accessToken={session?.accessToken}
        loading={loading}
        onCancel={() => setImportDialogOpen(false)}
        onConfirm={runImport}
      />
    </>
  );
}
