package application

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/excel"
	"github.com/your-org/eegfaktura-member-onboarding/internal/mail"
	"github.com/your-org/eegfaktura-member-onboarding/internal/pdf"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ApplicationListFilters holds optional filter parameters for the admin list endpoint.
type ApplicationListFilters struct {
	Status          *string
	ReferenceNumber *string
	Lastname        *string
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
}

// adminTransitions defines which status changes the admin endpoint may perform.
// Forward import transitions (approved→imported etc.) are handled by the dedicated
// import endpoint (PROJ-4). The reset transition import_failed→approved is an admin
// action (re-approve after failed import) and is handled here so the approval email
// is re-sent consistently via ChangeStatus.
var adminTransitions = map[shared.ApplicationStatus][]shared.ApplicationStatus{
	shared.StatusSubmitted:    {shared.StatusUnderReview},
	shared.StatusUnderReview:  {shared.StatusNeedsInfo, shared.StatusApproved, shared.StatusRejected},
	shared.StatusNeedsInfo:    {shared.StatusSubmitted},
	shared.StatusImportFailed: {shared.StatusApproved},
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
	}
}

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

	return &shared.AdminApplicationDetailResponse{
		Application:    *app,
		MeteringPoints: meteringPoints,
		StatusLog:      statusLog,
		Consents:       consentViews,
	}, nil
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
	if req.MemberNumber != nil {
		app.MemberNumber = req.MemberNumber
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
			points[i] = shared.MeteringPoint{
				MeteringPoint:       mp.MeteringPoint,
				Direction:           shared.MeterDirection(mp.Direction),
				ParticipationFactor: mp.ParticipationFactor,
				CreatedAt:           now,
				UpdatedAt:           now,
			}
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

	if requiresReason(toStatus) && reason == "" {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"reason": "a reason is required for this status transition",
		})
	}

	now := time.Now().UTC()

	// Timestamp columns that vary by target status.
	var submittedAt, approvedAt, rejectedAt *time.Time
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
	}

	var actorPtr *string
	if actorID != "" {
		actorPtr = &actorID
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.appRepo.UpdateStatusAdminTx(tx, id, toStatus, submittedAt, approvedAt, rejectedAt, needsInfoReason, actorPtr); err != nil {
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

	// Trigger approval notification asynchronously after a successful commit.
	if toStatus == shared.StatusApproved {
		appID := id
		go func() {
			acquireMailSem()
			defer releaseMailSem()
			reloadedApp, err := s.appRepo.GetByID(appID)
			if err != nil {
				slog.Error("approval mail: failed to reload app", "application_id", appID, "error", err)
				return
			}
			entrypoint, err := s.entrypointRepo.GetByRCNumber(reloadedApp.RCNumber)
			if err != nil {
				slog.Error("approval mail: failed to load entrypoint", "application_id", appID, "error", err)
				return
			}
			if entrypoint.ContactEmail == nil || *entrypoint.ContactEmail == "" {
				return
			}
			mps, err := s.meteringRepo.GetByApplicationID(appID)
			if err != nil {
				slog.Error("approval mail: failed to load metering points", "application_id", appID, "error", err)
				return
			}
			statusLog, err := s.statusLogRepo.GetByApplicationID(appID)
			if err != nil {
				slog.Error("approval mail: failed to load status log", "application_id", appID, "error", err)
				return
			}
			consents, err := s.consentRepo.GetByApplicationID(appID)
			if err != nil {
				slog.Error("approval mail: failed to load consents", "application_id", appID, "error", err)
				return
			}
			fieldConfig, fcErr := s.fieldConfigRepo.Get(reloadedApp.RCNumber)
			if fcErr != nil {
				slog.Warn("approval mail: failed to load field config", "application_id", appID, "error", fcErr)
				fieldConfig = map[string]FieldConfigEntry{}
			}

			pdfData := buildApprovalPDFData(reloadedApp, mps, statusLog, consents, entrypoint, toStateMap(fieldConfig))
			pdfBytes, pdfErr := s.approvalPDFGenerator.GenerateApproval(pdfData)
			pdfFailed := pdfErr != nil
			if pdfFailed {
				slog.Error("approval mail: failed to generate PDF", "application_id", appID, "error", pdfErr)
			}

			if err := s.mailService.SendApprovalEmail(reloadedApp, entrypoint, pdfBytes, pdfFailed); err != nil {
				slog.Error("approval mail: failed to send email", "application_id", appID, "error", err)
			}
		}()
	}

	return &shared.ChangeStatusResponse{
		ID:     id,
		Status: string(toStatus),
	}, nil
}

// BulkChangeStatus applies a status transition to multiple applications.
// Applications whose transition is not allowed (wrong current status, wrong tenant,
// or not found) are added to skipped instead of returning an error.
// allowedRCNumbers may be nil (superuser — no restriction) or a non-nil slice
// (tenant-admin — must match app.RCNumber).
func (s *AdminApplicationService) BulkChangeStatus(
	ids []uuid.UUID,
	toStatus shared.ApplicationStatus,
	reason, actorID string,
	allowedRCNumbers []string,
) (succeeded, skipped []uuid.UUID, err error) {
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
		mpPDFs[i] = pdf.MeteringPointPDF{
			MeteringPoint:       mp.MeteringPoint,
			Direction:           dir,
			ParticipationFactor: mp.ParticipationFactor,
		}
	}

	consentPDFs := make([]pdf.ConsentPDF, len(consents))
	for i, c := range consents {
		consentPDFs[i] = pdf.ConsentPDF{
			Title:       c.Title,
			URL:         c.URL,
			ConsentedAt: c.ConsentedAt,
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
		SepaMandateType:      approvalSepaMandateType(app, entrypoint),
		MeteringPoints:       mpPDFs,
		Consents:             consentPDFs,
		StatusLog:            slPDFs,
		ConfigurableFields:   buildApprovalConfigurableFields(app, fieldConfig),
		PrivacyAccepted:      app.PrivacyAccepted,
		PrivacyVersion:       derefStr(app.PrivacyVersion),
		AccuracyConfirmed:    app.AccuracyConfirmed,
		SepaMandateAccepted:  app.SepaMandateAccepted,
		MemberNumber:         app.MemberNumber,
	}
}

func approvalMemberTypeLabel(mt shared.MemberType) string {
	switch mt {
	case shared.MemberTypePrivate:
		return "Privatperson"
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
	pdfBytes, err := s.approvalPDFGenerator.GenerateApproval(pdfData)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate approval PDF: %w", err)
	}

	filename := "beitrittsbestaetigung-" + app.ReferenceNumber + ".pdf"
	return pdfBytes, filename, nil
}
