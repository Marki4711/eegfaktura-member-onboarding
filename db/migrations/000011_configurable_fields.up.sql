-- Add configurable application-level fields
ALTER TABLE member_onboarding.application
    ADD COLUMN membership_start_date    DATE,
    ADD COLUMN persons_in_household     INTEGER,
    ADD COLUMN consumption_previous_year INTEGER,
    ADD COLUMN consumption_forecast     INTEGER,
    ADD COLUMN feed_in_forecast         INTEGER,
    ADD COLUMN pv_power_kwp             NUMERIC(10, 3),
    ADD COLUMN heat_pump                BOOLEAN,
    ADD COLUMN electric_vehicle         BOOLEAN,
    ADD COLUMN electric_hot_water       BOOLEAN;

-- Add configurable metering-point-level fields
ALTER TABLE member_onboarding.metering_point
    ADD COLUMN transformer         VARCHAR(100),
    ADD COLUMN installation_number VARCHAR(50),
    ADD COLUMN installation_name   VARCHAR(100);

-- Sparse field configuration per EEG
-- Only non-default settings are stored; missing rows imply the default state.
CREATE TABLE member_onboarding.field_config (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    rc_number  VARCHAR(50)  NOT NULL REFERENCES member_onboarding.registration_entrypoint(rc_number) ON DELETE CASCADE,
    field_name VARCHAR(100) NOT NULL,
    state      VARCHAR(20)  NOT NULL CHECK (state IN ('hidden', 'optional', 'required')),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (rc_number, field_name)
);

CREATE INDEX idx_field_config_rc_number ON member_onboarding.field_config(rc_number);
