"use client";

import { useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { DataExportConfigsList } from "./configs-list";
import { DataExportJobsList } from "./jobs-list";
import { DataExportJobStatusModal } from "./job-status-modal";

interface Props {
  rcNumber: string;
}

export function DataExportSection({ rcNumber }: Props) {
  const [trackedJobId, setTrackedJobId] = useState<string | null>(null);

  return (
    <>
      <Tabs defaultValue="configs" className="w-full">
        <TabsList>
          <TabsTrigger value="configs">Konfigurationen</TabsTrigger>
          <TabsTrigger value="jobs">Jobs</TabsTrigger>
        </TabsList>
        <TabsContent value="configs" className="mt-4">
          <DataExportConfigsList rcNumber={rcNumber} />
        </TabsContent>
        <TabsContent value="jobs" className="mt-4">
          <DataExportJobsList rcNumber={rcNumber} onTrackJob={setTrackedJobId} />
        </TabsContent>
      </Tabs>

      <DataExportJobStatusModal
        rcNumber={rcNumber}
        jobId={trackedJobId}
        onClose={() => setTrackedJobId(null)}
        onRetried={setTrackedJobId}
      />
    </>
  );
}
