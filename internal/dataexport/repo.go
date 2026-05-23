package dataexport

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// =====================================================================
// CONFIG REPOSITORY
// =====================================================================

type ConfigRepository struct {
	db *sql.DB
}

func NewConfigRepository(db *sql.DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

// Create inserts a new config and returns its ID.
func (r *ConfigRepository) Create(cfg *shared.DataExportConfig) (uuid.UUID, error) {
	id := uuid.New()
	now := time.Now()
	_, err := r.db.Exec(`
		INSERT INTO member_onboarding.data_export_config
			(id, rc_number, plugin_type, name, config, is_obsolete, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		id, cfg.RCNumber, cfg.PluginType, cfg.Name, cfg.Config, cfg.IsObsolete, now)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert config: %w", err)
	}
	return id, nil
}

// GetByID returns the config with the given ID (excluding soft-deleted).
func (r *ConfigRepository) GetByID(id uuid.UUID) (*shared.DataExportConfig, error) {
	row := r.db.QueryRow(`
		SELECT id, rc_number, plugin_type, name, config, is_obsolete,
		       deleted_at, created_at, updated_at
		FROM member_onboarding.data_export_config
		WHERE id = $1 AND deleted_at IS NULL`, id)
	return scanConfig(row)
}

// GetByIDIncludingDeleted returns the config regardless of soft-delete status.
// Used for job execution that already has a snapshot — needed for audit display.
func (r *ConfigRepository) GetByIDIncludingDeleted(id uuid.UUID) (*shared.DataExportConfig, error) {
	row := r.db.QueryRow(`
		SELECT id, rc_number, plugin_type, name, config, is_obsolete,
		       deleted_at, created_at, updated_at
		FROM member_onboarding.data_export_config
		WHERE id = $1`, id)
	return scanConfig(row)
}

// ListByRCNumber returns all non-deleted configs for the given EEG.
func (r *ConfigRepository) ListByRCNumber(rcNumber string) ([]shared.DataExportConfig, error) {
	rows, err := r.db.Query(`
		SELECT id, rc_number, plugin_type, name, config, is_obsolete,
		       deleted_at, created_at, updated_at
		FROM member_onboarding.data_export_config
		WHERE rc_number = $1 AND deleted_at IS NULL
		ORDER BY plugin_type, name`, rcNumber)
	if err != nil {
		return nil, fmt.Errorf("query configs: %w", err)
	}
	defer rows.Close()

	var out []shared.DataExportConfig
	for rows.Next() {
		cfg, err := scanConfigRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *cfg)
	}
	return out, rows.Err()
}

// CountByRCNumber returns the number of non-deleted configs for the EEG.
// Used to enforce DataExportMaxConfigsPerEEG.
func (r *ConfigRepository) CountByRCNumber(rcNumber string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM member_onboarding.data_export_config
		WHERE rc_number = $1 AND deleted_at IS NULL`, rcNumber).Scan(&count)
	return count, err
}

// Update modifies an existing config.
func (r *ConfigRepository) Update(id uuid.UUID, name string, config []byte) error {
	res, err := r.db.Exec(`
		UPDATE member_onboarding.data_export_config
		SET name = $2, config = $3, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, id, name, config)
	if err != nil {
		return fmt.Errorf("update config: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// SoftDelete marks the config as deleted (deleted_at = NOW).
func (r *ConfigRepository) SoftDelete(id uuid.UUID) error {
	res, err := r.db.Exec(`
		UPDATE member_onboarding.data_export_config
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return fmt.Errorf("soft-delete config: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// MarkObsolete sets is_obsolete=true for all configs whose plugin_type is
// no longer in the registered list. Called at backend startup.
func (r *ConfigRepository) MarkObsolete(registeredTypes []string) (int64, error) {
	res, err := r.db.Exec(`
		UPDATE member_onboarding.data_export_config
		SET is_obsolete = TRUE, updated_at = NOW()
		WHERE deleted_at IS NULL
		  AND is_obsolete = FALSE
		  AND plugin_type <> ALL($1)`, pq.Array(registeredTypes))
	if err != nil {
		return 0, fmt.Errorf("mark obsolete: %w", err)
	}
	return res.RowsAffected()
}

// HardDeleteOldSoftDeleted removes config rows that were soft-deleted more
// than the threshold ago. Called by the cleanup cron (DSGVO § 132 BAO).
func (r *ConfigRepository) HardDeleteOldSoftDeleted(olderThan time.Time) (int64, error) {
	res, err := r.db.Exec(`
		DELETE FROM member_onboarding.data_export_config
		WHERE deleted_at IS NOT NULL AND deleted_at < $1`, olderThan)
	if err != nil {
		return 0, fmt.Errorf("hard delete old configs: %w", err)
	}
	return res.RowsAffected()
}

func scanConfig(row *sql.Row) (*shared.DataExportConfig, error) {
	var cfg shared.DataExportConfig
	err := row.Scan(&cfg.ID, &cfg.RCNumber, &cfg.PluginType, &cfg.Name, &cfg.Config,
		&cfg.IsObsolete, &cfg.DeletedAt, &cfg.CreatedAt, &cfg.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan config: %w", err)
	}
	return &cfg, nil
}

func scanConfigRows(rows *sql.Rows) (*shared.DataExportConfig, error) {
	var cfg shared.DataExportConfig
	err := rows.Scan(&cfg.ID, &cfg.RCNumber, &cfg.PluginType, &cfg.Name, &cfg.Config,
		&cfg.IsObsolete, &cfg.DeletedAt, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan config row: %w", err)
	}
	return &cfg, nil
}

// =====================================================================
// JOB REPOSITORY
// =====================================================================

type JobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{db: db}
}

// Create inserts a new job (status='queued') and returns its ID.
func (r *JobRepository) Create(job *shared.DataExportJob) (uuid.UUID, error) {
	id := uuid.New()
	_, err := r.db.Exec(`
		INSERT INTO member_onboarding.data_export_job
			(id, rc_number, config_id, config_snapshot, plugin_type,
			 application_ids, status, admin_user_id, total_count)
		VALUES ($1, $2, $3, $4, $5, $6, 'queued', $7, $8)`,
		id, job.RCNumber, job.ConfigID, job.ConfigSnapshot, job.PluginType,
		pq.Array(uuidsToStrings(job.ApplicationIDs)), job.AdminUserID, job.TotalCount)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert job: %w", err)
	}
	return id, nil
}

// GetByID returns the job by ID.
func (r *JobRepository) GetByID(id uuid.UUID) (*shared.DataExportJob, error) {
	row := r.db.QueryRow(`
		SELECT id, rc_number, config_id, config_snapshot, plugin_type,
		       application_ids, status, admin_user_id,
		       processed_count, total_count, result_summary, error_message,
		       retry_count, created_at, started_at, finished_at
		FROM member_onboarding.data_export_job
		WHERE id = $1`, id)
	return scanJob(row)
}

// CountActiveByRCNumber returns the number of jobs in status queued or running
// for the EEG. Used for concurrency-limit soft-check.
func (r *JobRepository) CountActiveByRCNumber(rcNumber string) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM member_onboarding.data_export_job
		WHERE rc_number = $1 AND status IN ('queued','running')`, rcNumber).Scan(&count)
	return count, err
}

// PickupQueued claims the oldest queued job atomically (FOR UPDATE SKIP LOCKED),
// updates status='running' + started_at=NOW(), and returns the job. Returns
// (nil, nil) when no queued job is available.
//
// Multi-replica-safe: two workers calling this concurrently will pick
// different jobs (or one returns nil).
func (r *JobRepository) PickupQueued() (*shared.DataExportJob, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin pickup tx: %w", err)
	}
	defer tx.Rollback()

	// Pick the oldest queued job, skipping any currently-locked rows.
	row := tx.QueryRow(`
		SELECT id, rc_number, config_id, config_snapshot, plugin_type,
		       application_ids, status, admin_user_id,
		       processed_count, total_count, result_summary, error_message,
		       retry_count, created_at, started_at, finished_at
		FROM member_onboarding.data_export_job
		WHERE status = 'queued'
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED`)
	job, err := scanJob(row)
	if errors.Is(err, shared.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Mark running.
	_, err = tx.Exec(`
		UPDATE member_onboarding.data_export_job
		SET status = 'running', started_at = NOW()
		WHERE id = $1`, job.ID)
	if err != nil {
		return nil, fmt.Errorf("mark running: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit pickup: %w", err)
	}

	job.Status = shared.DataExportJobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	return job, nil
}

// UpdateProgress updates only the processed_count for a running job.
// Worker calls this periodically (every ~50 items).
func (r *JobRepository) UpdateProgress(id uuid.UUID, processed int) error {
	_, err := r.db.Exec(`
		UPDATE member_onboarding.data_export_job
		SET processed_count = $2
		WHERE id = $1 AND status = 'running'`, id, processed)
	return err
}

// MarkDone finalises a successful run.
func (r *JobRepository) MarkDone(id uuid.UUID, processed int, summary []byte) error {
	_, err := r.db.Exec(`
		UPDATE member_onboarding.data_export_job
		SET status = 'done',
		    processed_count = $2,
		    result_summary = $3,
		    finished_at = NOW()
		WHERE id = $1`, id, processed, summary)
	if err != nil {
		return fmt.Errorf("mark done: %w", err)
	}
	return nil
}

// MarkFailed finalises a failed run.
func (r *JobRepository) MarkFailed(id uuid.UUID, errMsg string) error {
	_, err := r.db.Exec(`
		UPDATE member_onboarding.data_export_job
		SET status = 'failed',
		    error_message = $2,
		    finished_at = NOW()
		WHERE id = $1`, id, errMsg)
	return err
}

// RecoverZombies finds running jobs older than the threshold and marks
// them as failed (with retry_count++). Returns the affected row count.
func (r *JobRepository) RecoverZombies(threshold time.Time) (int64, error) {
	res, err := r.db.Exec(`
		UPDATE member_onboarding.data_export_job
		SET status = 'failed',
		    error_message = 'cleanup: zombie — worker did not finish',
		    retry_count = retry_count + 1,
		    finished_at = NOW()
		WHERE status = 'running'
		  AND started_at < $1`, threshold)
	if err != nil {
		return 0, fmt.Errorf("recover zombies: %w", err)
	}
	return res.RowsAffected()
}

// MarkExpired transitions done-jobs whose result BLOB was deleted to status='expired'.
func (r *JobRepository) MarkExpired(jobIDs []uuid.UUID) (int64, error) {
	if len(jobIDs) == 0 {
		return 0, nil
	}
	res, err := r.db.Exec(`
		UPDATE member_onboarding.data_export_job
		SET status = 'expired'
		WHERE id = ANY($1) AND status = 'done'`, pq.Array(uuidsToStrings(jobIDs)))
	if err != nil {
		return 0, fmt.Errorf("mark expired: %w", err)
	}
	return res.RowsAffected()
}

// ListByRCNumber returns jobs for the EEG with optional status filter + date range.
// Pagination via cursor (createdBefore).
func (r *JobRepository) ListByRCNumber(rcNumber string, status string, since, until *time.Time, createdBefore *time.Time, limit int) ([]shared.DataExportJob, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `SELECT id, rc_number, config_id, config_snapshot, plugin_type,
		       application_ids, status, admin_user_id,
		       processed_count, total_count, result_summary, error_message,
		       retry_count, created_at, started_at, finished_at
		FROM member_onboarding.data_export_job
		WHERE rc_number = $1`
	args := []interface{}{rcNumber}
	argN := 2

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, status)
		argN++
	}
	if since != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argN)
		args = append(args, *since)
		argN++
	}
	if until != nil {
		query += fmt.Sprintf(" AND created_at < $%d", argN)
		args = append(args, *until)
		argN++
	}
	if createdBefore != nil {
		query += fmt.Sprintf(" AND created_at < $%d", argN)
		args = append(args, *createdBefore)
		argN++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argN)
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs: %w", err)
	}
	defer rows.Close()

	var out []shared.DataExportJob
	for rows.Next() {
		job, err := scanJobRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *job)
	}
	return out, rows.Err()
}

// CountFailedSince returns the number of jobs with status='failed' since
// the given time (used for the BackOffice Failed-Jobs-Badge).
func (r *JobRepository) CountFailedSince(rcNumber string, since time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM member_onboarding.data_export_job
		WHERE rc_number = $1 AND status = 'failed' AND created_at >= $2`,
		rcNumber, since).Scan(&count)
	return count, err
}

