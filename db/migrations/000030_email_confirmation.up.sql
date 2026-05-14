-- PROJ-31: E-Mail-Adresse-Bestätigung
--
-- Adds an opt-in setting per EEG ("require_email_confirmation") and three
-- columns on the application to back the token-based confirmation flow.

ALTER TABLE member_onboarding.registration_entrypoint
    ADD COLUMN require_email_confirmation BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE member_onboarding.application
    ADD COLUMN email_confirmed_at TIMESTAMPTZ NULL,
    ADD COLUMN email_confirmation_used_at TIMESTAMPTZ NULL,
    ADD COLUMN email_confirmation_token_hash TEXT NULL,
    ADD COLUMN email_confirmation_token_expires_at TIMESTAMPTZ NULL;

-- Partial UNIQUE — token hashes must be unique while they are active. Once
-- the token is consumed and the column reset to NULL, the row is excluded.
CREATE UNIQUE INDEX IF NOT EXISTS uniq_application_email_confirmation_token_hash
    ON member_onboarding.application(email_confirmation_token_hash)
    WHERE email_confirmation_token_hash IS NOT NULL;

-- Pruning the auto-reject job: composite index for the cheap "find expired
-- pending confirmations" scan.
CREATE INDEX IF NOT EXISTS idx_application_email_confirmation_expiry
    ON member_onboarding.application(email_confirmation_token_expires_at)
    WHERE email_confirmation_token_hash IS NOT NULL AND email_confirmed_at IS NULL;
