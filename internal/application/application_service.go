package application

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/mail"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ApplicationService handles business logic for applications
type ApplicationService struct {
	db             *sql.DB
	appRepo        *ApplicationRepository
	meteringRepo   *MeteringPointRepository
	statusLogRepo  *StatusLogRepository
	entrypointRepo *RegistrationEntrypointRepository
	mailService    mail.MailService
}

// NewApplicationService creates a new application service
func NewApplicationService(
	db *sql.DB,
	appRepo *ApplicationRepository,
	meteringRepo *MeteringPointRepository,
	statusLogRepo *StatusLogRepository,
	entrypointRepo *RegistrationEntrypointRepository,
	mailService mail.MailService,
) *ApplicationService {
	return &ApplicationService{
		db:             db,
		appRepo:        appRepo,
		meteringRepo:   meteringRepo,
		statusLogRepo:  statusLogRepo,
		entrypointRepo: entrypointRepo,
		mailService:    mailService,
	}
}

// CreateApplication creates a new application wrapped in a single database transaction.
// If metering point insertion fails the application row is rolled back automatically.
func (s *ApplicationService) CreateApplication(req shared.CreateApplicationRequest) (*shared.ApplicationResponse, error) {
	// Resolve RC number via registration_entrypoint — never reads core tables
	ep, err := s.entrypointRepo.GetByRCNumber(strings.ToUpper(req.RCNumber))
	if err != nil {
		return nil, err
	}
	if !ep.IsActive {
		return nil, shared.ErrGone
	}

	// Build metering point list and check for duplicates within the request
	var meteringPoints []shared.MeteringPoint
	for _, mpReq := range req.MeteringPoints {
		meteringPoints = append(meteringPoints, shared.MeteringPoint{
			MeteringPoint: mpReq.MeteringPoint,
			Direction:     shared.MeterDirection(mpReq.Direction),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		})
	}
	if err = s.meteringRepo.ValidateUniqueMeteringPoints(uuid.Nil, meteringPoints); err != nil {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"meteringPoints": err.Error(),
		})
	}

	birthDate, err := parseDateString(req.BirthDate)
	if err != nil {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"birthDate": err.Error(),
		})
	}

	now := time.Now()
	privacyAcceptedAt := now
	iban := normalizeIBAN(req.IBAN)
	if !validateIBAN(iban) {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"iban": "Ungültige IBAN",
		})
	}
	var sepaMandateAcceptedAt *time.Time
	if req.SepaMandateAccepted {
		sepaMandateAcceptedAt = &now
	}

	phone := trimStringPtr(req.Phone)
	app := &shared.Application{
		ReferenceNumber:       s.generateReferenceNumber(),
		RCNumber:              strings.ToUpper(strings.TrimSpace(req.RCNumber)),
		Status:                shared.StatusDraft,
		StartedAt:             &now,
		MemberType:            shared.MemberType(strings.TrimSpace(req.MemberType)),
		Firstname:             trimStringPtr(req.Firstname),
		Lastname:              trimStringPtr(req.Lastname),
		BirthDate:             birthDate,
		CompanyName:           trimStringPtr(req.CompanyName),
		UIDNumber:             trimStringPtr(req.UIDNumber),
		RegisterNumber:        trimStringPtr(req.RegisterNumber),
		Email:                 strings.TrimSpace(req.Email),
		Phone:                 phone,
		ResidentStreet:        strings.TrimSpace(req.ResidentStreet),
		ResidentStreetNumber:  strings.TrimSpace(req.ResidentStreetNumber),
		ResidentZip:           strings.TrimSpace(req.ResidentZip),
		ResidentCity:          strings.TrimSpace(req.ResidentCity),
		PrivacyAccepted:       req.PrivacyAccepted,
		PrivacyVersion:        &req.PrivacyVersion,
		PrivacyAcceptedAt:     &privacyAcceptedAt,
		AccuracyConfirmed:     req.AccuracyConfirmed,
		IBAN:                  &iban,
		AccountHolder:         func() *string { s := strings.TrimSpace(req.AccountHolder); return &s }(),
		SepaMandateAccepted:   req.SepaMandateAccepted,
		SepaMandateAcceptedAt: sepaMandateAcceptedAt,
		CreatedAt:             now,
		UpdatedAt:             now,
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

	if err = s.appRepo.CreateTx(tx, app); err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	for i := range meteringPoints {
		meteringPoints[i].ApplicationID = app.ID
	}

	if err = s.meteringRepo.CreateBulkTx(tx, app.ID, meteringPoints); err != nil {
		return nil, fmt.Errorf("failed to create metering points: %w", err)
	}

	toStatus := string(shared.StatusDraft)
	statusLog := &shared.StatusLogEntry{
		ApplicationID: app.ID,
		FromStatus:    nil,
		ToStatus:      toStatus,
		CreatedAt:     now,
	}
	if err = s.statusLogRepo.CreateTx(tx, statusLog); err != nil {
		return nil, fmt.Errorf("failed to create status log: %w", err)
	}

	if err = tx.Commit(); err != nil {
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

// UpdateApplication updates an existing application in draft or needs_info status.
func (s *ApplicationService) UpdateApplication(id uuid.UUID, req shared.UpdateApplicationRequest) (*shared.ApplicationResponse, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if app.Status != shared.StatusDraft && app.Status != shared.StatusNeedsInfo {
		return nil, shared.ErrConflict
	}

	if req.MemberType != nil {
		app.MemberType = shared.MemberType(strings.TrimSpace(*req.MemberType))
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
		bd, bdErr := parseDateString(req.BirthDate)
		if bdErr != nil {
			return nil, shared.NewValidationError("Validation failed", map[string]string{
				"birthDate": bdErr.Error(),
			})
		}
		app.BirthDate = bd
	}
	if req.Email != nil {
		app.Email = strings.TrimSpace(*req.Email)
	}
	if req.Phone != nil {
		app.Phone = trimStringPtr(req.Phone)
	}
	if req.ResidentStreet != nil {
		app.ResidentStreet = strings.TrimSpace(*req.ResidentStreet)
	}
	if req.ResidentStreetNumber != nil {
		app.ResidentStreetNumber = strings.TrimSpace(*req.ResidentStreetNumber)
	}
	if req.ResidentZip != nil {
		app.ResidentZip = strings.TrimSpace(*req.ResidentZip)
	}
	if req.ResidentCity != nil {
		app.ResidentCity = strings.TrimSpace(*req.ResidentCity)
	}
	if req.PrivacyAccepted != nil {
		app.PrivacyAccepted = *req.PrivacyAccepted
	}
	if req.PrivacyVersion != nil {
		app.PrivacyVersion = req.PrivacyVersion
	}
	if req.AccuracyConfirmed != nil {
		app.AccuracyConfirmed = *req.AccuracyConfirmed
	}
	if req.IBAN != nil {
		normalized := normalizeIBAN(*req.IBAN)
		if !validateIBAN(normalized) {
			return nil, shared.NewValidationError("Validation failed", map[string]string{
				"iban": "Ungültige IBAN",
			})
		}
		app.IBAN = &normalized
	}
	if req.AccountHolder != nil {
		app.AccountHolder = trimStringPtr(req.AccountHolder)
	}
	if req.SepaMandateAccepted != nil {
		app.SepaMandateAccepted = *req.SepaMandateAccepted
		if *req.SepaMandateAccepted && app.SepaMandateAcceptedAt == nil {
			now := time.Now()
			app.SepaMandateAcceptedAt = &now
		}
	}

	clearMemberTypeFields(app)
	if err = validateMemberTypeFields(app); err != nil {
		return nil, err
	}

	var meteringPoints []shared.MeteringPoint
	if req.MeteringPoints != nil {
		for _, mpReq := range req.MeteringPoints {
			meteringPoints = append(meteringPoints, shared.MeteringPoint{
				ApplicationID: id,
				MeteringPoint: mpReq.MeteringPoint,
				Direction:     shared.MeterDirection(mpReq.Direction),
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			})
		}

		// Only check for duplicates within the new set — CreateBulkTx replaces all existing points
		if err = s.meteringRepo.ValidateUniqueMeteringPoints(uuid.Nil, meteringPoints); err != nil {
			return nil, shared.NewValidationError("Validation failed", map[string]string{
				"meteringPoints": err.Error(),
			})
		}
	}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if meteringPoints != nil {
		if err = s.meteringRepo.CreateBulkTx(tx, id, meteringPoints); err != nil {
			return nil, fmt.Errorf("failed to update metering points: %w", err)
		}
	}

	if err = s.appRepo.UpdateTx(tx, app); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
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

// SubmitApplication transitions an application from draft/needs_info to submitted.
func (s *ApplicationService) SubmitApplication(id uuid.UUID) (*shared.SubmitResponse, error) {
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if app.Status != shared.StatusDraft && app.Status != shared.StatusNeedsInfo {
		return nil, shared.ErrConflict
	}

	if !app.PrivacyAccepted || app.PrivacyVersion == nil || !app.AccuracyConfirmed {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"general": "Privacy consent and accuracy confirmation required for submission",
		})
	}
	if !app.SepaMandateAccepted {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"sepaMandateAccepted": "SEPA-Lastschriftmandat muss akzeptiert werden",
		})
	}
	if app.IBAN == nil || *app.IBAN == "" {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"iban": "IBAN ist erforderlich",
		})
	}
	if app.AccountHolder == nil || *app.AccountHolder == "" {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"accountHolder": "Kontoinhaber ist erforderlich",
		})
	}
	if err = validateMemberTypeFields(app); err != nil {
		return nil, err
	}

	meteringPoints, err := s.meteringRepo.GetByApplicationID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get metering points: %w", err)
	}
	if len(meteringPoints) == 0 {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"meteringPoints": "At least one metering point is required",
		})
	}

	now := time.Now()
	oldStatus := string(app.Status)

	if err = s.appRepo.UpdateStatus(id, shared.StatusSubmitted, &now); err != nil {
		return nil, err
	}

	statusLog := &shared.StatusLogEntry{
		ApplicationID: id,
		FromStatus:    &oldStatus,
		ToStatus:      string(shared.StatusSubmitted),
		CreatedAt:     now,
	}
	if err = s.statusLogRepo.Create(statusLog); err != nil {
		fmt.Printf("Failed to create status log: %v\n", err)
	}

	// Send submission emails only on first submission (draft → submitted).
	if oldStatus == string(shared.StatusDraft) {
		entrypoint, epErr := s.entrypointRepo.GetByRCNumber(app.RCNumber)
		if epErr != nil {
			fmt.Printf("mail: failed to load entrypoint for rc=%s: %v\n", app.RCNumber, epErr)
		} else {
			go s.mailService.SendSubmissionEmails(app, meteringPoints, entrypoint)
		}
	}

	return &shared.SubmitResponse{
		ID:              id,
		ReferenceNumber: app.ReferenceNumber,
		Status:          shared.StatusSubmitted,
		SubmittedAt:     now,
	}, nil
}

