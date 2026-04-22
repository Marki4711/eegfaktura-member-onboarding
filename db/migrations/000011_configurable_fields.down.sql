DROP TABLE IF EXISTS member_onboarding.field_config;

ALTER TABLE member_onboarding.metering_point
    DROP COLUMN IF EXISTS transformer,
    DROP COLUMN IF EXISTS installation_number,
    DROP COLUMN IF EXISTS installation_name;

ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS membership_start_date,
    DROP COLUMN IF EXISTS persons_in_household,
    DROP COLUMN IF EXISTS consumption_previous_year,
    DROP COLUMN IF EXISTS consumption_forecast,
    DROP COLUMN IF EXISTS feed_in_forecast,
    DROP COLUMN IF EXISTS pv_power_kwp,
    DROP COLUMN IF EXISTS heat_pump,
    DROP COLUMN IF EXISTS electric_vehicle,
    DROP COLUMN IF EXISTS electric_hot_water;
