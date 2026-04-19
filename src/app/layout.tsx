import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "eegFaktura Mitglieder-Onboarding",
  description: "Selbstregistrierung für neue EEG-Mitglieder",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="de">
      <body className="antialiased">{children}</body>
    </html>
  );
}
