"use client";

import { useCallback, useEffect, useState } from "react";
import { useSession } from "next-auth/react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Plus } from "lucide-react";
import {
  deleteDataExportConfig,
  listDataExportConfigs,
  listDataExportPlugins,
  type DataExportConfigResponse,
  type DataExportPluginInfo,
  type DataExportStandardConfigInfo,
} from "@/lib/api";
import { DataExportExcelEditor } from "./excel-editor";

interface Props {
  rcNumber: string;
}

type EditorState =
  | { mode: "closed" }
  | { mode: "create"; pluginType: string; template: DataExportStandardConfigInfo | null }
  | { mode: "edit"; config: DataExportConfigResponse };

export function DataExportConfigsList({ rcNumber }: Props) {
  const { data: session } = useSession();
  const [configs, setConfigs] = useState<DataExportConfigResponse[] | null>(null);
  const [plugins, setPlugins] = useState<DataExportPluginInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editor, setEditor] = useState<EditorState>({ mode: "closed" });
  const [deleteTarget, setDeleteTarget] = useState<DataExportConfigResponse | null>(null);
  const [deleting, setDeleting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [c, p] = await Promise.all([
        listDataExportConfigs(rcNumber, session?.accessToken),
        listDataExportPlugins(session?.accessToken),
      ]);
      setConfigs(c.configs);
      setPlugins(p.plugins);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Konfigurationen konnten nicht geladen werden.");
    } finally {
      setLoading(false);
    }
  }, [rcNumber, session?.accessToken]);

  useEffect(() => {
    if (!session?.accessToken) return;
    void load();
  }, [load, session?.accessToken]);

  async function handleDelete() {
    if (!deleteTarget) return;
    setDeleting(true);
    try {
      await deleteDataExportConfig(rcNumber, deleteTarget.id, session?.accessToken);
      setDeleteTarget(null);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Löschen fehlgeschlagen");
    } finally {
      setDeleting(false);
    }
  }

  if (loading) return <Skeleton className="h-48 w-full" />;
  if (error) return <p className="text-sm text-destructive">{error}</p>;

  const grouped = new Map<string, DataExportConfigResponse[]>();
  for (const c of configs ?? []) {
    const arr = grouped.get(c.pluginType) ?? [];
    arr.push(c);
    grouped.set(c.pluginType, arr);
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button size="sm">
              <Plus className="mr-1 h-4 w-4" />
              Neue Konfiguration
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-72">
            {plugins.map((p) => (
              <div key={p.type}>
                <DropdownMenuLabel>{p.displayName}</DropdownMenuLabel>
                <DropdownMenuItem
                  onClick={() => setEditor({ mode: "create", pluginType: p.type, template: null })}
                >
                  Leere Konfiguration
                </DropdownMenuItem>
                {p.standardConfigs.length > 0 && (
                  <>
                    <DropdownMenuLabel className="text-xs text-muted-foreground">
                      Aus Vorlage
                    </DropdownMenuLabel>
                    {p.standardConfigs.map((t) => (
                      <DropdownMenuItem
                        key={t.name}
                        onClick={() => setEditor({ mode: "create", pluginType: p.type, template: t })}
                      >
                        {t.name}
                      </DropdownMenuItem>
                    ))}
                  </>
                )}
                <DropdownMenuSeparator />
              </div>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {(configs ?? []).length === 0 && (
        <Card>
          <CardContent className="py-8 text-center text-sm text-muted-foreground">
            Noch keine Datenweiterleitungs-Konfigurationen angelegt.
          </CardContent>
        </Card>
      )}

      {Array.from(grouped.entries()).map(([pluginType, list]) => {
        const plugin = plugins.find((p) => p.type === pluginType);
        return (
          <div key={pluginType} className="space-y-2">
            <h3 className="text-base font-semibold">{plugin?.displayName ?? pluginType}</h3>
            <div className="grid gap-3 sm:grid-cols-2">
              {list.map((c) => (
                <Card key={c.id} className={c.isObsolete ? "opacity-60" : undefined}>
                  <CardHeader className="pb-2">
                    <CardTitle className="flex items-center justify-between text-base">
                      <span>{c.name}</span>
                      {c.isObsolete && <Badge variant="outline">Obsolet</Badge>}
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-2">
                    {c.isObsolete && (
                      <p className="text-xs text-muted-foreground">
                        Das Plugin {c.pluginType} ist im System nicht mehr verfügbar. Konfiguration kann nur noch gelöscht werden.
                      </p>
                    )}
                    <div className="flex justify-end gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        disabled={c.isObsolete}
                        onClick={() => setEditor({ mode: "edit", config: c })}
                      >
                        Bearbeiten
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        className="text-destructive border-destructive hover:bg-destructive hover:text-destructive-foreground"
                        onClick={() => setDeleteTarget(c)}
                      >
                        Löschen
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </div>
        );
      })}

      {/* Editor dialog — only excel-plugin supported in V1. */}
      <Dialog
        open={editor.mode !== "closed"}
        onOpenChange={(open) => {
          if (!open) setEditor({ mode: "closed" });
        }}
      >
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {editor.mode === "edit"
                ? `Bearbeiten: ${editor.config.name}`
                : editor.mode === "create"
                ? editor.template
                  ? `Neue Konfiguration aus Vorlage „${editor.template.name}"`
                  : "Neue Konfiguration"
                : ""}
            </DialogTitle>
          </DialogHeader>
          {editor.mode === "create" && editor.pluginType === "excel" && (
            <DataExportExcelEditor
              rcNumber={rcNumber}
              template={editor.template}
              onSaved={() => {
                setEditor({ mode: "closed" });
                void load();
              }}
              onCancel={() => setEditor({ mode: "closed" })}
            />
          )}
          {editor.mode === "edit" && editor.config.pluginType === "excel" && (
            <DataExportExcelEditor
              rcNumber={rcNumber}
              initial={editor.config}
              onSaved={() => {
                setEditor({ mode: "closed" });
                void load();
              }}
              onCancel={() => setEditor({ mode: "closed" })}
            />
          )}
          {editor.mode !== "closed" &&
            !(editor.mode === "create" && editor.pluginType === "excel") &&
            !(editor.mode === "edit" && editor.config.pluginType === "excel") && (
              <p className="text-sm text-muted-foreground">
                Für dieses Plugin ist noch kein Editor verfügbar.
              </p>
            )}
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null);
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Konfiguration löschen?</AlertDialogTitle>
            <AlertDialogDescription>
              Die Konfiguration <strong>{deleteTarget?.name}</strong> wird gelöscht. Bereits ausgeführte Jobs bleiben im Audit-Log erhalten.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Abbrechen</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              disabled={deleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleting ? "Wird gelöscht…" : "Löschen"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
