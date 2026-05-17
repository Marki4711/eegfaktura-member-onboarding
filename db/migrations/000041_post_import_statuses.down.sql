-- Rollback PROJ-46 Stage A. Schlägt fehl, wenn schon Anträge in den
-- neuen Stati existieren — dann müssen die erst auf einen Pre-PROJ-46-
-- Status zurückgesetzt werden (z.B. via UPDATE auf 'imported').

ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS activated_at,
    DROP COLUMN IF EXISTS bank_confirmed_at;

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
        'import_failed'
    ));
