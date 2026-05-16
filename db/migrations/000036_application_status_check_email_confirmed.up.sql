-- PROJ-31 follow-up: the `email_confirmed` status was added in code and in
-- migration 000030, but the CHECK constraint on application.status was never
-- updated. As a result every confirm-email POST hit Postgres error 23514
-- ("violates check constraint application_status_check") and returned 500.
--
-- Drop the legacy constraint and re-create it with the full status set as
-- documented in CLAUDE.md → "Allowed status values".

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
