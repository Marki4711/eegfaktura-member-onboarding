import { notFound } from "next/navigation";
import type { Metadata } from "next";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertCircle } from "lucide-react";
import { RegistrationForm } from "@/components/registration-form";
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

// Duck-type guard that works even when module bundling produces multiple class
// identities (a known Next.js Server Component edge case with instanceof).
function isApiResponseError(err: unknown): err is ApiResponseError {
  return (
    err instanceof Error &&
    err.name === "ApiResponseError" &&
    "apiError" in err &&
    typeof (err as ApiResponseError).apiError?.code === "string"
  );
}

export default async function RegisterPage({ params }: PageProps) {
  const { rc_number } = await params;

  let config: RegistrationConfig | null = null;
  let gone = false;
  let backendError = false;

  try {
    config = await getRegistrationConfig(rc_number);
  } catch (err) {
    if (isApiResponseError(err)) {
      const code = err.apiError.code;
      if (code === "not_found") {
        notFound(); // renders the nearest not-found.tsx / default 404 page
      } else if (code === "gone") {
        gone = true;
      } else {
        backendError = true;
      }
    } else {
      // Network error, DNS failure, backend unreachable, etc.
      backendError = true;
    }
  }

  if (gone) {
    return (
      <main className="min-h-screen bg-muted/40 py-12 px-4">
        <div className="max-w-2xl mx-auto">
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>Registrierung nicht verfügbar</AlertTitle>
            <AlertDescription>
              Diese Registrierung ist nicht mehr aktiv. Bitte wenden Sie sich an
              Ihren EEG-Administrator.
            </AlertDescription>
          </Alert>
        </div>
      </main>
    );
  }

  if (backendError || !config) {
    return (
      <main className="min-h-screen bg-muted/40 py-12 px-4">
        <div className="max-w-2xl mx-auto">
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>Dienst nicht verfügbar</AlertTitle>
            <AlertDescription>
              Die Registrierung konnte nicht geladen werden. Bitte versuchen Sie
              es später erneut.
            </AlertDescription>
          </Alert>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-muted/40 py-12 px-4">
      <div className="max-w-2xl mx-auto space-y-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">{config.title}</h1>
          <p className="text-muted-foreground mt-1">
            Füllen Sie das Formular aus, um Ihre Mitgliedschaft zu beantragen.
          </p>
        </div>
        <RegistrationForm config={config} />
      </div>
    </main>
  );
}
