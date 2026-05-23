package dataexport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ApplicationLoader is the dependency used to fetch applications + their
// metering points. Implemented by application.ApplicationRepository wrapper.
type ApplicationLoader interface {
	LoadForExport(ctx context.Context, rcNumber string, ids []uuid.UUID) ([]ApplicationSnapshot, error)
	LoadRecentImportedForPreview(ctx context.Context, rcNumber string, limit int) ([]ApplicationSnapshot, error)
}

// =====================================================================
// CONFIG SERVICE
// =====================================================================

type ConfigService struct {
	repo    *ConfigRepository
	appRepo ApplicationLoader
}

func NewConfigService(repo *ConfigRepository, appRepo ApplicationLoader) *ConfigService {
	return &ConfigService{repo: repo, appRepo: appRepo}
}

// CreateConfig validates and inserts a new plugin config.
func (s *ConfigService) CreateConfig(rcNumber string, req shared.DataExportConfigRequest) (*shared.DataExportConfig, error) {
	plugin := Get(req.PluginType)
	if plugin == nil {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"pluginType": fmt.Sprintf("unbekannter Plugin-Typ: %q", req.PluginType),
		})
	}
	if err := plugin.ValidateConfig(req.Config); err != nil {
		return nil, err
	}

	// Enforce per-EEG config limit.
	count, err := s.repo.CountByRCNumber(rcNumber)
	if err != nil {
		return nil, fmt.Errorf("count configs: %w", err)
	}
	if count >= shared.DataExportMaxConfigsPerEEG {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"limit": fmt.Sprintf("maximal %d Konfigurationen pro EEG erlaubt", shared.DataExportMaxConfigsPerEEG),
		})
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	cfg := &shared.DataExportConfig{
		RCNumber:   rcNumber,
		PluginType: req.PluginType,
		Name:       req.Name,
		Config:     configJSON,
	}
	id, err := s.repo.Create(cfg)
	if err != nil {
		return nil, err
	}
	cfg.ID = id
	cfg.CreatedAt = time.Now()
	cfg.UpdatedAt = cfg.CreatedAt
	return cfg, nil
}

// UpdateConfig validates and updates an existing config (tenant-checked).
func (s *ConfigService) UpdateConfig(id uuid.UUID, rcNumber string, req shared.DataExportConfigRequest) (*shared.DataExportConfig, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing.RCNumber != rcNumber {
		return nil, shared.ErrForbidden
	}
	if existing.PluginType != req.PluginType {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"pluginType": "Plugin-Typ kann nicht geändert werden",
		})
	}
	plugin := Get(req.PluginType)
	if plugin == nil {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"pluginType": "Plugin nicht mehr verfügbar",
		})
	}
	if err := plugin.ValidateConfig(req.Config); err != nil {
		return nil, err
	}

	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	if err := s.repo.Update(id, req.Name, configJSON); err != nil {
		return nil, err
	}
	return s.repo.GetByID(id)
}

// DeleteConfig soft-deletes a config (tenant-checked).
func (s *ConfigService) DeleteConfig(id uuid.UUID, rcNumber string) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if existing.RCNumber != rcNumber {
		return shared.ErrForbidden
	}
	return s.repo.SoftDelete(id)
}

// GetConfig retrieves a config (tenant-checked).
func (s *ConfigService) GetConfig(id uuid.UUID, rcNumber string) (*shared.DataExportConfig, error) {
	cfg, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if cfg.RCNumber != rcNumber {
		return nil, shared.ErrForbidden
	}
	return cfg, nil
}

// ListConfigs returns all non-deleted configs for the EEG.
func (s *ConfigService) ListConfigs(rcNumber string) ([]shared.DataExportConfig, error) {
	return s.repo.ListByRCNumber(rcNumber)
}

