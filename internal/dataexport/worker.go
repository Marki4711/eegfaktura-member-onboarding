package dataexport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/logfields"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// Worker polls the data_export_job queue and dispatches queued jobs to the
// appropriate plugin. Runs as an in-app goroutine pool started from
// cmd/server/main.go (no separate worker pod needed).
//
// Multi-replica-safe via FOR UPDATE SKIP LOCKED in JobRepository.PickupQueued.
type Worker struct {
	jobRepo    *JobRepository
	resultRepo *ResultRepository
	configRepo *ConfigRepository
	appLoader  ApplicationLoader
	mailer     FailureMailer
	poolSize   int
	pollEvery  time.Duration

	stop  chan struct{}
	wg    sync.WaitGroup
	once  sync.Once
}

// FailureMailer is invoked when a job fails. Implemented by mail.Service
// adapter in main.go.
type FailureMailer interface {
	SendDataExportFailure(ctx context.Context, job *shared.DataExportJob) error
}

// NoopFailureMailer is a fallback that just logs (used in tests or when
// SMTP is not configured).
type NoopFailureMailer struct{}

func (NoopFailureMailer) SendDataExportFailure(ctx context.Context, job *shared.DataExportJob) error {
	slog.Info("dataexport: failure mail suppressed (no mailer)", "job_id", job.ID)
	return nil
}

func NewWorker(jobRepo *JobRepository, resultRepo *ResultRepository, configRepo *ConfigRepository, appLoader ApplicationLoader, mailer FailureMailer, poolSize int, pollEvery time.Duration) *Worker {
	if poolSize <= 0 {
		poolSize = 3
	}
	if pollEvery <= 0 {
		pollEvery = 5 * time.Second
	}
	if mailer == nil {
		mailer = NoopFailureMailer{}
	}
	return &Worker{
		jobRepo:    jobRepo,
		resultRepo: resultRepo,
		configRepo: configRepo,
		appLoader:  appLoader,
		mailer:     mailer,
		poolSize:   poolSize,
		pollEvery:  pollEvery,
		stop:       make(chan struct{}),
	}
}

// Start launches the goroutine pool. Returns immediately.
func (w *Worker) Start(ctx context.Context) {
	for i := 0; i < w.poolSize; i++ {
		w.wg.Add(1)
		workerID := i
		go w.loop(ctx, workerID)
	}
	slog.Info("dataexport: worker pool started", "pool_size", w.poolSize, "poll_every", w.pollEvery)
}

// Stop signals all workers to stop accepting new jobs and waits for in-flight
// jobs to finish. Bounded by ctx.Done() — when the context cancels (e.g. K8s
// terminationGracePeriodSeconds elapsing) Stop returns even if jobs are
// still running; those jobs will be picked up as zombies by the cleanup
// CronJob within an hour.
//
// Called from main.go on SIGTERM BEFORE srv.Shutdown so the worker drains
// while HTTP is still up (admin can observe job-status until the very end).
// New TriggerJob/Retry calls are blocked in parallel via
// JobService.MarkShuttingDown so no fresh jobs land in the queue during the
// drain window — otherwise they would be guaranteed zombies.
func (w *Worker) Stop(ctx context.Context) error {
	w.once.Do(func() {
		close(w.stop)
	})
	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		slog.Info("dataexport: worker pool stopped (clean drain)")
		return nil
	case <-ctx.Done():
		slog.Warn("dataexport: worker pool stop deadline exceeded — in-flight jobs will be recovered by cleanup cron",
			"error", ctx.Err())
		return ctx.Err()
	}
}

func (w *Worker) loop(ctx context.Context, workerID int) {
	defer w.wg.Done()
	tick := time.NewTicker(w.pollEvery)
	defer tick.Stop()

	for {
		select {
		case <-w.stop:
			return
		case <-ctx.Done():
			return
		case <-tick.C:
			w.tryPickup(ctx, workerID)
		}
	}
}

func (w *Worker) tryPickup(ctx context.Context, workerID int) {
	job, err := w.jobRepo.PickupQueued()
	if err != nil {
		slog.Error("dataexport: pickup failed", "worker", workerID, "error", err)
		return
	}
	if job == nil {
		return // queue empty
	}
	slog.Info("dataexport: job picked up", "worker", workerID, "job_id", job.ID, "plugin", job.PluginType, "total", job.TotalCount)
	w.processJob(ctx, job)
}

