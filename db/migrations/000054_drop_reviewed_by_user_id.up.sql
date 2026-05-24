-- Data-Model-Slimming-Audit (2026-05-24, Audit-TODO §6): Spalte ist
-- echtes Tot-Datum.
--
-- application.reviewed_by_user_id wird per COALESCE in UpdateStatusAdminTx
-- gesetzt und in das ApplicationDetail-JSON serialisiert, aber NIRGENDS
-- konsumiert (kein Admin-UI-Rendering, kein PDF, kein Mail, keine
-- Geschäftslogik). Die echte Audit-Quelle "Wer hat den Status geändert"
-- ist status_log.changed_by_user_id (status_log_repo.go schreibt es,
-- admin-status-log.tsx rendert es).

ALTER TABLE member_onboarding.application
    DROP COLUMN IF EXISTS reviewed_by_user_id;
