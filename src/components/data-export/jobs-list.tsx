"use client";

import { useCallback, useEffect, useState } from "react";
import { useSession } from "next-auth/react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { AlertTriangle, Download, RotateCw } from "lucide-react";
import {
  downloadDataExportResult,
  listDataExportJobs,
  retryDataExportJob,
  type DataExportJobResponse,
  type DataExportJobStatus,
} from "@/lib/api";
import { formatDate } from "@/lib/datetime";
import { formatValidationError } from "./error-utils";

const STATUS_LABELS: Record<DataExportJobStatus, string> = {
  queued: "In Warteschlange",
  running: "Läuft",
  done: "Fertig",
  failed: "Fehlgeschlagen",
  expired: "Abgelaufen",
};

const STATUS_VARIANTS: Record<DataExportJobStatus, "default" | "secondary" | "destructive" | "outline"> = {
  queued: "secondary",
  running: "secondary",
  done: "default",
  failed: "destructive",
  expired: "outline",
};

interface Props {
  rcNumber: string;
  // When the user retries a job, we surface the new job-ID so a polling modal
  // can be opened in the parent.
  onTrackJob?: (jobId: string) => void;
  // Bump this number to force a reload (e.g. after a new job was triggered
  // elsewhere on the page).
  reloadKey?: number;
}

export function DataExportJobsList({ rcNumber, onTrackJob, reloadKey }: Props) {
  const { data: session } = useSession();
  const [jobs, setJobs] = useState<DataExportJobResponse[] | null>(null);
  const [failedCount, setFailedCount] = useState(0);
  const [nextCursor, setNextCursor] = useState<string | undefined>();
  const [loadingMore, setLoadingMore] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<"" | DataExportJobStatus>("");
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = useCallback(
    async (append: boolean, cursor?: string) => {
      if (append) setLoadingMore(true);
      else setLoading(true);
      setError(null);
      try {
        const res = await listDataExportJobs(
          rcNumber,
          { status: statusFilter || undefined, cursor, limit: 50 },
          session?.accessToken,
        );
        setFailedCount(res.failedLast7Days);
        setNextCursor(res.nextCursor);
        if (append && jobs) setJobs([...jobs, ...res.jobs]);
        else setJobs(res.jobs);
      } catch (err) {
        setError(formatValidationError(err).join(" — "));
      } finally {
        setLoading(false);
        setLoadingMore(false);
      }
    },
    [rcNumber, session?.accessToken, statusFilter, jobs],
  );

  useEffect(() => {
    void load(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rcNumber, session?.accessToken, statusFilter, reloadKey]);

  async function handleDownload(job: DataExportJobResponse) {
    setBusyId(job.id);
    try {
      const { blob, filename } = await downloadDataExportResult(rcNumber, job.id, session?.accessToken);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(formatValidationError(err).join(" — "));
    } finally {
      setBusyId(null);
    }
  }

  async function handleRetry(job: DataExportJobResponse) {
    setBusyId(job.id);
    try {
      const newJob = await retryDataExportJob(rcNumber, job.id, session?.accessToken);
      onTrackJob?.(newJob.id);
      await load(false);
    } catch (err) {
      setError(formatValidationError(err).join(" — "));
    } finally {
      setBusyId(null);
    }
  }

  if (loading) return <Skeleton className="h-48 w-full" />;
  if (error) return <p className="text-sm text-destructive">{error}</p>;

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        {failedCount > 0 && (
          <Badge variant="destructive" className="flex items-center gap-1">
            <AlertTriangle className="h-3 w-3" />
            {failedCount} fehlgeschlagene{" "}
            {failedCount === 1 ? "Job" : "Jobs"} in den letzten 7 Tagen
          </Badge>
        )}
        <div className="ml-auto flex items-center gap-2">
          <span className="text-sm text-muted-foreground">Status:</span>
          <Select
            value={statusFilter || "all"}
            onValueChange={(v) => setStatusFilter(v === "all" ? "" : (v as DataExportJobStatus))}
          >
            <SelectTrigger className="w-44">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Alle</SelectItem>
              <SelectItem value="queued">In Warteschlange</SelectItem>
              <SelectItem value="running">Läuft</SelectItem>
              <SelectItem value="done">Fertig</SelectItem>
              <SelectItem value="failed">Fehlgeschlagen</SelectItem>
              <SelectItem value="expired">Abgelaufen</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {(jobs ?? []).length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center text-sm text-muted-foreground">
            Keine Jobs vorhanden.
          </CardContent>
        </Card>
      ) : (
        <div className="rounded border overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Erstellt</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Plugin</TableHead>
                <TableHead className="text-right">Anträge</TableHead>
                <TableHead>Ergebnis</TableHead>
                <TableHead className="text-right">Aktionen</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(jobs ?? []).map((j) => (
                <TableRow key={j.id}>
                  <TableCell className="whitespace-nowrap text-xs">
                    {formatDate(j.createdAt)}
                  </TableCell>
                  <TableCell>
                    <Badge variant={STATUS_VARIANTS[j.status]}>
                      {STATUS_LABELS[j.status]}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-xs">{j.pluginType}</TableCell>
                  <TableCell className="text-right text-xs">
                    {j.processedCount}/{j.totalCount}
                  </TableCell>
                  <TableCell className="text-xs">
                    {j.status === "done" && j.resultFileName && (
                      <span>
                        {j.resultFileName}
                        {typeof j.resultFileSize === "number" && (
                          <> ({Math.ceil(j.resultFileSize / 1024)} KB)</>
                        )}
                      </span>
                    )}
                    {j.status === "failed" && j.errorMessage && (
                      <span className="text-destructive">{j.errorMessage}</span>
                    )}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-1">
                      {j.status === "done" && j.hasResult && (
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => handleDownload(j)}
                          disabled={busyId === j.id}
                          aria-label="Datei herunterladen"
                        >
                          <Download className="h-4 w-4" />
                        </Button>
                      )}
                      {(j.status === "failed" || j.status === "expired") && (
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => handleRetry(j)}
                          disabled={busyId === j.id}
                          aria-label="Erneut ausführen"
                        >
                          <RotateCw className="h-4 w-4" />
                        </Button>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {nextCursor && (
        <div className="flex justify-center">
          <Button
            variant="outline"
            size="sm"
            onClick={() => load(true, nextCursor)}
            disabled={loadingMore}
          >
            {loadingMore ? "Lädt…" : "Mehr laden"}
          </Button>
        </div>
      )}
    </div>
  );
}
