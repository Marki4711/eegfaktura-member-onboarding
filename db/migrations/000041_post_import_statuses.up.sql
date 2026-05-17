-- PROJ-46 Stage A: drei neue Stati nach Import-Erfolg + zwei Audit-
-- Timestamps für den Übergang zur EEG-Aktivierung.
--
-- Flow nach Import-Erfolg (Service-seitig automatisch):
--   einzugsart = 'b2b'  → awaiting_bank_confirmation  (Admin wartet auf
--                          Member-Rückmeldung zur Hausbank-Abstimmung)
--                       → ready_for_activation        (Admin manuell)
--                       → activated                   (Admin manuell ODER
--                          Activation-Check-Button in Stage D)
--   sonst               → ready_for_activation        (Auto-Skip)
--                       → activated                   (s.o.)
--
-- 'activated' ist strikter Endzustand: kein Reset möglich, keine
-- Rückwärts-Übergänge. Stages davor lassen sich via /reset-import wieder
-- auf 'approved' zurücksetzen (PROJ-30-Erweiterung).

ALTER TABLE member_onboarding.application
    DROP CONSTRAINT IF EXISTS application_status_check;

ALTER TABLE member_onboarding.application
    ADD CONSTRAINT application_status_check
    CHECK (status IN (
        'draft',
        'submitted',
        'email_confirmed',
        'under_review',
        'needs_info',
        'approved',
        'rejected',
        'imported',
        'import_failed',
        'awaiting_bank_confirmation',
        'ready_for_activation',
        'activated'
    ));

ALTER TABLE member_onboarding.application
    ADD COLUMN bank_confirmed_at  TIMESTAMPTZ NULL,
    ADD COLUMN activated_at       TIMESTAMPTZ NULL;
