import type { Metadata } from "next";
import { PublicHeader } from "@/components/public-header";
import { ConfirmEmailClient } from "@/components/confirm-email-client";

export const metadata: Metadata = {
  title: "E-Mail-Adresse bestätigen",
  // PROJ-31 Security M2: the page URL ends up in browser history; an
  // explicit no-referrer policy keeps the (URL-fragment-stripped) origin
  // out of Referer headers for any future external link we might add.
  other: {
    referrer: "no-referrer",
  },
};

export default function ConfirmEmailPage() {
  return (
    <div className="min-h-screen flex flex-col">
      <PublicHeader />
      <main className="flex-1 flex items-start justify-center px-4 py-12">
        <div className="w-full max-w-md">
          {/* Token comes from the URL fragment — handled client-side so the
              server never sees it (PROJ-31 Security M1: keeps the token
              out of reverse-proxy and CDN access logs). */}
          <ConfirmEmailClient />
        </div>
      </main>
    </div>
  );
}
