-- Rollback Migration 000053. Stellt den Vor-Audit-Zustand wieder her:
-- entfernt den neuen key_hash-Partial-Index und legt die zwei redundanten
-- Plain-Indizes wieder an (für strikte Symmetrie — sie waren funktional
-- redundant zu UNIQUE-Constraints, aber falls ein Roll-Forward jemals
-- nötig wird, bleiben Index-Namen identisch).

DROP INDEX IF EXISTS member_onboarding.idx_external_api_key_hash;

CREATE INDEX IF NOT EXISTS idx_application_reference_number
    ON member_onboarding.application(reference_number);

CREATE INDEX IF NOT EXISTS idx_registration_entrypoint_rc_number
    ON member_onboarding.registration_entrypoint(rc_number);
