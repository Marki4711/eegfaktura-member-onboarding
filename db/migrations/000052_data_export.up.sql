-- PROJ-60: Datenweiterleitung an externe Systeme — Plugin-Framework
-- mit Excel/CSV-Plugin als erster Implementierung.
--
-- Drei Tabellen:
--   1. data_export_config — Plugin-Konfigurationen pro EEG (mit Soft-Delete)
--   2. data_export_job    — Async-Job-Queue + langlebiger Audit-Trail
--   3. data_export_result — Datei-BLOBs mit 24h-TTL für DownloadResults

-- ============================================================
-- 1. CONFIG: Plugin-Konfigurationen pro EEG
-- ============================================================
CREATE TABLE member_onboarding.data_export_config (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rc_number    TEXT NOT NULL,
    plugin_type  TEXT NOT NULL,
    name         TEXT NOT NULL,
    config       JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_obsolete  BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at   TIMESTAMPTZ NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (rc_number) REFERENCES member_onboarding.registration_entrypoint(rc_number) ON DELETE CASCADE
);

CREATE UNIQUE INDEX uniq_data_export_config_name_per_eeg
    ON member_onboarding.data_export_config(rc_number, name)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_data_export_config_rc_plugin
    ON member_onboarding.data_export_config(rc_number, plugin_type)
    WHERE deleted_at IS NULL;

-- ============================================================
-- 2. JOB: Async-Job-Queue + Audit-Trail (langlebig)
-- ============================================================
CREATE TABLE member_onboarding.data_export_job (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rc_number        TEXT NOT NULL,
    config_id        UUID NULL,
    config_snapshot  JSONB NOT NULL,
    plugin_type      TEXT NOT NULL,
    application_ids  UUID[] NOT NULL,
    status           TEXT NOT NULL CHECK (status IN ('queued','running','done','failed','expired')),
    admin_user_id    TEXT NOT NULL,
    processed_count  INTEGER NOT NULL DEFAULT 0,
    total_count      INTEGER NOT NULL,
    result_summary   JSONB NULL,
    error_message    TEXT NULL,
    retry_count      INTEGER NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at       TIMESTAMPTZ NULL,
    finished_at      TIMESTAMPTZ NULL,
    FOREIGN KEY (rc_number) REFERENCES member_onboarding.registration_entrypoint(rc_number) ON DELETE CASCADE,
    FOREIGN KEY (config_id) REFERENCES member_onboarding.data_export_config(id) ON DELETE SET NULL
);

CREATE INDEX idx_data_export_job_pickup
    ON member_onboarding.data_export_job(status, created_at)
    WHERE status = 'queued';

CREATE INDEX idx_data_export_job_active_per_eeg
    ON member_onboarding.data_export_job(rc_number, status)
    WHERE status IN ('queued','running');

CREATE INDEX idx_data_export_job_rc_created
    ON member_onboarding.data_export_job(rc_number, created_at DESC);

CREATE INDEX idx_data_export_job_zombie
    ON member_onboarding.data_export_job(started_at)
    WHERE status = 'running';

-- ============================================================
-- 3. RESULT: Datei-BLOBs mit TTL
-- ============================================================
CREATE TABLE member_onboarding.data_export_result (
    job_id        UUID PRIMARY KEY,
    file_name     TEXT NOT NULL,
    mime_type     TEXT NOT NULL,
    file_bytes    BYTEA NOT NULL,
    file_size     INTEGER NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    downloaded_at TIMESTAMPTZ NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (job_id) REFERENCES member_onboarding.data_export_job(id) ON DELETE CASCADE
);

CREATE INDEX idx_data_export_result_expires
    ON member_onboarding.data_export_result(expires_at);

COMMENT ON TABLE member_onboarding.data_export_config IS 'PROJ-60: Plugin-Konfigurationen pro EEG für Datenweiterleitung.';
COMMENT ON TABLE member_onboarding.data_export_job IS 'PROJ-60: Async-Job-Queue + langlebiger Audit-Trail für Datenweiterleitungs-Läufe.';
COMMENT ON TABLE member_onboarding.data_export_result IS 'PROJ-60: Datei-BLOBs mit 24h-TTL für DownloadResult-Plugins.';
