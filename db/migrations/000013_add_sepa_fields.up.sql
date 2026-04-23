ALTER TABLE member_onboarding.registration_entrypoint
  ADD COLUMN eeg_name          TEXT         NULL,
  ADD COLUMN eeg_street        TEXT         NULL,
  ADD COLUMN eeg_street_number VARCHAR(20)  NULL,
  ADD COLUMN eeg_zip           VARCHAR(20)  NULL,
  ADD COLUMN eeg_city          TEXT         NULL,
  ADD COLUMN creditor_id       VARCHAR(35)  NULL,
  ADD COLUMN sepa_mandate_enabled BOOLEAN   NOT NULL DEFAULT FALSE;
