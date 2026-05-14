"use client";

import { useEffect, useRef, useState } from "react";
import { CheckCircle2, AlertCircle } from "lucide-react";
import { confirmEmail, type ConfirmEmailResponse } from "@/lib/api";

type State =
  | { kind: "loading" }
  | { kind: "success"; data: ConfirmEmailResponse }
  | { kind: "error"; message: string };

interface Props {
  token: string;
}

export function ConfirmEmailClient({ token }: Props) {
  const [state, setState] = useState<State>({ kind: "loading" });
  // StrictMode in dev runs effects twice — a one-shot guard makes sure
  // the confirm endpoint receives exactly one POST per page mount.
  const sentRef = useRef(false);

  useEffect(() => {
    if (sentRef.current) return;
    sentRef.current = true;
    confirmEmail(token)
      .then((data) => setState({ kind: "success", data }))
      .catch((err: Error) => setState({ kind: "error", message: err.message }));
  }, [token]);

  if (state.kind === "loading") {
    return (
      <div className="rounded-md border bg-card p-8 text-center">
        <div className="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-muted-foreground/30 border-t-foreground" />
        <p className="mt-4 text-sm text-muted-foreground">Bestätige deine E-Mail-Adresse …</p>
      </div>
    );
  }

  if (state.kind === "success") {
    return (
      <div className="rounded-md border border-green-200 bg-green-50 p-8 text-center text-green-900">
        <CheckCircle2 className="mx-auto h-10 w-10 text-green-600" />
        <h1 className="mt-3 text-xl font-semibold">Vielen Dank!</h1>
        <p className="mt-2 text-sm">
          Deine E-Mail-Adresse ist bestätigt.
          {state.data.eegName ? (
            <>
              {" "}
              Dein Antrag liegt jetzt bei <strong>{state.data.eegName}</strong> zur Prüfung.
            </>
          ) : (
            <> Dein Antrag liegt jetzt bei deiner Energiegemeinschaft zur Prüfung.</>
          )}
        </p>
        {state.data.eegContactEmail && (
          <p className="mt-4 text-xs text-green-800">
            Bei Rückfragen wende dich bitte direkt an{" "}
            <a className="underline" href={`mailto:${state.data.eegContactEmail}`}>
              {state.data.eegContactEmail}
            </a>
            .
          </p>
        )}
      </div>
    );
  }

  // state.kind === "error"
  return (
    <div className="rounded-md border border-red-200 bg-red-50 p-8 text-center text-red-900">
      <AlertCircle className="mx-auto h-10 w-10 text-red-600" />
      <h1 className="mt-3 text-xl font-semibold">Bestätigung nicht möglich</h1>
      <p className="mt-2 text-sm">{state.message}</p>
      <p className="mt-4 text-xs text-red-800">
        Falls du eine neue Bestätigungs-Mail benötigst, wende dich bitte an deine
        Energiegemeinschaft.
      </p>
    </div>
  );
}
