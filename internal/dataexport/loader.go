package dataexport

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// AppLoader is the concrete ApplicationLoader implementation that wraps
// the existing ApplicationRepository + MeteringPointRepository.
type AppLoader struct {
	appRepo   *application.ApplicationRepository
	meterRepo *application.MeteringPointRepository
}

func NewAppLoader(appRepo *application.ApplicationRepository, meterRepo *application.MeteringPointRepository) *AppLoader {
	return &AppLoader{appRepo: appRepo, meterRepo: meterRepo}
}

// LoadForExport fetches the given application IDs and their metering points.
// Filters out applications that don't belong to rcNumber (defence-in-depth
// against compromised job snapshots).
func (l *AppLoader) LoadForExport(ctx context.Context, rcNumber string, ids []uuid.UUID) ([]ApplicationSnapshot, error) {
	out := make([]ApplicationSnapshot, 0, len(ids))
	for _, id := range ids {
		app, err := l.appRepo.GetByID(id)
		if err != nil {
			// Skip missing applications (member was deleted between selection
			// and processing). Don't fail the whole job.
			continue
		}
		if app.RCNumber != rcNumber {
			// Cross-tenant check — should never happen if the trigger handler
			// validated, but defence-in-depth here.
			continue
		}
		meters, err := l.meterRepo.GetByApplicationID(id)
		if err != nil {
			return nil, fmt.Errorf("load meters for %s: %w", id, err)
		}
		if meters == nil {
			meters = nil
		}
		out = append(out, ApplicationSnapshot{
			Application:    app,
			MeteringPoints: meters,
		})
	}
	return out, nil
}

// postImportStatuses are the statuses that count as "imported" for the
// preview: the member has reached the Core, regardless of whether they're
// still pending bank-confirmation or already activated. Excludes draft,
// submitted, email_confirmed, under_review, needs_info, approved, rejected
// and import_failed — those are not yet useful for testing a column mapping
// against real data.
var postImportStatuses = []string{
	"imported",
	"awaiting_bank_confirmation",
	"ready_for_activation",
	"activated",
}

// LoadRecentImportedForPreview fetches up to `limit` most-recent imported
// applications for the EEG. Used by the live-preview feature in the config
// editor — admin wants to see real data through their column mapping.
// Filters strictly to post-import statuses (PROJ-60 spec); falls back to
// the plugin's synthetic sample when no imported members exist (handled
// one layer up in ConfigService.Preview).
func (l *AppLoader) LoadRecentImportedForPreview(ctx context.Context, rcNumber string, limit int) ([]ApplicationSnapshot, error) {
	if limit <= 0 {
		limit = 5
	}
	rcs := []string{rcNumber}

	// Fetch up to `limit` from each post-import status, then merge + take
	// the most recent `limit` overall. 4 small queries is cheaper than
	// adding a multi-status filter to the shared ApplicationListFilters.
	var pooled []shared.ApplicationListItem
	for _, status := range postImportStatuses {
		st := status
		filters := application.ApplicationListFilters{
			RCNumbers: &rcs,
			Status:    &st,
			Sort:      "submittedAt",
			Order:     "desc",
		}
		items, _, err := l.appRepo.List(filters, 1, limit)
		if err != nil {
			return nil, fmt.Errorf("list recent applications (status=%s): %w", status, err)
		}
		pooled = append(pooled, items...)
	}

	// Sort merged set by SubmittedAt desc; nil submitted_at sorts last so
	// fully-processed members with a real timestamp beat any stragglers.
	sort.Slice(pooled, func(i, j int) bool {
		if pooled[i].SubmittedAt == nil {
			return false
		}
		if pooled[j].SubmittedAt == nil {
			return true
		}
		return pooled[i].SubmittedAt.After(*pooled[j].SubmittedAt)
	})
	if len(pooled) > limit {
		pooled = pooled[:limit]
	}

	out := make([]ApplicationSnapshot, 0, len(pooled))
	for _, it := range pooled {
		app, err := l.appRepo.GetByID(it.ID)
		if err != nil {
			continue
		}
		meters, err := l.meterRepo.GetByApplicationID(it.ID)
		if err != nil {
			meters = nil
		}
		out = append(out, ApplicationSnapshot{
			Application:    app,
			MeteringPoints: meters,
		})
	}
	return out, nil
}
