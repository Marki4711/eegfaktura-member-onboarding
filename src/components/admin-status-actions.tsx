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
import { changeApplicationStatus, importApplication, resetImportApplication, reassignApplicationToEEG, ApiResponseError } from "@/lib/api";
import type { ApplicationStatus, MeteringPointDetail } from "@/lib/api";
import { ImportTariffDialog } from "@/components/import-tariff-dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

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
      "Diese Aktion setzt den Antrag zurück auf „Genehmigt\" und löscht die Verknüpfung zum Core-Teilnehmer " +
      "(inkl. Mitgliedsnummer und ggf. Bank-Bestätigung). " +
      "Verwende dies nur, wenn du den Teilnehmer vorher im eegFaktura-Core gelöscht hast — sonst werden beim Re-Import Dubletten erzeugt. " +
      "Anträge im Status „Aktiviert\" können hier nicht mehr zurückgesetzt werden — dazu muss das Mitglied zuerst im Core deaktiviert werden.",
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

  // PROJ-40: EEG-Umzuordnung. Admin sieht die Funktion nur, wenn er ≥ 2 EEGs
  // verwaltet (oder Superuser ist). Die Liste der möglichen Ziel-EEGs kommt
  // aus der Session und schließt die aktuelle EEG aus.
  const sessionTenants = ((session as unknown as { tenant?: string[] })?.tenant ?? []) as string[];
  const reassignableStatuses: ApplicationStatus[] = ["submitted", "email_confirmed", "under_review", "needs_info"];
  const availableTargetRcs = sessionTenants.filter((rc) => rc !== rcNumber);
  const canReassign = reassignableStatuses.includes(status) && availableTargetRcs.length > 0;
  const [reassignDialogOpen, setReassignDialogOpen] = useState(false);
  const [reassignTarget, setReassignTarget] = useState<string>("");
  const [reassignReason, setReassignReason] = useState("");
  const [reassignLoading, setReassignLoading] = useState(false);
  const [reassignError, setReassignError] = useState<string | null>(null);

  const staticNote = STATIC_NOTES[status];
  if (staticNote) {
    return (
      <p className="text-sm text-muted-foreground italic">{staticNote}</p>
    );
  }

  function handleActionError(err: unknown) {
    if (err instanceof ApiResponseError && err.apiError.code === "conflict") {
      setIsConflict(true);
      // Prefer the server's specific German message (e.g. PROJ-31's
      // "E-Mail-Adresse des Bewerbers ist noch nicht bestätigt …") over
      // the generic "Aktion nicht mehr gültig"-Text. The reload button
      // is still shown afterwards so the admin can refresh the view if
      // the conflict was about a stale local status.
      const serverMessage = err.apiError.message?.trim();
      setError(
        serverMessage && serverMessage.length > 0
          ? serverMessage
          : "Diese Aktion ist nicht mehr gültig. Bitte laden Sie die Seite neu, um den aktuellen Status zu sehen.",
      );
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
      const msg = err instanceof ApiResponseError
        ? (err.apiError.message || "Import fehlgeschlagen.")
        : (err instanceof Error ? err.message : "Import fehlgeschlagen.");
      setError(msg);
      // Surface as toast too — the inline error inside the open dialog is
      // the primary signal; the toast catches the case where the dialog
      // gets dismissed before the admin reads the inline message.
      toast.error(msg);
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

  function openReassignDialog() {
    setReassignTarget(availableTargetRcs[0] ?? "");
    setReassignReason("");
    setReassignError(null);
    setReassignDialogOpen(true);
  }

  function closeReassignDialog() {
    if (reassignLoading) return;
    setReassignDialogOpen(false);
    setReassignTarget("");
    setReassignReason("");
    setReassignError(null);
  }

  async function confirmReassign() {
    if (!reassignTarget || reassignReason.trim().length < 5) return;
    setReassignLoading(true);
    setReassignError(null);
    try {
      await reassignApplicationToEEG(applicationId, reassignTarget, reassignReason.trim(), session?.accessToken);
      toast.success(`Antrag wurde der EEG ${reassignTarget} zugeordnet.`);
      setReassignDialogOpen(false);
      setReassignTarget("");
      setReassignReason("");
      onRefresh();
    } catch (err: unknown) {
      const msg = err instanceof ApiResponseError
        ? (err.apiError.message || "Umzuordnung fehlgeschlagen.")
        : (err instanceof Error ? err.message : "Umzuordnung fehlgeschlagen.");
      setReassignError(msg);
    } finally {
      setReassignLoading(false);
    }
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

        {canReassign && (
          <Button
            variant="outline"
            onClick={openReassignDialog}
            disabled={loading}
          >
            EEG umzuordnen
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

      {/* PROJ-46: Post-Import-Stati. */}
      {status === "awaiting_bank_confirmation" && (
        <div className="space-y-3">
          <div className="rounded-md border border-amber-500/50 bg-amber-50 p-3 text-sm text-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
            <p className="font-medium mb-1">Warte auf Bank-Bestätigung</p>
            <p>
              Das Mitglied wurde gebeten, das B2B-SEPA-Mandat bei seiner Hausbank zu hinterlegen.
              Sobald die Rückmeldung kommt, hier auf <strong>„Bank-Bestätigung erhalten"</strong> klicken.
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="default"
              onClick={() => directAction("ready_for_activation")}
              disabled={loading}
            >
              {loading ? "Bitte warten..." : "Bank-Bestätigung erhalten"}
            </Button>
            <Button
              variant="outline"
              onClick={() => directAction("under_review")}
              disabled={loading}
            >
              Zurück in Prüfung
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => openDialog("reset_import")}
              disabled={loading}
            >
              Import zurücksetzen
            </Button>
          </div>
        </div>
      )}

      {status === "ready_for_activation" && (
        <div className="space-y-3">
          <div className="text-sm space-y-1">
            <p className="text-muted-foreground italic">
              Mitglied ist bereit zur Aktivierung in der EEG.
            </p>
            {targetParticipantId && (
              <p>
                <span className="text-muted-foreground">Participant-ID im Core: </span>
                <code className="font-mono">{targetParticipantId}</code>
              </p>
            )}
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              variant="default"
              className="bg-green-600 hover:bg-green-700"
              onClick={() => directAction("activated")}
              disabled={loading}
            >
              {loading ? "Bitte warten..." : "Als aktiv markieren"}
            </Button>
            <Button
              variant="outline"
              onClick={() => directAction("under_review")}
              disabled={loading}
            >
              Zurück in Prüfung
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => openDialog("reset_import")}
              disabled={loading}
            >
              Import zurücksetzen
            </Button>
          </div>
        </div>
      )}

      {status === "activated" && (
        <p className="text-sm text-muted-foreground italic">
          Mitglied ist aktiv in der EEG. Keine weiteren Aktionen verfügbar.
        </p>
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
          {(dialogTarget === "rejected" || dialogTarget === "needs_info") && (
            <div className="rounded-md border border-blue-500/40 bg-blue-50 p-3 text-sm text-blue-900 dark:bg-blue-950/30 dark:text-blue-200">
              {dialogTarget === "rejected"
                ? "Die hier eingegebene Begründung wird per E-Mail an den Beitrittswerber übermittelt."
                : "Der hier eingegebene Text wird per E-Mail an den Beitrittswerber übermittelt."}
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

      <Dialog open={reassignDialogOpen} onOpenChange={(open) => { if (!open) closeReassignDialog(); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Antrag einer anderen EEG zuordnen</DialogTitle>
          </DialogHeader>
          <div className="rounded-md border border-amber-500/50 bg-amber-50 p-3 text-sm text-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
            Beim Umzuordnen wird eine <strong>neue Referenznummer</strong> der Ziel-EEG vergeben.
            Die alte Referenznummer und EEG werden im Statusverlauf archiviert.
          </div>
          <div className="space-y-2 py-2">
            <Label htmlFor="reassign-target">Ziel-EEG</Label>
            <Select value={reassignTarget} onValueChange={setReassignTarget}>
              <SelectTrigger id="reassign-target">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {availableTargetRcs.map((rc) => (
                  <SelectItem key={rc} value={rc}>{rc}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2 py-2">
            <Label htmlFor="reassign-reason">Begründung *</Label>
            <Textarea
              id="reassign-reason"
              value={reassignReason}
              onChange={(e) => setReassignReason(e.target.value)}
              rows={3}
              className="resize-none"
            />
            {reassignError && (
              <p className="text-sm text-destructive">{reassignError}</p>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeReassignDialog} disabled={reassignLoading}>
              Abbrechen
            </Button>
            <Button
              onClick={confirmReassign}
              disabled={reassignLoading || !reassignTarget || reassignReason.trim().length < 5}
            >
              {reassignLoading ? "Wird umzuordnet..." : "Umzuordnen"}
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
        errorMessage={error}
        onCancel={() => setImportDialogOpen(false)}
        onConfirm={runImport}
      />
    </>
  );
}
