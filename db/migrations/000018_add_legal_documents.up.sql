CREATE TABLE member_onboarding.legal_document (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    rc_number  TEXT        NOT NULL REFERENCES member_onboarding.registration_entrypoint(rc_number) ON DELETE CASCADE,
    title      TEXT        NOT NULL,
    url        TEXT        NOT NULL,
    required   BOOLEAN     NOT NULL DEFAULT true,
    sort_order INTEGER     NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_legal_document_rc_number ON member_onboarding.legal_document(rc_number);

CREATE TABLE member_onboarding.document_consent (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id    UUID        NOT NULL REFERENCES member_onboarding.application(id) ON DELETE CASCADE,
    title             TEXT        NOT NULL,
    url               TEXT        NOT NULL,
    is_central_policy BOOLEAN     NOT NULL DEFAULT false,
    consented_at      TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_document_consent_application_id ON member_onboarding.document_consent(application_id);
