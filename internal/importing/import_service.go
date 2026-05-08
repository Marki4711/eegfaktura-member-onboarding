package importing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/coreclient"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ImportService orchestrates the import of an approved application into the
// eegFaktura core. The actual core call happens outside any DB transaction;
// only the bookkeeping writes (status, status_log) are transactional.
type ImportService struct {
	db            *sql.DB
	appRepo       *application.ApplicationRepository
	meteringRepo  *application.MeteringPointRepository
	statusLogRepo *application.StatusLogRepository
	coreClient    coreclient.CoreClient
}

// NewImportService wires the dependencies. coreClient may be a stub in tests.
func NewImportService(
	db *sql.DB,
	appRepo *application.ApplicationRepository,
	meteringRepo *application.MeteringPointRepository,
	statusLogRepo *application.StatusLogRepository,
	coreClient coreclient.CoreClient,
) *ImportService {
	return &ImportService{
		db:            db,
		appRepo:       appRepo,
		meteringRepo:  meteringRepo,
		statusLogRepo: statusLogRepo,
		coreClient:    coreClient,
	}
}

// ImportResult is what the handler returns to the API caller.
type ImportResult struct {
	ApplicationID       uuid.UUID
	Status              shared.ApplicationStatus
	TargetParticipantID string
	ErrorMessage        string
}

// Import runs one import attempt for the application identified by id.
// bearerToken is the caller's Keycloak JWT, forwarded to the core. actorID is
// the admin's username/sub used in the status_log entry. allowedTenants is
// the verified set of RC numbers the caller is allowed to act on, or nil for
// superusers (no tenant restriction). It is asserted as defense-in-depth on
// top of the handler-level tenant check.
//
// Pre-import validation errors (wrong status, no metering points) are returned
// as typed shared errors and the application is left untouched. Core failures
// transition the application into import_failed and return a wrapped error so
// the handler can render a 500 with the stored error message.
func (s *ImportService) Import(ctx context.Context, id uuid.UUID, bearerToken, actorID string, allowedTenants []string) (*ImportResult, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if allowedTenants != nil && !containsString(allowedTenants, app.RCNumber) {
		return nil, shared.ErrForbidden
	}
	if app.Status != shared.StatusApproved {
		return nil, shared.NewConflictError(
			fmt.Sprintf("only applications in approved status can be imported (current: %s)", app.Status),
		)
	}

	meteringPoints, err := s.meteringRepo.GetByApplicationID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load metering points: %w", err)
	}
	if len(meteringPoints) == 0 {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"meteringPoints": "application has no metering points to import",
		})
	}

	importStartedAt := time.Now()

	// Reserve the in-flight slot before calling the core. This both persists
	// import_started_at (so a crashed/timed-out attempt leaves a trail) and
	// prevents a concurrent request from triggering a duplicate participant
	// in the non-idempotent core. If we lose the race, return 409.
	reserved, err := s.appRepo.MarkImportInFlight(id, importStartedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to reserve import slot: %w", err)
	}
	if !reserved {
		return nil, shared.NewConflictError("another import is already in progress for this application")
	}

	payload := BuildPayload(app, meteringPoints, importStartedAt)

	participantID, coreErr := s.coreClient.CreateParticipant(ctx, payload, bearerToken, app.RCNumber)
	importFinishedAt := time.Now()

	if coreErr != nil {
		errMessage := normalizeError(coreErr)
		if persistErr := s.persistResult(id, app.Status, application.ImportResultUpdate{
			Status:             shared.StatusImportFailed,
			ImportStartedAt:    importStartedAt,
			ImportFinishedAt:   importFinishedAt,
			ImportErrorMessage: &errMessage,
		}, actorID); persistErr != nil {
			slog.Error("import: failed to persist failure outcome", "application_id", id, "error", persistErr)
			return nil, fmt.Errorf("import failed and bookkeeping failed: core_error=%v db_error=%w", coreErr, persistErr)
		}
		return &ImportResult{
			ApplicationID: id,
			Status:        shared.StatusImportFailed,
			ErrorMessage:  errMessage,
		}, coreErr
	}

	if err := s.persistResult(id, app.Status, application.ImportResultUpdate{
		Status:              shared.StatusImported,
		ImportStartedAt:     importStartedAt,
		ImportFinishedAt:    importFinishedAt,
		ImportedAt:          &importFinishedAt,
		TargetParticipantID: &participantID,
	}, actorID); err != nil {
		return nil, fmt.Errorf("import succeeded in core but bookkeeping failed: %w", err)
	}

	return &ImportResult{
		ApplicationID:       id,
		Status:              shared.StatusImported,
		TargetParticipantID: participantID,
	}, nil
}

// persistResult writes the application UPDATE and the status_log INSERT in a
// single transaction.
func (s *ImportService) persistResult(id uuid.UUID, fromStatus shared.ApplicationStatus, u application.ImportResultUpdate, actorID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin import-result transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.appRepo.UpdateImportResultTx(tx, id, u); err != nil {
		return err
	}

	from := string(fromStatus)
	to := string(u.Status)
	var actorPtr *string
	if actorID != "" {
		actorPtr = &actorID
	}
	logEntry := &shared.StatusLogEntry{
		ApplicationID:   id,
		FromStatus:      &from,
		ToStatus:        to,
		ChangedByUserID: actorPtr,
		CreatedAt:       u.ImportFinishedAt,
	}
	if err := s.statusLogRepo.CreateTx(tx, logEntry); err != nil {
		return fmt.Errorf("failed to write status log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit import-result transaction: %w", err)
	}
	return nil
}

func containsString(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// normalizeError converts a coreclient error into a human-readable string for
// storage in import_error_message. Bounded length is enforced at the column
// level (TEXT) and at the coreclient level (1000 chars on body).
func normalizeError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, coreclient.ErrCoreTimeout) {
		return "core service timeout"
	}
	if errors.Is(err, coreclient.ErrCoreNotConfigured) {
		return "core service not configured (CORE_BASE_URL is empty)"
	}
	var httpErr *coreclient.CoreHTTPError
	if errors.As(err, &httpErr) {
		return httpErr.Error()
	}
	var parseErr *coreclient.CoreParseError
	if errors.As(err, &parseErr) {
		return parseErr.Error()
	}
	return err.Error()
}
