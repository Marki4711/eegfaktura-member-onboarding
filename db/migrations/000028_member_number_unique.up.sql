-- +migrate Up
-- Defense-in-depth: a partial UNIQUE index on (rc_number, member_number)
-- catches accidental duplicate assignments (re-import after a bug, manual
-- DB edit, etc.). The new flow (assign at import time, pre-import check
-- against the core) should never produce duplicates on its own, but this
-- guards against future regressions and historical inconsistencies.
--
-- Partial WHERE clause excludes NULL members (draft/submitted/under_review
-- applications, plus all applications between submit and import).
CREATE UNIQUE INDEX IF NOT EXISTS uniq_application_rc_member_number
    ON member_onboarding.application(rc_number, member_number)
    WHERE member_number IS NOT NULL;
