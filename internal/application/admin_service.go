package application

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/excel"
	"github.com/your-org/eegfaktura-member-onboarding/internal/mail"
	"github.com/your-org/eegfaktura-member-onboarding/internal/metrics"
	"github.com/your-org/eegfaktura-member-onboarding/internal/pdf"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ApplicationListFilters holds optional filter parameters for the admin list endpoint.
type ApplicationListFilters struct {
	Status          *string
	ReferenceNumber *string
	// Name (PROJ-…) is a partial-match search across firstname, lastname and
	// company_name. The admin list column is itself a coalesce of these three,
	// so the filter has to match the same surface — otherwise typing the
	// firstname or a company's name yields nothing.
	Name            *string
	Email           *string
	MeteringPoint   *string
	SubmittedFrom   *time.Time
	SubmittedTo     *time.Time
	// RCNumbers restricts results to a specific set of RC numbers (tenant-admin scope).
	// When nil, no restriction is applied (superuser scope).
	RCNumbers *[]string
	// RCNumberFilter is an optional single-EEG filter chosen by the admin in the UI.
	// Must always be a subset of RCNumbers when set.
	RCNumberFilter *string
	// Sort is the column to sort by. Allowed values are whitelisted in the
	// repository layer (see allowedSortColumns). Empty defaults to "submittedAt".
	Sort string
	// Order is "asc" or "desc". Empty defaults to "desc".
	Order string
}

// adminTransitions defines which status changes the admin endpoint may perform.
// Forward import transitions (approved→imported etc.) are handled by the dedicated
// import endpoint (PROJ-4). The reset transition import_failed→approved is an admin
// action (re-approve after failed import) and is handled here so the approval email
// is re-sent consistently via ChangeStatus.
var adminTransitions = map[shared.ApplicationStatus][]shared.ApplicationStatus{
	// `submitted → rejected` is allowed even when the EEG requires e-mail
	// confirmation: the admin can dismiss obvious junk without waiting for
	// the member to click. The runtime guard below blocks all OTHER
	// submitted-targets when require_email_confirmation is on and the
	// e-mail has not yet been confirmed (PROJ-31).
	shared.StatusSubmitted:       {shared.StatusUnderReview, shared.StatusRejected},
	shared.StatusEmailConfirmed:  {shared.StatusUnderReview, shared.StatusNeedsInfo, shared.StatusApproved, shared.StatusRejected},
	shared.StatusUnderReview:     {shared.StatusNeedsInfo, shared.StatusApproved, shared.StatusRejected},
	shared.StatusNeedsInfo:       {shared.StatusSubmitted},
	shared.StatusImportFailed:    {shared.StatusApproved},
	// PROJ-46: post-import statuses.
	// `imported → awaiting_bank_confirmation` and `imported → ready_for_activation`
	// are auto-transitions triggered by the import service right after a
	// successful import (branched on einzugsart) — they are intentionally
	// NOT in this admin map. If a manual fallback is needed later, expose
	// a dedicated endpoint rather than widening the generic /status surface.
	shared.StatusAwaitingBankConfirmation: {shared.StatusReadyForActivation, shared.StatusUnderReview},
	shared.StatusReadyForActivation:       {shared.StatusActivated, shared.StatusUnderReview},
	// `activated` is a strict end state — no transitions out (User-Wunsch A).
}

// AdminApplicationService implements admin review business logic.
type AdminApplicationService struct {
	db                   *sql.DB
	appRepo              *ApplicationRepository
	meteringRepo         *MeteringPointRepository
	statusLogRepo        *StatusLogRepository
	fieldConfigRepo      *FieldConfigRepository
	entrypointRepo       *RegistrationEntrypointRepository
	consentRepo          *DocumentConsentRepository
	mailService          mail.MailService
	approvalPDFGenerator pdf.ApprovalPDFGenerator
	// PROJ-47: B2B-Firmenlastschrift-Mandat-PDF wird beim Import an die
	// Member-Mail angehängt (mit Mandatsreferenz=Mitgliedsnummer).
	sepaMandateGenerator pdf.SEPAMandateGenerator
	publicBaseURL        string
}

// NewAdminApplicationService creates an AdminApplicationService.
func NewAdminApplicationService(
	db *sql.DB,
	appRepo *ApplicationRepository,
	meteringRepo *MeteringPointRepository,
	statusLogRepo *StatusLogRepository,
	fieldConfigRepo *FieldConfigRepository,
	entrypointRepo *RegistrationEntrypointRepository,
	consentRepo *DocumentConsentRepository,
	mailService mail.MailService,
	approvalPDFGenerator pdf.ApprovalPDFGenerator,
	sepaMandateGenerator pdf.SEPAMandateGenerator,
	publicBaseURL string,
) *AdminApplicationService {
	return &AdminApplicationService{
		db:                   db,
		appRepo:              appRepo,
		meteringRepo:         meteringRepo,
		statusLogRepo:        statusLogRepo,
		fieldConfigRepo:      fieldConfigRepo,
		entrypointRepo:       entrypointRepo,
		consentRepo:          consentRepo,
		mailService:          mailService,
		approvalPDFGenerator: approvalPDFGenerator,
		sepaMandateGenerator: sepaMandateGenerator,
		publicBaseURL:        publicBaseURL,
	}
}

// resendThrottle is the minimum interval between two resend-email-confirmation
// clicks on the same application — protects against admin accidentally
// double-clicking and spamming the member's inbox (PROJ-31 Q6).
const resendThrottle = 5 * time.Minute

// GetFieldConfig returns the field configuration for a given RC number.
func (s *AdminApplicationService) GetFieldConfig(rcNumber string) (map[string]FieldConfigEntry, error) {
	return s.fieldConfigRepo.Get(rcNumber)
}

// SaveFieldConfig replaces the field configuration for a given RC number.
func (s *AdminApplicationService) SaveFieldConfig(rcNumber string, config map[string]FieldConfigEntry) error {
	return s.fieldConfigRepo.Save(rcNumber, config)
}

// ResendMemberConfirmation re-sends the member confirmation email for any application.
func (s *AdminApplicationService) ResendMemberConfirmation(id uuid.UUID) error {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return err
	}
	entrypoint, err := s.entrypointRepo.GetByRCNumber(app.RCNumber)
	if err != nil {
		return err
	}
	return s.mailService.SendMemberConfirmation(app, entrypoint)
}

