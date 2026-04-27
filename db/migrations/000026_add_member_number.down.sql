-- +migrate Down
DROP INDEX IF EXISTS member_onboarding.idx_application_rc_number_member_number;
ALTER TABLE member_onboarding.registration_entrypoint DROP COLUMN IF EXISTS member_number_start;
ALTER TABLE member_onboarding.application DROP COLUMN IF EXISTS member_number;
