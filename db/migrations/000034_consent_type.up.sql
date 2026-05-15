-- PROJ-36: track whether a consent snapshot represents an active user
-- action (`explicit`) or a passive "displayed for information"
-- acknowledgement (`informational`).
--
-- Existing entries all came from required-checkbox flows pre-PROJ-36 and
-- therefore qualify as `explicit` — the DEFAULT covers the backfill so
-- the migration is a single ALTER TABLE.

ALTER TABLE member_onboarding.document_consent
    ADD COLUMN consent_type TEXT NOT NULL DEFAULT 'explicit';

ALTER TABLE member_onboarding.document_consent
    ADD CONSTRAINT document_consent_consent_type_check
    CHECK (consent_type IN ('explicit', 'informational'));
