package dataexport

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

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

func NewWorker(jobRepo *JobRepository, resultRepo *ResultRepository, appLoader ApplicationLoader, mailer FailureMailer, poolSize int, pollEvery time.Duration) *Worker {
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
// jobs to finish. Called from main.go on SIGTERM.
func (w *Worker) Stop() {
	w.once.Do(func() {
		close(w.stop)
	})
	w.wg.Wait()
	slog.Info("dataexport: worker pool stopped")
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
		w.fail(ctx, job, fmt.Sprintf("plugin %q not registered (probably removed since job was queued)", job.PluginType))
		return
	}

	// Load applications.
	apps, err := w.appLoader.LoadForExport(ctx, job.RCNumber, job.ApplicationIDs)
	if err != nil {
		w.fail(ctx, job, fmt.Sprintf("load applications: %v", err))
		return
	}

	// Decode config snapshot.
	var configMap map[string]interface{}
	if err := json.Unmarshal(job.ConfigSnapshot, &configMap); err != nil {
		w.fail(ctx, job, fmt.Sprintf("decode config snapshot: %v", err))
		return
	}

	// Progress callback updates DB every ~50 items.
	progress := func(processed int) {
		if err := w.jobRepo.UpdateProgress(job.ID, processed); err != nil {
			slog.Warn("dataexport: progress update failed", "job_id", job.ID, "error", err)
		}
	}

	result, err := plugin.Process(ctx, configMap, apps, progress)
	if err != nil {
		w.fail(ctx, job, fmt.Sprintf("plugin process: %v", err))
		return
	}

	// Persist result (only for DownloadResult — SyncResult writes nothing to result table).
	if dl, ok := result.(DownloadResult); ok {
		exp := time.Now().Add(shared.DataExportResultTTL)
		if err := w.resultRepo.Create(&shared.DataExportResult{
			JobID:     job.ID,
			FileName:  dl.FileName,
			MimeType:  dl.MimeType,
			FileBytes: dl.Bytes,
			FileSize:  len(dl.Bytes),
			ExpiresAt: exp,
		}); err != nil {
			w.fail(ctx, job, fmt.Sprintf("persist result blob: %v", err))
			return
		}
	}

	summaryJSON, err := json.Marshal(result.Summary())
	if err != nil {
		w.fail(ctx, job, fmt.Sprintf("marshal summary: %v", err))
		return
	}
	if err := w.jobRepo.MarkDone(job.ID, len(apps), summaryJSON); err != nil {
		slog.Error("dataexport: mark done failed", "job_id", job.ID, "error", err)
		return
	}
	slog.Info("dataexport: job done", "job_id", job.ID, "processed", len(apps))
}

func (w *Worker) fail(ctx context.Context, job *shared.DataExportJob, errMsg string) {
	slog.Error("dataexport: job failed", "job_id", job.ID, "error", errMsg)
	if err := w.jobRepo.MarkFailed(job.ID, errMsg); err != nil {
		slog.Error("dataexport: mark failed write failed", "job_id", job.ID, "error", err)
	}
	job.Status = shared.DataExportJobStatusFailed
	job.ErrorMessage = &errMsg
	if mailErr := w.mailer.SendDataExportFailure(ctx, job); mailErr != nil {
		slog.Warn("dataexport: failure mail send failed (continuing)", "job_id", job.ID, "error", mailErr)
	}
}

// JobID is a tiny helper for tests / debug logging.
func JobID(id uuid.UUID) string { return id.String() }
