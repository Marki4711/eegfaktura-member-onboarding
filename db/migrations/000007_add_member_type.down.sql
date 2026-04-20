-- Restore NOT NULL on firstname/lastname (fill blanks first to avoid constraint violation)
UPDATE member_onboarding.application SET firstname = '' WHERE firstname IS NULL;
UPDATE member_onboarding.application SET lastname  = '' WHERE lastname  IS NULL;

ALTER TABLE member_onboarding.application
    ALTER COLUMN firstname SET NOT NULL,
    ALTER COLUMN lastname  SET NOT NULL;

-- Remove organisation fields
ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS member_type,
    DROP COLUMN IF EXISTS company_name,
    DROP COLUMN IF EXISTS uid_number,
    DROP COLUMN IF EXISTS register_number;
