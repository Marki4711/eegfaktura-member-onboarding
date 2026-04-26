-- Migration 000020: Widen energy consumption and feed-in fields from INTEGER to BIGINT.
-- INTEGER (INT4) overflows at ~2.1 billion. Large PV installations or annual totals in Wh can exceed this.
ALTER TABLE member_onboarding.application
    ALTER COLUMN consumption_previous_year TYPE BIGINT,
    ALTER COLUMN consumption_forecast      TYPE BIGINT,
    ALTER COLUMN feed_in_forecast          TYPE BIGINT;
