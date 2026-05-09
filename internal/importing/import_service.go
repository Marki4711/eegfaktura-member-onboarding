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
//
// TODO(coverage): Import() currently has no unit tests because the
// concrete repositories are not behind interfaces and the project does not
// depend on sqlmock. The Opus PROJ-4 review flagged this as Medium. Adding
// coverage requires either extracting repo interfaces or pulling in a
// sqlmock dependency — tracked as a follow-up.
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
		// The participant exists in the core but our DB couldn't record it.
		// Log the orphan participant ID so an operator can clean it up by
		// hand, and surface it to the caller in the result so the handler can
		// include it in the response (without leaking the raw DB error).
		slog.Error("import: bookkeeping failed after successful core insert; orphan participant created",
			"application_id", id,
			"target_participant_id", participantID,
			"db_error", err,
		)
		return &ImportResult{
				ApplicationID:       id,
				Status:              shared.StatusApproved, // unchanged on disk
				TargetParticipantID: participantID,
				ErrorMessage:        "import succeeded in core but the onboarding record could not be updated; participant created in core, manual cleanup required",
			},
			fmt.Errorf("import succeeded in core but bookkeeping failed: %w", err)
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

// importErrorMessageMaxLen caps the length of strings stored in
// import_error_message. The coreclient already truncates the HTTP body,
// but other error sources (parse failures, network errors) can be longer
// than what we want to surface to admins or store in the DB column.
const importErrorMessageMaxLen = 1000

// normalizeError converts a coreclient error into a human-readable string for
// storage in import_error_message. The result is always bounded to
// importErrorMessageMaxLen characters, regardless of error source.
func normalizeError(err error) string {
	if err == nil {
		return ""
	}
	var msg string
	switch {
	case errors.Is(err, coreclient.ErrCoreTimeout):
		msg = "core service timeout"
	case errors.Is(err, coreclient.ErrCoreNotConfigured):
		msg = "core service not configured (CORE_BASE_URL is empty)"
	default:
		var httpErr *coreclient.CoreHTTPError
		var parseErr *coreclient.CoreParseError
		switch {
		case errors.As(err, &httpErr):
			msg = httpErr.Error()
		case errors.As(err, &parseErr):
			msg = parseErr.Error()
		default:
			msg = err.Error()
		}
	}
	return truncateRunes(msg, importErrorMessageMaxLen)
}

// truncateRunes shortens s to at most maxRunes runes, appending an ellipsis
// when truncated. Slicing by runes (not bytes) avoids cutting a multi-byte
// UTF-8 sequence in half.
func truncateRunes(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "…"
}
