"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Separator } from "@/components/ui/separator";
import { AdminStatusBadge } from "@/components/admin-status-badge";
import { AdminMeteringPointTable } from "@/components/admin-metering-point-table";
import { AdminStatusLog } from "@/components/admin-status-log";
import { AdminStatusActions } from "@/components/admin-status-actions";
import { AdminImportUnstuckBanner } from "@/components/admin-import-unstuck-banner";
import { AdminNoteEditor } from "@/components/admin-note-editor";
import { AdminEditForm } from "@/components/admin-edit-form";
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
import { Alert, AlertDescription } from "@/components/ui/alert";
import { getApplicationDetail, resendMemberConfirmation, resendEmailConfirmation, deleteApplication, downloadApplicationExcel, downloadApprovalPDF, ApiResponseError } from "@/lib/api";
import type { AdminApplicationDetail, MemberType, DocumentConsentView } from "@/lib/api";
import { formatPlainDate as formatDate, formatDateTime } from "@/lib/datetime";

const EINZUGSART_LABELS: Record<string, string> = {
  core:      "Core (Standard)",
  b2b:       "B2B",
  kein_sepa: "Kein SEPA",
};

const MEMBER_TYPE_LABELS: Record<MemberType, string> = {
  private:         "Privatperson",
  sole_proprietor: "Kleinunternehmer",
  farmer:          "Pauschalierter Landwirt",
  municipality:    "Gemeinde / öffentl. Körperschaft",
  company:         "Unternehmen",
  association:     "Verein",
};

interface Props {
  id: string;
  returnTo: string;
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
  const { data: session } = useSession();
  const [application, setApplication] = useState<AdminApplicationDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [resending, setResending] = useState(false);
  const [resendResult, setResendResult] = useState<"ok" | "error" | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [downloadingExcel, setDownloadingExcel] = useState(false);
  const [excelError, setExcelError] = useState<string | null>(null);
  const [downloadingPDF, setDownloadingPDF] = useState(false);
  const [pdfError, setPdfError] = useState<string | null>(null);

