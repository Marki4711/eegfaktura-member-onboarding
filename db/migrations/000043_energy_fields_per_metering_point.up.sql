-- PROJ-49: Energie-Felder wandern von application zu metering_point.
-- Bestandswerte werden verworfen (Entscheidung 2c, abgestimmt 2026-05-17).

-- 1. Neue MP-Spalten anlegen
ALTER TABLE member_onboarding.metering_point
    ADD COLUMN consumption_previous_year BIGINT NULL,
    ADD COLUMN consumption_forecast      BIGINT NULL,
    ADD COLUMN feed_in_forecast          BIGINT NULL,
    ADD COLUMN pv_power_kwp              NUMERIC(7,2) NULL,
    ADD COLUMN feed_in_limit_present     BOOLEAN NULL,
    ADD COLUMN feed_in_limit_kw          NUMERIC(7,2) NULL;

-- 2. Alte Application-Level-Spalten droppen (Werte verworfen)
ALTER TABLE member_onboarding.application
    DROP COLUMN consumption_previous_year,
    DROP COLUMN consumption_forecast,
    DROP COLUMN feed_in_forecast,
    DROP COLUMN pv_power_kwp;

-- 3. Alte field_config-Einträge mit denselben Namen löschen.
-- Diese hatten Application-Scope; nach dem Refactoring werden sie als
-- MP-Scope neu angelegt — initial alle implicit hidden (kein DB-Eintrag
-- nötig, Default greift). EEGs aktivieren bewusst neu, was sie brauchen.
DELETE FROM member_onboarding.field_config
WHERE field_name IN (
    'consumption_previous_year',
    'consumption_forecast',
    'feed_in_forecast',
    'pv_power_kwp'
);
