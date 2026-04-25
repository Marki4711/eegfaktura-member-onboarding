"use client";

import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useSession } from "next-auth/react";
import { AdminFilterPanel } from "@/components/admin-filter-panel";
import { AdminApplicationTable } from "@/components/admin-application-table";
import { listApplications, deleteDraftApplications } from "@/lib/api";
import type { ApplicationListItem } from "@/lib/api";
import { Button } from "@/components/ui/button";
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

export function ApplicationsPageContent() {
  const searchParams = useSearchParams();
  const { data: session, status } = useSession();
  const [items, setItems] = useState<ApplicationListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [draftCount, setDraftCount] = useState<number | null>(null);
  const [deletingDrafts, setDeletingDrafts] = useState(false);

  const rcNumbers: string[] = (session as unknown as { tenant?: string[] })?.tenant ?? [];

  const page = parseInt(searchParams.get("page") ?? "1", 10) || 1;
  const pageSize = parseInt(searchParams.get("page_size") ?? "20", 10) || 20;

  const fetchApplications = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await listApplications({
        status: searchParams.get("status") ?? undefined,
        lastname: searchParams.get("lastname") ?? undefined,
        email: searchParams.get("email") ?? undefined,
        rc_number: searchParams.get("rc_number") ?? undefined,
        submitted_from: searchParams.get("submitted_from") ?? undefined,
        submitted_to: searchParams.get("submitted_to") ?? undefined,
        page,
        page_size: pageSize,
      }, session?.accessToken);
      setItems(result.items);
      setTotal(result.total);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Fehler beim Laden der Anträge";
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [searchParams, page, pageSize, session?.accessToken]);

  const fetchDraftCount = useCallback(async () => {
    if (!session?.accessToken) return;
    try {
      const result = await listApplications({ status: "draft", page: 1, page_size: 1 }, session.accessToken);
      setDraftCount(result.total);
    } catch {
      setDraftCount(null);
    }
  }, [session?.accessToken]);

  useEffect(() => {
    if (status === "loading") return;
    fetchApplications();
    fetchDraftCount();
  }, [fetchApplications, fetchDraftCount, status]);

  const handleDeleteDrafts = async () => {
    setDeletingDrafts(true);
    try {
      await deleteDraftApplications(session?.accessToken);
      await fetchApplications();
      await fetchDraftCount();
    } finally {
      setDeletingDrafts(false);
    }
  };

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
      <AdminApplicationTable
        items={items}
        total={total}
        page={page}
        pageSize={pageSize}
        loading={loading}
        error={error}
        onRetry={fetchApplications}
      />
    </div>
  );
}
