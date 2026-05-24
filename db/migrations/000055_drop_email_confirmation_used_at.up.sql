-- Data-Model-Slimming-Audit (2026-05-24, Audit-TODO §6): Spalte ist
-- 100 % redundant zu email_confirmed_at.
--
-- MarkEmailConfirmedTx setzt BEIDE Spalten auf denselben NOW(), der
-- einzige Reader (application_service.go:825) prüft `UsedAt != nil` zur
-- Idempotenz-Erkennung — funktional identisch zu `ConfirmedAt != nil`.
-- Code-Refactor auf `email_confirmed_at != nil` umgestellt im selben
-- Commit; nach der Migration läuft der Idempotenz-Check sauber gegen
-- die verbleibende Spalte.

ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS email_confirmation_used_at;