// ResendEmailConfirmation rotates the e-mail confirmation token for a still-
// pending application and re-sends the confirmation mail (PROJ-31). The
// original token (if any) is invalidated. Throttled to one resend every
// resendThrottle to prevent the admin from spamming the member's inbox.
//
// Concurrency-safe: the precondition checks and the token write happen in a
// single transaction with SELECT … FOR UPDATE so two simultaneous admin
// clicks don't both sneak past the throttle (PROJ-31 security finding L2).
func (s *AdminApplicationService) ResendEmailConfirmation(id uuid.UUID) error {
	// Read-only side: load entrypoint (separate row, no need to lock).
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return err
	}
	entrypoint, err := s.entrypointRepo.GetByRCNumber(app.RCNumber)
	if err != nil {
		return err
	}
	if !entrypoint.RequireEmailConfirmation {
		return shared.NewConflictError("EEG does not require e-mail confirmation")
	}
	if s.publicBaseURL == "" {
		return fmt.Errorf("public base URL not configured — cannot build confirmation link")
	}

	plaintext, hash, tokErr := GenerateEmailConfirmationToken()
	if tokErr != nil {
		return fmt.Errorf("token generation: %w", tokErr)
	}
	now := time.Now()
	expiresAt := now.Add(emailConfirmationTokenLifetime)

	tx, txErr := s.db.Begin()
	if txErr != nil {
		return fmt.Errorf("begin tx: %w", txErr)
	}
	defer tx.Rollback()

	// Lock the application row + read the fields the throttle check needs.
	// Holding the row lock for the rest of the transaction makes a
	// concurrent second resend wait here, and then see the just-rotated
	// token expiry — which fails its own throttle check.
	var lockedStatus shared.ApplicationStatus
	var lockedConfirmedAt sql.NullTime
	var lockedExpiresAt sql.NullTime
	if err := tx.QueryRow(`
		SELECT status, email_confirmed_at, email_confirmation_token_expires_at
		FROM member_onboarding.application
		WHERE id = $1
		FOR UPDATE`, id).Scan(&lockedStatus, &lockedConfirmedAt, &lockedExpiresAt); err != nil {
		if err == sql.ErrNoRows {
			return shared.ErrNotFound
		}
		return fmt.Errorf("lock application: %w", err)
	}
	if lockedStatus != shared.StatusSubmitted {
		return shared.NewConflictError("application is not in submitted status")
	}
	if lockedConfirmedAt.Valid {
		return shared.NewConflictError("application e-mail is already confirmed")
	}
	if lockedExpiresAt.Valid {
		issuedAt := lockedExpiresAt.Time.Add(-emailConfirmationTokenLifetime)
		if time.Since(issuedAt) < resendThrottle {
			return shared.NewConflictError(fmt.Sprintf("bitte warten Sie noch %s vor dem nächsten Versand", resendThrottle))
		}
	}

	if err := s.appRepo.AssignEmailConfirmationTokenTx(tx, id, hash, expiresAt); err != nil {
		return err
	}

	logReason := "Bestätigungs-Mail erneut versendet"
	systemActor := "admin"
	statusLog := &shared.StatusLogEntry{
		ApplicationID:   id,
		FromStatus:      stringPtr(string(shared.StatusSubmitted)),
		ToStatus:        string(shared.StatusSubmitted),
		ChangedByUserID: &systemActor,
		Reason:          &logReason,
		CreatedAt:       now,
	}
	if err := s.statusLogRepo.CreateTx(tx, statusLog); err != nil {
		return fmt.Errorf("status log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	url := BuildEmailConfirmationURL(s.publicBaseURL, plaintext)
	metrics.EmailConfirmationsTotal.WithLabelValues("resent").Inc()
	go func() {
		acquireMailSem()
		defer releaseMailSem()
		meteringPoints, mpErr := s.meteringRepo.GetByApplicationID(id)
		if mpErr != nil {
			slog.Error("resend-confirmation: failed to load metering points", "application_id", id, "error", mpErr)
			return
		}
		fieldConfig, _ := s.fieldConfigRepo.Get(strings.ToUpper(app.RCNumber))
		var consents []shared.DocumentConsent
		if s.consentRepo != nil {
			consents, _ = s.consentRepo.GetByApplicationID(id)
		}
		// Re-uses SendSubmissionEmails so the member mail keeps the same
		// shape it had at first submit — only the confirmation URL is new.
		// EEG-notification is deferred again (same condition as initial submit).
		s.mailService.SendSubmissionEmails(app, meteringPoints, entrypoint, toStateMap(fieldConfig), nil, consents, url)
	}()
	return nil
}

func stringPtr(s string) *string { return &s }

// ListApplications returns a paginated, filtered list of applications for admin review.
func (s *AdminApplicationService) ListApplications(filters ApplicationListFilters, page, pageSize int) (*shared.ApplicationListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	} else if pageSize > 100 {
		pageSize = 100
	}

	items, total, err := s.appRepo.List(filters, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}


	return &shared.ApplicationListResponse{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}, nil
}

// GetApplicationDetail returns the full detail view for a single application.
func (s *AdminApplicationService) GetApplicationDetail(id uuid.UUID) (*shared.AdminApplicationDetailResponse, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	meteringPoints, err := s.meteringRepo.GetByApplicationID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch metering points: %w", err)
	}
	if meteringPoints == nil {
		meteringPoints = []shared.MeteringPoint{}
	}

	statusLog, err := s.statusLogRepo.GetByApplicationID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch status log: %w", err)
	}
	if statusLog == nil {
		statusLog = []shared.StatusLogEntry{}
	}

	consentRows, err := s.consentRepo.GetByApplicationID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch consents: %w", err)
	}
	consentViews := make([]shared.DocumentConsentView, 0, len(consentRows))
	for _, c := range consentRows {
		consentViews = append(consentViews, shared.DocumentConsentView{
			ID:              c.ID,
			Title:           c.Title,
			URL:             c.URL,
			IsCentralPolicy: c.IsCentralPolicy,
			ConsentedAt:     c.ConsentedAt,
		})
	}

	resp := &shared.AdminApplicationDetailResponse{
		Application:    *app,
		MeteringPoints: meteringPoints,
		StatusLog:      statusLog,
		Consents:       consentViews,
		ImportStuck:    shared.IsImportStuck(app, time.Now()),
	}
	// PROJ-37: join in the EEG-level cooperative-shares config so the
	// admin detail can render the block with current amount × count =
	// total without an extra round-trip. Failure to load the entrypoint
	// is logged but does NOT fail the detail load — the block just stays
	// collapsed in that case.
	if ep, epErr := s.entrypointRepo.GetByRCNumber(app.RCNumber); epErr == nil {
		resp.CooperativeSharesEnabled = ep.CooperativeSharesEnabled
		resp.CooperativeRequiredShares = ep.CooperativeRequiredShares
		resp.CooperativeShareAmountCents = ep.CooperativeShareAmountCents
	}
	return resp, nil
}