func (w *Worker) processJob(ctx context.Context, job *shared.DataExportJob) {
	plugin := Get(job.PluginType)
	if plugin == nil {
		w.fail(ctx, job, "Plugin nicht mehr verfügbar — wurde seit Job-Erstellung entfernt", nil)
		return
	}

	// Load applications.
	apps, err := w.appLoader.LoadForExport(ctx, job.RCNumber, job.ApplicationIDs)
	if err != nil {
		w.fail(ctx, job, "Anträge konnten nicht geladen werden", err)
		return
	}

	// Decode config snapshot.
	var configMap map[string]interface{}
	if err := json.Unmarshal(job.ConfigSnapshot, &configMap); err != nil {
		w.fail(ctx, job, "Konfigurations-Snapshot ist beschädigt", err)
		return
	}

	// DSGVO audit log: detect exports containing sensitive personal data
	// (IBAN, Geburtsdatum). Emitted before processing so the audit trail
	// exists even if the job later fails. Admin-User-ID is part of the job
	// row, so the slog event ties admin → sensitive-export.
	if sens := detectSensitiveFields(configMap); len(sens) > 0 {
		slog.Info("dataexport: sensitive-export",
			logfields.Classification, logfields.ClassSensitiveExport,
			logfields.JobID, job.ID,
			logfields.RCNumber, job.RCNumber,
			logfields.AdminUserID, job.AdminUserID,
			logfields.PluginType, job.PluginType,
			logfields.ApplicationCount, len(apps),
			"sensitive_fields", sens,
		)
	}

	// Progress callback updates DB every ~50 items.
	progress := func(processed int) {
		if err := w.jobRepo.UpdateProgress(job.ID, processed); err != nil {
			slog.Warn("dataexport: progress update failed", "job_id", job.ID, "error", err)
		}
	}

	result, err := plugin.Process(ctx, configMap, apps, progress)
	if err != nil {
		w.fail(ctx, job, "Plugin-Verarbeitung fehlgeschlagen", err)
		return
	}

	// Persist result (only for DownloadResult — SyncResult writes nothing to result table).
	if dl, ok := result.(DownloadResult); ok {
		// Spec-conformant filename: {rc_number}-{config_name}-{YYYY-MM-DD}.{ext}.
		// Config name comes from configRepo (including soft-deleted, so an
		// admin-deleted config mid-job still yields a meaningful name);
		// fall back to plugin's original filename if the lookup fails.
		dl.FileName = w.buildFileName(job, dl.FileName)

		exp := time.Now().Add(shared.DataExportResultTTL)
		if err := w.resultRepo.Create(&shared.DataExportResult{
			JobID:     job.ID,
			FileName:  dl.FileName,
			MimeType:  dl.MimeType,
			FileBytes: dl.Bytes,
			FileSize:  len(dl.Bytes),
			ExpiresAt: exp,
		}); err != nil {
			w.fail(ctx, job, "Ergebnis konnte nicht gespeichert werden", err)
			return
		}
	}

	summaryJSON, err := json.Marshal(result.Summary())
	if err != nil {
		w.fail(ctx, job, "Ergebnis-Zusammenfassung konnte nicht serialisiert werden", err)
		return
	}
	if err := w.jobRepo.MarkDone(job.ID, len(apps), summaryJSON); err != nil {
		slog.Error("dataexport: mark done failed", "job_id", job.ID, "error", err)
		return
	}
	slog.Info("dataexport: job done", "job_id", job.ID, "processed", len(apps))
}

// fail records a user-safe message in the job row and logs internal details
// separately. The userMsg is shown to the admin in the BackOffice UI; cause
// is the wrapped Go error (may contain DB internals, library names, etc.)
// and only ever appears in structured logs.
func (w *Worker) fail(ctx context.Context, job *shared.DataExportJob, userMsg string, cause error) {
	slog.Error("dataexport: job failed",
		"job_id", job.ID,
		"user_msg", userMsg,
		"cause", cause,
	)
	if err := w.jobRepo.MarkFailed(job.ID, userMsg); err != nil {
		slog.Error("dataexport: mark failed write failed", "job_id", job.ID, "error", err)
	}
	job.Status = shared.DataExportJobStatusFailed
	job.ErrorMessage = &userMsg
	if mailErr := w.mailer.SendDataExportFailure(ctx, job); mailErr != nil {
		slog.Warn("dataexport: failure mail send failed (continuing)", "job_id", job.ID, "error", mailErr)
	}
}

// buildFileName returns `{rc_number}-{config_name}-{YYYY-MM-DD}.{ext}` per
// the PROJ-60 spec. config_name is fetched from configRepo (including
// soft-deleted rows); ext is inferred from the plugin's original filename.
// All path-traversal characters are stripped before assembly.
func (w *Worker) buildFileName(job *shared.DataExportJob, pluginFileName string) string {
	ext := ".bin"
	if i := strings.LastIndex(pluginFileName, "."); i >= 0 {
		ext = pluginFileName[i:]
	}

	configName := "export"
	if job.ConfigID != nil && w.configRepo != nil {
		if cfg, err := w.configRepo.GetByIDIncludingDeleted(*job.ConfigID); err == nil {
			configName = cfg.Name
		}
	}

	rc := sanitiseFilenameSegment(job.RCNumber)
	name := sanitiseFilenameSegment(configName)
	date := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s-%s-%s%s", rc, name, date, ext)
}

// sanitiseFilenameSegment replaces any character that could traverse the
// filesystem (slashes, dots, control bytes) or break Content-Disposition
// quoting (quote, backslash) with `_`. Result is bounded to 64 chars so
// pathological config names cannot produce > MAX_PATH filenames downstream.
func sanitiseFilenameSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "export"
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9',
			r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('_')
		default:
			b.WriteByte('_')
		}
	}
	out := b.String()
	if len(out) > 64 {
		out = out[:64]
	}
	if out == "" {
		out = "export"
	}
	return out
}

// detectSensitiveFields returns the list of sensitive field-keys present in
// an Excel-plugin config (currently the only plugin that exposes them).
// Other plugins return an empty list. Used for DSGVO audit-trail logging.
func detectSensitiveFields(configSnapshot map[string]interface{}) []string {
	cols, ok := configSnapshot["columns"].([]interface{})
	if !ok {
		return nil
	}
	sensitive := map[string]bool{
		"iban":       true,
		"birth_date": true,
	}
	var found []string
	for _, c := range cols {
		m, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		field, _ := m["field"].(string)
		if sensitive[field] {
			found = append(found, field)
		}
	}
	return found
}

// JobID is a tiny helper for tests / debug logging.
func JobID(id uuid.UUID) string { return id.String() }