  const handleExcelDownload = async () => {
    if (!application) return;
    setDownloadingExcel(true);
    setExcelError(null);
    try {
      const { blob, filename } = await downloadApplicationExcel(application.id, session?.accessToken);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch {
      setExcelError("Excel-Download fehlgeschlagen.");
    } finally {
      setDownloadingExcel(false);
    }
  };

  const handlePDFDownload = async () => {
    if (!application) return;
    setDownloadingPDF(true);
    setPdfError(null);
    try {
      const { blob, filename } = await downloadApprovalPDF(application.id, session?.accessToken);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
    } catch {
      setPdfError("PDF-Download fehlgeschlagen.");
    } finally {
      setDownloadingPDF(false);
    }
  };

  const handleDelete = async () => {
    if (!application) return;
    setDeleting(true);
    try {
      await deleteApplication(application.id, session?.accessToken);
      router.push(returnTo);
    } catch {
      setDeleting(false);
    }
  };

  const handleResend = async () => {
    if (!application) return;
    setResending(true);
    setResendResult(null);
    try {
      await resendMemberConfirmation(application.id, session?.accessToken);
      setResendResult("ok");
    } catch {
      setResendResult("error");
    } finally {
      setResending(false);
    }
  };

  const handleResendEmailConfirmation = async () => {
    if (!application) return;
    setResending(true);
    setResendResult(null);
    try {
      await resendEmailConfirmation(application.id, session?.accessToken);
      setResendResult("ok");
    } catch (err) {
      setResendResult("error");
      if (err instanceof ApiResponseError && err.apiError.message) {
        setError(err.apiError.message);
      }
    } finally {
      setResending(false);
    }
  };

  // PROJ-31: backend sets emailConfirmationPending=true when a confirmation
  // token is active and unconsumed. In that state we show a dedicated
  // resend button that ROTATES the token and suppress the generic
  // "Bestätigung erneut senden" button to avoid confusion.
  const isPendingEmailConfirmation = application?.emailConfirmationPending === true;

  const fetchApplication = useCallback(async (signal?: AbortSignal) => {
    setLoading(true);
    setError(null);
    setNotFound(false);
    try {
      const data = await getApplicationDetail(id, session?.accessToken, signal);
      setApplication(data);
    } catch (err: unknown) {
      if (err instanceof DOMException && err.name === "AbortError") return;
      if (err instanceof ApiResponseError && err.apiError.code === "not_found") {
        setNotFound(true);
      } else {
        const msg = err instanceof Error ? err.message : "Fehler beim Laden des Antrags";
        setError(msg);
      }
    } finally {
      setLoading(false);
    }
  }, [id, session?.accessToken]);

  useEffect(() => {
    // Cancel an in-flight detail fetch if the admin navigates to a different
    // application before the previous one returned.
    const ac = new AbortController();
    fetchApplication(ac.signal);
    return () => ac.abort();
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
          <Button variant="outline" onClick={() => fetchApplication()}>
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
        <div className="flex items-center gap-2">
          {resendResult === "ok" && (
            <span className="text-xs text-green-600">E-Mail gesendet</span>
          )}
          {resendResult === "error" && (
            <span className="text-xs text-destructive">Fehler beim Senden</span>
          )}
          {excelError && (
            <span className="text-xs text-destructive">{excelError}</span>
          )}
          {pdfError && (
            <span className="text-xs text-destructive">{pdfError}</span>
          )}
          {isPendingEmailConfirmation ? (
            <Button variant="outline" size="sm" onClick={handleResendEmailConfirmation} disabled={resending}>
              {resending ? "Wird gesendet…" : "Bestätigungs-Link erneut senden"}
            </Button>
          ) : (
            <Button variant="outline" size="sm" onClick={handleResend} disabled={resending}>
              {resending ? "Wird gesendet…" : "Bestätigung erneut senden"}
            </Button>
          )}
          {(application.status === "approved" || application.status === "imported" || application.status === "import_failed") && (
            <Button variant="outline" size="sm" onClick={handleExcelDownload} disabled={downloadingExcel}>
              {downloadingExcel ? "Wird erstellt…" : "Excel herunterladen"}
            </Button>
          )}
          {(application.status === "approved" || application.status === "imported" || application.status === "import_failed") && (
            <Button variant="outline" size="sm" onClick={handlePDFDownload} disabled={downloadingPDF}>
              {downloadingPDF ? "Wird erstellt…" : "Beitrittsbestätigung herunterladen"}
            </Button>
          )}
          {(application.status === "draft" || application.status === "rejected") && (
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="destructive" size="sm" disabled={deleting}>
                  Löschen
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Antrag löschen?</AlertDialogTitle>
                  <AlertDialogDescription>
                    Der Antrag <strong>{application.referenceNumber}</strong> wird unwiderruflich gelöscht.
                    Diese Aktion kann nicht rückgängig gemacht werden.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Abbrechen</AlertDialogCancel>
                  <AlertDialogAction onClick={handleDelete} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
                    Endgültig löschen
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          )}
          <Button onClick={() => setEditOpen(true)}>Bearbeiten</Button>
        </div>
      </div>

      <div className="space-y-6">
        {/* E-Mail confirmation banner (PROJ-31) */}
        {isPendingEmailConfirmation && (
          <Alert className="border-orange-300 bg-orange-50 text-orange-900">
            <AlertDescription>
              <strong>E-Mail-Adresse noch nicht bestätigt.</strong>{" "}
              Das Mitglied hat den Bestätigungs-Link noch nicht angeklickt. Der Antrag kann
              erst nach Bestätigung weiterbearbeitet werden — bis dahin ist nur „Ablehnen"
              als Status-Aktion möglich.
              Wenn die Bestätigungs-Mail im Spam-Ordner gelandet ist, schicken Sie den Link
              über „Bestätigungs-Link erneut senden" oben rechts neu.
            </AlertDescription>
          </Alert>
        )}
        {application.status === "email_confirmed" && (
          <Alert className="border-teal-300 bg-teal-50 text-teal-900">
            <AlertDescription>
              <strong>E-Mail-Adresse bestätigt</strong>
              {application.emailConfirmedAt && (
                <> am {formatDateTime(application.emailConfirmedAt)}</>
              )}.{" "}
              Der Antrag liegt jetzt zur Prüfung bereit.
            </AlertDescription>
          </Alert>
        )}

        {/* Status Actions */}
        {application.importStuck && (
          <AdminImportUnstuckBanner
            applicationId={application.id}
            importStartedAt={application.importStartedAt}
            targetParticipantId={application.targetParticipantId}
            onRefresh={fetchApplication}
          />
        )}

        <Card>
          <CardHeader>
            <CardTitle className="text-base">Statusaktionen</CardTitle>
          </CardHeader>
          <CardContent>
            <AdminStatusActions
              applicationId={application.id}
              rcNumber={application.rcNumber}
              status={application.status}
              targetParticipantId={application.targetParticipantId}
              importErrorMessage={application.importErrorMessage}
              meteringPoints={application.meteringPoints}
              onRefresh={fetchApplication}
              emailConfirmationPending={isPendingEmailConfirmation}
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
              <Field label="Mitgliedsnummer" value={application.memberNumber != null ? String(application.memberNumber) : null} />
              <Field label="Mitgliedstyp" value={MEMBER_TYPE_LABELS[application.memberType] ?? application.memberType} />
            </dl>
            {(application.memberType === "private" || application.memberType === "farmer") ? (
              <dl className="grid grid-cols-2 sm:grid-cols-3 gap-4">
                {application.titel && <Field label="Titel" value={application.titel} />}
                <Field label="Vorname" value={application.firstname} />
                <Field label="Nachname" value={application.lastname} />
                <Field label="Geburtsdatum" value={formatDate(application.birthDate)} />
                <Field label="E-Mail" value={application.email} />
                <Field label="Telefon" value={application.phone} />
              </dl>
            ) : (
              <dl className="grid grid-cols-2 sm:grid-cols-3 gap-4">
                <Field
                  label={
                    application.memberType === "municipality"
                      ? "Organisationsname"
                      : application.memberType === "association"
                      ? "Vereinsname"
                      : "Firmenname"
                  }
                  value={application.companyName}
                />
                {application.memberType !== "sole_proprietor" && (
                  <Field label="UID-Nummer" value={application.uidNumber} />
                )}
                {(application.memberType === "company" || application.memberType === "association") && (
                  <Field
                    label={application.memberType === "association" ? "Vereinsnummer" : "Firmenbuch-/Vereinsnummer"}
                    value={application.registerNumber}
                  />
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
              <Field label="Kontoinhaber:in" value={application.accountHolder} />
              <BoolField label="SEPA-Mandat akzeptiert" value={application.sepaMandateAccepted} />
              <Field label="SEPA-Mandat akzeptiert am" value={formatDateTime(application.sepaMandateAcceptedAt)} />
            </dl>
            <Separator className="my-4" />
            <dl className="grid grid-cols-2 sm:grid-cols-3 gap-4">
              <Field label="Einzugsart" value={EINZUGSART_LABELS[application.einzugsart] ?? application.einzugsart} />
              {application.einzugsart !== "kein_sepa" && (
                <>
                  <Field label="Bankverbindung (Admin)" value={application.bankName} />
                  <Field label="Mandatsreferenz" value={application.mandateReference} />
                  <Field label="Mandatsdatum" value={formatDate(application.mandateDate ?? null)} />
                </>
              )}
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
            {application.consents && application.consents.length > 0 && (
              <>
                <Separator className="my-4" />
                <p className="text-xs text-muted-foreground mb-3">Dokument-Einwilligungen</p>
                <div className="space-y-2">
                  {(application.consents as DocumentConsentView[]).map((c) => (
                    <div key={c.id} className="flex items-start gap-3 text-sm">
                      <div className="flex-1">
                        <a href={c.url} target="_blank" rel="noopener noreferrer" className="underline hover:text-foreground">
                          {c.title}
                        </a>
                        {c.isCentralPolicy && (
                          <span className="ml-2 text-xs text-muted-foreground">(Zentrale Datenschutzerklärung)</span>
                        )}
                      </div>
                      <span className="text-xs text-muted-foreground whitespace-nowrap">{formatDateTime(c.consentedAt)}</span>
                    </div>
                  ))}
                </div>
              </>
            )}
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
