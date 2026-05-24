-- Rollback Migration 000055. Backfill aus email_confirmed_at, da
-- semantisch identisch (beide wurden in MarkEmailConfirmedTx auf
-- denselben NOW() gesetzt).

ALTER TABLE member_onboarding.application
    ADD COLUMN IF NOT EXISTS email_confirmation_used_at TIMESTAMPTZ NULL;

UPDATE member_onboarding.application
   SET email_confirmation_used_at = email_confirmed_at
 WHERE email_confirmed_at IS NOT NULL;