// AdminUpdateApplication applies a partial admin update to a draft or needs_info application.
func (s *AdminApplicationService) AdminUpdateApplication(id uuid.UUID, req shared.AdminUpdateApplicationRequest) (*shared.ApplicationResponse, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Admin may edit applications in pre-import active states.
	// draft, rejected, and imported are not editable by admin.
	allowed := map[shared.ApplicationStatus]bool{
		shared.StatusSubmitted:    true,
		shared.StatusUnderReview:  true,
		shared.StatusNeedsInfo:    true,
		shared.StatusApproved:     true,
		shared.StatusImportFailed: true,
	}
	if !allowed[app.Status] {
		return nil, shared.NewConflictError("application cannot be edited in its current status")
	}

	// Apply partial updates.
	if req.MemberType != nil {
		app.MemberType = shared.MemberType(*req.MemberType)
	}
	if req.Titel != nil {
		app.Titel = trimStringPtr(req.Titel)
	}
	if req.TitelNach != nil {
		app.TitelNach = trimStringPtr(req.TitelNach)
	}
	if req.Firstname != nil {
		app.Firstname = trimStringPtr(req.Firstname)
	}
	if req.Lastname != nil {
		app.Lastname = trimStringPtr(req.Lastname)
	}
	if req.CompanyName != nil {
		app.CompanyName = trimStringPtr(req.CompanyName)
	}
	if req.UIDNumber != nil {
		app.UIDNumber = trimStringPtr(req.UIDNumber)
	}
	if req.RegisterNumber != nil {
		app.RegisterNumber = trimStringPtr(req.RegisterNumber)
	}
	if req.BirthDate != nil {
		parsed, err := parseDateString(req.BirthDate)
		if err != nil {
			return nil, shared.NewValidationError("Validation failed", map[string]string{"birthDate": err.Error()})
		}
		app.BirthDate = parsed
	}
	if req.Email != nil {
		app.Email = *req.Email
	}
	if req.Phone != nil {
		app.Phone = req.Phone
	}
	if req.ResidentStreet != nil {
		app.ResidentStreet = *req.ResidentStreet
	}
	if req.ResidentStreetNumber != nil {
		app.ResidentStreetNumber = *req.ResidentStreetNumber
	}
	if req.ResidentZip != nil {
		app.ResidentZip = *req.ResidentZip
	}
	if req.ResidentCity != nil {
		app.ResidentCity = *req.ResidentCity
	}
	if req.AdminNote != nil {
		app.AdminNote = req.AdminNote
	}
	if req.Einzugsart != nil {
		app.Einzugsart = *req.Einzugsart
	}
	if req.BankName != nil {
		app.BankName = trimStringPtr(req.BankName)
	}
	if req.MandateReference != nil {
		app.MandateReference = trimStringPtr(req.MandateReference)
	}
	if req.MandateDate != nil {
		parsed, err := parseDateString(req.MandateDate)
		if err != nil {
			return nil, shared.NewValidationError("Validation failed", map[string]string{"mandateDate": err.Error()})
		}
		app.MandateDate = parsed
	}
	clearMemberTypeFields(app)
	if err = validateMemberTypeFields(app); err != nil {
		return nil, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if req.MeteringPoints != nil {
		now := time.Now().UTC()
		points := make([]shared.MeteringPoint, len(req.MeteringPoints))
		for i, mp := range req.MeteringPoints {
			normalized := strings.ToUpper(strings.ReplaceAll(mp.MeteringPoint, " ", ""))
			if !validateMeteringPointFormat(normalized) {
				return nil, shared.NewValidationError("Validation failed", map[string]string{
					"meteringPoints": fmt.Sprintf("Zählpunkt %q muss mit AT beginnen und 31 Ziffern enthalten (33 Zeichen gesamt)", mp.MeteringPoint),
				})
			}
			points[i] = shared.MeteringPoint{
				MeteringPoint:       normalized,
				Direction:           shared.MeterDirection(mp.Direction),
				ParticipationFactor: mp.ParticipationFactor,
				Transformer:         trimStringPtr(mp.Transformer),
				InstallationNumber:  trimStringPtr(mp.InstallationNumber),
				InstallationName:    trimStringPtr(mp.InstallationName),
				AddressStreet:       trimStringPtr(mp.AddressStreet),
				AddressStreetNumber: trimStringPtr(mp.AddressStreetNumber),
				AddressZip:          trimStringPtr(mp.AddressZip),
				AddressCity:         trimStringPtr(mp.AddressCity),
				CreatedAt:           now,
				UpdatedAt:           now,
			}
		}
		if err := validateMeteringPointAddresses(points); err != nil {
			return nil, err
		}
		if err := s.meteringRepo.CreateBulkTx(tx, id, points); err != nil {
			return nil, fmt.Errorf("failed to update metering points: %w", err)
		}
	}

	if err := s.appRepo.UpdateAdminTx(tx, app); err != nil {
		return nil, fmt.Errorf("failed to update application: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &shared.ApplicationResponse{
		ID:              app.ID,
		ReferenceNumber: app.ReferenceNumber,
		Status:          string(app.Status),
		CreatedAt:       app.CreatedAt,
		UpdatedAt:       app.UpdatedAt,
	}, nil
}

// ChangeStatus performs an admin status transition and writes a status_log entry.
// actorID is the Keycloak user ID of the reviewer; pass "" until PROJ-4 adds auth.
func (s *AdminApplicationService) ChangeStatus(id uuid.UUID, toStatus shared.ApplicationStatus, reason, actorID string) (*shared.ChangeStatusResponse, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !isAdminTransitionAllowed(app.Status, toStatus) {
		return nil, shared.NewConflictError(
			fmt.Sprintf("transition from %s to %s is not allowed", app.Status, toStatus),
		)
	}

	// PROJ-31: when the EEG requires e-mail confirmation, block all moves
	// out of `submitted` except `rejected` until the member has clicked the
	// confirmation link. The transition map keeps `submitted → under_review`
	// because EEGs without the setting still need it.
	if app.Status == shared.StatusSubmitted && toStatus != shared.StatusRejected {
		entrypoint, epErr := s.entrypointRepo.GetByRCNumber(app.RCNumber)
		if epErr == nil && entrypoint.RequireEmailConfirmation && app.EmailConfirmedAt == nil {
			return nil, shared.NewConflictError("E-Mail-Adresse des Bewerbers ist noch nicht bestätigt — der Antrag kann erst nach Bestätigung weiterbearbeitet werden.")
		}
	}

	if requiresReason(toStatus) && reason == "" {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"reason": "a reason is required for this status transition",
		})
	}

	now := time.Now().UTC()

	// Timestamp columns that vary by target status.
	var submittedAt, approvedAt, rejectedAt, bankConfirmedAt, activatedAt *time.Time
	var needsInfoReason *string

	switch toStatus {
	case shared.StatusSubmitted:
		submittedAt = &now
	case shared.StatusApproved:
		approvedAt = &now
	case shared.StatusRejected:
		rejectedAt = &now
	case shared.StatusNeedsInfo:
		r := reason
		needsInfoReason = &r
	case shared.StatusReadyForActivation:
		// PROJ-46: only stamp bank_confirmed_at when coming from
		// awaiting_bank_confirmation (the b2b path). Auto-skip from
		// imported leaves bank_confirmed_at NULL because no bank
		// confirmation took place.
		if app.Status == shared.StatusAwaitingBankConfirmation {
			bankConfirmedAt = &now
		}
	case shared.StatusActivated:
		activatedAt = &now
	}

	var actorPtr *string
	if actorID != "" {
		actorPtr = &actorID
	}

	// PROJ-41 + PROJ-43 hard-fail: render + send the member mail BEFORE
	// opening any DB transaction. SMTP latency or backpressure can be
	// several hundred ms; doing this in-TX would hold row-locks on the
	// `application` and `status_log` rows for the whole send and block
	// concurrent admin reads/writes on the same application. If the mail
	// fails the status change never happens (we return early — no TX yet).
	// Trade-off: the mail goes out before the row is committed; if the
	// subsequent commit fails (very rare on a validated UPDATE), the
	// applicant has a mail for a state that wasn't persisted. Admin would
	// retry, applicant gets a second mail — annoying but not data-corrupting.
	if toStatus == shared.StatusRejected || toStatus == shared.StatusNeedsInfo {
		entrypoint, epErr := s.entrypointRepo.GetByRCNumber(app.RCNumber)
		if epErr != nil {
			return nil, fmt.Errorf("status-change mail: load entrypoint: %w", epErr)
		}
		sendErr := func() error {
			acquireMailSem()
			defer releaseMailSem()
			if toStatus == shared.StatusRejected {
				return s.mailService.SendRejectedNotification(app, entrypoint, reason)
			}
			return s.mailService.SendNeedsInfoNotification(app, entrypoint, reason)
		}()
		if sendErr != nil {
			return nil, fmt.Errorf("status-change mail: send failed (status change not applied): %w", sendErr)
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.appRepo.UpdateStatusAdminTx(tx, id, app.Status, toStatus, submittedAt, approvedAt, rejectedAt, needsInfoReason, actorPtr, bankConfirmedAt, activatedAt); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	fromStatus := string(app.Status)
	toStatusStr := string(toStatus)
	var reasonPtr *string
	if reason != "" {
		reasonPtr = &reason
	}
	logEntry := &shared.StatusLogEntry{
		ApplicationID:   id,
		FromStatus:      &fromStatus,
		ToStatus:        toStatusStr,
		ChangedByUserID: actorPtr,
		Reason:          reasonPtr,
		CreatedAt:       now,
	}
	if err := s.statusLogRepo.CreateTx(tx, logEntry); err != nil {
		return nil, fmt.Errorf("failed to write status log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// PROJ-46 Stage B: the approval mail (with PDF) is no longer sent at
	// `→ approved`. The PDF is now generated at import time when the
	// member_number is set (required for B2B-SEPA mandate references) —
	// see SendPostImportNotification, called from the import HTTP handler.

	// PROJ-46 Stage B: welcome mail on activation.
	if toStatus == shared.StatusActivated {
		appID := id
		go func() {
			acquireMailSem()
			defer releaseMailSem()
			reloadedApp, err := s.appRepo.GetByID(appID)
			if err != nil {
				slog.Error("activated mail: failed to reload app", "application_id", appID, "error", err)
				return
			}
			entrypoint, err := s.entrypointRepo.GetByRCNumber(reloadedApp.RCNumber)
			if err != nil {
				slog.Error("activated mail: failed to load entrypoint", "application_id", appID, "error", err)
				return
			}
			if err := s.mailService.SendActivatedNotification(reloadedApp, entrypoint); err != nil {
				slog.Error("activated mail: send failed", "application_id", appID, "error", err)
			}
		}()
	}

	// (PROJ-41/43 member mails are sent synchronously pre-commit above.)

	return &shared.ChangeStatusResponse{
		ID:     id,
		Status: string(toStatus),
	}, nil
}

// MarkImportedManually completes a stuck import by writing the operator-
// provided participant-ID + member-number, transitioning approved (with
// in-flight slot set) → imported. PROJ-34 recovery path for the orphan
// scenario where the core created the participant but the bookkeeping
// transaction failed and left the in-flight slot stuck.
//
// `targetParticipantID` and `memberNumber` are both mandatory — the admin
// reads them from eegFaktura. `reason` is appended to the status_log entry
// for the audit trail.
func (s *AdminApplicationService) MarkImportedManually(id uuid.UUID, targetParticipantID, memberNumber, reason, actorID string) (*shared.Application, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !shared.IsImportStuck(app, time.Now()) {
		return nil, shared.NewConflictError("application is not in a stuck import state")
	}
	targetParticipantID = strings.TrimSpace(targetParticipantID)
	memberNumber = strings.TrimSpace(memberNumber)
	if targetParticipantID == "" || memberNumber == "" {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"targetParticipantId": "Teilnehmer-UUID aus eegFaktura ist erforderlich",
			"memberNumber":        "Mitgliedsnummer ist erforderlich",
		})
	}

	now := time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin manual-import transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.appRepo.MarkImportedManuallyTx(tx, id, targetParticipantID, memberNumber, now); err != nil {
		return nil, err
	}

	fromStatus := string(shared.StatusApproved)
	toStatus := string(shared.StatusImported)
	var actorPtr *string
	if actorID != "" {
		actorPtr = &actorID
	}
	fullReason := strings.TrimSpace(reason)
	if fullReason == "" {
		fullReason = "Manuell als importiert markiert (Orphan-Recovery)"
	}
	fullReason += fmt.Sprintf("\n[system] target_participant_id=%s, member_number=%s", targetParticipantID, memberNumber)
	logEntry := &shared.StatusLogEntry{
		ApplicationID:   id,
		FromStatus:      &fromStatus,
		ToStatus:        toStatus,
		ChangedByUserID: actorPtr,
		Reason:          &fullReason,
		CreatedAt:       now,
	}
	if err := s.statusLogRepo.CreateTx(tx, logEntry); err != nil {
		return nil, fmt.Errorf("failed to write status log: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit manual-import: %w", err)
	}

	slog.Info("import: marked imported manually (orphan recovery)",
		"application_id", id, "actor", actorID,
		"target_participant_id", targetParticipantID, "member_number", memberNumber)

	return s.appRepo.GetByID(id)
}

// ClearImportLock releases the in-flight slot on a stuck application
// without touching its status. Risk: the original attempt may have already
// created a participant in the core — a retry then produces a duplicate.
// The admin confirms this risk in the UI. PROJ-34 fallback for the case
// where the operator cannot or does not want to recover via
// MarkImportedManually.
func (s *AdminApplicationService) ClearImportLock(id uuid.UUID, reason, actorID string) (*shared.Application, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if !shared.IsImportStuck(app, time.Now()) {
		return nil, shared.NewConflictError("application is not in a stuck import state")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"reason": "Begründung ist erforderlich",
		})
	}

	now := time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin clear-lock transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.appRepo.ClearImportLockTx(tx, id); err != nil {
		return nil, err
	}

	// status_log: from=to=approved — the row state didn't change, but the
	// audit trail records the operator intervention.
	approvedStr := string(shared.StatusApproved)
	var actorPtr *string
	if actorID != "" {
		actorPtr = &actorID
	}
	fullReason := "Import-Lock manuell zurückgesetzt: " + reason
	if app.TargetParticipantID != nil && *app.TargetParticipantID != "" {
		fullReason += fmt.Sprintf("\n[system] previous target_participant_id=%s — Duplikatsrisiko bei erneutem Import", *app.TargetParticipantID)
	}
	logEntry := &shared.StatusLogEntry{
		ApplicationID:   id,
		FromStatus:      &approvedStr,
		ToStatus:        approvedStr,
		ChangedByUserID: actorPtr,
		Reason:          &fullReason,
		CreatedAt:       now,
	}
	if err := s.statusLogRepo.CreateTx(tx, logEntry); err != nil {
		return nil, fmt.Errorf("failed to write status log: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit clear-lock: %w", err)
	}

	slog.Warn("import: lock cleared manually — duplicate risk",
		"application_id", id, "actor", actorID,
		"previous_target_participant_id", derefStr(app.TargetParticipantID))

	return s.appRepo.GetByID(id)
}

