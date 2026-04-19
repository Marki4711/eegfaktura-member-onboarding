"use client";

import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { AdminFilterPanel } from "@/components/admin-filter-panel";
import { AdminApplicationTable } from "@/components/admin-application-table";
import { listApplications } from "@/lib/api";
import type { ApplicationListItem } from "@/lib/api";

export function ApplicationsPageContent() {
  const searchParams = useSearchParams();
  const [items, setItems] = useState<ApplicationListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

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
        metering_point: searchParams.get("metering_point") ?? undefined,
        submitted_from: searchParams.get("submitted_from") ?? undefined,
        submitted_to: searchParams.get("submitted_to") ?? undefined,
        page,
        page_size: pageSize,
      });
      setItems(result.items);
      setTotal(result.total);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Fehler beim Laden der Anträge";
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [searchParams, page, pageSize]);

  useEffect(() => {
    fetchApplications();
  }, [fetchApplications]);

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold text-gray-900">Anträge</h1>
      <AdminFilterPanel />
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
