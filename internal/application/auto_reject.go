package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/metrics"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// AutoRejectExpiredEmailConfirmations scans for applications stuck in
// `submitted` with an expired e-mail-confirmation token (PROJ-31) and moves
// them to `rejected`. Idempotent — re-running after a partial run only
// touches rows that are still expired-and-unconfirmed.
//
// Returns the number of applications transitioned. Errors mid-batch are
// logged but do not abort the loop; the remaining candidates still get
// processed.
func (s *AdminApplicationService) AutoRejectExpiredEmailConfirmations(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	ids, err := s.appRepo.ListExpiredEmailConfirmationPendingIDs(now, 200)
	if err != nil {
		return 0, fmt.Errorf("list expired: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	slog.Info("auto-reject: processing expired e-mail-confirmation applications", "count", len(ids))

	processed := 0
	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		if err := s.autoRejectOne(id, now); err != nil {
			if errors.Is(err, shared.ErrConflict) {
				// Row moved out of `submitted` since we listed — that's
				// fine, somebody else handled it.
				continue
			}
			slog.Error("auto-reject: failed to reject application", "application_id", id, "error", err)
			continue
		}
		processed++
		metrics.EmailConfirmationsTotal.WithLabelValues("expired").Inc()
	}
	if processed > 0 {
		slog.Info("auto-reject: completed", "rejected", processed)
	}
	return processed, nil
}

func (s *AdminApplicationService) autoRejectOne(id uuid.UUID, now time.Time) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.appRepo.AutoRejectExpiredEmailConfirmationTx(tx, id, now); err != nil {
		return err
	}

	systemActor := "system"
	logReason := "E-Mail-Bestätigung ausgeblieben (Auto-Reject nach 30 Tagen)"
	logEntry := &shared.StatusLogEntry{
		ApplicationID:   id,
		FromStatus:      stringPtr(string(shared.StatusSubmitted)),
		ToStatus:        string(shared.StatusRejected),
		ChangedByUserID: &systemActor,
		Reason:          &logReason,
		CreatedAt:       now,
	}
	if err := s.statusLogRepo.CreateTx(tx, logEntry); err != nil {
		return fmt.Errorf("status log: %w", err)
	}
	return tx.Commit()
}

// RunAutoRejectLoop blocks until ctx is cancelled, running
// AutoRejectExpiredEmailConfirmations on the given interval. Intended to be
// launched as a goroutine from cmd/server/main.go right before the HTTP
// server starts. interval==0 disables the loop (returns immediately).
func (s *AdminApplicationService) RunAutoRejectLoop(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		slog.Info("auto-reject: loop disabled (interval=0)")
		return
	}
	slog.Info("auto-reject: loop started", "interval", interval.String())

	// One immediate pass on start so the first run doesn't have to wait the
	// full interval after a server reboot.
	if _, err := s.AutoRejectExpiredEmailConfirmations(ctx); err != nil {
		slog.Error("auto-reject: initial pass failed", "error", err)
	}

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("auto-reject: loop stopped")
			return
		case <-t.C:
			if _, err := s.AutoRejectExpiredEmailConfirmations(ctx); err != nil {
				slog.Error("auto-reject: pass failed", "error", err)
			}
		}
	}
}
