"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const STATUS_OPTIONS = [
  { value: "all", label: "Alle Status" },
  { value: "draft", label: "Entwurf" },
  { value: "submitted", label: "Eingereicht" },
  { value: "under_review", label: "In Prüfung" },
  { value: "needs_info", label: "Info benötigt" },
  { value: "approved", label: "Genehmigt" },
  { value: "rejected", label: "Abgelehnt" },
  { value: "imported", label: "Importiert" },
  { value: "import_failed", label: "Import fehlgeschlagen" },
];

export function AdminFilterPanel() {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [status, setStatus] = useState(searchParams.get("status") ?? "all");
  const [lastname, setLastname] = useState(searchParams.get("lastname") ?? "");
  const [email, setEmail] = useState(searchParams.get("email") ?? "");
  const [meteringPoint, setMeteringPoint] = useState(
    searchParams.get("metering_point") ?? ""
  );
  const [submittedFrom, setSubmittedFrom] = useState(
    searchParams.get("submitted_from") ?? ""
  );
  const [submittedTo, setSubmittedTo] = useState(
    searchParams.get("submitted_to") ?? ""
  );

  function applyFilters() {
    const params = new URLSearchParams();
    if (status && status !== "all") params.set("status", status);
    if (lastname.trim()) params.set("lastname", lastname.trim());
    if (email.trim()) params.set("email", email.trim());
    if (meteringPoint.trim()) params.set("metering_point", meteringPoint.trim());
    if (submittedFrom) params.set("submitted_from", submittedFrom);
    if (submittedTo) params.set("submitted_to", submittedTo);
    params.set("page", "1");
    router.push(`/admin/applications?${params.toString()}`);
  }

  function clearFilters() {
    setStatus("all");
    setLastname("");
    setEmail("");
    setMeteringPoint("");
    setSubmittedFrom("");
    setSubmittedTo("");
    router.push("/admin/applications");
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter") applyFilters();
  }

  return (
    <div className="bg-white rounded-lg border p-4 space-y-4">
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
        <div className="space-y-1">
          <Label htmlFor="filter-status">Status</Label>
          <Select value={status} onValueChange={setStatus}>
            <SelectTrigger id="filter-status">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {STATUS_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-1">
          <Label htmlFor="filter-lastname">Nachname</Label>
          <Input
            id="filter-lastname"
            value={lastname}
            onChange={(e) => setLastname(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Brandstätter"
          />
        </div>

        <div className="space-y-1">
          <Label htmlFor="filter-email">E-Mail</Label>
          <Input
            id="filter-email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="max@example.org"
          />
        </div>

        <div className="space-y-1">
          <Label htmlFor="filter-mp">Zählpunkt</Label>
          <Input
            id="filter-mp"
            value={meteringPoint}
            onChange={(e) => setMeteringPoint(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="AT003..."
          />
        </div>

        <div className="space-y-1">
          <Label htmlFor="filter-from">Eingereicht ab</Label>
          <Input
            id="filter-from"
            type="date"
            value={submittedFrom}
            onChange={(e) => setSubmittedFrom(e.target.value)}
          />
        </div>

        <div className="space-y-1">
          <Label htmlFor="filter-to">Eingereicht bis</Label>
          <Input
            id="filter-to"
            type="date"
            value={submittedTo}
            onChange={(e) => setSubmittedTo(e.target.value)}
          />
        </div>
      </div>

      <div className="flex gap-2">
        <Button onClick={applyFilters}>Filter anwenden</Button>
        <Button variant="outline" onClick={clearFilters}>
          Zurücksetzen
        </Button>
      </div>
    </div>
  );
}
