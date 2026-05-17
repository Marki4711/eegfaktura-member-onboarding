-- PROJ-44: Netzbetreiber-Vollmacht.
-- Per-EEG konfigurierbar via field_config (Standard: hidden).
-- Default FALSE für Bestandsdaten — eine fehlende Spalte heißt
-- explizit „nicht erteilt", nicht „unbekannt".

ALTER TABLE member_onboarding.application
    ADD COLUMN network_operator_authorization     BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN network_operator_authorization_at  TIMESTAMPTZ NULL;
