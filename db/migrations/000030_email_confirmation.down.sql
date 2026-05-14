DROP INDEX IF EXISTS member_onboarding.idx_application_email_confirmation_expiry;
DROP INDEX IF EXISTS member_onboarding.uniq_application_email_confirmation_token_hash;

ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS email_confirmation_token_expires_at,
    DROP COLUMN IF EXISTS email_confirmation_token_hash,
    DROP COLUMN IF EXISTS email_confirmation_used_at,
    DROP COLUMN IF EXISTS email_confirmed_at;

ALTER TABLE member_onboarding.registration_entrypoint
    DROP COLUMN IF EXISTS require_email_confirmation;
