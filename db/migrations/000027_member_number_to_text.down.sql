-- +migrate Down
-- Reverting will fail on rows whose member_number contains non-numeric
-- characters. Operator must clean those rows by hand before rolling back.
ALTER TABLE member_onboarding.application
    ALTER COLUMN member_number TYPE INT
    USING NULLIF(member_number, '')::INT;
