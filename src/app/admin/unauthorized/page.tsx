import Link from "next/link";
import { Button } from "@/components/ui/button";

export default function UnauthorizedPage() {
  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className="text-center space-y-4 max-w-md px-6">
        <div className="w-16 h-16 bg-destructive/10 rounded-full flex items-center justify-center mx-auto">
          <svg viewBox="0 0 24 24" fill="none" className="w-8 h-8 text-destructive" aria-hidden="true">
            <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </div>
        <h1 className="text-2xl font-semibold text-foreground">Kein Zugriff</h1>
        <p className="text-muted-foreground">
          Ihr Konto hat keine Berechtigung für den Admin-Bereich. Bitte wenden Sie
          sich an Ihren Administrator.
        </p>
        <div className="flex gap-3 justify-center pt-2">
          <Button asChild variant="outline">
            <Link href="/api/auth/signin">Anderes Konto verwenden</Link>
          </Button>
          <Button asChild variant="ghost">
            <Link href="/">Zurück zur Startseite</Link>
          </Button>
        </div>
      </div>
    </div>
  );
}
