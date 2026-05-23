"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Progress } from "@/components/ui/progress";
import { CheckCircle2, Download, Loader2, XCircle } from "lucide-react";
import {
  downloadDataExportResult,
  getDataExportJob,
  retryDataExportJob,
  type DataExportJobResponse,
} from "@/lib/api";

interface Props {
  rcNumber: string;
  jobId: string | null;
  onClose: () => void;
}

const POLL_FAST_MS = 2000;
const POLL_SLOW_MS = 5000;

export function DataExportJobStatusModal({ rcNumber, jobId, onClose }: Props) {
  const { data: session } = useSession();
  const [job, setJob] = useState<DataExportJobResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [downloading, setDownloading] = useState(false);
  const [retrying, setRetrying] = useState(false);
  const activeJobIdRef = useRef<string | null>(null);

  const fetchJob = useCallback(
    async (id: string, signal: AbortSignal) => {
      try {
        const j = await getDataExportJob(rcNumber, id, session?.accessToken, signal);
        setJob(j);
        setError(null);
        return j;
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") return null;
        setError(err instanceof Error ? err.message : "Status konnte nicht geladen werden.");
        return null;
      }
    },
    [rcNumber, session?.accessToken],
  );

  useEffect(() => {
    activeJobIdRef.current = jobId;
    if (!jobId) {
      setJob(null);
      setError(null);
      return;
    }
    const ac = new AbortController();
    let timeout: number | null = null;

    const poll = async () => {
      if (activeJobIdRef.current !== jobId) return;
      const fetched = await fetchJob(jobId, ac.signal);
      if (activeJobIdRef.current !== jobId) return;
      if (!fetched) return;
      if (fetched.status === "queued" || fetched.status === "running") {
        const interval = fetched.totalCount && fetched.totalCount < 100 ? POLL_FAST_MS : POLL_SLOW_MS;
        timeout = window.setTimeout(poll, interval);
      }
    };

    void poll();

    return () => {
      ac.abort();
      if (timeout !== null) window.clearTimeout(timeout);
    };
  }, [jobId, fetchJob]);

  async function handleDownload() {
    if (!job) return;
    setDownloading(true);
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
      setError(err instanceof Error ? err.message : "Download fehlgeschlagen");
    } finally {
      setDownloading(false);
    }
  }

  async function handleRetry() {
    if (!job) return;
    setRetrying(true);
    try {
      const newJob = await retryDataExportJob(rcNumber, job.id, session?.accessToken);
      activeJobIdRef.current = newJob.id;
      setJob(newJob);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Retry fehlgeschlagen");
    } finally {
      setRetrying(false);
    }
  }

  const open = jobId !== null;
  const status = job?.status;
  const progressPct = job && job.totalCount > 0 ? Math.min(100, Math.round((job.processedCount / job.totalCount) * 100)) : 0;

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) onClose();
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            {(!status || status === "queued" || status === "running") && (
              <Loader2 className="h-5 w-5 animate-spin" />
            )}
            {status === "done" && <CheckCircle2 className="h-5 w-5 text-green-600" />}
            {(status === "failed" || status === "expired") && <XCircle className="h-5 w-5 text-destructive" />}
            Datenweiterleitung
          </DialogTitle>
          <DialogDescription>
            {status === "queued" && "Job in der Warteschlange — wird gleich verarbeitet…"}
            {status === "running" &&
              (job?.totalCount
                ? `Wird verarbeitet: ${job.processedCount} von ${job.totalCount}`
                : "Wird verarbeitet…")}
            {status === "done" && "Job erfolgreich abgeschlossen."}
            {status === "failed" && "Job ist fehlgeschlagen."}
            {status === "expired" && "Datei-Download ist abgelaufen. Bitte erneut ausführen."}
            {!status && "Status wird geladen…"}
          </DialogDescription>
        </DialogHeader>

        {(status === "queued" || status === "running") && job && (
          <Progress value={progressPct} className="my-2" />
        )}

        {status === "done" && job?.resultSummary && (
          <div className="rounded border bg-muted/40 px-3 py-2 text-sm">
            {Object.entries(job.resultSummary).map(([k, v]) => (
              <div key={k} className="flex justify-between">
                <span className="text-muted-foreground">{k}:</span>
                <span className="font-medium">{String(v)}</span>
              </div>
            ))}
          </div>
        )}

        {status === "failed" && job?.errorMessage && (
          <div className="rounded border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
            {job.errorMessage}
          </div>
        )}

        {error && <p className="text-sm text-destructive">{error}</p>}

        <DialogFooter className="gap-2 sm:gap-2">
          {status === "done" && job?.hasResult && (
            <Button onClick={handleDownload} disabled={downloading}>
              <Download className="mr-1 h-4 w-4" />
              {downloading ? "Lädt…" : `${job.resultFileName ?? "Datei"} herunterladen`}
            </Button>
          )}
          {status === "failed" && (
            <Button onClick={handleRetry} disabled={retrying} variant="default">
              {retrying ? "Wird neu gestartet…" : "Erneut ausführen"}
            </Button>
          )}
          <Button variant="outline" onClick={onClose}>
            {status === "queued" || status === "running"
              ? "Im Hintergrund weiterlaufen lassen"
              : "Schließen"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