// SendPostImportNotification (PROJ-46 Stage B) loads everything needed to
// render the Beitrittsbestätigungs-PDF, generates it (with member_number),
// and sends the member welcome mail + EEG copy via MailService.
//
// Best-effort: errors are logged but never propagated — the import was
// already persisted by the caller. Use this from the HTTP import handler
// after a successful Import() returns.
func (s *AdminApplicationService) SendPostImportNotification(appID uuid.UUID) {
	acquireMailSem()
	defer releaseMailSem()

	reloadedApp, err := s.appRepo.GetByID(appID)
	if err != nil {
		slog.Error("imported mail: failed to reload app", "application_id", appID, "error", err)
		return
	}
	entrypoint, err := s.entrypointRepo.GetByRCNumber(reloadedApp.RCNumber)
	if err != nil {
		slog.Error("imported mail: failed to load entrypoint", "application_id", appID, "error", err)
		return
	}
	mps, err := s.meteringRepo.GetByApplicationID(appID)
	if err != nil {
		slog.Error("imported mail: failed to load metering points", "application_id", appID, "error", err)
		return
	}
	statusLog, err := s.statusLogRepo.GetByApplicationID(appID)
	if err != nil {
		slog.Error("imported mail: failed to load status log", "application_id", appID, "error", err)
		return
	}
	consents, err := s.consentRepo.GetByApplicationID(appID)
	if err != nil {
		slog.Error("imported mail: failed to load consents", "application_id", appID, "error", err)
		return
	}
	fieldConfig, fcErr := s.fieldConfigRepo.Get(reloadedApp.RCNumber)
	if fcErr != nil {
		slog.Warn("imported mail: failed to load field config", "application_id", appID, "error", fcErr)
		fieldConfig = map[string]FieldConfigEntry{}
	}

	pdfData := buildApprovalPDFData(reloadedApp, mps, statusLog, consents, entrypoint, toStateMap(fieldConfig))
	if logoBytes, logoMime, logoErr := s.entrypointRepo.GetLogo(reloadedApp.RCNumber); logoErr == nil && len(logoBytes) > 0 {
		pdfData.LogoBytes = logoBytes
		pdfData.LogoMIME = logoMime
	}
	pdfBytes, pdfErr := s.approvalPDFGenerator.GenerateApproval(pdfData)
	pdfFailed := pdfErr != nil
	if pdfFailed {
		slog.Error("imported mail: failed to generate PDF", "application_id", appID, "error", pdfErr)
	}

	// PROJ-47 / PROJ-48: SEPA-Mandat als zweiter Anhang an die Import-Mail,
	// mit ausgefüllter Mandatsreferenz = Mitgliedsnummer. Zwei Pfade:
	//
	//   - einzugsart=b2b → Firmenlastschrift-Mandat (PROJ-47, unverändert)
	//   - einzugsart=core UND EEG-Setting sepa_mandate_at_import=TRUE →
	//     Basis-Lastschriftmandat mit Mandatsreferenz (PROJ-48, neu)
	//
	// In allen anderen Fällen (kein_sepa, oder core ohne at-import-Setting)
	// gibt es keinen zweiten Anhang — die Submit-Mail hat das Basis-Mandat
	// dann bereits (ohne Mandatsreferenz) ausgeliefert.
	//
	// Best-effort: ein PDF-Fehler blockiert die Hauptmail nicht.
	var mandatePDF []byte
	wantsB2B := reloadedApp.Einzugsart == "b2b"
	wantsCoreAtImport := reloadedApp.Einzugsart == "core" && entrypoint.SEPAMandateAtImport
	if wantsB2B || wantsCoreAtImport {
		if mandate := buildSEPAMandateData(reloadedApp, entrypoint); mandate != nil {
			if reloadedApp.MemberNumber != nil {
				mandate.MandateReference = *reloadedApp.MemberNumber
			}
			// Bei B2B: Debtor-Name muss der Firmenname sein.
			if wantsB2B && reloadedApp.CompanyName != nil && *reloadedApp.CompanyName != "" {
				mandate.MemberName = *reloadedApp.CompanyName
			}
			if logoBytes, logoMime, logoErr := s.entrypointRepo.GetLogo(reloadedApp.RCNumber); logoErr == nil && len(logoBytes) > 0 {
				mandate.LogoBytes = logoBytes
				mandate.LogoMIME = logoMime
			}
			var mandateBytes []byte
			var mandateErr error
			if wantsB2B {
				mandateBytes, mandateErr = s.sepaMandateGenerator.GenerateCompany(*mandate)
			} else {
				mandateBytes, mandateErr = s.sepaMandateGenerator.Generate(*mandate)
			}
			if mandateErr != nil {
				slog.Warn("imported mail: failed to generate SEPA mandate", "application_id", appID, "einzugsart", reloadedApp.Einzugsart, "error", mandateErr)
			} else {
				mandatePDF = mandateBytes
			}
		} else {
			slog.Info("imported mail: skipping SEPA mandate (EEG missing required fields)", "application_id", appID, "rc", reloadedApp.RCNumber)
		}
	}

	if err := s.mailService.SendImportedNotification(reloadedApp, entrypoint, pdfBytes, pdfFailed, mandatePDF); err != nil {
		slog.Error("imported mail: send failed", "application_id", appID, "error", err)
	}
}

