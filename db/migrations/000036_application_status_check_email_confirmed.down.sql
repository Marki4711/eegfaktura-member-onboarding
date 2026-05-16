-- Reverse of 000036: shrink the constraint back to the original 8-value set.
-- Requires no application rows in status 'email_confirmed' (manually clean
-- them up before running this down migration).

ALTER TABLE member_onboarding.application
    DROP CONSTRAINT IF EXISTS application_status_check;

ALTER TABLE member_onboarding.application
    ADD CONSTRAINT application_status_check
    CHECK (status IN (
        'draft',
        'submitted',
        'under_review',
        'needs_info',
        'approved',
        'rejected',
        'imported',
        'import_failed'
    ));
