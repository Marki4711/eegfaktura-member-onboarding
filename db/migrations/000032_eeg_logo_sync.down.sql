ALTER TABLE member_onboarding.registration_entrypoint
    DROP COLUMN IF EXISTS eeg_logo_bytes,
    DROP COLUMN IF EXISTS eeg_logo_mime,
    DROP COLUMN IF EXISTS eeg_logo_synced_at;