// parseDateString parses an optional "YYYY-MM-DD" string into *time.Time.
func parseDateString(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD")
	}
	return &t, nil
}

// generateReferenceNumber generates a unique reference number
func (s *ApplicationService) generateReferenceNumber() string {
	now := time.Now()
	return fmt.Sprintf("MO-%s-%06d", now.Format("2006"), now.Unix()%1000000)
}

// trimStringPtr trims whitespace from a *string, returning nil if the pointer is nil.
func trimStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	return &trimmed
}

// normalizeIBAN strips whitespace and uppercases an IBAN string.
func normalizeIBAN(iban string) string {
	return strings.ToUpper(strings.ReplaceAll(iban, " ", ""))
}

// validateIBAN checks IBAN structure and MOD-97 checksum.
func validateIBAN(iban string) bool {
	if len(iban) < 15 || len(iban) > 34 {
		return false
	}
	// Move first 4 chars to end, convert letters to digits (A=10 … Z=35)
	rearranged := iban[4:] + iban[:4]
	var numeric strings.Builder
	for _, c := range rearranged {
		if c >= 'A' && c <= 'Z' {
			numeric.WriteString(fmt.Sprintf("%d", int(c-'A'+10)))
		} else if c >= '0' && c <= '9' {
			numeric.WriteByte(byte(c))
		} else {
			return false
		}
	}
	// MOD-97 on large number processed in chunks
	digits := numeric.String()
	remainder := 0
	for _, ch := range digits {
		remainder = (remainder*10 + int(ch-'0')) % 97
	}
	return remainder == 1
}