// ResetImport returns an imported (or post-import) application to status
// `approved` so the admin can re-import after the participant was deleted
// in the eegFaktura core. The transition is deliberately NOT in
// adminTransitions — it is only reachable through this dedicated method to
// keep the generic /status endpoint conservative. See PROJ-30.
//
// PROJ-46 extension: allowed source statuses are now `imported`,
// `awaiting_bank_confirmation`, and `ready_for_activation`. `activated`
// is intentionally excluded — an active member must be deactivated in the
// core first, not silently un-onboarded here (User-Wunsch A).
//
// The reason is mandatory (Q3 of PROJ-30) and is written to the status_log.
// The previous target_participant_id is appended to the reason as
// `[system] previous target_participant_id=<uuid>` so the audit trail
// preserves the lost UUID (Q1).
//
// PROJ-38 note: the PROJ-31 e-mail-confirmation gate intentionally does
// NOT apply here. The application already went through `approved →
// imported`, which means it was once vetted (either pre-PROJ-31 or with
// confirmation). Re-vetting the member's e-mail at reset time would only
// matter if the EEG retroactively turned the toggle on, which equally
// affects every historical `approved` row and is by design out of scope.
func (s *AdminApplicationService) ResetImport(id uuid.UUID, reason, actorID string) (*shared.Application, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	resetable := map[shared.ApplicationStatus]bool{
		shared.StatusImported:                 true,
		shared.StatusAwaitingBankConfirmation: true,
		shared.StatusReadyForActivation:       true,
	}
	if !resetable[app.Status] {
		return nil, shared.NewConflictError(
			fmt.Sprintf("only imported / awaiting_bank_confirmation / ready_for_activation applications can be reset (current: %s)", app.Status),
		)
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"reason": "Begründung ist erforderlich",
		})
	}

	previousParticipantID := ""
	previousMemberNumber := ""
	fullReason := reason
	if app.TargetParticipantID != nil && *app.TargetParticipantID != "" {
		previousParticipantID = *app.TargetParticipantID
		fullReason = fmt.Sprintf("%s\n[system] previous target_participant_id=%s", fullReason, previousParticipantID)
	}
	if app.MemberNumber != nil && *app.MemberNumber != "" {
		previousMemberNumber = *app.MemberNumber
		fullReason = fmt.Sprintf("%s\n[system] previous member_number=%s", fullReason, previousMemberNumber)
	}

	now := time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin reset transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.appRepo.ResetImportTx(tx, id); err != nil {
		return nil, err
	}

	fromStatus := string(app.Status)
	toStatus := string(shared.StatusApproved)
	var actorPtr *string
	if actorID != "" {
		actorPtr = &actorID
	}
	logEntry := &shared.StatusLogEntry{
		ApplicationID:   id,
		FromStatus:      &fromStatus,
		ToStatus:        toStatus,
		ChangedByUserID: actorPtr,
		Reason:          &fullReason,
		CreatedAt:       now,
	}
	if err := s.statusLogRepo.CreateTx(tx, logEntry); err != nil {
		return nil, fmt.Errorf("failed to write status log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit reset: %w", err)
	}

	slog.Info("import: reset to approved",
		"application_id", id,
		"actor", actorID,
		"previous_target_participant_id", previousParticipantID,
		"previous_member_number", previousMemberNumber,
	)

	return s.appRepo.GetByID(id)
}

