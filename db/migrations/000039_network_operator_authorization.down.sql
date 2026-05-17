ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS network_operator_authorization_at,
    DROP COLUMN IF EXISTS network_operator_authorization;
