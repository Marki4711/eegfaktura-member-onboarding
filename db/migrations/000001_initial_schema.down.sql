-- +migrate Down
-- Drop triggers
DROP TRIGGER IF EXISTS update_metering_point_updated_at ON member_onboarding.metering_point;
DROP TRIGGER IF EXISTS update_application_updated_at ON member_onboarding.application;

-- Drop function
DROP FUNCTION IF EXISTS member_onboarding.update_updated_at_column();

-- Drop tables
DROP TABLE IF EXISTS member_onboarding.status_log;
DROP TABLE IF EXISTS member_onboarding.metering_point;
DROP TABLE IF EXISTS member_onboarding.application;

-- Drop schema
DROP SCHEMA IF EXISTS member_onboarding;