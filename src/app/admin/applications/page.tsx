import { Suspense } from "react";
import { Skeleton } from "@/components/ui/skeleton";
import { ApplicationsPageContent } from "./applications-page-content";

export default function ApplicationsPage() {
  return (
    <Suspense
      fallback={
        <div className="space-y-4">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-64 w-full" />
        </div>
      }
    >
      <ApplicationsPageContent />
    </Suspense>
  );
}
