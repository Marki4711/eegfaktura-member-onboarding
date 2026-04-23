"use client";

import { useMemo } from "react";
import DOMPurify from "dompurify";

const DEFAULT_TEXT = "Füllen Sie das Formular aus, um Ihre Mitgliedschaft zu beantragen.";

interface Props {
  introText?: string | null;
}

export function IntroTextDisplay({ introText }: Props) {
  const sanitized = useMemo(() => {
    if (typeof window === "undefined") return null;
    if (!introText || introText.trim() === "") return null;
    return DOMPurify.sanitize(introText, {
      ALLOWED_TAGS: ["p", "br", "strong", "b", "em", "i", "ul", "ol", "li", "a"],
      ALLOWED_ATTR: ["href", "target", "rel"],
      FORCE_BODY: true,
    });
  }, [introText]);

  if (!sanitized) {
    return (
      <p className="text-sm text-muted-foreground">{DEFAULT_TEXT}</p>
    );
  }

  return (
    <div
      className="prose prose-sm dark:prose-invert max-w-none text-foreground [&_a]:text-primary [&_a]:underline"
      dangerouslySetInnerHTML={{ __html: sanitized }}
    />
  );
}
