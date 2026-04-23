ALTER TABLE member_onboarding.registration_entrypoint
  DROP COLUMN IF EXISTS eeg_name,
  DROP COLUMN IF EXISTS eeg_street,
  DROP COLUMN IF EXISTS eeg_street_number,
  DROP COLUMN IF EXISTS eeg_zip,
  DROP COLUMN IF EXISTS eeg_city,
  DROP COLUMN IF EXISTS creditor_id,
  DROP COLUMN IF EXISTS sepa_mandate_enabled;
