ALTER TABLE member_onboarding.metering_point
    DROP CONSTRAINT IF EXISTS metering_point_generation_type_check;

ALTER TABLE member_onboarding.metering_point
    DROP COLUMN IF EXISTS inverter_manufacturer,
    DROP COLUMN IF EXISTS battery_size_kwh,
    DROP COLUMN IF EXISTS generation_type;
