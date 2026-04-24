CREATE TABLE member_onboarding.external_api_key (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    rc_number         TEXT         NOT NULL UNIQUE
                                   REFERENCES member_onboarding.registration_entrypoint(rc_number)
                                   ON DELETE CASCADE,
    key_hash          VARCHAR(64)  NOT NULL,
    revoked_at        TIMESTAMPTZ,
    last_generated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    daily_count       INT          NOT NULL DEFAULT 0,
    quota_date        DATE
);
