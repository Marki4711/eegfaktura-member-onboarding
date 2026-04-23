import type { Metadata } from "next";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertCircle } from "lucide-react";
import { RegistrationForm } from "@/components/registration-form";
import { PublicHeader } from "@/components/public-header";
import { getRegistrationConfig, ApiResponseError, type RegistrationConfig } from "@/lib/api";

interface PageProps {
  params: Promise<{ rc_number: string }>;
}

export async function generateMetadata({ params }: PageProps): Promise<Metadata> {
  const { rc_number } = await params;
  return {
    title: `Mitglied werden – ${rc_number}`,
  };
}

function isApiResponseError(err: unknown): err is ApiResponseError {
  return (
    err instanceof Error &&
    err.name === "ApiResponseError" &&
    "apiError" in err &&
    typeof (err as ApiResponseError).apiError?.code === "string"
  );
}

function PublicPageShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex flex-col">
      <PublicHeader />
      <main className="flex-1 py-10 px-4">
        <div className="max-w-2xl mx-auto space-y-6">{children}</div>
      </main>
      <footer className="py-4 px-4 border-t border-border text-center text-xs text-muted-foreground">
        © {new Date().getFullYear()} eegFaktura — Energiegemeinschaften einfach verwalten
      </footer>
    </div>
  );
}

export default async function RegisterPage({ params }: PageProps) {
  const { rc_number } = await params;

  let config: RegistrationConfig | null = null;
  let errorKind: "not_found" | "gone" | "backend" | null = null;

  try {
    config = await getRegistrationConfig(rc_number);
  } catch (err) {
    if (isApiResponseError(err)) {
      const code = err.apiError.code;
      if (code === "not_found") {
        errorKind = "not_found";
      } else if (code === "gone") {
        errorKind = "gone";
      } else {
        errorKind = "backend";
      }
    } else {
      errorKind = "backend";
    }
  }

  if (errorKind === "not_found") {
    return (
      <PublicPageShell>
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Registrierungslink ungültig</AlertTitle>
          <AlertDescription>
            Der Registrierungslink <strong>{rc_number.toUpperCase()}</strong> ist
            nicht bekannt. Bitte überprüfen Sie den Link oder wenden Sie sich an
            Ihren EEG-Administrator.
          </AlertDescription>
        </Alert>
      </PublicPageShell>
    );
  }

  if (errorKind === "gone") {
    return (
      <PublicPageShell>
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Registrierung nicht verfügbar</AlertTitle>
          <AlertDescription>
            Diese Registrierung ist nicht mehr aktiv. Bitte wenden Sie sich an
            Ihren EEG-Administrator.
          </AlertDescription>
        </Alert>
      </PublicPageShell>
    );
  }

  if (errorKind === "backend" || !config) {
    return (
      <PublicPageShell>
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertTitle>Dienst nicht verfügbar</AlertTitle>
          <AlertDescription>
            Die Registrierung konnte nicht geladen werden. Bitte versuchen Sie
            es später erneut.
          </AlertDescription>
        </Alert>
      </PublicPageShell>
    );
  }

  return (
    <PublicPageShell>
      <h1 className="text-2xl font-bold tracking-tight text-foreground">
        {config.title}
      </h1>
      <RegistrationForm config={config} />
    </PublicPageShell>
  );
}
