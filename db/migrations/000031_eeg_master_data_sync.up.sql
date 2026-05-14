-- PROJ-32: track when the EEG master-data was last synced from the
-- eegFaktura core. NULL means "never synced yet" — the admin UI then
-- shows a bootstrap prompt instead of a stand-vom-Anzeige.

ALTER TABLE member_onboarding.registration_entrypoint
    ADD COLUMN last_synced_from_core_at TIMESTAMPTZ NULL;