func scanJob(row *sql.Row) (*shared.DataExportJob, error) {
	var job shared.DataExportJob
	var appIDs pq.StringArray
	err := row.Scan(&job.ID, &job.RCNumber, &job.ConfigID, &job.ConfigSnapshot, &job.PluginType,
		&appIDs, &job.Status, &job.AdminUserID,
		&job.ProcessedCount, &job.TotalCount, &job.ResultSummary, &job.ErrorMessage,
		&job.RetryCount, &job.CreatedAt, &job.StartedAt, &job.FinishedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan job: %w", err)
	}
	job.ApplicationIDs = stringsToUUIDs(appIDs)
	return &job, nil
}

func scanJobRows(rows *sql.Rows) (*shared.DataExportJob, error) {
	var job shared.DataExportJob
	var appIDs pq.StringArray
	err := rows.Scan(&job.ID, &job.RCNumber, &job.ConfigID, &job.ConfigSnapshot, &job.PluginType,
		&appIDs, &job.Status, &job.AdminUserID,
		&job.ProcessedCount, &job.TotalCount, &job.ResultSummary, &job.ErrorMessage,
		&job.RetryCount, &job.CreatedAt, &job.StartedAt, &job.FinishedAt)
	if err != nil {
		return nil, fmt.Errorf("scan job row: %w", err)
	}
	job.ApplicationIDs = stringsToUUIDs(appIDs)
	return &job, nil
}

