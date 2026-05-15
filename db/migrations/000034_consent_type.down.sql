ALTER TABLE member_onboarding.document_consent
    DROP CONSTRAINT IF EXISTS document_consent_consent_type_check;

ALTER TABLE member_onboarding.document_consent
    DROP COLUMN IF EXISTS consent_type;
