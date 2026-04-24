ALTER TABLE member_onboarding.registration_entrypoint
  ADD COLUMN use_company_sepa_mandate BOOLEAN NOT NULL DEFAULT FALSE;