// Preview executes the plugin against up to 5 recent imported applications
// and returns a structured preview the admin UI can render directly.
func (s *ConfigService) Preview(ctx context.Context, rcNumber, pluginType string, config map[string]interface{}) (*shared.DataExportPreviewResponse, error) {
	plugin := Get(pluginType)
	if plugin == nil {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"pluginType": "unbekannter Plugin-Typ",
		})
	}
	if err := plugin.ValidateConfig(config); err != nil {
		return nil, err
	}

	apps, err := s.appRepo.LoadRecentImportedForPreview(ctx, rcNumber, 5)
	if err != nil {
		return nil, fmt.Errorf("load preview applications: %w", err)
	}

	note := ""
	if len(apps) == 0 {
		// Fall back to plugin's sample data if available.
		apps = plugin.PreviewSample()
		note = "Beispiel-Daten — sobald Sie Mitglieder importiert haben, sehen Sie hier echte Vorschau"
	}

	// Build a structured table by going through the same field-extraction
	// path the renderer uses, but emitting rows as maps instead of file bytes.
	headers, rows, err := buildPreviewTable(plugin, config, apps)
	if err != nil {
		return nil, err
	}
	return &shared.DataExportPreviewResponse{
		Headers: headers,
		Rows:    rows,
		Note:    note,
	}, nil
}

// MarkObsoletePluginsOnStartup is called by main.go after plugin registration
// to flag configs whose plugin_type is no longer registered.
func (s *ConfigService) MarkObsoletePluginsOnStartup() error {
	plugins := List()
	types := make([]string, len(plugins))
	for i, p := range plugins {
		types[i] = p.Type()
	}
	n, err := s.repo.MarkObsolete(types)
	if err != nil {
		return err
	}
	if n > 0 {
		slog.Info("dataexport: marked configs as obsolete", "count", n, "active_plugins", types)
	}
	return nil
}

// buildPreviewTable extracts structured preview data. Currently delegated
// to the excel plugin's renderer-equivalent (could be generalised later
// if other Download-style plugins emerge).
func buildPreviewTable(plugin Plugin, config map[string]interface{}, apps []ApplicationSnapshot) ([]string, []map[string]interface{}, error) {
	// We re-use the plugin's Process() to render bytes — but for preview we
	// want structured rows, not file bytes. Two paths:
	//  - Generic path: ask the plugin for a "preview-mode" via a separate
	//    interface (not yet defined)
	//  - Pragmatic path: only the Excel plugin needs preview today; we hard-
	//    code the call here.
	//
	// Since the framework is meant to be plugin-agnostic but only Excel
	// implements preview right now, we cast via a type-assertion to a
	// PreviewBuilder interface that the Excel plugin satisfies. Future
	// plugins can opt-in by implementing the same interface.
	pb, ok := plugin.(PreviewBuilder)
	if !ok {
		return nil, nil, shared.NewValidationError("Validation failed", map[string]string{
			"pluginType": "Vorschau ist für diesen Plugin-Typ nicht verfügbar",
		})
	}
	return pb.BuildPreviewTable(config, apps)
}

// PreviewBuilder is an opt-in interface for plugins that support live
// structured preview (Excel does; future push-plugins like Zoho do not
// need to).
type PreviewBuilder interface {
	BuildPreviewTable(config map[string]interface{}, apps []ApplicationSnapshot) (headers []string, rows []map[string]interface{}, err error)
}

// =====================================================================
// JOB SERVICE
// =====================================================================

type JobService struct {
	configRepo *ConfigRepository
	jobRepo    *JobRepository
	resultRepo *ResultRepository
	appRepo    ApplicationLoader
}

func NewJobService(configRepo *ConfigRepository, jobRepo *JobRepository, resultRepo *ResultRepository, appRepo ApplicationLoader) *JobService {
	return &JobService{
		configRepo: configRepo,
		jobRepo:    jobRepo,
		resultRepo: resultRepo,
		appRepo:    appRepo,
	}
}