// clearMemberTypeFields nils out fields not applicable to the current member type.
func clearMemberTypeFields(app *shared.Application) {
	switch app.MemberType {
	case shared.MemberTypePrivate, shared.MemberTypeFarmer:
		app.CompanyName = nil
		app.UIDNumber = nil
		app.RegisterNumber = nil
	case shared.MemberTypeMunicipality, shared.MemberTypeCompany, shared.MemberTypeAssociation:
		app.Firstname = nil
		app.Lastname = nil
		app.BirthDate = nil
	}
}

// validateMemberTypeFields checks that all required fields for the member type are present.
func validateMemberTypeFields(app *shared.Application) error {
	switch app.MemberType {
	case shared.MemberTypePrivate, shared.MemberTypeFarmer:
		if app.Firstname == nil || strings.TrimSpace(*app.Firstname) == "" {
			return shared.NewValidationError("Validation failed", map[string]string{
				"firstname": "Vorname ist erforderlich",
			})
		}
		if app.Lastname == nil || strings.TrimSpace(*app.Lastname) == "" {
			return shared.NewValidationError("Validation failed", map[string]string{
				"lastname": "Nachname ist erforderlich",
			})
		}
		if app.BirthDate == nil {
			return shared.NewValidationError("Validation failed", map[string]string{
				"birthDate": "Geburtsdatum ist erforderlich",
			})
		}
	case shared.MemberTypeMunicipality:
		if app.CompanyName == nil || strings.TrimSpace(*app.CompanyName) == "" {
			return shared.NewValidationError("Validation failed", map[string]string{
				"companyName": "Organisationsname ist erforderlich",
			})
		}
	case shared.MemberTypeAssociation:
		if app.CompanyName == nil || strings.TrimSpace(*app.CompanyName) == "" {
			return shared.NewValidationError("Validation failed", map[string]string{
				"companyName": "Vereinsname ist erforderlich",
			})
		}
		if app.RegisterNumber == nil || strings.TrimSpace(*app.RegisterNumber) == "" {
			return shared.NewValidationError("Validation failed", map[string]string{
				"registerNumber": "Vereinsnummer ist erforderlich",
			})
		}
	case shared.MemberTypeCompany:
		if app.CompanyName == nil || strings.TrimSpace(*app.CompanyName) == "" {
			return shared.NewValidationError("Validation failed", map[string]string{
				"companyName": "Firmenname ist erforderlich",
			})
		}
		if app.UIDNumber == nil || strings.TrimSpace(*app.UIDNumber) == "" {
			return shared.NewValidationError("Validation failed", map[string]string{
				"uidNumber": "UID-Nummer ist erforderlich",
			})
		}
		if app.RegisterNumber == nil || strings.TrimSpace(*app.RegisterNumber) == "" {
			return shared.NewValidationError("Validation failed", map[string]string{
				"registerNumber": "Firmenbuch-/Vereinsnummer ist erforderlich",
			})
		}
	default:
		return shared.NewValidationError("Validation failed", map[string]string{
			"memberType": "Ungültiger Mitgliedstyp",
		})
	}
	return nil
}