// ReassignEEG moves an application from its current EEG to a different EEG
// during admin review (PROJ-40). The admin must be authorized for both
// source and target (or be a superuser → allowedRCNumbers == nil). The
// reference number is regenerated from the target EEG's per-year counter
// (PROJ-35), so the member-facing identifier matches the new EEG. Old
// rc_number + old reference_number are archived in the status_log reason.
//
// Reassignable statuses (enforced both here and in UpdateRCNumberTx):
// submitted, email_confirmed, under_review, needs_info. Anything past
// approval — or `rejected`/`import_failed`/`draft` — is rejected with 409.
//
// No member notification in V1 (see PROJ-40 spec § Q5).
func (s *AdminApplicationService) ReassignEEG(id uuid.UUID, targetRCNumber, reason, actorID string, allowedRCNumbers []string) (*shared.Application, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	targetRCNumber = strings.ToUpper(strings.TrimSpace(targetRCNumber))
	reason = strings.TrimSpace(reason)
	if targetRCNumber == "" {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"targetRcNumber": "Ziel-EEG ist erforderlich",
		})
	}
	if reason == "" || len(reason) < 5 {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"reason": "Begründung ist erforderlich (mindestens 5 Zeichen)",
		})
	}
	if targetRCNumber == app.RCNumber {
		return nil, shared.NewConflictError("Quelle und Ziel sind identisch")
	}

	reassignable := map[shared.ApplicationStatus]bool{
		shared.StatusSubmitted:       true,
		shared.StatusEmailConfirmed:  true,
		shared.StatusUnderReview:     true,
		shared.StatusNeedsInfo:       true,
	}
	if !reassignable[app.Status] {
		return nil, shared.NewConflictError(
			fmt.Sprintf("status %s kann nicht umzuordnet werden — nur submitted/email_confirmed/under_review/needs_info", app.Status))
	}

	// Tenant check on BOTH source and target. allowedRCNumbers == nil
	// means superuser → unrestricted.
	if allowedRCNumbers != nil {
		hasSource, hasTarget := false, false
		for _, rc := range allowedRCNumbers {
			if rc == app.RCNumber {
				hasSource = true
			}
			if rc == targetRCNumber {
				hasTarget = true
			}
		}
		if !hasSource || !hasTarget {
			return nil, shared.ErrForbidden
		}
	}

	targetEP, err := s.entrypointRepo.GetByRCNumber(targetRCNumber)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewValidationError("Validation failed", map[string]string{
				"targetRcNumber": "Ziel-EEG existiert nicht",
			})
		}
		return nil, fmt.Errorf("failed to load target entrypoint: %w", err)
	}
	if !targetEP.IsActive {
		return nil, shared.NewConflictError("Ziel-EEG ist nicht aktiv")
	}

	// Mint the new reference number on the target's counter BEFORE the TX
	// so a failure here doesn't dirty the application row. NextReferenceNumber
	// already runs in its own atomic INSERT … ON CONFLICT — no race.
	newRef, err := s.appRepo.NextReferenceNumber(targetRCNumber, time.Now().Year())
	if err != nil {
		return nil, fmt.Errorf("failed to mint new reference number: %w", err)
	}

	oldRC := app.RCNumber
	oldRef := app.ReferenceNumber
	fullReason := fmt.Sprintf("%s\n[system] previous rc_number=%s\n[system] previous reference_number=%s",
		reason, oldRC, oldRef)
	now := time.Now().UTC()

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin reassign transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.appRepo.UpdateRCNumberTx(tx, id, oldRC, targetRCNumber, newRef); err != nil {
		return nil, err
	}

	// Log entry: status unchanged (same on both sides) — the reassign
	// itself isn't a status change, but the audit trail needs to record it.
	currentStatus := string(app.Status)
	var actorPtr *string
	if actorID != "" {
		actorPtr = &actorID
	}
	logEntry := &shared.StatusLogEntry{
		ApplicationID:   id,
		FromStatus:      &currentStatus,
		ToStatus:        currentStatus,
		ChangedByUserID: actorPtr,
		Reason:          &fullReason,
		CreatedAt:       now,
	}
	if err := s.statusLogRepo.CreateTx(tx, logEntry); err != nil {
		return nil, fmt.Errorf("failed to write status log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit reassign: %w", err)
	}

	slog.Info("application reassigned",
		"application_id", id,
		"actor", actorID,
		"from_rc", oldRC,
		"to_rc", targetRCNumber,
		"old_reference_number", oldRef,
		"new_reference_number", newRef,
	)

	return s.appRepo.GetByID(id)
}

// BulkChangeStatus applies a status transition to multiple applications.
// Applications whose transition is not allowed (wrong current status, wrong tenant,
// or not found) are added to skipped instead of returning an error.
// allowedRCNumbers may be nil (superuser — no restriction) or a non-nil slice
// (tenant-admin — must match app.RCNumber).
const bulkActionMaxIDs = 50

func (s *AdminApplicationService) BulkChangeStatus(
	ids []uuid.UUID,
	toStatus shared.ApplicationStatus,
	reason, actorID string,
	allowedRCNumbers []string,
) (succeeded, skipped []uuid.UUID, err error) {
	if len(ids) > bulkActionMaxIDs {
		return nil, nil, shared.NewValidationError("Validation failed", map[string]string{
			"ids": fmt.Sprintf("bulk action is limited to %d applications", bulkActionMaxIDs),
		})
	}
	for _, id := range ids {
		app, appErr := s.appRepo.GetByID(id)
		if appErr != nil {
			skipped = append(skipped, id)
			continue
		}
		if allowedRCNumbers != nil && !containsStr(allowedRCNumbers, app.RCNumber) {
			skipped = append(skipped, id)
			continue
		}
		if !isAdminTransitionAllowed(app.Status, toStatus) {
			skipped = append(skipped, id)
			continue
		}
		if _, changeErr := s.ChangeStatus(id, toStatus, reason, actorID); changeErr != nil {
			skipped = append(skipped, id)
			continue
		}
		succeeded = append(succeeded, id)
	}
	if succeeded == nil {
		succeeded = []uuid.UUID{}
	}
	if skipped == nil {
		skipped = []uuid.UUID{}
	}
	return succeeded, skipped, nil
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// DeleteApplication permanently removes an application.
// Only draft and rejected applications may be deleted.
func (s *AdminApplicationService) DeleteApplication(id uuid.UUID) error {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return err
	}

	deletable := map[shared.ApplicationStatus]bool{
		shared.StatusDraft:    true,
		shared.StatusRejected: true,
	}
	if !deletable[app.Status] {
		return shared.NewConflictError(
			fmt.Sprintf("applications in status %s cannot be deleted", app.Status),
		)
	}

	return s.appRepo.Delete(id)
}

