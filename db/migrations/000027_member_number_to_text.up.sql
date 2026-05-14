-- +migrate Up
-- Member numbers are now sourced from the eegFaktura core, where the
-- participantNumber column is VARCHAR — values like "A005" or "M-12" are
-- legitimate. Promote our member_number column from INT to TEXT so we can
-- store whatever the admin picked in the import dialog.
ALTER TABLE member_onboarding.application
    ALTER COLUMN member_number TYPE TEXT
    USING member_number::TEXT;
