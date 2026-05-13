"use client";

import { useEffect, useState } from "react";
import { useSession } from "next-auth/react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
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
import { getApiKeyStatus, generateApiKey, revokeApiKey, type ApiKeyStatus } from "@/lib/api";
import { formatDateTime as formatDate } from "@/lib/datetime";

interface Props {
  rcNumber: string;
}

export function AdminApiKeyEditor({ rcNumber }: Props) {
  const { data: session } = useSession();
  const [status, setStatus] = useState<ApiKeyStatus | null>(null);
  const [loaded, setLoaded] = useState(false);

  const [generating, setGenerating] = useState(false);
  const [revoking, setRevoking] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [generatedKey, setGeneratedKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!rcNumber || !session?.accessToken) return;
    setLoaded(false);
    setError(null);
    getApiKeyStatus(rcNumber, session.accessToken)
      .then((s) => { setStatus(s); setLoaded(true); })
      .catch(() => { setStatus(null); setLoaded(true); });
  }, [rcNumber, session?.accessToken]);

  const handleGenerate = async () => {
    setGenerating(true);
    setError(null);
    try {
      const res = await generateApiKey(rcNumber, session?.accessToken);
      setGeneratedKey(res.apiKey);
      setCopied(false);
      setStatus({ active: true, lastGeneratedAt: new Date().toISOString() });
    } catch {
      setError("API-Key konnte nicht generiert werden. Bitte erneut versuchen.");
    } finally {
      setGenerating(false);
    }
  };

  const handleRevoke = async () => {
    setRevoking(true);
    setError(null);
    try {
      await revokeApiKey(rcNumber, session?.accessToken);
      setStatus((prev) => prev ? { ...prev, active: false } : null);
    } catch {
      setError("Key konnte nicht widerrufen werden. Bitte erneut versuchen.");
    } finally {
      setRevoking(false);
    }
  };

  const handleCopy = async () => {
    if (!generatedKey) return;
    await navigator.clipboard.writeText(generatedKey);
    setCopied(true);
  };

  if (!loaded) {
    return <p className="text-xs text-muted-foreground">Lädt…</p>;
  }

  return (
    <div className="space-y-4">
      {/* Status-Anzeige */}
      <div className="flex flex-wrap items-center gap-x-6 gap-y-1 text-sm">
        <span>
          Status:{" "}
          <span className={status?.active ? "text-green-600 font-medium" : "text-muted-foreground"}>
            {status?.active ? "Aktiv" : "Kein aktiver Key"}
          </span>
        </span>
        {status?.lastGeneratedAt && (
          <span className="text-muted-foreground text-xs">
            Zuletzt generiert: {formatDate(status.lastGeneratedAt)}
          </span>
        )}
      </div>

      {error && (
        <p className="text-sm text-destructive">{error}</p>
      )}

      {/* Aktionen */}
      <div className="flex flex-wrap gap-2">
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button variant="outline" size="sm" disabled={generating}>
              {generating ? "Wird generiert…" : status?.active ? "Neuen Key generieren" : "API-Key generieren"}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {status?.active ? "Neuen API-Key generieren?" : "API-Key generieren?"}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {status?.active
                  ? "Der bestehende Key wird sofort ungültig. Alle Integrationen, die diesen Key verwenden, müssen auf den neuen Key umgestellt werden."
                  : "Es wird ein neuer API-Key für diese EEG generiert. Der Key wird einmalig angezeigt und danach nicht mehr lesbar."}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>Abbrechen</AlertDialogCancel>
              <AlertDialogAction onClick={handleGenerate}>Generieren</AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {status?.active && (
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" disabled={revoking}>
                {revoking ? "Wird widerrufen…" : "Key widerrufen"}
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>API-Key widerrufen?</AlertDialogTitle>
                <AlertDialogDescription>
                  Der aktive Key wird sofort ungültig. Es wird kein neuer Key erzeugt.
                  Alle Integrationen, die diesen Key verwenden, erhalten ab sofort HTTP 401.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Abbrechen</AlertDialogCancel>
                <AlertDialogAction
                  onClick={handleRevoke}
                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                >
                  Widerrufen
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}
      </div>

      {/* Einmaliger Key-Dialog */}
      <Dialog open={!!generatedKey} onOpenChange={(open) => { if (!open) setGeneratedKey(null); }}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>API-Key generiert</DialogTitle>
            <DialogDescription>
              Kopieren Sie den Key jetzt. Er wird nach dem Schließen dieses Dialogs{" "}
              <strong>nicht mehr angezeigt</strong>.
            </DialogDescription>
          </DialogHeader>

          <div className="my-2 rounded-md border bg-muted p-3">
            <code className="text-xs break-all select-all font-mono">{generatedKey}</code>
          </div>

          <p className="text-xs text-muted-foreground">
            Speichern Sie den Key sicher ab (z.B. als Umgebungsvariable auf Ihrem Server).
            Der Key darf niemals in Browser-seitigem Code verwendet werden.
          </p>

          <DialogFooter className="gap-2">
            <Button variant="outline" size="sm" onClick={handleCopy}>
              {copied ? "Kopiert!" : "Kopieren"}
            </Button>
            <Button size="sm" onClick={() => setGeneratedKey(null)}>
              Schließen
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
