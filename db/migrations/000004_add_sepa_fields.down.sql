ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS iban,
    DROP COLUMN IF EXISTS account_holder,
    DROP COLUMN IF EXISTS sepa_mandate_accepted,
    DROP COLUMN IF EXISTS sepa_mandate_accepted_at;
