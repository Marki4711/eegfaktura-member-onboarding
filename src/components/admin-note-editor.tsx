"use client";

import { useState } from "react";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "sonner";
import { updateApplication } from "@/lib/api";
import type { AdminApplicationDetail } from "@/lib/api";

interface Props {
  application: AdminApplicationDetail;
  onRefresh: () => void;
}

export function AdminNoteEditor({ application, onRefresh }: Props) {
  const { data: session } = useSession();
  const [editing, setEditing] = useState(false);
  const [note, setNote] = useState(application.adminNote ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function startEdit() {
    setNote(application.adminNote ?? "");
    setError(null);
    setEditing(true);
  }

  function cancel() {
    setEditing(false);
    setError(null);
  }

  async function save() {
    setSaving(true);
    setError(null);
    try {
      await updateApplication(application.id, {
        firstname: application.firstname ?? undefined,
        lastname: application.lastname ?? undefined,
        birthDate: application.birthDate?.slice(0, 10) ?? undefined,
        email: application.email,
        phone: application.phone ?? undefined,
        residentStreet: application.residentStreet,
        residentStreetNumber: application.residentStreetNumber,
        residentZip: application.residentZip,
        residentCity: application.residentCity,
        adminNote: note,
        meteringPoints: application.meteringPoints.map((mp) => ({
          meteringPoint: mp.meteringPoint,
          direction: mp.direction,
        })),
      }, session?.accessToken);
      toast.success("Notiz gespeichert");
      setEditing(false);
      onRefresh();
    } catch (err: unknown) {
      const msg =
        err instanceof Error ? err.message : "Fehler beim Speichern der Notiz";
      setError(msg);
    } finally {
      setSaving(false);
    }
  }

  if (!editing) {
    return (
      <div className="space-y-2">
        <div className="min-h-[3rem] rounded-md border bg-muted/30 p-3 text-sm">
          {application.adminNote ? (
            <p className="whitespace-pre-wrap">{application.adminNote}</p>
          ) : (
            <p className="text-muted-foreground italic">Keine Admin-Notiz vorhanden.</p>
          )}
        </div>
        <Button variant="outline" size="sm" onClick={startEdit}>
          Notiz bearbeiten
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <Textarea
        value={note}
        onChange={(e) => setNote(e.target.value)}
        rows={4}
        placeholder="Interne Notiz für Kollegen..."
        className="resize-none"
      />
      {error && <p className="text-sm text-destructive">{error}</p>}
      <div className="flex gap-2">
        <Button size="sm" onClick={save} disabled={saving}>
          {saving ? "Speichern..." : "Speichern"}
        </Button>
        <Button size="sm" variant="outline" onClick={cancel} disabled={saving}>
          Abbrechen
        </Button>
      </div>
    </div>
  );
}
