ALTER TABLE member_onboarding.application DROP COLUMN IF EXISTS eeg_id;
DROP INDEX IF EXISTS member_onboarding.idx_application_eeg_id;

ALTER TABLE member_onboarding.registration_entrypoint DROP COLUMN IF EXISTS eeg_id;
