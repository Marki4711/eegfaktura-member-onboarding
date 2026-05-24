-- Rollback Migration 000054.
-- Bewusst KEIN Backfill aus status_log: die Spalte war nie ausgewertet,
-- ein Rollback würde bestehende Rows mit NULL belassen, was dem Vor-Audit-
-- Verhalten entspricht (Default war NULL).

ALTER TABLE member_onboarding.application
    ADD COLUMN IF NOT EXISTS reviewed_by_user_id VARCHAR(255);
