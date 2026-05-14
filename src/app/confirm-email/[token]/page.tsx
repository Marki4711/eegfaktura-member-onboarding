import type { Metadata } from "next";
import { PublicHeader } from "@/components/public-header";
import { ConfirmEmailClient } from "@/components/confirm-email-client";

export const metadata: Metadata = {
  title: "E-Mail-Adresse bestätigen",
};

interface PageProps {
  params: Promise<{ token: string }>;
}

export default async function ConfirmEmailPage({ params }: PageProps) {
  const { token } = await params;
  return (
    <div className="min-h-screen flex flex-col">
      <PublicHeader />
      <main className="flex-1 flex items-start justify-center px-4 py-12">
        <div className="w-full max-w-md">
          <ConfirmEmailClient token={token} />
        </div>
      </main>
    </div>
  );
}
