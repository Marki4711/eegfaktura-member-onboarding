-- DB-Performance-Audit (2026-05-24): drei index-bezogene Folge-Fixes.
--
-- 1. HIGH: fehlender Index auf external_api_key.key_hash. Jeder externe
--    API-Call (Bearer moak_*) macht aktuell einen Seq-Scan. Partial-Index
--    auf revoked_at IS NULL, weil widerrufene Keys sowieso 401 zurückgeben
--    und gar nicht erst gesucht werden müssen.
--
-- 2./3. LOW: zwei redundante Indizes droppen. Postgres legt für UNIQUE-
--    Constraints automatisch einen Index an; die zusätzlichen plain-B-Tree-
--    Indizes sind Duplikate und kosten beim INSERT/UPDATE doppelte
--    Write-Amplification.
--
-- Migration ist additiv-sicher: alle drei Statements sind idempotent
-- (IF NOT EXISTS / IF EXISTS).

CREATE INDEX IF NOT EXISTS idx_external_api_key_hash
    ON member_onboarding.external_api_key(key_hash)
    WHERE revoked_at IS NULL;

DROP INDEX IF EXISTS member_onboarding.idx_application_reference_number;

DROP INDEX IF EXISTS member_onboarding.idx_registration_entrypoint_rc_number;