// DeleteDrafts deletes all draft applications for the given RC numbers and returns the count.
func (s *AdminApplicationService) DeleteDrafts(rcNumbers []string) (int64, error) {
	return s.appRepo.DeleteDraftsByRCNumbers(rcNumbers)
}

// DeleteAllDrafts deletes every draft application across every EEG. Reserved
// for superusers — tenant-scoped admins must use DeleteDrafts instead.
func (s *AdminApplicationService) DeleteAllDrafts() (int64, error) {
	return s.appRepo.DeleteAllDrafts()
}

// GetRCNumberByID is a thin pass-through for tenant-access checks so the
// HTTP layer doesn't have to load the full application detail just to compare
// rc_number against the calling admin's allowed RC list.
func (s *AdminApplicationService) GetRCNumberByID(id uuid.UUID) (string, error) {
	return s.appRepo.GetRCNumberByID(id)
}

// UpdateAdminNote replaces only the admin_note column for the given
// application. Used by the dedicated PATCH endpoint to avoid touching any
// other field (PROJ-7/15 attributes, metering points with their
// participation factors, etc.) when the admin just wants to edit a note.
func (s *AdminApplicationService) UpdateAdminNote(id uuid.UUID, note string) error {
	const maxNoteLen = 2000
	if len(note) > maxNoteLen {
		return shared.NewValidationError("Validation failed", map[string]string{
			"note": fmt.Sprintf("Notiz darf maximal %d Zeichen lang sein", maxNoteLen),
		})
	}
	return s.appRepo.UpdateAdminNote(id, strings.TrimSpace(note))
}

func buildApprovalPDFData(
	app *shared.Application,
	meteringPoints []shared.MeteringPoint,
	statusLog []shared.StatusLogEntry,
	consents []shared.DocumentConsent,
	entrypoint *shared.RegistrationEntrypoint,
	fieldConfig map[string]string,
) pdf.ApprovalPDFData {
	eegName := derefStr(entrypoint.EEGName)

	approvedAt := time.Now()
	if app.ApprovedAt != nil {
		approvedAt = *app.ApprovedAt
	}

	mpPDFs := make([]pdf.MeteringPointPDF, len(meteringPoints))
	for i, mp := range meteringPoints {
		dir := "Verbrauch"
		if mp.Direction == shared.DirectionProduction {
			dir = "Einspeisung"
		}
		addrLine := ""
		if mp.HasDeviatingAddress() {
			street := derefStr(mp.AddressStreet)
			streetNumber := derefStr(mp.AddressStreetNumber)
			zip := derefStr(mp.AddressZip)
			city := derefStr(mp.AddressCity)
			addrLine = strings.TrimSpace(street+" "+streetNumber) + ", " + strings.TrimSpace(zip+" "+city)
		}
		mpPDFs[i] = pdf.MeteringPointPDF{
			MeteringPoint:       mp.MeteringPoint,
			Direction:           dir,
			ParticipationFactor: mp.ParticipationFactor,
			AddressLine:         addrLine,
			GenerationLine:      mail.FormatGenerationLine(&meteringPoints[i]),
		}
	}

	consentPDFs := make([]pdf.ConsentPDF, len(consents))
	for i, c := range consents {
		consentPDFs[i] = pdf.ConsentPDF{
			Title:         c.Title,
			URL:           c.URL,
			ConsentedAt:   c.ConsentedAt,
			Informational: c.ConsentType == shared.ConsentTypeInformational,
		}
	}

	slPDFs := make([]pdf.StatusLogPDF, len(statusLog))
	for i, sl := range statusLog {
		from := ""
		if sl.FromStatus != nil {
			from = *sl.FromStatus
		}
		reason := ""
		if sl.Reason != nil {
			reason = *sl.Reason
		}
		slPDFs[i] = pdf.StatusLogPDF{
			FromStatus: from,
			ToStatus:   sl.ToStatus,
			Timestamp:  sl.CreatedAt,
			Reason:     reason,
		}
	}

	memberTypeLabel := approvalMemberTypeLabel(app.MemberType)

	return pdf.ApprovalPDFData{
		EEGName:              eegName,
		RCNumber:             app.RCNumber,
		ApprovedAt:           approvedAt,
		ReferenceNumber:      app.ReferenceNumber,
		MemberType:           memberTypeLabel,
		Titel:                derefStr(app.Titel),
		TitelNach:            derefStr(app.TitelNach),
		Firstname:            derefStr(app.Firstname),
		Lastname:             derefStr(app.Lastname),
		BirthDate:            app.BirthDate,
		CompanyName:          derefStr(app.CompanyName),
		UIDNumber:            derefStr(app.UIDNumber),
		RegisterNumber:       derefStr(app.RegisterNumber),
		Email:                app.Email,
		Phone:                derefStr(app.Phone),
		ResidentStreet:       app.ResidentStreet,
		ResidentStreetNumber: app.ResidentStreetNumber,
		ResidentZip:          app.ResidentZip,
		ResidentCity:         app.ResidentCity,
		IBAN:                 derefStr(app.IBAN),
		AccountHolder:        derefStr(app.AccountHolder),
		BankName:             derefStr(app.BankName),
		SepaMandateType:      approvalSepaMandateType(app, entrypoint),
		MeteringPoints:       mpPDFs,
		Consents:             consentPDFs,
		StatusLog:            slPDFs,
		ConfigurableFields:   buildApprovalConfigurableFields(app, fieldConfig),
		PrivacyAccepted:       app.PrivacyAccepted,
		PrivacyVersion:        derefStr(app.PrivacyVersion),
		PrivacyAcceptedAt:     app.PrivacyAcceptedAt,
		AccuracyConfirmed:     app.AccuracyConfirmed,
		AccuracyConfirmedAt:   app.SubmittedAt, // accuracy is validated at submit-time
		SepaMandateAccepted:   app.SepaMandateAccepted,
		SepaMandateAcceptedAt: app.SepaMandateAcceptedAt,
		SEPAMandateEnabled:    entrypoint.SEPAMandateEnabled,
		MemberNumber:         app.MemberNumber,
		// PROJ-37: only set both fields together — the PDF render skips
		// the section if either is missing. EEG entrypoint provides the
		// price; the application carries the count. When the EEG feature
		// is off, both stay nil and no section is rendered.
		CooperativeSharesCount:      cooperativeSharesPDFFields(app, entrypoint),
		CooperativeShareAmountCents: cooperativeShareAmountPDFField(app, entrypoint),
	}
}

// cooperativeSharesPDFFields returns the count to render in the PDF, or
// nil when the EEG hasn't enabled the feature (so the PDF section is
// skipped regardless of any legacy non-zero count). Companion helper:
// cooperativeShareAmountPDFField for the same gate on the price field.
func cooperativeSharesPDFFields(app *shared.Application, ep *shared.RegistrationEntrypoint) *int {
	if ep == nil || !ep.CooperativeSharesEnabled {
		return nil
	}
	return app.CooperativeSharesCount
}

func cooperativeShareAmountPDFField(app *shared.Application, ep *shared.RegistrationEntrypoint) *int64 {
	if ep == nil || !ep.CooperativeSharesEnabled || app.CooperativeSharesCount == nil {
		return nil
	}
	return ep.CooperativeShareAmountCents
}

