ALTER TABLE member_onboarding.registration_entrypoint ADD COLUMN IF NOT EXISTS eeg_id VARCHAR(255);

ALTER TABLE member_onboarding.application ADD COLUMN IF NOT EXISTS eeg_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_application_eeg_id ON member_onboarding.application(eeg_id);
