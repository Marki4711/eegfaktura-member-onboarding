import type { Metadata } from "next";
import { Roboto } from "next/font/google";
import { Providers } from "./providers";
import "./globals.css";

const roboto = Roboto({
  subsets: ["latin"],
  weight: ["400", "700"],
  variable: "--font-roboto",
});

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
    <html lang="de" className={roboto.variable}>
      <body className="antialiased font-[var(--font-roboto)]">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
