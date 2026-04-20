"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { AdminStatusBadge } from "@/components/admin-status-badge";
import { AdminMeteringPointTable } from "@/components/admin-metering-point-table";
import { AdminStatusLog } from "@/components/admin-status-log";
import { AdminStatusActions } from "@/components/admin-status-actions";
import { AdminNoteEditor } from "@/components/admin-note-editor";
import { AdminEditForm } from "@/components/admin-edit-form";
import { getApplicationDetail, ApiResponseError } from "@/lib/api";
import type { AdminApplicationDetail, MemberType } from "@/lib/api";

const MEMBER_TYPE_LABELS: Record<MemberType, string> = {
  private:      "Privatperson",
  farmer:       "Pauschalierter Landwirt",
  municipality: "Gemeinde / öffentl. Körperschaft",
  company:      "Unternehmen",
  association:  "Verein / Kleinunternehmer",
};

interface Props {
  id: string;
  returnTo: string;
}

function formatDate(iso: string | null) {
  if (!iso) return "—";
  // Parse only the YYYY-MM-DD portion to avoid UTC→local timezone shifts
  // (e.g. "1962-06-06T00:00:00Z" would show as "05.06.1962" in UTC-1).
  const [year, month, day] = iso.slice(0, 10).split("-").map(Number);
  return new Date(year, month - 1, day).toLocaleDateString("de-AT", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });
}

function formatDateTime(iso: string | null) {
  if (!iso) return "—";
  return new Date(iso).toLocaleString("de-AT", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function Field({ label, value }: { label: string; value: string | null | undefined }) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="text-sm mt-0.5">{value || "—"}</dd>
    </div>
  );
}

function BoolField({ label, value }: { label: string; value: boolean }) {
  return (
    <div>
      <dt className="text-xs text-muted-foreground">{label}</dt>
      <dd className="text-sm mt-0.5">{value ? "Ja" : "Nein"}</dd>
    </div>
  );
}

