-- PROJ-37: Genossenschaftsanteile (cooperative shares).
--
-- Three new columns on registration_entrypoint configure the feature
-- per EEG (toggle + mandatory minimum + price per share), one column
-- on application stores the count the member subscribed. Data stays
-- inside the onboarding — not part of the eegFaktura core payload or
-- Excel export.
--
-- All values are NULL when the feature is off for the EEG (toggle FALSE).
-- When toggle=TRUE both config values become mandatory at the
-- application layer (validated in SaveEEGSettings, not enforced via DB
-- — admin might first enable and then fill the values).

ALTER TABLE member_onboarding.registration_entrypoint
    ADD COLUMN cooperative_shares_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN cooperative_required_shares INT NULL
        CHECK (cooperative_required_shares IS NULL OR cooperative_required_shares > 0),
    ADD COLUMN cooperative_share_amount_cents BIGINT NULL
        CHECK (cooperative_share_amount_cents IS NULL OR cooperative_share_amount_cents > 0);

ALTER TABLE member_onboarding.application
    ADD COLUMN cooperative_shares_count INT NULL
        CHECK (cooperative_shares_count IS NULL OR cooperative_shares_count > 0);
