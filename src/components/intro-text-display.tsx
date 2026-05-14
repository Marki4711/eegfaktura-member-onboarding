"use client";

import { useEffect, useState } from "react";

const DEFAULT_TEXT = "Füllen Sie das Formular aus, um Ihre Mitgliedschaft zu beantragen.";

interface Props {
  introText?: string | null;
}

export function IntroTextDisplay({ introText }: Props) {
  const [sanitized, setSanitized] = useState<string | null>(null);

  useEffect(() => {
    if (!introText || introText.trim() === "") {
      setSanitized(null);
      return;
    }
    let cancelled = false;
    void import("dompurify").then(({ default: DOMPurify }) => {
      if (cancelled) return;
      setSanitized(
        DOMPurify.sanitize(introText, {
          ALLOWED_TAGS: ["p", "br", "strong", "b", "em", "i", "ul", "ol", "li", "a"],
          ALLOWED_ATTR: ["href", "target", "rel"],
          FORCE_BODY: true,
        })
      );
    });
    return () => {
      cancelled = true;
    };
  }, [introText]);

  if (!sanitized) {
    return (
      <p className="text-sm text-muted-foreground">{DEFAULT_TEXT}</p>
    );
  }

  return (
    <div
      className="prose prose-sm dark:prose-invert max-w-none text-muted-foreground [&_a]:text-primary [&_a]:underline"
      dangerouslySetInnerHTML={{ __html: sanitized }}
    />
  );
}