export function AdminApplicationDetail({ id, returnTo }: Props) {
  const router = useRouter();
  const [application, setApplication] = useState<AdminApplicationDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [editOpen, setEditOpen] = useState(false);

  const fetchApplication = useCallback(async () => {
    setLoading(true);
    setError(null);
    setNotFound(false);
    try {
      const data = await getApplicationDetail(id);
      setApplication(data);
    } catch (err: unknown) {
      if (err instanceof ApiResponseError && err.apiError.code === "not_found") {
        setNotFound(true);
      } else {
        const msg = err instanceof Error ? err.message : "Fehler beim Laden des Antrags";
        setError(msg);
      }
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => {
    fetchApplication();
  }, [fetchApplication]);

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-48 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (notFound) {
    return (
      <Card>
        <CardContent className="py-12 text-center space-y-4">
          <p className="text-muted-foreground">
            Dieser Antrag wurde nicht gefunden.
          </p>
          <Button variant="outline" onClick={() => router.push(returnTo)}>
            Zurück zur Liste
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Card>
        <CardContent className="py-12 text-center space-y-4">
          <p className="text-sm text-destructive">{error}</p>
          <Button variant="outline" onClick={fetchApplication}>
            Erneut versuchen
          </Button>
        </CardContent>
      </Card>
    );
  }

  if (!application) return null;

  return (
    <>
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-4 mb-6">
        <div className="space-y-1">
          <Button
            variant="ghost"
            size="sm"
            className="text-muted-foreground -ml-2 mb-1"
            onClick={() => router.push(returnTo)}
          >
            ← Zurück zur Liste
          </Button>
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-semibold">{application.referenceNumber}</h1>
            <AdminStatusBadge status={application.status} />
          </div>
          <p className="text-sm text-muted-foreground">
            {application.memberType === "private" || application.memberType === "farmer"
              ? `${application.firstname ?? ""} ${application.lastname ?? ""}`.trim()
              : (application.companyName ?? "")}
            {" · "}{application.email}
          </p>
        </div>
        <Button onClick={() => setEditOpen(true)}>Bearbeiten</Button>
      </div>

      <div className="space-y-6">
        {/* Status Actions */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Statusaktionen</CardTitle>
          </CardHeader>
          <CardContent>
            <AdminStatusActions
              applicationId={application.id}
              status={application.status}
              onRefresh={fetchApplication}
            />
          </CardContent>
        </Card>

        {/* Member Data */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Mitgliedsdaten</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 sm:grid-cols-3 gap-4 mb-4">
              <Field label="Mitgliedstyp" value={MEMBER_TYPE_LABELS[application.memberType] ?? application.memberType} />
            </dl>
            {(application.memberType === "private" || application.memberType === "farmer") ? (
              <dl className="grid grid-cols-2 sm:grid-cols-3 gap-4">
                <Field label="Vorname" value={application.firstname} />
                <Field label="Nachname" value={application.lastname} />
                <Field label="Geburtsdatum" value={formatDate(application.birthDate)} />
                <Field label="E-Mail" value={application.email} />
                <Field label="Telefon" value={application.phone} />
              </dl>
            ) : (
              <dl className="grid grid-cols-2 sm:grid-cols-3 gap-4">
                <Field label={application.memberType === "municipality" ? "Organisationsname" : "Firmenname"} value={application.companyName} />
                <Field label="UID-Nummer" value={application.uidNumber} />
                {application.memberType === "company" && (
                  <Field label="Firmenbuch-/Vereinsnummer" value={application.registerNumber} />
                )}
                <Field label="E-Mail" value={application.email} />
                <Field label="Telefon" value={application.phone} />
              </dl>
            )}
            <Separator className="my-4" />
            <dl className="grid grid-cols-2 sm:grid-cols-3 gap-4">
              <Field label="Straße" value={application.residentStreet} />
              <Field label="Hausnummer" value={application.residentStreetNumber} />
              <Field label="PLZ" value={application.residentZip} />
              <Field label="Ort" value={application.residentCity} />
            </dl>
          </CardContent>
        </Card>

        {/* Bank account */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Bankverbindung</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 sm:grid-cols-3 gap-4">
              <Field label="IBAN" value={application.iban} />
              <Field label="Kontoinhaber" value={application.accountHolder} />
              <BoolField label="SEPA-Mandat akzeptiert" value={application.sepaMandateAccepted} />
              <Field label="SEPA-Mandat akzeptiert am" value={formatDateTime(application.sepaMandateAcceptedAt)} />
            </dl>
          </CardContent>
        </Card>

        {/* Consent */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Einwilligungen</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 sm:grid-cols-3 gap-4">
              <BoolField label="Datenschutz akzeptiert" value={application.privacyAccepted} />
              <Field label="Datenschutz-Version" value={application.privacyVersion} />
              <Field label="Akzeptiert am" value={formatDateTime(application.privacyAcceptedAt)} />
              <BoolField label="Richtigkeit bestätigt" value={application.accuracyConfirmed} />
            </dl>
          </CardContent>
        </Card>

        {/* Metadata */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Antragsdaten</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 sm:grid-cols-3 gap-4">
              <Field label="Referenznummer" value={application.referenceNumber} />
              <Field label="EEG-ID" value={application.eegId} />
              <Field label="RC-Nummer" value={application.rcNumber} />
              <Field label="Erstellt am" value={formatDateTime(application.createdAt)} />
              <Field label="Angelegt am" value={formatDateTime(application.startedAt)} />
              <Field label="Eingereicht am" value={formatDateTime(application.submittedAt)} />
              <Field label="Genehmigt am" value={formatDateTime(application.approvedAt)} />
              <Field label="Abgelehnt am" value={formatDateTime(application.rejectedAt)} />
            </dl>
            {application.needsInfoReason && (
              <div className="mt-4 p-3 bg-amber-500/10 border border-amber-500/20 rounded-md">
                <p className="text-xs text-muted-foreground mb-1">Anforderung (Info benötigt)</p>
                <p className="text-sm">{application.needsInfoReason}</p>
              </div>
            )}
            {(application.importedAt || application.importErrorMessage) && (
              <>
                <Separator className="my-4" />
                <dl className="grid grid-cols-2 sm:grid-cols-3 gap-4">
                  <Field label="Importiert am" value={formatDateTime(application.importedAt)} />
                  <Field label="Ziel-Teilnehmer-ID" value={application.targetParticipantId} />
                  <Field label="Import gestartet" value={formatDateTime(application.importStartedAt)} />
                  <Field label="Import abgeschlossen" value={formatDateTime(application.importFinishedAt)} />
                </dl>
                {application.importErrorMessage && (
                  <div className="mt-4 p-3 bg-destructive/10 border border-destructive/20 rounded-md">
                    <p className="text-xs text-muted-foreground mb-1">Import-Fehlermeldung</p>
                    <p className="text-sm font-mono break-all">{application.importErrorMessage}</p>
                  </div>
                )}
              </>
            )}
          </CardContent>
        </Card>

        {/* Metering Points */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Zählpunkte</CardTitle>
          </CardHeader>
          <CardContent>
            <AdminMeteringPointTable meteringPoints={application.meteringPoints} />
          </CardContent>
        </Card>

        {/* Admin Note */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Admin-Notiz</CardTitle>
          </CardHeader>
          <CardContent>
            <AdminNoteEditor
              application={application}
              onRefresh={fetchApplication}
            />
          </CardContent>
        </Card>

        {/* Status Log */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Statusverlauf</CardTitle>
          </CardHeader>
          <CardContent>
            <AdminStatusLog entries={application.statusLog} />
          </CardContent>
        </Card>
      </div>

      {editOpen && (
        <AdminEditForm
          open={editOpen}
          application={application}
          onClose={() => setEditOpen(false)}
          onRefresh={fetchApplication}
        />
      )}
    </>
  );
}
