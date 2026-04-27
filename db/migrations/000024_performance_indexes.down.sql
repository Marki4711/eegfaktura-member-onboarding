-- +migrate Down
DROP INDEX IF EXISTS member_onboarding.idx_metering_point_metering_point_trgm;
DROP INDEX IF EXISTS member_onboarding.idx_application_reference_number_trgm;
DROP INDEX IF EXISTS member_onboarding.idx_application_email_trgm;
DROP INDEX IF EXISTS member_onboarding.idx_application_lastname_trgm;
DROP INDEX IF EXISTS member_onboarding.idx_application_submitted_at;
DROP INDEX IF EXISTS member_onboarding.idx_application_rc_number_status_created_at;
DROP INDEX IF EXISTS member_onboarding.idx_application_rc_number_created_at;
-- pg_trgm extension is intentionally not dropped — it may be used by other schemas
