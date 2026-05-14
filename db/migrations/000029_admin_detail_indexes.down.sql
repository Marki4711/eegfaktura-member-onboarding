-- +migrate Down
DROP INDEX IF EXISTS member_onboarding.idx_status_log_app_created;
DROP INDEX IF EXISTS member_onboarding.idx_document_consent_app_consented;
DROP INDEX IF EXISTS member_onboarding.idx_metering_point_app_created;
