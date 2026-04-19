export function PublicHeader() {
  return (
    <header className="bg-card border-b border-border">
      <div className="max-w-4xl mx-auto px-4 py-4 flex items-center gap-3">
        <div className="w-8 h-8 bg-primary flex items-center justify-center shrink-0">
          <svg viewBox="0 0 24 24" fill="none" className="w-5 h-5" aria-hidden="true">
            <path
              d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"
              fill="currentColor"
              className="text-primary-foreground"
            />
          </svg>
        </div>
        <div>
          <div className="text-foreground font-bold text-sm leading-none tracking-wide">
            eegFaktura
          </div>
          <div className="text-primary text-xs leading-none mt-1 font-normal">
            Mitglieder-Onboarding
          </div>
        </div>
      </div>
    </header>
  );
}
