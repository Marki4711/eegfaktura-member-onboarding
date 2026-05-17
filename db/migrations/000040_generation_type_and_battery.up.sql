-- PROJ-45: Erzeugungsform + Batterie-Felder pro Erzeugungs-Zählpunkt.
--
-- generation_type ist Pflicht für PRODUCTION (Default 'pv' bei Backfill,
-- weil das der häufigste Fall ist) und MUSS NULL bei CONSUMPTION sein.
-- battery_size_kwh + inverter_manufacturer sind optional und nur sinnvoll
-- wenn generation_type='pv' — wird im Service-Layer durchgesetzt
-- (kein DB-Check, weil das die Logik unnötig hart macht).

ALTER TABLE member_onboarding.metering_point
    ADD COLUMN generation_type        VARCHAR(20) NULL,
    ADD COLUMN battery_size_kwh       NUMERIC(7,2) NULL,
    ADD COLUMN inverter_manufacturer  VARCHAR(100) NULL;

UPDATE member_onboarding.metering_point
SET generation_type = 'pv'
WHERE direction = 'PRODUCTION';

ALTER TABLE member_onboarding.metering_point
    ADD CONSTRAINT metering_point_generation_type_check
    CHECK (
        (direction = 'CONSUMPTION' AND generation_type IS NULL)
        OR
        (direction = 'PRODUCTION' AND generation_type IN ('pv','hydro','wind','biomass'))
    );
