"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { toast } from "sonner";
import { updateApplication } from "@/lib/api";
import type { AdminApplicationDetail, MeteringPointRequest } from "@/lib/api";

interface Props {
  open: boolean;
  application: AdminApplicationDetail;
  onClose: () => void;
  onRefresh: () => void;
}

interface FormMeteringPoint {
  key: number;
  meteringPoint: string;
  direction: "CONSUMPTION" | "PRODUCTION";
}

function validateEmail(email: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

// The API returns birthDate as RFC3339 ("1962-06-06T00:00:00Z").
// <input type="date"> requires "YYYY-MM-DD". Slice the first 10 chars.
function toDateInputValue(iso: string | null | undefined): string {
  if (!iso) return "";
  return iso.slice(0, 10);
}

let mpKeyCounter = 0;

export function AdminEditForm({ open, application, onClose, onRefresh }: Props) {
  const [firstname, setFirstname] = useState(application.firstname);
  const [lastname, setLastname] = useState(application.lastname);
  const [birthDate, setBirthDate] = useState(toDateInputValue(application.birthDate));
  const [email, setEmail] = useState(application.email);
  const [phone, setPhone] = useState(application.phone ?? "");
  const [residentStreet, setResidentStreet] = useState(application.residentStreet);
  const [residentStreetNumber, setResidentStreetNumber] = useState(application.residentStreetNumber);
  const [residentZip, setResidentZip] = useState(application.residentZip);
  const [residentCity, setResidentCity] = useState(application.residentCity);
  const [residentCountry, setResidentCountry] = useState(application.residentCountry);
  const [adminNote, setAdminNote] = useState(application.adminNote ?? "");
  const [meteringPoints, setMeteringPoints] = useState<FormMeteringPoint[]>(
    application.meteringPoints.map((mp) => ({
      key: ++mpKeyCounter,
      meteringPoint: mp.meteringPoint,
      direction: mp.direction,
    }))
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function addRow() {
    setMeteringPoints((prev) => [
      ...prev,
      { key: ++mpKeyCounter, meteringPoint: "", direction: "CONSUMPTION" },
    ]);
  }

  function removeRow(key: number) {
    setMeteringPoints((prev) => prev.filter((mp) => mp.key !== key));
  }

  function updateRow(key: number, field: keyof Omit<FormMeteringPoint, "key">, value: string) {
    setMeteringPoints((prev) =>
      prev.map((mp) => (mp.key === key ? { ...mp, [field]: value } : mp))
    );
  }

  function validate(): string | null {
    if (!firstname.trim()) return "Vorname ist erforderlich.";
    if (!lastname.trim()) return "Nachname ist erforderlich.";
    if (!email.trim()) return "E-Mail ist erforderlich.";
    if (!validateEmail(email.trim())) return "Ungültige E-Mail-Adresse.";
    if (!residentStreet.trim()) return "Straße ist erforderlich.";
    if (!residentStreetNumber.trim()) return "Hausnummer ist erforderlich.";
    if (!residentZip.trim()) return "PLZ ist erforderlich.";
    if (!residentCity.trim()) return "Ort ist erforderlich.";
    if (!residentCountry.trim()) return "Land ist erforderlich.";
    if (meteringPoints.length === 0) return "Mindestens ein Zählpunkt ist erforderlich.";
    for (const mp of meteringPoints) {
      if (!mp.meteringPoint.trim()) return "Alle Zählpunktnummern müssen ausgefüllt sein.";
    }
    return null;
  }

  async function save() {
    const validationError = validate();
    if (validationError) {
      setError(validationError);
      return;
    }

    setSaving(true);
    setError(null);

    const payload: MeteringPointRequest[] = meteringPoints.map((mp) => ({
      meteringPoint: mp.meteringPoint.trim(),
      direction: mp.direction,
    }));

    try {
      await updateApplication(application.id, {
        firstname: firstname.trim(),
        lastname: lastname.trim(),
        birthDate: birthDate || undefined,
        email: email.trim(),
        phone: phone.trim() || undefined,
        residentStreet: residentStreet.trim(),
        residentStreetNumber: residentStreetNumber.trim(),
        residentZip: residentZip.trim(),
        residentCity: residentCity.trim(),
        residentCountry: residentCountry.trim(),
        adminNote: adminNote,
        meteringPoints: payload,
      });
      toast.success("Änderungen gespeichert");
      onClose();
      onRefresh();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Fehler beim Speichern";
      setError(msg);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={(isOpen) => { if (!isOpen) onClose(); }}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Antrag bearbeiten</DialogTitle>
        </DialogHeader>

        <div className="space-y-6 py-2">
          {/* Personal data */}
          <div>
            <h3 className="text-sm font-semibold mb-3">Persönliche Daten</h3>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1">
                <Label htmlFor="edit-firstname">Vorname *</Label>
                <Input
                  id="edit-firstname"
                  value={firstname}
                  onChange={(e) => setFirstname(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="edit-lastname">Nachname *</Label>
                <Input
                  id="edit-lastname"
                  value={lastname}
                  onChange={(e) => setLastname(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="edit-birthdate">Geburtsdatum</Label>
                <Input
                  id="edit-birthdate"
                  type="date"
                  value={birthDate}
                  onChange={(e) => setBirthDate(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="edit-email">E-Mail *</Label>
                <Input
                  id="edit-email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="edit-phone">Telefon</Label>
                <Input
                  id="edit-phone"
                  value={phone}
                  onChange={(e) => setPhone(e.target.value)}
                />
              </div>
            </div>
          </div>

          <Separator />

          {/* Address */}
          <div>
            <h3 className="text-sm font-semibold mb-3">Adresse</h3>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1 col-span-2 sm:col-span-1">
                <Label htmlFor="edit-street">Straße *</Label>
                <Input
                  id="edit-street"
                  value={residentStreet}
                  onChange={(e) => setResidentStreet(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="edit-street-nr">Hausnummer *</Label>
                <Input
                  id="edit-street-nr"
                  value={residentStreetNumber}
                  onChange={(e) => setResidentStreetNumber(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="edit-zip">PLZ *</Label>
                <Input
                  id="edit-zip"
                  value={residentZip}
                  onChange={(e) => setResidentZip(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="edit-city">Ort *</Label>
                <Input
                  id="edit-city"
                  value={residentCity}
                  onChange={(e) => setResidentCity(e.target.value)}
                />
              </div>
              <div className="space-y-1">
                <Label htmlFor="edit-country">Land *</Label>
                <Input
                  id="edit-country"
                  value={residentCountry}
                  onChange={(e) => setResidentCountry(e.target.value)}
                />
              </div>
            </div>
          </div>

          <Separator />

          {/* Metering points */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold">Zählpunkte *</h3>
              <Button type="button" variant="outline" size="sm" onClick={addRow}>
                + Hinzufügen
              </Button>
            </div>
            {meteringPoints.length === 0 && (
              <p className="text-sm text-destructive mb-2">
                Mindestens ein Zählpunkt ist erforderlich.
              </p>
            )}
            <div className="space-y-2">
              {meteringPoints.map((mp) => (
                <div key={mp.key} className="flex gap-2 items-center">
                  <Input
                    value={mp.meteringPoint}
                    onChange={(e) => updateRow(mp.key, "meteringPoint", e.target.value)}
                    placeholder="AT003100000000000000000000000000"
                    className="font-mono text-sm flex-1"
                  />
                  <Select
                    value={mp.direction}
                    onValueChange={(v) => updateRow(mp.key, "direction", v)}
                  >
                    <SelectTrigger className="w-40">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="CONSUMPTION">Bezug</SelectItem>
                      <SelectItem value="PRODUCTION">Einspeisung</SelectItem>
                    </SelectContent>
                  </Select>
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => removeRow(mp.key)}
                    className="text-destructive hover:text-destructive"
                  >
                    ✕
                  </Button>
                </div>
              ))}
            </div>
          </div>

          <Separator />

          {/* Admin note */}
          <div className="space-y-1">
            <Label htmlFor="edit-admin-note">Admin-Notiz</Label>
            <Textarea
              id="edit-admin-note"
              value={adminNote}
              onChange={(e) => setAdminNote(e.target.value)}
              rows={3}
              placeholder="Interne Notiz für Kollegen..."
              className="resize-none"
            />
          </div>

          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={saving}>
            Abbrechen
          </Button>
          <Button onClick={save} disabled={saving}>
            {saving ? "Speichern..." : "Speichern"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
