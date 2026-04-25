"use client";

import { useEffect, useState } from "react";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { Label } from "@/components/ui/label";
import {
  listLegalDocuments,
  createLegalDocument,
  updateLegalDocument,
  deleteLegalDocument,
  reorderLegalDocuments,
  getEEGSettings,
  saveEEGSettings,
  ApiResponseError,
  type LegalDocumentItem,
  type CreateLegalDocumentRequest,
  type EEGSettings,
} from "@/lib/api";

interface Props {
  rcNumber: string;
}

interface DocForm {
  title: string;
  url: string;
  required: boolean;
}

const emptyForm: DocForm = { title: "", url: "", required: true };

export function AdminLegalDocumentsEditor({ rcNumber }: Props) {
  const { data: session } = useSession();
  const [docs, setDocs] = useState<LegalDocumentItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingDoc, setEditingDoc] = useState<LegalDocumentItem | null>(null);
  const [form, setForm] = useState<DocForm>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const [showCentralPolicy, setShowCentralPolicy] = useState(true);
  const [eegSettings, setEegSettings] = useState<EEGSettings | null>(null);
  const [policyToggleSaving, setPolicyToggleSaving] = useState(false);

  const loadDocs = async () => {
    setLoading(true);
    setError(null);
    try {
      const [data, settings] = await Promise.all([
        listLegalDocuments(rcNumber, session?.accessToken),
        getEEGSettings(rcNumber, session?.accessToken),
      ]);
      setDocs(data);
      setEegSettings(settings);
      setShowCentralPolicy(settings.showCentralPolicy ?? true);
    } catch {
      setError("Dokumente konnten nicht geladen werden.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadDocs();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rcNumber, session?.accessToken]);

  async function handlePolicyToggle(checked: boolean) {
    if (!eegSettings) return;
    setPolicyToggleSaving(true);
    setShowCentralPolicy(checked);
    try {
      await saveEEGSettings(rcNumber, { ...eegSettings, showCentralPolicy: checked }, session?.accessToken);
      setEegSettings({ ...eegSettings, showCentralPolicy: checked });
    } catch {
      setShowCentralPolicy(!checked);
    } finally {
      setPolicyToggleSaving(false);
    }
  }

  function openAdd() {
    setEditingDoc(null);
    setForm(emptyForm);
    setFormError(null);
    setDialogOpen(true);
  }

  function openEdit(doc: LegalDocumentItem) {
    setEditingDoc(doc);
    setForm({ title: doc.title, url: doc.url, required: doc.required });
    setFormError(null);
    setDialogOpen(true);
  }

  async function handleSave() {
    if (!form.title.trim()) { setFormError("Titel ist erforderlich."); return; }
    if (!form.url.trim()) { setFormError("URL ist erforderlich."); return; }
    setSaving(true);
    setFormError(null);
    const req: CreateLegalDocumentRequest = { title: form.title.trim(), url: form.url.trim(), required: form.required };
    try {
      if (editingDoc) {
        await updateLegalDocument(editingDoc.id, req, session?.accessToken);
      } else {
        await createLegalDocument(rcNumber, req, session?.accessToken);
      }
      setDialogOpen(false);
      await loadDocs();
    } catch (err) {
      const msg = err instanceof ApiResponseError ? err.message : "Speichern fehlgeschlagen.";
      setFormError(msg);
    } finally {
      setSaving(false);
    }
  }

  async function handleDelete(id: string) {
    try {
      await deleteLegalDocument(id, session?.accessToken);
      await loadDocs();
    } catch {
      setError("Dokument konnte nicht gelöscht werden.");
    }
  }

  async function handleReorder(id: string, direction: "up" | "down") {
    const idx = docs.findIndex((d) => d.id === id);
    if (idx === -1) return;
    const newDocs = [...docs];
    const targetIdx = direction === "up" ? idx - 1 : idx + 1;
    if (targetIdx < 0 || targetIdx >= newDocs.length) return;
    [newDocs[idx], newDocs[targetIdx]] = [newDocs[targetIdx], newDocs[idx]];
    setDocs(newDocs);
    try {
      await reorderLegalDocuments(rcNumber, newDocs.map((d) => d.id), session?.accessToken);
    } catch {
      await loadDocs();
    }
  }

  if (loading) {
    return <p className="text-sm text-muted-foreground">Lade Dokumente …</p>;
  }

  if (error) {
    return (
      <div className="space-y-2">
        <p className="text-sm text-destructive">{error}</p>
        <Button variant="outline" size="sm" onClick={loadDocs}>Erneut versuchen</Button>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {docs.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          Noch keine EEG-spezifischen Dokumente konfiguriert. Die zentrale Datenschutzerklärung wird über Servereinstellungen konfiguriert.
        </p>
      ) : (
        <div className="space-y-2">
          {docs.map((doc, idx) => (
            <Card key={doc.id}>
              <CardContent className="py-3 px-4 flex items-center gap-3">
                <div className="flex flex-col gap-0.5">
                  <Button variant="ghost" size="icon" className="h-5 w-5" onClick={() => handleReorder(doc.id, "up")} disabled={idx === 0} aria-label="Nach oben">▲</Button>
                  <Button variant="ghost" size="icon" className="h-5 w-5" onClick={() => handleReorder(doc.id, "down")} disabled={idx === docs.length - 1} aria-label="Nach unten">▼</Button>
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{doc.title}</p>
                  <a href={doc.url} target="_blank" rel="noopener noreferrer" className="text-xs text-muted-foreground hover:underline truncate block">{doc.url}</a>
                </div>
                <span className={`text-xs px-2 py-0.5 rounded-full ${doc.required ? "bg-red-100 text-red-700" : "bg-gray-100 text-gray-600"}`}>
                  {doc.required ? "Pflicht" : "Optional"}
                </span>
                <Button variant="ghost" size="sm" onClick={() => openEdit(doc)}>Bearbeiten</Button>
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive">Löschen</Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Dokument löschen?</AlertDialogTitle>
                      <AlertDialogDescription>
                        <strong>{doc.title}</strong> wird dauerhaft aus der Registrierung entfernt.
                        Bestehende Einwilligungen bleiben erhalten.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Abbrechen</AlertDialogCancel>
                      <AlertDialogAction onClick={() => handleDelete(doc.id)} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
                        Löschen
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Button variant="outline" size="sm" onClick={openAdd}>+ Dokument hinzufügen</Button>

      <div className="border-t border-border pt-4 mt-2">
        <div className="flex items-center gap-3">
          <Switch
            id="show-central-policy"
            checked={showCentralPolicy}
            onCheckedChange={handlePolicyToggle}
            disabled={policyToggleSaving || !eegSettings}
          />
          <Label htmlFor="show-central-policy" className="cursor-pointer font-normal">
            Standard-Datenschutzerklärung im Registrierungsformular anzeigen
          </Label>
        </div>
        <p className="text-xs text-muted-foreground mt-1 ml-10">
          Deaktivieren wenn diese EEG eine eigene Datenschutzerklärung als Dokument oben konfiguriert hat.
        </p>
      </div>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editingDoc ? "Dokument bearbeiten" : "Neues Dokument"}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-2">
            <div className="space-y-1.5">
              <Label htmlFor="doc-title">Titel *</Label>
              <Input
                id="doc-title"
                value={form.title}
                onChange={(e) => setForm({ ...form, title: e.target.value })}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="doc-url">URL *</Label>
              <Input
                id="doc-url"
                value={form.url}
                onChange={(e) => setForm({ ...form, url: e.target.value })}
                type="url"
              />
            </div>
            <div className="flex items-center gap-3">
              <Switch
                id="doc-required"
                checked={form.required}
                onCheckedChange={(v) => setForm({ ...form, required: v })}
              />
              <Label htmlFor="doc-required" className="cursor-pointer">
                Zustimmung erforderlich
              </Label>
            </div>
            {formError && <p className="text-sm text-destructive">{formError}</p>}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)} disabled={saving}>Abbrechen</Button>
            <Button onClick={handleSave} disabled={saving}>
              {saving ? "Wird gespeichert …" : "Speichern"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
