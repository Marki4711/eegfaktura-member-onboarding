-- +migrate Up
-- Per-application member number (admin-only, auto-assigned at first submission).
ALTER TABLE member_onboarding.application
    ADD COLUMN member_number INT;

-- Per-EEG starting value for the auto-increment counter.
-- The first member number assigned for this EEG will be member_number_start.
ALTER TABLE member_onboarding.registration_entrypoint
    ADD COLUMN member_number_start INT NOT NULL DEFAULT 1;

CREATE INDEX idx_application_rc_number_member_number
    ON member_onboarding.application(rc_number, member_number);
