-- Add member type and organisation-specific fields
ALTER TABLE member_onboarding.application
    ADD COLUMN member_type     VARCHAR(50)  NOT NULL DEFAULT 'private',
    ADD COLUMN company_name    VARCHAR(255),
    ADD COLUMN uid_number      VARCHAR(50),
    ADD COLUMN register_number VARCHAR(50);

-- firstname and lastname are no longer required at DB level;
-- business rules enforce them per member_type in the application layer.
ALTER TABLE member_onboarding.application
    ALTER COLUMN firstname DROP NOT NULL,
    ALTER COLUMN lastname  DROP NOT NULL;
