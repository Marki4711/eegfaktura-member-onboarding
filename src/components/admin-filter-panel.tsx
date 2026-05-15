"use client";

import { useEffect, useState } from "react";
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

interface Props {
  rcNumbers?: string[];
}

export function AdminFilterPanel({ rcNumbers = [] }: Props) {
  const router = useRouter();
  const searchParams = useSearchParams();

  const [status, setStatus] = useState(searchParams.get("status") ?? "all");
  const [name, setLastname] = useState(searchParams.get("name") ?? "");
  const [email, setEmail] = useState(searchParams.get("email") ?? "");
  const [rcNumber, setRcNumber] = useState(searchParams.get("rc_number") ?? "all");
  const [submittedFrom, setSubmittedFrom] = useState(
    searchParams.get("submitted_from") ?? ""
  );
  const [submittedTo, setSubmittedTo] = useState(
    searchParams.get("submitted_to") ?? ""
  );

  useEffect(() => {
    setStatus(searchParams.get("status") ?? "all");
    setLastname(searchParams.get("name") ?? "");
    setEmail(searchParams.get("email") ?? "");
    setRcNumber(searchParams.get("rc_number") ?? "all");
    setSubmittedFrom(searchParams.get("submitted_from") ?? "");
    setSubmittedTo(searchParams.get("submitted_to") ?? "");
  }, [searchParams]);

  function applyFilters() {
    const params = new URLSearchParams();
    if (status && status !== "all") params.set("status", status);
    if (name.trim()) params.set("name", name.trim());
    if (email.trim()) params.set("email", email.trim());
    if (rcNumber && rcNumber !== "all") params.set("rc_number", rcNumber);
    if (submittedFrom) params.set("submitted_from", submittedFrom);
    if (submittedTo) params.set("submitted_to", submittedTo);
    // Preserve current sort selection across filter changes.
    const currentSort = searchParams.get("sort");
    const currentOrder = searchParams.get("order");
    if (currentSort) params.set("sort", currentSort);
    if (currentOrder) params.set("order", currentOrder);
    params.set("page", "1");
    router.push(`/admin/applications?${params.toString()}`);
  }

  function clearFilters() {
    setStatus("all");
    setLastname("");
    setEmail("");
    setRcNumber("all");
    setSubmittedFrom("");
    setSubmittedTo("");
    router.push("/admin/applications");
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Enter") applyFilters();
  }

  const showEEGFilter = rcNumbers.length > 1;

  return (
    <div className="bg-card rounded-lg border p-4 space-y-4">
      <div className={`grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 ${showEEGFilter ? "lg:grid-cols-6" : "lg:grid-cols-5"} gap-4`}>
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
          <Label htmlFor="filter-name">Name</Label>
          <Input
            id="filter-name"
            value={name}
            onChange={(e) => setLastname(e.target.value)}
            onKeyDown={handleKeyDown}
          />
        </div>

        <div className="space-y-1">
          <Label htmlFor="filter-email">E-Mail</Label>
          <Input
            id="filter-email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            onKeyDown={handleKeyDown}
          />
        </div>

        {showEEGFilter && (
          <div className="space-y-1">
            <Label htmlFor="filter-eeg">EEG</Label>
            <Select value={rcNumber} onValueChange={setRcNumber}>
              <SelectTrigger id="filter-eeg">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Alle EEGs</SelectItem>
                {rcNumbers.map((rc) => (
                  <SelectItem key={rc} value={rc}>{rc}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

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
