import Link from "next/link";
import { Toaster } from "@/components/ui/sonner";

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-background">
      <header className="bg-card border-b border-border">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center gap-6">
          <div className="flex items-center gap-2">
            <div className="w-6 h-6 bg-primary flex items-center justify-center shrink-0">
              <svg viewBox="0 0 24 24" fill="none" className="w-4 h-4" aria-hidden="true">
                <path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" fill="currentColor" className="text-primary-foreground" />
              </svg>
            </div>
            <span className="font-bold text-foreground text-sm tracking-wide">
              eegFaktura
            </span>
            <span className="text-primary text-xs font-normal">Admin</span>
          </div>
          <nav className="flex gap-4">
            <Link
              href="/admin/applications"
              className="text-sm text-muted-foreground hover:text-primary transition-colors"
            >
              Anträge
            </Link>
          </nav>
        </div>
      </header>
      <main className="max-w-7xl mx-auto px-6 py-8">{children}</main>
      <Toaster />
    </div>
  );
}