// TriggerJob creates a new queued job from a config + list of application IDs.
// Performs soft concurrency-limit check (overshoot up to ~4-5 tolerated).
func (s *JobService) TriggerJob(rcNumber string, configID uuid.UUID, applicationIDs []uuid.UUID, adminUserID string) (*shared.DataExportJob, error) {
	if len(applicationIDs) == 0 {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"applicationIds": "mindestens eine Antrags-ID erforderlich",
		})
	}
	if len(applicationIDs) > shared.DataExportMaxApplications {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"applicationIds": fmt.Sprintf("maximal %d Anträge pro Bulk-Aktion", shared.DataExportMaxApplications),
		})
	}

	cfg, err := s.configRepo.GetByID(configID)
	if err != nil {
		return nil, err
	}
	if cfg.RCNumber != rcNumber {
		return nil, shared.ErrForbidden
	}
	if cfg.IsObsolete {
		return nil, shared.NewConflictError("Plugin nicht mehr verfügbar — Konfiguration kann nicht ausgeführt werden")
	}

	// Soft concurrency check.
	active, err := s.jobRepo.CountActiveByRCNumber(rcNumber)
	if err != nil {
		return nil, fmt.Errorf("count active jobs: %w", err)
	}
	if active >= shared.DataExportConcurrencyLimit {
		// Soft limit: still allow insert; queue grows beyond the soft cap.
		// Worker drains in FIFO. We log so the operator notices spikes.
		slog.Info("dataexport: concurrency soft limit reached, queueing",
			"rc_number", rcNumber, "active", active, "limit", shared.DataExportConcurrencyLimit)
	}

	job := &shared.DataExportJob{
		RCNumber:       rcNumber,
		ConfigID:       &cfg.ID,
		ConfigSnapshot: cfg.Config,
		PluginType:     cfg.PluginType,
		ApplicationIDs: applicationIDs,
		AdminUserID:    adminUserID,
		TotalCount:     len(applicationIDs),
	}
	id, err := s.jobRepo.Create(job)
	if err != nil {
		return nil, err
	}
	job.ID = id
	job.Status = shared.DataExportJobStatusQueued
	job.CreatedAt = time.Now()
	return job, nil
}

// GetJob returns a job (tenant-checked).
func (s *JobService) GetJob(id uuid.UUID, rcNumber string) (*shared.DataExportJob, error) {
	job, err := s.jobRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if job.RCNumber != rcNumber {
		return nil, shared.ErrForbidden
	}
	return job, nil
}

// ListJobs returns jobs for the EEG with optional filters + cursor.
func (s *JobService) ListJobs(rcNumber, status string, since, until, createdBefore *time.Time, limit int) ([]shared.DataExportJob, error) {
	return s.jobRepo.ListByRCNumber(rcNumber, status, since, until, createdBefore, limit)
}

// CountFailedSince returns the count of failed jobs in the time-window.
func (s *JobService) CountFailedSince(rcNumber string, since time.Time) (int, error) {
	return s.jobRepo.CountFailedSince(rcNumber, since)
}

// Retry creates a new queued job with the same snapshot + application IDs as
// the original. The original job is unaffected (audit-trail).
func (s *JobService) Retry(originalJobID uuid.UUID, rcNumber, adminUserID string) (*shared.DataExportJob, error) {
	original, err := s.GetJob(originalJobID, rcNumber)
	if err != nil {
		return nil, err
	}
	job := &shared.DataExportJob{
		RCNumber:       rcNumber,
		ConfigID:       original.ConfigID,
		ConfigSnapshot: original.ConfigSnapshot,
		PluginType:     original.PluginType,
		ApplicationIDs: original.ApplicationIDs,
		AdminUserID:    adminUserID,
		TotalCount:     original.TotalCount,
	}
	id, err := s.jobRepo.Create(job)
	if err != nil {
		return nil, err
	}
	job.ID = id
	job.Status = shared.DataExportJobStatusQueued
	job.CreatedAt = time.Now()
	return job, nil
}

// LoadResult returns the result BLOB (tenant-checked via the job).
func (s *JobService) LoadResult(jobID uuid.UUID, rcNumber string) (*shared.DataExportResult, error) {
	job, err := s.GetJob(jobID, rcNumber)
	if err != nil {
		return nil, err
	}
	if job.Status != shared.DataExportJobStatusDone {
		return nil, shared.NewConflictError(fmt.Sprintf("job ist im Status %s, kein Ergebnis verfügbar", job.Status))
	}
	res, err := s.resultRepo.GetByJobID(jobID)
	if err != nil {
		return nil, err
	}
	// Best-effort mark as downloaded.
	_ = s.resultRepo.MarkDownloaded(jobID)
	return res, nil
}

// GetResultMetadata returns just file_name + size + exists flag for a job,
// without loading the BLOB (cheap for status/listing endpoints).
func (s *JobService) GetResultMetadata(jobID uuid.UUID) (fileName string, fileSize int, exists bool) {
	fn, fs, ex, err := s.resultRepo.GetMetadataByJobID(jobID)
	if err != nil {
		return "", 0, false
	}
	return fn, fs, ex
}
