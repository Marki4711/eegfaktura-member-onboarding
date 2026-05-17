-- PROJ-42: Detail-Erfassung für E-Fahrzeuge.
-- Beide Felder sind nur sinnvoll wenn electric_vehicle = TRUE.
-- Service-Layer setzt sie auf NULL, falls electric_vehicle != TRUE.
-- Keine DB-Constraints — der Service ist die einzige Schreibstelle.

ALTER TABLE member_onboarding.application
    ADD COLUMN electric_vehicle_count      INT,
    ADD COLUMN electric_vehicle_annual_km  INT;
