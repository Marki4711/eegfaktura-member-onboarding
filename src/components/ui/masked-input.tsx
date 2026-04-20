"use client";

import { IMaskInput } from "react-imask";
import { cn } from "@/lib/utils";

const inputClass =
  "flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-base ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 md:text-sm";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function MaskedInput({ className, ...props }: { className?: string } & Record<string, any>) {
  return <IMaskInput className={cn(inputClass, className)} {...props} />;
}