func approvalMemberTypeLabel(mt shared.MemberType) string {
	switch mt {
	case shared.MemberTypePrivate:
		return "Privatperson"
	case shared.MemberTypeSoleProprietor:
		return "Kleinunternehmer"
	case shared.MemberTypeFarmer:
		return "Landwirt"
	case shared.MemberTypeCompany:
		return "Unternehmen"
	case shared.MemberTypeMunicipality:
		return "Gemeinde"
	case shared.MemberTypeAssociation:
		return "Verein"
	default:
		return string(mt)
	}
}

func approvalSepaMandateType(app *shared.Application, ep *shared.RegistrationEntrypoint) string {
	if !app.SepaMandateAccepted {
		return "Per E-Mail"
	}
	if ep.UseCompanySEPAMandate &&
		(app.MemberType == shared.MemberTypeCompany || app.MemberType == shared.MemberTypeAssociation) {
		return "Firmenlastschrift"
	}
	return "Basislastschrift"
}

func buildApprovalConfigurableFields(app *shared.Application, fieldConfig map[string]string) []pdf.ConfigurableFieldPDF {
	var result []pdf.ConfigurableFieldPDF

	add := func(name, label, value string) {
		state := fieldConfig[name]
		if state == "hidden" || state == "" {
			return
		}
		if value == "" {
			return
		}
		result = append(result, pdf.ConfigurableFieldPDF{Label: label, Value: value})
	}

	if app.HeatPump != nil {
		v := "Nein"
		if *app.HeatPump {
			v = "Ja"
		}
		add("heat_pump", "Wärmepumpe vorhanden", v)
	}
	if app.ElectricVehicle != nil {
		v := "Nein"
		if *app.ElectricVehicle {
			v = "Ja"
		}
		add("electric_vehicle", "Elektrofahrzeug vorhanden", v)
	}
	if app.ElectricVehicleCount != nil {
		add("electric_vehicle_count", "Anzahl E-Fahrzeuge", fmt.Sprintf("%d", *app.ElectricVehicleCount))
	}
	if app.ElectricVehicleAnnualKm != nil {
		add("electric_vehicle_annual_km", "Jahres-Kilometer (E-Fahrzeuge)", fmt.Sprintf("%d km", *app.ElectricVehicleAnnualKm))
	}
	if app.ElectricHotWater != nil {
		v := "Nein"
		if *app.ElectricHotWater {
			v = "Ja"
		}
		add("electric_hot_water", "Warmwasser elektrisch", v)
	}
	if app.PersonsInHousehold != nil {
		add("persons_in_household", "Personen im Haushalt", fmt.Sprintf("%d", *app.PersonsInHousehold))
	}
	if app.ConsumptionPreviousYear != nil {
		add("consumption_previous_year", "Verbrauch Vorjahr (kWh)", fmt.Sprintf("%d", *app.ConsumptionPreviousYear))
	}
	if app.ConsumptionForecast != nil {
		add("consumption_forecast", "Verbrauch Prognose (kWh)", fmt.Sprintf("%d", *app.ConsumptionForecast))
	}
	if app.FeedInForecast != nil {
		add("feed_in_forecast", "Einspeisung Prognose (kWh)", fmt.Sprintf("%d", *app.FeedInForecast))
	}
	if app.PvPowerKwp != nil {
		add("pv_power_kwp", "PV-Leistung (kWp)", fmt.Sprintf("%.2f", *app.PvPowerKwp))
	}
	if app.MembershipStartDate != nil {
		add("membership_start_date", "Beitrittsdatum", app.MembershipStartDate.Format("02.01.2006"))
	}
	// PROJ-44: Netzbetreiber-Vollmacht. Wir rendern nur "Ja" (erteilt) —
	// "Nein" wird unterdrückt, weil das Feld bei Bestandsanträgen per
	// Default FALSE ist und nicht auf der Genehmigungs-PDF auftauchen soll.
	if app.NetworkOperatorAuthorization {
		add("network_operator_authorization", "Netzbetreiber-Vollmacht erteilt", "Ja")
	}
	return result
}

func isAdminTransitionAllowed(from, to shared.ApplicationStatus) bool {
	allowed, ok := adminTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

func requiresReason(status shared.ApplicationStatus) bool {
	return status == shared.StatusNeedsInfo || status == shared.StatusRejected
}

// ExportApplicationExcel generates an xlsx file for a given application in
// eegFaktura import format. Only applications in approved, imported, or
// import_failed status can be exported.
func (s *AdminApplicationService) ExportApplicationExcel(id uuid.UUID) ([]byte, string, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, "", err
	}

	exportable := map[shared.ApplicationStatus]bool{
		shared.StatusApproved:     true,
		shared.StatusImported:     true,
		shared.StatusImportFailed: true,
	}
	if !exportable[app.Status] {
		return nil, "", shared.NewConflictError(
			fmt.Sprintf("excel export not available for applications in status %s", app.Status),
		)
	}

	meteringPoints, err := s.meteringRepo.GetByApplicationID(id)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch metering points: %w", err)
	}
	if len(meteringPoints) == 0 {
		return nil, "", shared.NewUnprocessableEntityError("application has no metering points")
	}

	ep, err := s.entrypointRepo.GetByRCNumber(app.RCNumber)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch registration entrypoint: %w", err)
	}

	eegID := ""
	if ep.EegID != nil {
		eegID = *ep.EegID
	}

	data, err := excel.GenerateExcel(app, meteringPoints, eegID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate excel: %w", err)
	}

	filename := app.ReferenceNumber + ".xlsx"
	return data, filename, nil
}

var approvalPDFStatuses = map[shared.ApplicationStatus]bool{
	shared.StatusApproved:     true,
	shared.StatusImported:     true,
	shared.StatusImportFailed: true,
}

func (s *AdminApplicationService) GenerateApprovalPDF(id uuid.UUID) ([]byte, string, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, "", err
	}
	if !approvalPDFStatuses[app.Status] {
		return nil, "", shared.NewConflictError(
			fmt.Sprintf("approval PDF not available for applications in status %s", app.Status),
		)
	}

	mps, err := s.meteringRepo.GetByApplicationID(id)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch metering points: %w", err)
	}
	statusLog, err := s.statusLogRepo.GetByApplicationID(id)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch status log: %w", err)
	}
	consents, err := s.consentRepo.GetByApplicationID(id)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch consents: %w", err)
	}
	entrypoint, err := s.entrypointRepo.GetByRCNumber(app.RCNumber)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch entrypoint: %w", err)
	}
	fieldConfig, err := s.fieldConfigRepo.Get(app.RCNumber)
	if err != nil {
		fieldConfig = map[string]FieldConfigEntry{}
	}

	pdfData := buildApprovalPDFData(app, mps, statusLog, consents, entrypoint, toStateMap(fieldConfig))
	// PROJ-33: embed the cached EEG logo. Optional — same fallback story
	// as in the approval-mail path: a missing logo simply renders without it.
	if logoBytes, logoMime, logoErr := s.entrypointRepo.GetLogo(app.RCNumber); logoErr == nil && len(logoBytes) > 0 {
		pdfData.LogoBytes = logoBytes
		pdfData.LogoMIME = logoMime
	}
	pdfBytes, err := s.approvalPDFGenerator.GenerateApproval(pdfData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate approval PDF: %w", err)
	}

	filename := "beitrittsbestaetigung-" + app.ReferenceNumber + ".pdf"
	return pdfBytes, filename, nil
}
