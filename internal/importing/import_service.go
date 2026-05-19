package importing

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/coreclient"
	"github.com/your-org/eegfaktura-member-onboarding/internal/metrics"
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
	db             *sql.DB
	appRepo        *application.ApplicationRepository
	meteringRepo   *application.MeteringPointRepository
	statusLogRepo  *application.StatusLogRepository
	entrypointRepo *application.RegistrationEntrypointRepository
	coreClient     coreclient.CoreClient
}

// ListTariffs proxies the core's GET /eeg/tariff for the admin tariff-selection
// dialog at import time (PROJ-27). The caller's bearer token is forwarded; the
// tenant comes from the admin's allowed RC numbers (validated by the HTTP
// handler before calling here).
func (s *ImportService) ListTariffs(ctx context.Context, bearerToken, tenant string) ([]coreclient.CoreTariff, error) {
	return s.coreClient.ListTariffs(ctx, bearerToken, tenant)
}

// ActivationCheckResult summarises one batch run of CheckActivations.
type ActivationCheckResult struct {
	Checked   int      `json:"checked"`
	Activated int      `json:"activated"`
	Errors    []string `json:"errors,omitempty"`
}

// CheckActivations (PROJ-46 Stage D) inspects every application in status
// `ready_for_activation` (restricted to the admin's tenants when
// allowedRCNumbers != nil) and transitions those whose linked core
// participant is now `ACTIVE` to status `activated`.
//
// Per-tenant batch: groups the candidate apps by rc_number and calls the
// core's GET /participant once per tenant — keeps the number of upstream
// requests bounded by O(#tenants), not O(#apps). Failures for a single
// tenant are recorded in Result.Errors but don't abort the whole batch.
func (s *ImportService) CheckActivations(ctx context.Context, bearerToken string, allowedRCNumbers []string) (*ActivationCheckResult, error) {
	rows, err := s.appRepo.ListReadyForActivation(allowedRCNumbers)
	if err != nil {
		return nil, fmt.Errorf("list ready-for-activation: %w", err)
	}
	if len(rows) == 0 {
		return &ActivationCheckResult{}, nil
	}

	// Group by tenant.
	byTenant := map[string][]application.ReadyForActivationRow{}
	for _, row := range rows {
		byTenant[row.RCNumber] = append(byTenant[row.RCNumber], row)
	}

	result := &ActivationCheckResult{Checked: len(rows)}

	for tenant, tenantRows := range byTenant {
		// PROJ-53: Aktivierungs-Modus pro EEG. Default participant_active
		// = heutiges Verhalten (Core-Teilnehmer-Status ACTIVE).
		mode := shared.ActivationModeParticipantActive
		if ep, epErr := s.entrypointRepo.GetByRCNumber(tenant); epErr != nil {
			result.Errors = append(result.Errors,
				fmt.Sprintf("tenant %s: load entrypoint failed: %s", tenant, epErr))
			continue
		} else if shared.IsValidActivationMode(ep.ActivationMode) {
			mode = ep.ActivationMode
		}

		participants, err := s.coreClient.ListParticipants(ctx, bearerToken, tenant)
		if err != nil {
			result.Errors = append(result.Errors,
				fmt.Sprintf("tenant %s: %s", tenant, normalizeError(err)))
			continue
		}

		// Index by participant ID for O(1) lookup.
		participantByID := map[string]coreclient.CoreParticipantSummary{}
		for _, p := range participants {
			participantByID[p.ID] = p
		}

		for _, row := range tenantRows {
			if row.TargetParticipantID == nil || *row.TargetParticipantID == "" {
				result.Errors = append(result.Errors,
					fmt.Sprintf("app %s: target_participant_id is empty", row.ID))
				continue
			}
			p, ok := participantByID[*row.TargetParticipantID]
			if !ok {
				// Participant not found in core — could mean it was deleted.
				// Skip silently; admin can reset/re-import if needed.
				continue
			}
			if !shouldActivate(mode, p) {
				continue
			}
			if err := s.markActivated(row.ID, mode); err != nil {
				result.Errors = append(result.Errors,
					fmt.Sprintf("app %s: mark activated failed: %s", row.ID, err))
				continue
			}
			result.Activated++
		}
	}
	return result, nil
}

