ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS has_contact_person,
    DROP COLUMN IF EXISTS contact_person_name,
    DROP COLUMN IF EXISTS contact_person_email,
    DROP COLUMN IF EXISTS contact_person_phone;
