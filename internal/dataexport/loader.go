package dataexport

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// AppLoader is the concrete ApplicationLoader implementation that wraps
// the existing ApplicationRepository + MeteringPointRepository +
// RegistrationEntrypointRepository.
type AppLoader struct {
	appRepo        *application.ApplicationRepository
	meterRepo      *application.MeteringPointRepository
	entrypointRepo *application.RegistrationEntrypointRepository
}

func NewAppLoader(
	appRepo *application.ApplicationRepository,
	meterRepo *application.MeteringPointRepository,
	entrypointRepo *application.RegistrationEntrypointRepository,
) *AppLoader {
	return &AppLoader{appRepo: appRepo, meterRepo: meterRepo, entrypointRepo: entrypointRepo}
}

// LoadForExport fetches the given application IDs and their metering points
// using two batched queries (one for applications, one for metering points),
// regardless of the input size. Replaces the previous N+1 implementation
// that issued 2×N sequential round-trips and dominated the latency of
// 1000-application exports.
//
// Filters out applications that don't belong to rcNumber as defence-in-depth
// against compromised job snapshots. Missing applications (e.g. member
// deleted after job was queued) are silently skipped — the export proceeds
// with whatever IDs are still resolvable. Transient DB errors during the
// initial batched fetch are surfaced so the worker can fail the job and the
// admin sees a real error rather than a silently smaller result.
func (l *AppLoader) LoadForExport(_ context.Context, rcNumber string, ids []uuid.UUID) ([]ApplicationSnapshot, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	apps, err := l.appRepo.GetByIDs(ids)
	if err != nil {
		return nil, fmt.Errorf("batch load applications: %w", err)
	}

	// Tenant-filter + collect the IDs we'll need metering points for.
	keepIDs := make([]uuid.UUID, 0, len(apps))
	keepApps := make([]*shared.Application, 0, len(apps))
	skippedTenant := 0
	for _, app := range apps {
		if app.RCNumber != rcNumber {
			skippedTenant++
			continue
		}
		keepIDs = append(keepIDs, app.ID)
		keepApps = append(keepApps, app)
	}
	if skippedTenant > 0 {
		slog.Warn("dataexport: skipped applications with non-matching RC number",
			"rc_number", rcNumber, "skipped", skippedTenant)
	}
	// Hard-fail when the tenant filter wiped EVERYTHING but the caller
	// asked for ≥1 ID. Reaching this branch implies a compromised /
	// reassigned-EEG / forged-snapshot scenario — silently producing a
	// 0-row "successful" export would hand the admin a misleading file and
	// hide the integrity issue.
	if skippedTenant > 0 && len(keepApps) == 0 {
		// Bewusst KEIN rcNumber in der User-Message — die Details landen
		// via slog.Warn oben im Audit-Log. RC-Leak im Error wäre Info-Disclosure
		// gegenüber Cross-Tenant-Callern (Worker schreibt die Message in
		// job.error_message, das bleibt zwar tenant-gefiltert sichtbar, aber
		// generischer Wortlaut ist defensiver.)
		return nil, fmt.Errorf("alle %d Anträge gehören nicht (mehr) zur EEG dieses Jobs — Snapshot ist veraltet, bitte neu auslösen", skippedTenant)
	}

	meters, err := l.meterRepo.GetByApplicationIDs(keepIDs)
	if err != nil {
		return nil, fmt.Errorf("batch load metering points: %w", err)
	}

	// One job = one RC = one entrypoint row. Load once and share the
	// pointer across snapshots — keeps EEG-Stammdaten-Felder im Export
	// verfügbar ohne pro-Antrag-Roundtrip.
	entrypoint, err := l.entrypointRepo.GetByRCNumber(rcNumber)
	if err != nil {
		return nil, fmt.Errorf("load entrypoint for export: %w", err)
	}

	out := make([]ApplicationSnapshot, 0, len(keepApps))
	for _, app := range keepApps {
		out = append(out, ApplicationSnapshot{
			Application:    app,
			MeteringPoints: meters[app.ID], // nil-safe: missing key returns nil slice
			Entrypoint:     entrypoint,
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
//
// Now uses GetByIDs + GetByApplicationIDs for the detail fetch (2 queries
// total after the 4 per-status List() calls). Previous implementation did
// 4 + 2×limit queries per editor keystroke — fine for limit=5 but the
// underlying loader pattern matters for future preview-size growth.
func (l *AppLoader) LoadRecentImportedForPreview(_ context.Context, rcNumber string, limit int) ([]ApplicationSnapshot, error) {
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

	if len(pooled) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, len(pooled))
	for i, it := range pooled {
		ids[i] = it.ID
	}
	apps, err := l.appRepo.GetByIDs(ids)
	if err != nil {
		return nil, fmt.Errorf("batch load preview applications: %w", err)
	}
	meters, err := l.meterRepo.GetByApplicationIDs(ids)
	if err != nil {
		return nil, fmt.Errorf("batch load preview metering points: %w", err)
	}

	// Entrypoint einmal pro RC (siehe LoadForExport). Preview-Pfad nutzt
	// die EEG-Stammdaten in derselben Form wie der echte Export.
	entrypoint, err := l.entrypointRepo.GetByRCNumber(rcNumber)
	if err != nil {
		return nil, fmt.Errorf("load entrypoint for preview: %w", err)
	}

	// Preserve sort order from the pooled List() result by mapping by ID.
	byID := make(map[uuid.UUID]*shared.Application, len(apps))
	for _, app := range apps {
		byID[app.ID] = app
	}
	out := make([]ApplicationSnapshot, 0, len(pooled))
	for _, it := range pooled {
		app, ok := byID[it.ID]
		if !ok {
			continue
		}
		out = append(out, ApplicationSnapshot{
			Application:    app,
			MeteringPoints: meters[it.ID],
			Entrypoint:     entrypoint,
		})
	}
	return out, nil
}

