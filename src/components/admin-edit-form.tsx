"use client";

import { useState } from "react";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
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
  // PROJ-45: pro PRODUCTION-Zählpunkt admin-editierbar; CONSUMPTION ignoriert das.
  generationType?: "pv" | "hydro" | "wind" | "biomass";
  batterySizeKwh?: number;
  inverterManufacturer?: string;
  // PROJ-49: Energie-Felder pro Zählpunkt — durchgereicht beim Update,
  // damit Member-Eingaben beim Admin-Edit nicht stillschweigend gelöscht
  // werden (das Backend ersetzt die Zählpunkte vollständig).
  consumptionPreviousYear?: number;
  consumptionForecast?: number;
  feedInForecast?: number;
  pvPowerKwp?: number;
  feedInLimitPresent?: boolean;
  feedInLimitKw?: number;
  // PROJ-49 follow-up: „Speichersteuerung im Sinne der EEG vorstellbar?"
  batteryControlAcceptable?: boolean;
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
  const [titelNach, setTitelNach] = useState(application.titelNach ?? "");
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
  const [einzugsart, setEinzugsart] = useState(application.einzugsart ?? "core");
  const [bankName, setBankName] = useState(application.bankName ?? "");
  const [mandateReference, setMandateReference] = useState(application.mandateReference ?? "");
  const [mandateDate, setMandateDate] = useState(
    application.mandateDate ? application.mandateDate.split("T")[0] : ""
  );
  // PROJ-56: Admin kann die Netzbetreiber-Info-Felder editieren — werden
  // im UI nur gerendert wenn die Vollmacht aktiv ist (semantisch ohne
  // Vollmacht nicht sinnvoll).
  const [networkOperatorCustomerNumber, setNetworkOperatorCustomerNumber] = useState(
    application.networkOperatorCustomerNumber ?? ""
  );
  const [meterInventoryNumber, setMeterInventoryNumber] = useState(
    application.meterInventoryNumber ?? ""
  );
  // PROJ-57: Ansprechperson — Admin kann Toggle + drei Felder editieren.
  // Sichtbarkeit im UI: nur Org-Mitgliedstypen (company/association/municipality).
  const [hasContactPerson, setHasContactPerson] = useState(application.hasContactPerson ?? false);
  const [contactPersonName, setContactPersonName] = useState(application.contactPersonName ?? "");
  const [contactPersonEmail, setContactPersonEmail] = useState(application.contactPersonEmail ?? "");
  const [contactPersonPhone, setContactPersonPhone] = useState(application.contactPersonPhone ?? "");
  const isOrgType = memberType === "company" || memberType === "association" || memberType === "municipality";
  const [meteringPoints, setMeteringPoints] = useState<FormMeteringPoint[]>(
    application.meteringPoints.map((mp) => ({
      key: ++mpKeyCounter,
      meteringPoint: mp.meteringPoint,
      direction: mp.direction,
      participationFactor: mp.participationFactor ?? 100,
      generationType: (mp.generationType ?? undefined) as FormMeteringPoint["generationType"],
      batterySizeKwh: mp.batterySizeKwh ?? undefined,
      inverterManufacturer: mp.inverterManufacturer ?? undefined,
      consumptionPreviousYear: mp.consumptionPreviousYear ?? undefined,
      consumptionForecast: mp.consumptionForecast ?? undefined,
      feedInForecast: mp.feedInForecast ?? undefined,
      pvPowerKwp: mp.pvPowerKwp ?? undefined,
      feedInLimitPresent: mp.feedInLimitPresent ?? undefined,
      feedInLimitKw: mp.feedInLimitKw ?? undefined,
      batteryControlAcceptable: mp.batteryControlAcceptable ?? undefined,
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
    } else if (value === "sole_proprietor") {
      // PROJ-28: Kleinunternehmer captures only the company name.
      setTitel("");
      setTitelNach("");
      setFirstname("");
      setLastname("");
      setBirthDate("");
      setUidNumber("");
      setRegisterNumber("");
    } else {
      setTitel("");
      setTitelNach("");
      setFirstname("");
      setLastname("");
      setBirthDate("");
    }
  }

  function addRow() {
    setMeteringPoints((prev) => [
      ...prev,
      { key: ++mpKeyCounter, meteringPoint: "", direction: "CONSUMPTION", participationFactor: 100, generationType: "pv" },
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
        if (field === "batterySizeKwh") {
          const n = parseFloat(value);
          return { ...mp, batterySizeKwh: isNaN(n) ? undefined : n };
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
        // Firmenbuchnummer ist optional — siehe registration-form.tsx
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

    const payload: MeteringPointRequest[] = meteringPoints.map((mp) => {
      const isProduction = mp.direction === "PRODUCTION";
      const isPv = isProduction && (mp.generationType ?? "pv") === "pv";
      const isConsumption = mp.direction === "CONSUMPTION";
      return {
        meteringPoint: mp.meteringPoint.trim(),
        direction: mp.direction,
        participationFactor: mp.participationFactor,
        // PROJ-45: server normalisiert nochmal — CONSUMPTION ⇒ generation_type
        // wird auf NULL gecleart, non-pv ⇒ battery/inverter NULL.
        generationType: isProduction ? (mp.generationType ?? "pv") : undefined,
        batterySizeKwh: isPv ? mp.batterySizeKwh : undefined,
        inverterManufacturer: isPv ? (mp.inverterManufacturer || undefined) : undefined,
        // PROJ-49: Energie-Felder durchreichen. Sichtbarkeit-Gates analog
        // zum Public-Form, damit Admin-Update keine Member-Eingaben löscht
        // (Backend ersetzt die MP-Reihen vollständig).
        consumptionPreviousYear: isConsumption ? mp.consumptionPreviousYear : undefined,
        consumptionForecast: isConsumption ? mp.consumptionForecast : undefined,
        feedInForecast: isProduction ? mp.feedInForecast : undefined,
        pvPowerKwp: isPv ? mp.pvPowerKwp : undefined,
        feedInLimitPresent: isPv ? mp.feedInLimitPresent : undefined,
        feedInLimitKw: isPv && mp.feedInLimitPresent ? mp.feedInLimitKw : undefined,
        batteryControlAcceptable: isPv ? mp.batteryControlAcceptable : undefined,
      };
    });

    try {
      await updateApplication(application.id, {
        memberType,
        titel: isPerson ? titel.trim() || undefined : undefined,
        titelNach: isPerson ? titelNach.trim() || undefined : undefined,
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
        // PROJ-56: Backend cleart die Felder, wenn die Vollmacht nicht
        // aktiv ist — wir senden sie aber trotzdem so wie der Admin sie
        // gerade im UI sieht.
        networkOperatorCustomerNumber: networkOperatorCustomerNumber.trim() || undefined,
        meterInventoryNumber: meterInventoryNumber.trim() || undefined,
        // PROJ-57: Ansprechperson. Toggle + drei Felder. Backend cleart die
        // Felder serverseitig wenn der Toggle aus ist oder der Mitgliedstyp
        // nicht in der Org-Liste liegt.
        hasContactPerson: isOrgType ? hasContactPerson : undefined,
        contactPersonName: isOrgType && hasContactPerson ? (contactPersonName.trim() || undefined) : undefined,
        contactPersonEmail: isOrgType && hasContactPerson ? (contactPersonEmail.trim() || undefined) : undefined,
        contactPersonPhone: isOrgType && hasContactPerson ? (contactPersonPhone.trim() || undefined) : undefined,
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
          {/* Mitgliedsnummer wird nicht mehr im Onboarding verwaltet —
              sie wird zum Import-Zeitpunkt im Tarif-Dialog vergeben
              (vorausgefüllt mit max+1 aus dem Core, editierbar). */}

          {/* Member type */}
          <div>
            <h3 className="text-sm font-semibold mb-3">Mitgliedstyp</h3>
            <Select value={memberType} onValueChange={(v) => onMemberTypeChange(v as MemberType)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="private">Privatperson</SelectItem>
                <SelectItem value="sole_proprietor">Kleinunternehmer</SelectItem>
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
                  <Label htmlFor="edit-titel">Titel vor</Label>
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
                <div className="space-y-1 col-span-2">
                  <Label htmlFor="edit-titel-nach">Titel nach</Label>
                  <Input
                    id="edit-titel-nach"
                    value={titelNach}
                    onChange={(e) => setTitelNach(e.target.value)}
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
                      {memberType === "association" ? "Vereinsnummer *" : "Firmenbuchnummer"}
                    </Label>
                    <Input
                      id="edit-register"
                      value={registerNumber}
                      onChange={(e) => setRegisterNumber(e.target.value)}
                    />
                  </div>
                )}
                {(memberType === "company" || memberType === "municipality" || memberType === "association") && (
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
                )}
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
                <div key={mp.key} className="space-y-1">
                  <div className="flex gap-2 items-center">
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
                  {mp.direction === "PRODUCTION" && (
                    <div className="ml-1 flex gap-2 items-center text-xs text-muted-foreground">
                      <span>Erzeugung:</span>
                      <Select
                        value={mp.generationType ?? "pv"}
                        onValueChange={(v) => updateRow(mp.key, "generationType", v)}
                      >
                        <SelectTrigger className="w-40 h-8 text-xs">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="pv">PV (Photovoltaik)</SelectItem>
                          <SelectItem value="hydro">Wasser</SelectItem>
                          <SelectItem value="wind">Wind</SelectItem>
                          <SelectItem value="biomass">Biomasse</SelectItem>
                        </SelectContent>
                      </Select>
                      {(mp.generationType ?? "pv") === "pv" && (
                        <>
                          <Input
                            type="number"
                            min={0}
                            step={0.1}
                            value={mp.batterySizeKwh ?? ""}
                            onChange={(e) => updateRow(mp.key, "batterySizeKwh", e.target.value)}
                            className="w-24 h-8 text-xs"
                            title="Größe Batterie (kWh)"
                          />
                          <span>kWh</span>
                          <Input
                            value={mp.inverterManufacturer ?? ""}
                            onChange={(e) => updateRow(mp.key, "inverterManufacturer", e.target.value)}
                            className="w-40 h-8 text-xs"
                            title="Hersteller Wechselrichter"
                          />
                        </>
                      )}
                    </div>
                  )}
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

          {/* PROJ-57: Ansprechperson-Block — nur für Org-Mitgliedstypen.
              Toggle + drei Detail-Felder. Admin kann Toggle umschalten und
              die Werte editieren. */}
          {isOrgType && (
            <>
              <Separator />
              <div className="space-y-3">
                <div className="flex items-center gap-2">
                  <Checkbox
                    id="edit-has-contact-person"
                    checked={hasContactPerson}
                    onCheckedChange={(v) => setHasContactPerson(v === true)}
                  />
                  <Label htmlFor="edit-has-contact-person" className="cursor-pointer">
                    Ansprechperson angeben
                  </Label>
                </div>
                {hasContactPerson && (
                  <div className="space-y-3 pl-6">
                    <div className="space-y-1">
                      <Label htmlFor="edit-contact-person-name">Name *</Label>
                      <Input
                        id="edit-contact-person-name"
                        autoComplete="name"
                        value={contactPersonName}
                        onChange={(e) => setContactPersonName(e.target.value)}
                      />
                    </div>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                      <div className="space-y-1">
                        <Label htmlFor="edit-contact-person-email">E-Mail *</Label>
                        <Input
                          id="edit-contact-person-email"
                          type="email"
                          autoComplete="email"
                          value={contactPersonEmail}
                          onChange={(e) => setContactPersonEmail(e.target.value)}
                        />
                      </div>
                      <div className="space-y-1">
                        <Label htmlFor="edit-contact-person-phone">Telefon *</Label>
                        <Input
                          id="edit-contact-person-phone"
                          type="tel"
                          autoComplete="tel"
                          value={contactPersonPhone}
                          onChange={(e) => setContactPersonPhone(e.target.value)}
                        />
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </>
          )}

          {/* PROJ-56: Netzbetreiber-Info-Felder. Nur sichtbar wenn das
              Mitglied die Vollmacht beim Submit erteilt hat — sonst
              semantisch nicht sinnvoll. Editierbar damit Admin Tippfehler
              korrigieren kann. */}
          {application.networkOperatorAuthorization && (
            <>
              <Separator />
              <div className="space-y-3">
                <h3 className="text-sm font-semibold">Netzbetreiber-Informationen</h3>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <Label htmlFor="edit-network-operator-customer-number">
                      Netzbetreiber Kundennummer
                    </Label>
                    <Input
                      id="edit-network-operator-customer-number"
                      value={networkOperatorCustomerNumber}
                      onChange={(e) => setNetworkOperatorCustomerNumber(e.target.value)}
                    />
                  </div>
                  <div className="space-y-1">
                    <Label htmlFor="edit-meter-inventory-number">
                      Inventarnummer eines Zählers
                    </Label>
                    <Input
                      id="edit-meter-inventory-number"
                      value={meterInventoryNumber}
                      onChange={(e) => setMeterInventoryNumber(e.target.value)}
                    />
                  </div>
                </div>
              </div>
            </>
          )}

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