// shouldActivate (PROJ-53) decides whether the given core participant
// satisfies the per-EEG activation criterion.
//   - participant_active: classic — participant.status == ACTIVE
//   - any_meter_registration_started: at least one meter has processState
//     in {PENDING, APPROVED, ACTIVE} (Netzbetreiber has at minimum
//     bestätigt receipt of the online-registration request)
func shouldActivate(mode string, p coreclient.CoreParticipantSummary) bool {
	switch mode {
	case shared.ActivationModeAnyMeterRegistrationStarted:
		for _, m := range p.Meters {
			switch m.ProcessState {
			case "PENDING", "APPROVED", "ACTIVE":
				return true
			}
		}
		return false
	default: // ActivationModeParticipantActive
		return p.Status == "ACTIVE"
	}
}

// markActivated transitions an application from ready_for_activation to
// activated, stamps activated_at, and writes the status_log entry. PROJ-53:
// the activation mode is recorded in the log reason so debugging post-hoc
// is possible ("warum hat der Batch das aktiviert?").
func (s *ImportService) markActivated(id uuid.UUID, mode string) error {
	now := time.Now().UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()
	system := "system:activation-check"
	if err := s.appRepo.UpdateStatusAdminTx(
		tx, id, shared.StatusReadyForActivation, shared.StatusActivated,
		nil, nil, nil, nil, &system, nil, &now,
	); err != nil {
		return err
	}
	from := string(shared.StatusReadyForActivation)
	reason := fmt.Sprintf("activation-check batch (mode=%s)", mode)
	if err := s.statusLogRepo.CreateTx(tx, &shared.StatusLogEntry{
		ApplicationID:   id,
		FromStatus:      &from,
		ToStatus:        string(shared.StatusActivated),
		ChangedByUserID: &system,
		Reason:          &reason,
		CreatedAt:       now,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

// SuggestNextMemberNumber pre-fills the member-number input in the import
// dialog with the next free value in the tenant's dominant numbering pattern.
//
// Algorithm:
//   1. For each existing participantNumber, split off the trailing run of
//      digits — the rest is the "prefix" (e.g. "A005" → prefix="A", n=5;
//      "M-12" → "M-"/12; "123" → ""/123).
//   2. Group by prefix; track max(n) and the longest digit-padding seen.
//   3. Pick the largest group (most populous pattern wins). Tiebreak by the
//      longer prefix so "A001" beats "" when both have one entry.
//   4. Emit "<prefix><n+1>" with the group's padding (zero-pad).
//
// Returns "1" when no participantNumber has a parseable digit suffix.
func (s *ImportService) SuggestNextMemberNumber(ctx context.Context, bearerToken, tenant string) (string, error) {
	participants, err := s.coreClient.ListParticipants(ctx, bearerToken, tenant)
	if err != nil {
		metrics.MemberNumberLookupTotal.WithLabelValues("core_error").Inc()
		return "", err
	}
	metrics.MemberNumberLookupTotal.WithLabelValues("success").Inc()

	type group struct {
		prefix  string
		padding int
		maxN    int
		count   int
	}
	groups := map[string]*group{}

	for _, p := range participants {
		if p.ParticipantNumber == nil {
			continue
		}
		v := strings.TrimSpace(*p.ParticipantNumber)
		if v == "" {
			continue
		}
		prefix, digits := splitTrailingDigits(v)
		if digits == "" {
			continue
		}
		n, err := strconv.Atoi(digits)
		if err != nil {
			continue
		}
		g, ok := groups[prefix]
		if !ok {
			g = &group{prefix: prefix}
			groups[prefix] = g
		}
		g.count++
		if n > g.maxN {
			g.maxN = n
		}
		if len(digits) > g.padding {
			g.padding = len(digits)
		}
	}

	if len(groups) == 0 {
		return "1", nil
	}

	var best *group
	for _, g := range groups {
		if best == nil ||
			g.count > best.count ||
			(g.count == best.count && len(g.prefix) > len(best.prefix)) {
			best = g
		}
	}
	return fmt.Sprintf("%s%0*d", best.prefix, best.padding, best.maxN+1), nil
}

// MemberNumberTaken checks whether the given value (e.g. "A006") is already
// used by an existing participant in the tenant. Compared as raw strings.
func (s *ImportService) MemberNumberTaken(ctx context.Context, bearerToken, tenant, number string) (bool, error) {
	participants, err := s.coreClient.ListParticipants(ctx, bearerToken, tenant)
	if err != nil {
		return false, err
	}
	target := strings.TrimSpace(number)
	for _, p := range participants {
		if p.ParticipantNumber == nil {
			continue
		}
		if strings.TrimSpace(*p.ParticipantNumber) == target {
			return true, nil
		}
	}
	return false, nil
}

// splitTrailingDigits returns the prefix and the trailing run of decimal
// digits. "A005" → ("A", "005"), "123" → ("", "123"), "M-foo" → ("M-foo", "").
func splitTrailingDigits(s string) (prefix, digits string) {
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	return s[:i], s[i:]
}

// NewImportService wires the dependencies. coreClient may be a stub in tests.
func NewImportService(
	db *sql.DB,
	appRepo *application.ApplicationRepository,
	meteringRepo *application.MeteringPointRepository,
	statusLogRepo *application.StatusLogRepository,
	entrypointRepo *application.RegistrationEntrypointRepository,
	coreClient coreclient.CoreClient,
) *ImportService {
	return &ImportService{
		db:             db,
		appRepo:        appRepo,
		meteringRepo:   meteringRepo,
		statusLogRepo:  statusLogRepo,
		entrypointRepo: entrypointRepo,
		coreClient:     coreClient,
	}
}

// ImportResult is what the handler returns to the API caller.
type ImportResult struct {
	ApplicationID       uuid.UUID
	Status              shared.ApplicationStatus
	TargetParticipantID string
	ErrorMessage        string
	// MemberTariffWarning is set when participant creation succeeded but the
	// follow-up call to assign the member-level tariff (PROJ-27, PUT
	// /participant/v2/{id}) failed. The application remains `imported`; the
	// admin can re-assign the member tariff manually in the core.
	MemberTariffWarning string
}

// TariffSelection captures the admin's choices made in the import dialog (PROJ-27).
// Empty strings mean "no tariff" — the field is then omitted from the core call.
type TariffSelection struct {
	MemberTariffID string            // applied via PUT /participant/v2/{id} after creation
	MeterTariffIDs map[string]string // metering_point -> tariff UUID; goes into POST body
}

// Import runs one import attempt for the application identified by id.
// bearerToken is the caller's Keycloak JWT, forwarded to the core. actorID is
// the admin's username/sub used in the status_log entry. allowedTenants is
// the verified set of RC numbers the caller is allowed to act on, or nil for
// superusers (no tenant restriction). It is asserted as defense-in-depth on
// top of the handler-level tenant check.
//
// memberNumber is the value chosen by the admin in the import dialog (the
// frontend pre-fills it from `SuggestNextMemberNumber` but lets the admin
// override). The number must be > 0 and is checked against the core's
// existing participants to refuse duplicates before sending POST /participant.
// On success it is written to application.member_number so the approval PDF
// can render it.
//
// Pre-import validation errors (wrong status, no metering points, member
// number conflict) are returned as typed shared errors and the application
// is left untouched. Core failures transition the application into
// import_failed and return a wrapped error so the handler can render a 500
// with the stored error message.
func (s *ImportService) Import(ctx context.Context, id uuid.UUID, bearerToken, actorID string, allowedTenants []string, selection TariffSelection, memberNumber string) (*ImportResult, error) {
	memberNumber = strings.TrimSpace(memberNumber)
	if memberNumber == "" {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"memberNumber": "Mitgliedsnummer darf nicht leer sein",
		})
	}

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

	// Pre-import duplicate check: catches the race between two admins picking
	// the same suggested number, or an admin overriding to a value the core
	// already uses. The core does not enforce uniqueness on participantNumber,
	// so the guard has to live here.
	taken, err := s.MemberNumberTaken(ctx, bearerToken, app.RCNumber, memberNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to verify member number against core: %w", err)
	}
	if taken {
		return nil, shared.NewConflictError(
			fmt.Sprintf("Mitgliedsnummer %q ist im Core bereits vergeben", memberNumber),
		)
	}

	// PROJ-34 Stage B: defense-in-depth local check. The partial UNIQUE
	// index uniq_application_rc_member_number (migration 28) would
	// otherwise blow the bookkeeping transaction AFTER the core insert
	// has succeeded, producing a half-written state. By checking here
	// we refuse the import BEFORE talking to the core, so no orphan
	// participant is ever created from this failure mode.
	usedLocally, conflictingRef, err := s.appRepo.MemberNumberUsedLocally(app.RCNumber, memberNumber, id)
	if err != nil {
		return nil, fmt.Errorf("failed to verify member number against local db: %w", err)
	}
	if usedLocally {
		return nil, shared.NewConflictError(
			fmt.Sprintf("Mitgliedsnummer %q ist im Onboarding bereits dem Antrag %s zugeordnet — bitte eine andere Nummer wählen", memberNumber, conflictingRef),
		)
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

	// Detach the context for the core phase. Once we've reserved the in-flight
	// slot, the operation MUST complete to a terminal state (imported or
	// import_failed); otherwise the slot stays "in flight" and every future
	// import returns 409 until manual cleanup. If the caller cancels mid-call
	// (browser closed, network drop) we still want to finish what we started.
	// 2 minutes is a generous safety net; the coreClient's HTTP timeout is
	// the actual cap.
	coreCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	payload := BuildPayload(app, meteringPoints, importStartedAt, selection.MeterTariffIDs)
	// Member number is provided per import call now (no longer auto-assigned
	// at submit time), so override whatever BuildPayload picked up from the
	// stale `application.member_number` column.
	payload.ParticipantNumber = memberNumber

	participantID, coreErr := s.coreClient.CreateParticipant(coreCtx, payload, bearerToken, app.RCNumber)
	importFinishedAt := time.Now()

	if coreErr != nil {
		metrics.ImportsTotal.WithLabelValues("failed").Inc()
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

	metrics.ImportsTotal.WithLabelValues("success").Inc()

	if err := s.persistResult(id, app.Status, application.ImportResultUpdate{
		Status:              shared.StatusImported,
		ImportStartedAt:     importStartedAt,
		ImportFinishedAt:    importFinishedAt,
		ImportedAt:          &importFinishedAt,
		TargetParticipantID: &participantID,
		MemberNumber:        &memberNumber,
	}, actorID); err != nil {
		// The participant exists in the core but our DB couldn't link it
		// (typical cause: the local uniq_application_rc_member_number
		// partial index from migration 28 catches a duplicate that the
		// core didn't because Core has no uniqueness constraint on
		// participantNumber).
		//
		// PROJ-34 Stage A: take the application out of in-flight into a
		// clean `import_failed` end state in a SEPARATE transaction. That
		// keeps `target_participant_id` for later operator linkage, lets
		// the existing PROJ-30 reset-import flow recover the row, and
		// — critically — prevents the in-flight slot from staying set
		// forever, which used to brick the admin button.
		dbErr := err
		errMessage := "Core hat Teilnehmer " + participantID +
			" angelegt, lokale Verknüpfung fehlgeschlagen: " + normalizeError(dbErr)
		if fallbackErr := s.persistResult(id, app.Status, application.ImportResultUpdate{
			Status:              shared.StatusImportFailed,
			ImportStartedAt:     importStartedAt,
			ImportFinishedAt:    importFinishedAt,
			TargetParticipantID: &participantID,
			ImportErrorMessage:  &errMessage,
			// MemberNumber stays unchanged (nil) — we don't know if the
			// number we tried to assign is now reserved in the core.
		}, actorID); fallbackErr != nil {
			// Second-layer failure: the orphan-fallback transaction itself
			// died. The in-flight slot now stays set. Log loudly so the
			// operator sees this in monitoring; manual SQL cleanup needed.
			slog.Error("import: orphan fallback failed; application stuck in-flight",
				"application_id", id,
				"target_participant_id", participantID,
				"original_db_error", dbErr,
				"fallback_db_error", fallbackErr,
			)
			return &ImportResult{
					ApplicationID:       id,
					Status:              shared.StatusApproved,
					TargetParticipantID: participantID,
					ErrorMessage:        "import succeeded in core but onboarding bookkeeping AND the recovery write failed — manual cleanup required",
				},
				fmt.Errorf("import succeeded in core but bookkeeping failed: %w (fallback also failed: %v)", dbErr, fallbackErr)
		}
		slog.Error("import: bookkeeping failed after successful core insert; application marked import_failed",
			"application_id", id,
			"target_participant_id", participantID,
			"db_error", dbErr,
		)
		return &ImportResult{
				ApplicationID:       id,
				Status:              shared.StatusImportFailed,
				TargetParticipantID: participantID,
				ErrorMessage:        errMessage,
			},
			fmt.Errorf("import succeeded in core but bookkeeping failed: %w", dbErr)
	}

	result := &ImportResult{
		ApplicationID:       id,
		Status:              shared.StatusImported,
		TargetParticipantID: participantID,
	}

	// PROJ-46: auto-branch out of `imported` to the post-import status.
	// `imported` is intentionally a transient landing zone for the import
	// bookkeeping — within milliseconds it transitions to the correct
	// post-import state. b2b needs admin confirmation that the member
	// coordinated with their bank; everything else skips straight to
	// ready_for_activation.
	postImportStatus := shared.StatusReadyForActivation
	if app.Einzugsart == "b2b" {
		postImportStatus = shared.StatusAwaitingBankConfirmation
	}
	if err := s.autoTransitionAfterImport(id, postImportStatus, actorID); err != nil {
		// Do NOT fail the import — the application is correctly in
		// `imported`, the Core insert succeeded, and an admin can resolve
		// the next step manually (PROJ-34 stuck-import recovery covers it).
		slog.Warn("import: post-import auto-transition failed; application stays in 'imported'",
			"application_id", id, "target_status", postImportStatus, "error", err)
	} else {
		result.Status = postImportStatus
	}

	// PROJ-27: member-level tariff cannot be set via POST /participant
	// (goqu:"skipinsert" on EegParticipantBase.TariffId). Apply it as a
	// follow-up partial update. A failure here does not roll back the
	// import — the admin can re-assign the member tariff in the core UI.
	if selection.MemberTariffID != "" {
		if err := s.coreClient.UpdateParticipantField(coreCtx, bearerToken, app.RCNumber, participantID, "tariffId", selection.MemberTariffID); err != nil {
			warn := normalizeError(err)
			slog.Warn("import: member tariff assignment failed after participant creation",
				"application_id", id,
				"target_participant_id", participantID,
				"tariff_id", selection.MemberTariffID,
				"error", warn,
			)
			result.MemberTariffWarning = warn
		}
	}

	return result, nil
}

// autoTransitionAfterImport moves the application out of `imported` to the
// post-import status (PROJ-46). Always runs after a successful import. Uses
// the guarded UpdateStatusAdminTx so a concurrent admin action that already
// moved the row to e.g. `ready_for_activation` cannot be overwritten.
func (s *ImportService) autoTransitionAfterImport(id uuid.UUID, toStatus shared.ApplicationStatus, actorID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback()

	if err := s.appRepo.UpdateStatusAdminTx(
		tx, id, shared.StatusImported, toStatus,
		nil, nil, nil, nil, ptrOrNil(actorID), nil, nil,
	); err != nil {
		return err
	}

	from := string(shared.StatusImported)
	var actorPtr *string
	if actorID != "" {
		actorPtr = &actorID
	}
	if err := s.statusLogRepo.CreateTx(tx, &shared.StatusLogEntry{
		ApplicationID:   id,
		FromStatus:      &from,
		ToStatus:        string(toStatus),
		ChangedByUserID: actorPtr,
		CreatedAt:       time.Now().UTC(),
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func ptrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
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
//
// HTTP 400 from the core is special-cased: the eegFaktura backend logs the
// SQL constraint violation reason on its side but returns an empty `{}`
// body to us, so the raw error message is unhelpful. We attach the most
// common cause as a hint — duplicate metering point — so the admin
// doesn't need to chase down the operator for the server log.
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
			msg = annotateCoreHTTPError(httpErr)
		case errors.As(err, &parseErr):
			msg = parseErr.Error()
		default:
			msg = err.Error()
		}
	}
	return truncateRunes(msg, importErrorMessageMaxLen)
}

// annotateCoreHTTPError translates the opaque "core returned HTTP <code>:
// <body>" message into something the admin can act on. The core typically
// hides the SQL-constraint reason in its own log and returns an empty body
// to us, so a literal echo of the HTTP envelope wastes the admin's time.
//
// 400 with empty/short body during import almost always means "duplicate
// metering point on the (metering_point_id, active=1, tenant) unique index"
// — that's the only validation the core enforces server-side that we
// can't pre-check from our end. We add that hint as the German message,
// keeping the raw HTTP echo as a technical postscript.
//
// Other status codes (401/403/5xx) are passed through unchanged — they
// already carry enough info or are not actionable from the admin UI.
func annotateCoreHTTPError(httpErr *coreclient.CoreHTTPError) string {
	if httpErr.StatusCode != 400 {
		return httpErr.Error()
	}
	body := strings.TrimSpace(httpErr.Body)
	if body == "" || body == "{}" || body == "<empty>" {
		return "Import abgelehnt vom eegFaktura-Core (HTTP 400). " +
			"Wahrscheinlichste Ursache: einer der Zählpunkte ist im Core bereits " +
			"einem aktiven Teilnehmer zugeordnet. Bitte im eegFaktura prüfen, ob die " +
			"Zählpunkte schon einer anderen Mitgliedschaft gehören; ggf. den bestehenden " +
			"Teilnehmer deaktivieren oder den Zählpunkt im Antrag korrigieren."
	}
	return httpErr.Error()
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
