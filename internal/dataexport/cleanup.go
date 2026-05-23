package dataexport

import (
	"log/slog"
	"time"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// CleanupRunner orchestrates the three tasks the data-export-cleanup
// K8s-CronJob runs every ~10 minutes:
//   1. Zombie-Recovery: status='running' older than threshold → failed
//   2. BLOB-TTL: delete data_export_result rows past expires_at, mark
//      the associated jobs as 'expired'
//   3. Config-Hard-Delete: data_export_config rows soft-deleted longer
//      than DataExportConfigHardDeleteAge → hard delete (§ 132 BAO)
//
// All tasks are idempotent. Order is irrelevant.
type CleanupRunner struct {
	jobRepo    *JobRepository
	resultRepo *ResultRepository
	configRepo *ConfigRepository
}

func NewCleanupRunner(jobRepo *JobRepository, resultRepo *ResultRepository, configRepo *ConfigRepository) *CleanupRunner {
	return &CleanupRunner{
		jobRepo:    jobRepo,
		resultRepo: resultRepo,
		configRepo: configRepo,
	}
}

// CleanupResult reports the per-task counts of the run.
type CleanupResult struct {
	Zombies       int64
	ExpiredBlobs  int64
	DeletedConfigs int64
}

// Run executes all three cleanup tasks sequentially and returns the counts.
func (c *CleanupRunner) Run() (CleanupResult, error) {
	var out CleanupResult
	now := time.Now()

	// 1. Zombie-Recovery
	n, err := c.jobRepo.RecoverZombies(now.Add(-shared.DataExportZombieThreshold))
	if err != nil {
		return out, err
	}
	out.Zombies = n

	// 2. BLOB-TTL: delete expired results, mark jobs as expired
	expiredJobs, err := c.resultRepo.DeleteExpired(now)
	if err != nil {
		return out, err
	}
	out.ExpiredBlobs = int64(len(expiredJobs))
	if len(expiredJobs) > 0 {
		if _, err := c.jobRepo.MarkExpired(expiredJobs); err != nil {
			return out, err
		}
	}

	// 3. Config-Hard-Delete (DSGVO § 132 BAO)
	dn, err := c.configRepo.HardDeleteOldSoftDeleted(now.Add(-shared.DataExportConfigHardDeleteAge))
	if err != nil {
		return out, err
	}
	out.DeletedConfigs = dn

	slog.Info("dataexport: cleanup run complete",
		"zombies_recovered", out.Zombies,
		"expired_blobs_deleted", out.ExpiredBlobs,
		"old_configs_hard_deleted", out.DeletedConfigs)

	return out, nil
}
