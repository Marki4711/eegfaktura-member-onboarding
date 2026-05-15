"use client";

import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useSession } from "next-auth/react";
import { AdminFilterPanel } from "@/components/admin-filter-panel";
import { AdminApplicationTable } from "@/components/admin-application-table";
import { listApplications, deleteDraftApplications, bulkAction } from "@/lib/api";
import type { ApplicationListItem, BulkAction as BulkActionType, SortColumn, SortOrder } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

const BULK_ACTION_LABELS: Record<BulkActionType, string> = {
  approve: "Genehmigen",
  reject: "Ablehnen",
  under_review: "Zur Prüfung",
};

export function ApplicationsPageContent() {
  const searchParams = useSearchParams();
  const { data: session, status } = useSession();
  const [items, setItems] = useState<ApplicationListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [draftCount, setDraftCount] = useState<number | null>(null);
  const [deletingDrafts, setDeletingDrafts] = useState(false);

  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const [pendingAction, setPendingAction] = useState<BulkActionType | null>(null);
  const [rejectReason, setRejectReason] = useState("");
  const [bulkRunning, setBulkRunning] = useState(false);
  const [bulkResult, setBulkResult] = useState<{ succeeded: number; skipped: number } | null>(null);

  const sessionRCNumbers: string[] = (session as unknown as { tenant?: string[] })?.tenant ?? [];
  const rcNumbers = [...new Set([...sessionRCNumbers, ...items.map((i) => i.rcNumber)])].sort();

  const page = parseInt(searchParams.get("page") ?? "1", 10) || 1;
  const pageSize = parseInt(searchParams.get("page_size") ?? "20", 10) || 20;
  const ALLOWED_SORT: ReadonlyArray<SortColumn> = ["referenceNumber", "name", "email", "rcNumber", "status", "submittedAt"];
  const rawSort = searchParams.get("sort") ?? "submittedAt";
  const sort: SortColumn = (ALLOWED_SORT as ReadonlyArray<string>).includes(rawSort) ? (rawSort as SortColumn) : "submittedAt";
  const rawOrder = searchParams.get("order") ?? "desc";
  const order: SortOrder = rawOrder === "asc" ? "asc" : "desc";

  const fetchApplications = useCallback(async (signal?: AbortSignal) => {
    setLoading(true);
    setError(null);
    try {
      const result = await listApplications({
        status: searchParams.get("status") ?? undefined,
        name: searchParams.get("name") ?? searchParams.get("lastname") ?? undefined,
        email: searchParams.get("email") ?? undefined,
        rc_number: searchParams.get("rc_number") ?? undefined,
        submitted_from: searchParams.get("submitted_from") ?? undefined,
        submitted_to: searchParams.get("submitted_to") ?? undefined,
        page,
        page_size: pageSize,
        sort,
        order,
      }, session?.accessToken, signal);
      setItems(result.items);
      setTotal(result.total);
    } catch (err: unknown) {
      // Aborted requests during rapid navigation are expected — swallow.
      if (err instanceof DOMException && err.name === "AbortError") return;
      const msg = err instanceof Error ? err.message : "Fehler beim Laden der Anträge";
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [searchParams, page, pageSize, sort, order, session?.accessToken]);

  const fetchDraftCount = useCallback(async () => {
    if (!session?.accessToken) return;
    try {
      // Respect the active rc_number filter so the dialog shows (and the
      // delete button removes) only drafts in the EEG the admin is looking
      // at. Multi-EEG admins viewing "all" still see the union.
      const rcFilter = searchParams.get("rc_number") ?? undefined;
      const result = await listApplications(
        { status: "draft", page: 1, page_size: 1, rc_number: rcFilter },
        session.accessToken,
      );
      setDraftCount(result.total);
    } catch {
      setDraftCount(null);
    }
  }, [searchParams, session?.accessToken]);

  useEffect(() => {
    if (status === "loading") return;
    // AbortController so rapid filter/sort/page changes don't leave older
    // responses landing after newer ones and repainting stale data.
    const ac = new AbortController();
    fetchApplications(ac.signal);
    fetchDraftCount();
    return () => ac.abort();
  }, [fetchApplications, fetchDraftCount, status]);

  const handleDeleteDrafts = async () => {
    setDeletingDrafts(true);
    try {
      const rcFilter = searchParams.get("rc_number") ?? undefined;
      await deleteDraftApplications(session?.accessToken, rcFilter);
      await fetchApplications();
      await fetchDraftCount();
    } finally {
      setDeletingDrafts(false);
    }
  };

  const handleBulkActionConfirm = async () => {
    if (!pendingAction) return;
    setBulkRunning(true);
    try {
      const result = await bulkAction(
        pendingAction,
        Array.from(selectedIds),
        pendingAction === "reject" ? rejectReason : "",
        session?.accessToken
      );
      setBulkResult({ succeeded: result.succeeded.length, skipped: result.skipped.length });
      setSelectedIds(new Set());
      await fetchApplications();
      await fetchDraftCount();
    } finally {
      setBulkRunning(false);
      setPendingAction(null);
      setRejectReason("");
    }
  };

  const selectedCount = selectedIds.size;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Anträge</h1>
        {draftCount !== null && draftCount > 0 && (
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="outline" size="sm" className="text-destructive border-destructive hover:bg-destructive hover:text-destructive-foreground">
                Alle Entwürfe löschen
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Alle Entwürfe löschen?</AlertDialogTitle>
                <AlertDialogDescription>
                  Es werden <strong>{draftCount} {draftCount === 1 ? "Entwurf" : "Entwürfe"}</strong> unwiderruflich gelöscht.
                  Entwürfe sind nicht eingereichte Anträge, auf die kein Bewerber mehr zugreifen kann.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Abbrechen</AlertDialogCancel>
                <AlertDialogAction
                  onClick={handleDeleteDrafts}
                  disabled={deletingDrafts}
                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                >
                  {deletingDrafts ? "Wird gelöscht…" : `${draftCount} ${draftCount === 1 ? "Entwurf" : "Entwürfe"} löschen`}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}
      </div>

      <AdminFilterPanel rcNumbers={rcNumbers} />

      {/* Bulk action bar */}
      {selectedCount > 0 && (
        <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-muted/50 px-4 py-2">
          <span className="text-sm font-medium text-foreground">
            {selectedCount} {selectedCount === 1 ? "Antrag" : "Anträge"} ausgewählt
          </span>
          <div className="ml-auto flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="outline"
              onClick={() => setPendingAction("under_review")}
            >
              Zur Prüfung
            </Button>
            <Button
              size="sm"
              variant="outline"
              className="text-destructive border-destructive hover:bg-destructive hover:text-destructive-foreground"
              onClick={() => setPendingAction("reject")}
            >
              Ablehnen
            </Button>
            <Button
              size="sm"
              onClick={() => setPendingAction("approve")}
            >
              Genehmigen
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => setSelectedIds(new Set())}
            >
              Auswahl aufheben
            </Button>
          </div>
        </div>
      )}

      {/* Bulk result summary */}
      {bulkResult && (
        <div className="rounded-lg border bg-card px-4 py-3 text-sm flex items-center justify-between">
          <span>
            <strong>{bulkResult.succeeded}</strong> erfolgreich verarbeitet
            {bulkResult.skipped > 0 && (
              <>, <strong>{bulkResult.skipped}</strong> übersprungen (ungültige Transition oder kein Zugriff)</>
            )}
          </span>
          <Button size="sm" variant="ghost" onClick={() => setBulkResult(null)}>
            ✕
          </Button>
        </div>
      )}

      <AdminApplicationTable
        items={items}
        total={total}
        page={page}
        pageSize={pageSize}
        sort={sort}
        order={order}
        loading={loading}
        error={error}
        onRetry={fetchApplications}
        selectedIds={selectedIds}
        onSelectionChange={setSelectedIds}
      />

      {/* Bulk action confirmation dialog */}
      <Dialog open={pendingAction !== null} onOpenChange={(open) => { if (!open) { setPendingAction(null); setRejectReason(""); } }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {pendingAction ? `${BULK_ACTION_LABELS[pendingAction]}: ${selectedCount} ${selectedCount === 1 ? "Antrag" : "Anträge"}` : ""}
            </DialogTitle>
            <DialogDescription>
              {pendingAction === "approve" && `${selectedCount} ${selectedCount === 1 ? "Antrag" : "Anträge"} genehmigen? Anträge mit ungültiger Statusübergang werden übersprungen.`}
              {pendingAction === "reject" && `${selectedCount} ${selectedCount === 1 ? "Antrag" : "Anträge"} ablehnen? Ein Ablehnungsgrund ist erforderlich.`}
              {pendingAction === "under_review" && `${selectedCount} ${selectedCount === 1 ? "Antrag" : "Anträge"} zur Prüfung setzen? Anträge mit ungültiger Statusübergang werden übersprungen.`}
            </DialogDescription>
          </DialogHeader>

          {pendingAction === "reject" && (
            <div className="space-y-2">
              <Label htmlFor="bulk-reject-reason">Ablehnungsgrund</Label>
              <Input
                id="bulk-reject-reason"
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                placeholder="Bitte Grund angeben…"
                autoFocus
              />
            </div>
          )}

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => { setPendingAction(null); setRejectReason(""); }}
              disabled={bulkRunning}
            >
              Abbrechen
            </Button>
            <Button
              onClick={handleBulkActionConfirm}
              disabled={bulkRunning || (pendingAction === "reject" && rejectReason.trim() === "")}
              variant={pendingAction === "reject" ? "destructive" : "default"}
            >
              {bulkRunning ? "Wird verarbeitet…" : pendingAction ? BULK_ACTION_LABELS[pendingAction] : ""}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
