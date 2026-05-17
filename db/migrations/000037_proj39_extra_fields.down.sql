-- Rollback PROJ-39. Daten in den neuen Spalten gehen verloren.

ALTER TABLE member_onboarding.metering_point
    DROP COLUMN IF EXISTS address_city,
    DROP COLUMN IF EXISTS address_zip,
    DROP COLUMN IF EXISTS address_street_number,
    DROP COLUMN IF EXISTS address_street;

ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS titel_nach;
