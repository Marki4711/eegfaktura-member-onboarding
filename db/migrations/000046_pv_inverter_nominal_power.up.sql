-- Optionales Feld pro Zählpunkt: Nennleistung des PV-Wechselrichters in kW.
-- Nur relevant für PRODUCTION-Zählpunkte mit generation_type='pv'. Service-
-- Layer cleart das Feld in allen anderen Fällen (analog zu pv_power_kwp).
--
-- Per PROJ-8 konfigurierbar (siehe knownConfigurableFields). Default
-- ist "hidden" — EEGs aktivieren das Feld pro Bedarf.

ALTER TABLE member_onboarding.metering_point
    ADD COLUMN inverter_power_kw NUMERIC NULL;

COMMENT ON COLUMN member_onboarding.metering_point.inverter_power_kw IS
    'Nennleistung PV-Wechselrichter in kW. NULL für CONSUMPTION oder Non-PV-Erzeuger. Service-Layer enforce.';
