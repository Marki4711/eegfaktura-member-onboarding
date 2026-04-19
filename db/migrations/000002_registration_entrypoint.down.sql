DROP INDEX IF EXISTS idx_application_rc_number;

ALTER TABLE member_onboarding.application
    RENAME COLUMN rc_number TO registration_slug;

CREATE INDEX idx_application_registration_slug ON member_onboarding.application(registration_slug);

DROP TRIGGER IF EXISTS update_registration_entrypoint_updated_at ON member_onboarding.registration_entrypoint;

DROP INDEX IF EXISTS idx_registration_entrypoint_rc_number;

DROP TABLE IF EXISTS member_onboarding.registration_entrypoint;
