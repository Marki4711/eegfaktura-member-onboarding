-- +migrate Down
DROP INDEX IF EXISTS member_onboarding.uniq_application_rc_member_number;
