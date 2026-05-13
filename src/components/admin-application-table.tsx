"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { ArrowDown, ArrowUp, ArrowUpDown } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { AdminStatusBadge } from "@/components/admin-status-badge";
import type { ApplicationListItem, SortColumn, SortOrder } from "@/lib/api";
import { formatDate } from "@/lib/datetime";

const PAGE_SIZE_OPTIONS = [10, 20, 50];

interface Props {
  items: ApplicationListItem[];
  total: number;
  page: number;
  pageSize: number;
  sort: SortColumn;
  order: SortOrder;
  loading: boolean;
  error: string | null;
  onRetry: () => void;
  selectedIds: Set<string>;
  onSelectionChange: (ids: Set<string>) => void;
}

export function AdminApplicationTable({
  items,
  total,
  page,
  pageSize,
  sort,
  order,
  loading,
  error,
  onRetry,
  selectedIds,
  onSelectionChange,
}: Props) {
  const router = useRouter();
  const searchParams = useSearchParams();

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  function navigateTo(id: string) {
    const current = searchParams.toString();
    const returnTo = current ? `/admin/applications?${current}` : "/admin/applications";
    router.push(`/admin/applications/${id}?returnTo=${encodeURIComponent(returnTo)}`);
  }

  function setPage(p: number) {
    const params = new URLSearchParams(searchParams.toString());
    params.set("page", String(p));
    router.push(`/admin/applications?${params.toString()}`);
  }

  function changePageSize(size: number) {
    const params = new URLSearchParams(searchParams.toString());
    params.set("page_size", String(size));
    params.set("page", "1");
    router.push(`/admin/applications?${params.toString()}`);
  }

  // Click toggles asc<->desc on the active column; click on a new column
  // sets it as the active sort, defaulting to desc (matches the "newest first"
  // intuition for date columns and "Z..A" for text — users override with a
  // second click).
  function changeSort(column: SortColumn) {
    const params = new URLSearchParams(searchParams.toString());
    const nextOrder: SortOrder = sort === column && order === "desc" ? "asc" : "desc";
    params.set("sort", column);
    params.set("order", nextOrder);
    params.set("page", "1");
    router.push(`/admin/applications?${params.toString()}`);
  }

  function SortHeader({ column, label, className }: { column: SortColumn; label: string; className?: string }) {
    const active = sort === column;
    const Icon = !active ? ArrowUpDown : order === "asc" ? ArrowUp : ArrowDown;
    return (
      <TableHead className={className}>
        <button
          type="button"
          onClick={() => changeSort(column)}
          className="inline-flex items-center gap-1 hover:text-foreground transition-colors"
        >
          {label}
          <Icon className={`h-3.5 w-3.5 ${active ? "text-foreground" : "text-muted-foreground/60"}`} />
        </button>
      </TableHead>
    );
  }

  const allVisibleIds = items.map((i) => i.id);
  const allSelected = allVisibleIds.length > 0 && allVisibleIds.every((id) => selectedIds.has(id));
  const someSelected = allVisibleIds.some((id) => selectedIds.has(id));

  function toggleAll() {
    if (allSelected) {
      const next = new Set(selectedIds);
      allVisibleIds.forEach((id) => next.delete(id));
      onSelectionChange(next);
    } else {
      const next = new Set(selectedIds);
      allVisibleIds.forEach((id) => next.add(id));
      onSelectionChange(next);
    }
  }

  function toggleOne(id: string) {
    const next = new Set(selectedIds);
    if (next.has(id)) {
      next.delete(id);
    } else {
      next.add(id);
    }
    onSelectionChange(next);
  }

  if (error) {
    return (
      <div className="bg-card rounded-lg border p-8 text-center space-y-3">
        <p className="text-sm text-destructive">{error}</p>
        <Button variant="outline" onClick={onRetry}>
          Erneut versuchen
        </Button>
      </div>
    );
  }

  return (
    <div className="bg-card rounded-lg border overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-10">
              <Checkbox
                checked={allSelected ? true : someSelected ? "indeterminate" : false}
                onCheckedChange={toggleAll}
                aria-label="Alle auswählen"
                disabled={loading || items.length === 0}
              />
            </TableHead>
            <SortHeader column="referenceNumber" label="Referenz" />
            <SortHeader column="name" label="Name" />
            <SortHeader column="email" label="E-Mail" />
            <SortHeader column="rcNumber" label="EEG" />
            <SortHeader column="status" label="Status" />
            <SortHeader column="submittedAt" label="Eingereicht am" />
          </TableRow>
        </TableHeader>
        <TableBody>
          {loading ? (
            Array.from({ length: 5 }).map((_, i) => (
              <TableRow key={i}>
                {Array.from({ length: 7 }).map((_, j) => (
                  <TableCell key={j}>
                    <Skeleton className="h-4 w-full" />
                  </TableCell>
                ))}
              </TableRow>
            ))
          ) : items.length === 0 ? (
            <TableRow>
              <TableCell colSpan={7} className="text-center py-12 text-muted-foreground">
                Keine Anträge gefunden. Passen Sie die Filter an.
              </TableCell>
            </TableRow>
          ) : (
            items.map((item) => (
              <TableRow
                key={item.id}
                className={`cursor-pointer hover:bg-muted/50 ${selectedIds.has(item.id) ? "bg-muted/30" : ""}`}
              >
                <TableCell
                  className="w-10"
                  onClick={(e) => { e.stopPropagation(); toggleOne(item.id); }}
                >
                  <Checkbox
                    checked={selectedIds.has(item.id)}
                    onCheckedChange={() => toggleOne(item.id)}
                    aria-label={`Antrag ${item.referenceNumber} auswählen`}
                  />
                </TableCell>
                <TableCell className="font-mono text-sm" onClick={() => navigateTo(item.id)}>
                  {item.referenceNumber}
                </TableCell>
                <TableCell onClick={() => navigateTo(item.id)}>
                  {item.memberType === "private" || item.memberType === "farmer"
                    ? `${item.firstname ?? ""} ${item.lastname ?? ""}`.trim()
                    : (item.companyName ?? "")}
                </TableCell>
                <TableCell className="text-sm" onClick={() => navigateTo(item.id)}>{item.email}</TableCell>
                <TableCell className="text-sm text-muted-foreground font-mono" onClick={() => navigateTo(item.id)}>{item.rcNumber}</TableCell>
                <TableCell onClick={() => navigateTo(item.id)}>
                  <AdminStatusBadge status={item.status} />
                </TableCell>
                <TableCell className="text-sm text-muted-foreground" onClick={() => navigateTo(item.id)}>
                  {formatDate(item.submittedAt)}
                </TableCell>
              </TableRow>
            ))
          )}
        </TableBody>
      </Table>

      {!loading && !error && (
        <div className="flex flex-wrap items-center justify-between gap-3 px-4 py-3 border-t text-sm text-muted-foreground">
          <div className="flex items-center gap-2">
            <span>Einträge pro Seite:</span>
            <Select
              value={String(pageSize)}
              onValueChange={(v) => changePageSize(Number(v))}
            >
              <SelectTrigger className="h-7 w-16 text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {PAGE_SIZE_OPTIONS.map((n) => (
                  <SelectItem key={n} value={String(n)}>
                    {n}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <span>
              {total === 0
                ? "— Keine Einträge"
                : `— ${total} gesamt, Seite ${page} von ${totalPages}`}
            </span>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page <= 1}
              onClick={() => setPage(page - 1)}
            >
              Zurück
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
            >
              Weiter
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
