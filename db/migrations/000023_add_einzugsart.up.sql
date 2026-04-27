ALTER TABLE member_onboarding.application
    ADD COLUMN einzugsart        VARCHAR(20)  NOT NULL DEFAULT 'core',
    ADD COLUMN bank_name         VARCHAR(255),
    ADD COLUMN mandate_reference VARCHAR(255),
    ADD COLUMN mandate_date      DATE;
