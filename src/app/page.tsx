"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { PublicHeader } from "@/components/public-header";

export default function HomePage() {
  const router = useRouter();
  const [rcNumber, setRcNumber] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const trimmed = rcNumber.trim();
    if (trimmed) {
      router.push(`/register/${encodeURIComponent(trimmed)}`);
    }
  }

  return (
    <div className="min-h-screen flex flex-col">
      <PublicHeader />
      <main className="flex-1 flex items-center justify-center p-4">
        <Card className="w-full max-w-sm">
          <CardHeader>
            <CardTitle className="text-foreground">Mitglied werden</CardTitle>
            <CardDescription className="text-muted-foreground">
              Geben Sie Ihre Registrierungsnummer (RC-Nummer) ein, die Sie von
              Ihrer EEG erhalten haben.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="rc">RC-Nummer</Label>
                <Input
                  id="rc"
                  placeholder="RC123456"
                  value={rcNumber}
                  onChange={(e) => setRcNumber(e.target.value)}
                  autoFocus
                />
              </div>
              <Button
                type="submit"
                className="w-full"
                disabled={!rcNumber.trim()}
              >
                Weiter
              </Button>
            </form>
          </CardContent>
        </Card>
      </main>
      <footer className="py-4 px-4 border-t border-border text-center text-xs text-muted-foreground">
        © {new Date().getFullYear()} eegFaktura — Energiegemeinschaften einfach verwalten
      </footer>
    </div>
  );
}
