import Link from "next/link";
import { Toaster } from "@/components/ui/sonner";

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-6 py-4 flex items-center gap-6">
          <span className="font-semibold text-gray-900">
            eegFaktura Onboarding
          </span>
          <nav className="flex gap-4">
            <Link
              href="/admin/applications"
              className="text-sm text-gray-600 hover:text-gray-900 transition-colors"
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
