ALTER TABLE member_onboarding.registration_entrypoint
    DROP CONSTRAINT IF EXISTS registration_entrypoint_activation_mode_valid;

ALTER TABLE member_onboarding.registration_entrypoint
    DROP COLUMN IF EXISTS activation_mode;
