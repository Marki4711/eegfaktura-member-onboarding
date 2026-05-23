package dataexport

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
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

// LoadRecentImportedForPreview fetches up to `limit` most-recent applications
// for the EEG (any status). Used by the live-preview feature in the
// config editor — admin just wants to see *some* real data to test the
// mapping. Status doesn't matter for preview.
func (l *AppLoader) LoadRecentImportedForPreview(ctx context.Context, rcNumber string, limit int) ([]ApplicationSnapshot, error) {
	if limit <= 0 {
		limit = 5
	}
	rcs := []string{rcNumber}
	filters := application.ApplicationListFilters{
		RCNumbers: &rcs,
	}
	items, _, err := l.appRepo.List(filters, 1, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent applications: %w", err)
	}
	out := make([]ApplicationSnapshot, 0, len(items))
	for _, item := range items {
		app, err := l.appRepo.GetByID(item.ID)
		if err != nil {
			continue
		}
		meters, err := l.meterRepo.GetByApplicationID(item.ID)
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
