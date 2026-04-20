ALTER TABLE member_onboarding.application
    ADD COLUMN iban VARCHAR(34),
    ADD COLUMN account_holder VARCHAR(255),
    ADD COLUMN sepa_mandate_accepted BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN sepa_mandate_accepted_at TIMESTAMP WITH TIME ZONE;
