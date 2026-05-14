-- PROJ-33: cache the EEG logo from the eegFaktura-billing service so the
-- approval + SEPA mandate PDFs can embed it without a per-render HTTP call.
--
-- BYTEA + MIME + timestamp. NULL bytes = no logo yet synced (or core has
-- none). PostgreSQL auto-TOASTs values > ~2 KB, so the page-level overhead
-- stays at zero for entrypoints without a logo. Onboarding code caps the
-- upload at 256 KB; a typical logo is 15–150 KB.

ALTER TABLE member_onboarding.registration_entrypoint
    ADD COLUMN eeg_logo_bytes BYTEA NULL,
    ADD COLUMN eeg_logo_mime TEXT NULL,
    ADD COLUMN eeg_logo_synced_at TIMESTAMPTZ NULL;
