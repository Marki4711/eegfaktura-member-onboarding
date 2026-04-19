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
    <main className="min-h-screen bg-muted/40 flex items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <CardHeader>
          <CardTitle>Mitglied werden</CardTitle>
          <CardDescription>
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
            <Button type="submit" className="w-full" disabled={!rcNumber.trim()}>
              Weiter
            </Button>
          </form>
        </CardContent>
      </Card>
    </main>
  );
}
