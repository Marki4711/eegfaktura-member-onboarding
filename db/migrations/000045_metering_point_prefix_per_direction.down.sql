ALTER TABLE member_onboarding.registration_entrypoint
    DROP CONSTRAINT IF EXISTS registration_entrypoint_prefix_consumption_format,
    DROP CONSTRAINT IF EXISTS registration_entrypoint_prefix_production_format;

ALTER TABLE member_onboarding.registration_entrypoint
    DROP COLUMN IF EXISTS metering_point_prefix_consumption,
    DROP COLUMN IF EXISTS metering_point_prefix_production;
