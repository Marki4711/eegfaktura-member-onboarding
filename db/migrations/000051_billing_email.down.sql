ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS has_billing_email,
    DROP COLUMN IF EXISTS billing_email;
