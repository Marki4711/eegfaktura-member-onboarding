-- PROJ-49 down migration. Restore Application-Level-Spalten als NULL;
-- Bestandsdaten auf metering_point werden NICHT zurückgespielt.

ALTER TABLE member_onboarding.metering_point
    DROP COLUMN consumption_previous_year,
    DROP COLUMN consumption_forecast,
    DROP COLUMN feed_in_forecast,
    DROP COLUMN pv_power_kwp,
    DROP COLUMN feed_in_limit_present,
    DROP COLUMN feed_in_limit_kw;

ALTER TABLE member_onboarding.application
    ADD COLUMN consumption_previous_year BIGINT NULL,
    ADD COLUMN consumption_forecast      BIGINT NULL,
    ADD COLUMN feed_in_forecast          BIGINT NULL,
    ADD COLUMN pv_power_kwp              NUMERIC(7,2) NULL;
