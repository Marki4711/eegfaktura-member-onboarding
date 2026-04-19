-- Create registration_entrypoint table
CREATE TABLE member_onboarding.registration_entrypoint (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    eeg_id     VARCHAR(255) NOT NULL,
    rc_number  VARCHAR(100) NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT uq_registration_entrypoint_rc_number UNIQUE (rc_number)
);

CREATE INDEX idx_registration_entrypoint_rc_number ON member_onboarding.registration_entrypoint(rc_number);

CREATE TRIGGER update_registration_entrypoint_updated_at
    BEFORE UPDATE ON member_onboarding.registration_entrypoint
    FOR EACH ROW EXECUTE FUNCTION member_onboarding.update_updated_at_column();

-- Rename registration_slug -> rc_number in application table
ALTER TABLE member_onboarding.application
    RENAME COLUMN registration_slug TO rc_number;

DROP INDEX IF EXISTS idx_application_registration_slug;

CREATE INDEX idx_application_rc_number ON member_onboarding.application(rc_number);
