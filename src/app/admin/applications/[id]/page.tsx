"use client";

import { use } from "react";
import { useSearchParams } from "next/navigation";
import { AdminApplicationDetail } from "@/components/admin-application-detail";

interface Props {
  params: Promise<{ id: string }>;
}

export default function ApplicationDetailPage({ params }: Props) {
  const { id } = use(params);
  const searchParams = useSearchParams();
  const raw = searchParams.get("returnTo") ?? "";
  const returnTo = raw.startsWith("/") ? raw : "/admin/applications";

  return <AdminApplicationDetail id={id} returnTo={returnTo} />;
}