// =====================================================================
// RESULT REPOSITORY (file BLOBs with TTL)
// =====================================================================

type ResultRepository struct {
	db *sql.DB
}

func NewResultRepository(db *sql.DB) *ResultRepository {
	return &ResultRepository{db: db}
}

// Create inserts the result BLOB for a job.
func (r *ResultRepository) Create(res *shared.DataExportResult) error {
	_, err := r.db.Exec(`
		INSERT INTO member_onboarding.data_export_result
			(job_id, file_name, mime_type, file_bytes, file_size, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		res.JobID, res.FileName, res.MimeType, res.FileBytes, res.FileSize, res.ExpiresAt)
	if err != nil {
		return fmt.Errorf("insert result: %w", err)
	}
	return nil
}

// GetByJobID returns the result for a job, or shared.ErrNotFound if expired/missing.
func (r *ResultRepository) GetByJobID(jobID uuid.UUID) (*shared.DataExportResult, error) {
	row := r.db.QueryRow(`
		SELECT job_id, file_name, mime_type, file_bytes, file_size,
		       expires_at, downloaded_at, created_at
		FROM member_onboarding.data_export_result
		WHERE job_id = $1`, jobID)
	var res shared.DataExportResult
	err := row.Scan(&res.JobID, &res.FileName, &res.MimeType, &res.FileBytes, &res.FileSize,
		&res.ExpiresAt, &res.DownloadedAt, &res.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, shared.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan result: %w", err)
	}
	return &res, nil
}

// GetMetadataByJobID returns the file_name + size without loading the BLOB.
// Used by job-status responses to populate result fields without DB pressure.
func (r *ResultRepository) GetMetadataByJobID(jobID uuid.UUID) (fileName string, fileSize int, exists bool, err error) {
	err = r.db.QueryRow(`
		SELECT file_name, file_size
		FROM member_onboarding.data_export_result
		WHERE job_id = $1`, jobID).Scan(&fileName, &fileSize)
	if errors.Is(err, sql.ErrNoRows) {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, err
	}
	return fileName, fileSize, true, nil
}

// MarkDownloaded sets downloaded_at to now. Best-effort, idempotent.
func (r *ResultRepository) MarkDownloaded(jobID uuid.UUID) error {
	_, err := r.db.Exec(`
		UPDATE member_onboarding.data_export_result
		SET downloaded_at = NOW()
		WHERE job_id = $1 AND downloaded_at IS NULL`, jobID)
	return err
}

// DeleteExpired removes expired results and returns the IDs of jobs whose
// result was deleted (so the caller can mark them as 'expired').
func (r *ResultRepository) DeleteExpired(now time.Time) ([]uuid.UUID, error) {
	rows, err := r.db.Query(`
		DELETE FROM member_onboarding.data_export_result
		WHERE expires_at < $1
		RETURNING job_id`, now)
	if err != nil {
		return nil, fmt.Errorf("delete expired results: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// =====================================================================
// Helpers
// =====================================================================

func uuidsToStrings(in []uuid.UUID) []string {
	out := make([]string, len(in))
	for i, u := range in {
		out[i] = u.String()
	}
	return out
}

func stringsToUUIDs(in pq.StringArray) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(in))
	for _, s := range in {
		if u, err := uuid.Parse(s); err == nil {
			out = append(out, u)
		}
	}
	return out
}

// MarshalJSONB is a helper for callers that need to wrap a Go map as JSONB.
func MarshalJSONB(m map[string]interface{}) ([]byte, error) {
	return json.Marshal(m)
}
