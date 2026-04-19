"use client";

import { useState } from "react";
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
import { changeApplicationStatus } from "@/lib/api";
import type { ApplicationStatus } from "@/lib/api";

interface Props {
  applicationId: string;
  status: ApplicationStatus;
  onRefresh: () => void;
}

type DialogTarget = "rejected" | "needs_info";

const STATIC_NOTES: Partial<Record<ApplicationStatus, string>> = {
  approved:      "Antrag genehmigt — Import über PROJ-4 verfügbar.",
  rejected:      "Antrag abgelehnt. Keine weiteren Aktionen verfügbar.",
  imported:      "Antrag wurde erfolgreich importiert.",
  import_failed: "Import fehlgeschlagen — Reset über PROJ-4 verfügbar.",
};

const DIALOG_LABELS: Record<DialogTarget, { title: string; placeholder: string; confirm: string }> = {
  rejected:   { title: "Antrag ablehnen", placeholder: "Begründung der Ablehnung...", confirm: "Ablehnen" },
  needs_info: { title: "Informationen anfordern", placeholder: "Welche Informationen werden benötigt?", confirm: "Anforderung senden" },
};

export function AdminStatusActions({ applicationId, status, onRefresh }: Props) {
  const [dialogTarget, setDialogTarget] = useState<DialogTarget | null>(null);
  const [reason, setReason] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const staticNote = STATIC_NOTES[status];
  if (staticNote) {
    return (
      <p className="text-sm text-muted-foreground italic">{staticNote}</p>
    );
  }

  async function directAction(toStatus: string) {
    setLoading(true);
    setError(null);
    try {
      await changeApplicationStatus(applicationId, { toStatus });
      toast.success("Status erfolgreich geändert");
      onRefresh();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Fehler bei der Statusänderung";
      setError(msg);
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
    try {
      await changeApplicationStatus(applicationId, {
        toStatus: dialogTarget,
        reason: reason.trim(),
      });
      toast.success("Status erfolgreich geändert");
      closeDialog();
      onRefresh();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Fehler bei der Statusänderung";
      setError(msg);
    } finally {
      setLoading(false);
    }
  }

  return (
    <>
      <div className="flex flex-wrap gap-2">
        {status === "submitted" && (
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
      </div>

      {error && (
        <p className="text-sm text-destructive mt-2">{error}</p>
      )}

      <Dialog open={dialogTarget !== null} onOpenChange={(open) => { if (!open) closeDialog(); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {dialogTarget ? DIALOG_LABELS[dialogTarget].title : ""}
            </DialogTitle>
          </DialogHeader>
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
            {error && <p className="text-sm text-destructive">{error}</p>}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeDialog} disabled={loading}>
              Abbrechen
            </Button>
            <Button
              onClick={confirmDialog}
              disabled={loading || !reason.trim()}
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
    </>
  );
}
