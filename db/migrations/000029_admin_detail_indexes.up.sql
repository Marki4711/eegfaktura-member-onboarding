-- +migrate Up
-- Composite indexes on (application_id, created_at) for the three "list this
-- application's children, ordered by time" queries that fire on every admin
-- detail load (status_log + document_consent + metering_point). Without
-- them, Postgres falls back to heap-fetch + sort per query — death by a
-- thousand cuts as detail-view traffic grows.
CREATE INDEX IF NOT EXISTS idx_status_log_app_created
    ON member_onboarding.status_log(application_id, created_at);

CREATE INDEX IF NOT EXISTS idx_document_consent_app_consented
    ON member_onboarding.document_consent(application_id, consented_at);

CREATE INDEX IF NOT EXISTS idx_metering_point_app_created
    ON member_onboarding.metering_point(application_id, created_at);
