"use client";

import { useEffect, useState } from "react";
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
import { Skeleton } from "@/components/ui/skeleton";
import { FileSpreadsheet } from "lucide-react";
import {
  listDataExportConfigs,
  triggerDataExportJob,
  type DataExportConfigResponse,
} from "@/lib/api";

interface Props {
  rcNumber: string;
  applicationIds: string[];
  open: boolean;
  onClose: () => void;
  onJobStarted: (jobId: string) => void;
}

// Plugin-type → icon. Falls back to FileSpreadsheet (excel covers V1).
const PLUGIN_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  excel: FileSpreadsheet,
};

export function DataExportTriggerDialog({
  rcNumber,
  applicationIds,
  open,
  onClose,
  onJobStarted,
}: Props) {
  const { data: session } = useSession();
  const [configs, setConfigs] = useState<DataExportConfigResponse[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [triggeringId, setTriggeringId] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setConfigs(null);
      setError(null);
      setTriggeringId(null);
      return;
    }
    setLoading(true);
    setError(null);
    listDataExportConfigs(rcNumber, session?.accessToken)
      .then((res) => {
        // hide obsolete configs from the trigger dialog (spec criterion).
        setConfigs(res.configs.filter((c) => !c.isObsolete));
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Konfigurationen konnten nicht geladen werden."))
      .finally(() => setLoading(false));
  }, [open, rcNumber, session?.accessToken]);

  async function handleTrigger(c: DataExportConfigResponse) {
    setTriggeringId(c.id);
    setError(null);
    try {
      const job = await triggerDataExportJob(
        rcNumber,
        { configId: c.id, applicationIds },
        session?.accessToken,
      );
      onJobStarted(job.id);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Job konnte nicht gestartet werden.");
    } finally {
      setTriggeringId(null);
    }
  }

  const count = applicationIds.length;

  return (
    <Dialog open={open} onOpenChange={(o) => { if (!o) onClose(); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Datenweiterleitung</DialogTitle>
          <DialogDescription>
            {count === 1
              ? "1 Antrag wird weitergeleitet. Wähle die Ziel-Konfiguration:"
              : `${count} Anträge werden weitergeleitet. Wähle die Ziel-Konfiguration:`}
          </DialogDescription>
        </DialogHeader>

        {loading && (
          <div className="space-y-2">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        )}

        {!loading && configs && configs.length === 0 && (
          <p className="text-sm text-muted-foreground">
            Noch keine Datenweiterleitungs-Konfigurationen vorhanden. Lege sie in den Einstellungen unter <em>Datenweiterleitung</em> an.
          </p>
        )}

        {!loading && configs && configs.length > 0 && (
          <div className="space-y-2 max-h-80 overflow-y-auto">
            {configs.map((c) => {
              const Icon = PLUGIN_ICONS[c.pluginType] ?? FileSpreadsheet;
              return (
                <button
                  key={c.id}
                  type="button"
                  disabled={triggeringId !== null}
                  onClick={() => handleTrigger(c)}
                  className="flex w-full items-center gap-3 rounded-md border px-3 py-2 text-left text-sm hover:bg-muted/50 disabled:opacity-50"
                >
                  <Icon className="h-5 w-5 text-muted-foreground" />
                  <div className="flex-1">
                    <div className="font-medium">{c.name}</div>
                    <div className="text-xs text-muted-foreground">{c.pluginType}</div>
                  </div>
                  {triggeringId === c.id && (
                    <span className="text-xs text-muted-foreground">Wird gestartet…</span>
                  )}
                </button>
              );
            })}
          </div>
        )}

        {error && <p className="text-sm text-destructive">{error}</p>}

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={triggeringId !== null}>
            Abbrechen
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
