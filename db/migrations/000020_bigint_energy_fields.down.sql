ALTER TABLE member_onboarding.application
    ALTER COLUMN consumption_previous_year TYPE INTEGER,
    ALTER COLUMN consumption_forecast      TYPE INTEGER,
    ALTER COLUMN feed_in_forecast          TYPE INTEGER;
