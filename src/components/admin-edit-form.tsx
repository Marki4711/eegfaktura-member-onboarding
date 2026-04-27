"use client";

import { useState } from "react";
import { useSession } from "next-auth/react";
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
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Info } from "lucide-react";
import { toast } from "sonner";
import { updateApplication } from "@/lib/api";
import type { AdminApplicationDetail, MeteringPointRequest, MemberType } from "@/lib/api";

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
  participationFactor: number;
}

function validateEmail(email: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

function toDateInputValue(iso: string | null | undefined): string {
  if (!iso) return "";
  return iso.slice(0, 10);
}

let mpKeyCounter = 0;

export function AdminEditForm({ open, application, onClose, onRefresh }: Props) {
  const { data: session } = useSession();
  const [memberType, setMemberType] = useState<MemberType>(application.memberType ?? "private");
  const [titel, setTitel] = useState(application.titel ?? "");
  const [firstname, setFirstname] = useState(application.firstname ?? "");
  const [lastname, setLastname] = useState(application.lastname ?? "");
  const [birthDate, setBirthDate] = useState(toDateInputValue(application.birthDate));
  const [companyName, setCompanyName] = useState(application.companyName ?? "");
  const [uidNumber, setUidNumber] = useState(application.uidNumber ?? "");
  const [registerNumber, setRegisterNumber] = useState(application.registerNumber ?? "");
  const [email, setEmail] = useState(application.email);
  const [phone, setPhone] = useState(application.phone ?? "");
  const [residentStreet, setResidentStreet] = useState(application.residentStreet);
  const [residentStreetNumber, setResidentStreetNumber] = useState(application.residentStreetNumber);
  const [residentZip, setResidentZip] = useState(application.residentZip);
  const [residentCity, setResidentCity] = useState(application.residentCity);
  const [adminNote, setAdminNote] = useState(application.adminNote ?? "");
  const [memberNumber, setMemberNumber] = useState(
    application.memberNumber != null ? String(application.memberNumber) : ""
  );
  const [einzugsart, setEinzugsart] = useState(application.einzugsart ?? "core");
  const [bankName, setBankName] = useState(application.bankName ?? "");
  const [mandateReference, setMandateReference] = useState(application.mandateReference ?? "");
  const [mandateDate, setMandateDate] = useState(
    application.mandateDate ? application.mandateDate.split("T")[0] : ""
  );
  const [meteringPoints, setMeteringPoints] = useState<FormMeteringPoint[]>(
    application.meteringPoints.map((mp) => ({
      key: ++mpKeyCounter,
      meteringPoint: mp.meteringPoint,
      direction: mp.direction,
      participationFactor: mp.participationFactor ?? 100,
    }))
  );
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isPerson = memberType === "private" || memberType === "farmer";

  function onMemberTypeChange(value: MemberType) {
    setMemberType(value);
    // Clear the group that is no longer relevant
    if (value === "private" || value === "farmer") {
      setCompanyName("");
      setUidNumber("");
      setRegisterNumber("");
    } else {
      setTitel("");
      setFirstname("");
      setLastname("");
      setBirthDate("");
    }
  }

  function addRow() {
    setMeteringPoints((prev) => [
      ...prev,
      { key: ++mpKeyCounter, meteringPoint: "", direction: "CONSUMPTION", participationFactor: 100 },
    ]);
  }

  function removeRow(key: number) {
    setMeteringPoints((prev) => prev.filter((mp) => mp.key !== key));
  }

  function updateRow(key: number, field: keyof Omit<FormMeteringPoint, "key">, value: string) {
    setMeteringPoints((prev) =>
      prev.map((mp) => {
        if (mp.key !== key) return mp;
        if (field === "participationFactor") {
          return { ...mp, participationFactor: parseInt(value, 10) };
        }
        return { ...mp, [field]: value };
      })
    );
  }

  function validate(): string | null {
    if (!email.trim()) return "E-Mail ist erforderlich.";
    if (!validateEmail(email.trim())) return "Ungültige E-Mail-Adresse.";
    if (isPerson) {
      if (!firstname.trim()) return "Vorname ist erforderlich.";
      if (!lastname.trim()) return "Nachname ist erforderlich.";
    } else {
      const orgLabel = memberType === "municipality" ? "Organisationsname"
        : memberType === "association" ? "Vereinsname"
        : "Firmenname";
      if (!companyName.trim()) return `${orgLabel} ist erforderlich.`;
      if (memberType === "company") {
        if (!uidNumber.trim()) return "UID-Nummer ist erforderlich.";
        if (!registerNumber.trim()) return "Firmenbuchnummer ist erforderlich.";
      }
      if (memberType === "association") {
        if (!registerNumber.trim()) return "Vereinsnummer ist erforderlich.";
      }
    }
    if (!residentStreet.trim()) return "Straße ist erforderlich.";
    if (!residentStreetNumber.trim()) return "Hausnummer ist erforderlich.";
    if (!residentZip.trim()) return "PLZ ist erforderlich.";
    if (!residentCity.trim()) return "Ort ist erforderlich.";
    if (meteringPoints.length === 0) return "Mindestens ein Zählpunkt ist erforderlich.";
    for (const mp of meteringPoints) {
      if (!mp.meteringPoint.trim()) return "Alle Zählpunktnummern müssen ausgefüllt sein.";
      if (!Number.isFinite(mp.participationFactor) || mp.participationFactor < 1 || mp.participationFactor > 100) {
        return "Teilnahmefaktor muss zwischen 1 und 100 liegen.";
      }
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
      participationFactor: mp.participationFactor,
    }));

    try {
      await updateApplication(application.id, {
        memberType,
        titel: isPerson ? titel.trim() || undefined : undefined,
        firstname: isPerson ? firstname.trim() || undefined : undefined,
        lastname: isPerson ? lastname.trim() || undefined : undefined,
        birthDate: isPerson ? birthDate || undefined : undefined,
        companyName: !isPerson ? companyName.trim() || undefined : undefined,
        uidNumber: !isPerson ? uidNumber.trim() || undefined : undefined,
        registerNumber: !isPerson ? registerNumber.trim() || undefined : undefined,
        email: email.trim(),
        phone: phone.trim() || undefined,
        residentStreet: residentStreet.trim(),
        residentStreetNumber: residentStreetNumber.trim(),
        residentZip: residentZip.trim(),
        residentCity: residentCity.trim(),
        adminNote: adminNote,
        einzugsart: einzugsart,
        bankName: bankName.trim() || undefined,
        mandateReference: mandateReference.trim() || undefined,
        mandateDate: mandateDate || undefined,
        memberNumber: memberNumber.trim() ? parseInt(memberNumber.trim(), 10) || undefined : undefined,
        meteringPoints: payload,
      }, session?.accessToken);
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
          {/* Mitgliedsnummer */}
          <div className="space-y-1">
            <Label htmlFor="edit-member-number" className="flex items-center gap-1">
              Mitgliedsnummer
              <Popover>
                <PopoverTrigger type="button" className="cursor-help">
                  <Info className="h-3.5 w-3.5 text-muted-foreground" />
                </PopoverTrigger>
                <PopoverContent className="max-w-60 text-sm">
                  Wird bei der ersten Einreichung automatisch vergeben (fortlaufend pro EEG), kann hier aber manuell angepasst werden.
                </PopoverContent>
              </Popover>
            </Label>
            <Input
              id="edit-member-number"
              type="number"
              min={1}
              value={memberNumber}
              onChange={(e) => setMemberNumber(e.target.value)}
            />
          </div>

          <Separator />

          {/* Member type */}
          <div>
            <h3 className="text-sm font-semibold mb-3">Mitgliedstyp</h3>
            <Select value={memberType} onValueChange={(v) => onMemberTypeChange(v as MemberType)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="private">Privatperson / Kleinunternehmer</SelectItem>
                <SelectItem value="farmer">Pauschalierter Landwirt</SelectItem>
                <SelectItem value="municipality">Gemeinde / öffentl. Körperschaft</SelectItem>
                <SelectItem value="company">Unternehmen</SelectItem>
                <SelectItem value="association">Verein</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <Separator />

          {/* Member / organisation data */}
          <div>
            <h3 className="text-sm font-semibold mb-3">
              {isPerson ? "Persönliche Daten" : "Organisationsdaten"}
            </h3>
            {isPerson ? (
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1 col-span-2">
                  <Label htmlFor="edit-titel">Titel</Label>
                  <Input
                    id="edit-titel"
                    value={titel}
                    onChange={(e) => setTitel(e.target.value)}
                  />
                </div>
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
              </div>
            ) : (
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1 col-span-2">
                  <Label htmlFor="edit-company-name">
                    {memberType === "municipality"
                      ? "Organisationsname *"
                      : memberType === "association"
                      ? "Vereinsname *"
                      : "Firmenname *"}
                  </Label>
                  <Input
                    id="edit-company-name"
                    value={companyName}
                    onChange={(e) => setCompanyName(e.target.value)}
                  />
                </div>
                {(memberType === "company" || memberType === "association") && (
                  <div className="space-y-1">
                    <Label htmlFor="edit-register">
                      {memberType === "association" ? "Vereinsnummer *" : "Firmenbuchnummer *"}
                    </Label>
                    <Input
                      id="edit-register"
                      value={registerNumber}
                      onChange={(e) => setRegisterNumber(e.target.value)}
                    />
                  </div>
                )}
                <div className="space-y-1">
                  <Label htmlFor="edit-uid">
                    UID-Nummer{memberType === "company" ? " *" : ""}
                  </Label>
                  <Input
                    id="edit-uid"
                    value={uidNumber}
                    onChange={(e) => setUidNumber(e.target.value)}
                  />
                </div>
              </div>
            )}

            <div className="grid grid-cols-2 gap-4 mt-4">
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
                    <SelectTrigger className="w-36">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="CONSUMPTION">Bezug</SelectItem>
                      <SelectItem value="PRODUCTION">Einspeisung</SelectItem>
                    </SelectContent>
                  </Select>
                  <div className="flex items-center gap-1 shrink-0">
                    <Input
                      type="number"
                      min={1}
                      max={100}
                      value={mp.participationFactor}
                      onChange={(e) => updateRow(mp.key, "participationFactor", e.target.value)}
                      className="w-20 text-sm"
                      title="Teilnahmefaktor (%)"
                    />
                    <span className="text-sm text-muted-foreground">%</span>
                  </div>
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

          {/* Einzugsart */}
          <div>
            <h3 className="text-sm font-semibold mb-3">Einzugsart</h3>
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1 col-span-2">
                <Label htmlFor="edit-einzugsart">Einzugsart</Label>
                <Select value={einzugsart} onValueChange={setEinzugsart}>
                  <SelectTrigger id="edit-einzugsart">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="core">Core</SelectItem>
                    <SelectItem value="b2b">B2B</SelectItem>
                    <SelectItem value="kein_sepa">Kein Sepa</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {einzugsart !== "kein_sepa" && (
                <>
                  <div className="space-y-1 col-span-2">
                    <Label htmlFor="edit-bank-name">Bankverbindung</Label>
                    <Input
                      id="edit-bank-name"
                      value={bankName}
                      onChange={(e) => setBankName(e.target.value)}
                      placeholder="z.B. Raiffeisen Wien"
                    />
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor="edit-mandate-reference">Mandatsreferenz</Label>
                    <Input
                      id="edit-mandate-reference"
                      value={mandateReference}
                      onChange={(e) => setMandateReference(e.target.value)}
                    />
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor="edit-mandate-date">Mandatsdatum</Label>
                    <Input
                      id="edit-mandate-date"
                      type="date"
                      value={mandateDate}
                      onChange={(e) => setMandateDate(e.target.value)}
                    />
                  </div>
                </>
              )}
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
