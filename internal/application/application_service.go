package application

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/mail"
	"github.com/your-org/eegfaktura-member-onboarding/internal/pdf"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ApplicationService handles business logic for applications
type ApplicationService struct {
	db                  *sql.DB
	appRepo             *ApplicationRepository
	meteringRepo        *MeteringPointRepository
	statusLogRepo       *StatusLogRepository
	entrypointRepo      *RegistrationEntrypointRepository
	fieldConfigRepo     *FieldConfigRepository
	consentRepo         *DocumentConsentRepository
	mailService         mail.MailService
	pdfGenerator        pdf.SEPAMandateGenerator
}

// NewApplicationService creates a new application service
func NewApplicationService(
	db *sql.DB,
	appRepo *ApplicationRepository,
	meteringRepo *MeteringPointRepository,
	statusLogRepo *StatusLogRepository,
	entrypointRepo *RegistrationEntrypointRepository,
	fieldConfigRepo *FieldConfigRepository,
	consentRepo *DocumentConsentRepository,
	mailService mail.MailService,
	pdfGenerator pdf.SEPAMandateGenerator,
) *ApplicationService {
	return &ApplicationService{
		db:              db,
		appRepo:         appRepo,
		meteringRepo:    meteringRepo,
		statusLogRepo:   statusLogRepo,
		entrypointRepo:  entrypointRepo,
		fieldConfigRepo: fieldConfigRepo,
		consentRepo:     consentRepo,
		mailService:     mailService,
		pdfGenerator:    pdfGenerator,
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

	// Explicit consent check — must not store personal data without agreement.
	if !req.PrivacyAccepted || !req.AccuracyConfirmed {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"privacyAccepted": "Datenschutzerklärung und Richtigkeit müssen bestätigt werden",
		})
	}

	// Load field config (best-effort — fail open so a DB error doesn't block registrations)
	fieldConfig, fcErr := s.fieldConfigRepo.Get(strings.ToUpper(req.RCNumber))
	if fcErr != nil {
		slog.Warn("failed to load field config", "rc", req.RCNumber, "error", fcErr)
		fieldConfig = map[string]FieldConfigEntry{}
	}

	// Build metering point list and check for duplicates within the request
	var meteringPoints []shared.MeteringPoint
	for _, mpReq := range req.MeteringPoints {
		meteringPoints = append(meteringPoints, shared.MeteringPoint{
			MeteringPoint:       mpReq.MeteringPoint,
			Direction:           shared.MeterDirection(mpReq.Direction),
			ParticipationFactor: mpReq.ParticipationFactor,
			Transformer:         trimStringPtr(mpReq.Transformer),
			InstallationNumber:  trimStringPtr(mpReq.InstallationNumber),
			InstallationName:    trimStringPtr(mpReq.InstallationName),
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
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
	membershipStartDate, err := parseDateString(req.MembershipStartDate)
	if err != nil {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"membershipStartDate": err.Error(),
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
		ReferenceNumber:         s.generateReferenceNumber(),
		RCNumber:                strings.ToUpper(strings.TrimSpace(req.RCNumber)),
		Status:                  shared.StatusDraft,
		StartedAt:               &now,
		MemberType:              shared.MemberType(strings.TrimSpace(req.MemberType)),
		Titel:                   trimStringPtr(req.Titel),
		Firstname:               trimStringPtr(req.Firstname),
		Lastname:                trimStringPtr(req.Lastname),
		BirthDate:               birthDate,
		CompanyName:             trimStringPtr(req.CompanyName),
		UIDNumber:               trimStringPtr(req.UIDNumber),
		RegisterNumber:          trimStringPtr(req.RegisterNumber),
		Email:                   strings.TrimSpace(req.Email),
		Phone:                   phone,
		ResidentStreet:          strings.TrimSpace(req.ResidentStreet),
		ResidentStreetNumber:    strings.TrimSpace(req.ResidentStreetNumber),
		ResidentZip:             strings.TrimSpace(req.ResidentZip),
		ResidentCity:            strings.TrimSpace(req.ResidentCity),
		PrivacyAccepted:         req.PrivacyAccepted,
		PrivacyVersion:          &req.PrivacyVersion,
		PrivacyAcceptedAt:       &privacyAcceptedAt,
		AccuracyConfirmed:       req.AccuracyConfirmed,
		IBAN:                    &iban,
		AccountHolder:           func() *string { s := strings.TrimSpace(req.AccountHolder); return &s }(),
		SepaMandateAccepted:     req.SepaMandateAccepted,
		SepaMandateAcceptedAt:   sepaMandateAcceptedAt,
		Einzugsart:              "core",
		CreatedAt:               now,
		UpdatedAt:               now,
		MembershipStartDate:     membershipStartDate,
		PersonsInHousehold:      req.PersonsInHousehold,
		ConsumptionPreviousYear: req.ConsumptionPreviousYear,
		ConsumptionForecast:     req.ConsumptionForecast,
		FeedInForecast:          req.FeedInForecast,
		PvPowerKwp:              req.PvPowerKwp,
		HeatPump:                req.HeatPump,
		ElectricVehicle:         req.ElectricVehicle,
		ElectricHotWater:        req.ElectricHotWater,
	}
	applyAdminValues(app, fieldConfig)
	clearMemberTypeFields(app)
	if err = validateMemberTypeFields(app); err != nil {
		return nil, err
	}
	if err = validateConfigurableRequiredFields(app, fieldConfig); err != nil {
		return nil, err
	}
	if err = validateConfigurableMeteringPointFields(meteringPoints, fieldConfig); err != nil {
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
				ApplicationID:       id,
				MeteringPoint:       mpReq.MeteringPoint,
				Direction:           shared.MeterDirection(mpReq.Direction),
				ParticipationFactor: mpReq.ParticipationFactor,
				CreatedAt:           time.Now(),
				UpdatedAt:           time.Now(),
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

// SubmitApplication transitions an application from draft/needs_info to submitted
// and persists the provided consent snapshots.
func (s *ApplicationService) SubmitApplication(id uuid.UUID, consents []shared.ConsentInput) (*shared.SubmitResponse, error) {
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

	fieldConfig, fcErr := s.fieldConfigRepo.Get(strings.ToUpper(app.RCNumber))
	if fcErr != nil {
		slog.Warn("failed to load field config", "rc", app.RCNumber, "error", fcErr)
		fieldConfig = map[string]FieldConfigEntry{}
	}
	if err = validateConfigurableRequiredFields(app, fieldConfig); err != nil {
		return nil, err
	}
	if err = validateConfigurableMeteringPointFields(meteringPoints, fieldConfig); err != nil {
		return nil, err
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
		slog.Error("failed to create status log", "application_id", id, "error", err)
	}

	// Persist consent snapshots when provided.
	if len(consents) > 0 && s.consentRepo != nil {
		consentRows := make([]shared.DocumentConsent, 0, len(consents))
		for _, c := range consents {
			consentRows = append(consentRows, shared.DocumentConsent{
				ID:              uuid.New(),
				ApplicationID:   id,
				Title:           c.Title,
				URL:             c.URL,
				IsCentralPolicy: c.IsCentralPolicy,
				ConsentedAt:     now,
			})
		}
		tx, txErr := s.db.Begin()
		if txErr == nil {
			if txErr = s.consentRepo.CreateBulkTx(tx, consentRows); txErr != nil {
				tx.Rollback()
				slog.Error("failed to save consents", "application_id", id, "error", txErr)
			} else {
				tx.Commit()
			}
		} else {
			slog.Error("failed to begin consent transaction", "application_id", id, "error", txErr)
		}
	}

	// Assign member number on first submission (draft → submitted).
	if oldStatus == string(shared.StatusDraft) {
		tx, txErr := s.db.Begin()
		if txErr != nil {
			slog.Error("member number: failed to begin tx", "application_id", id, "error", txErr)
		} else if assignErr := s.appRepo.AssignMemberNumberTx(tx, id, app.RCNumber); assignErr != nil {
			tx.Rollback()
			slog.Error("member number: failed to assign", "application_id", id, "error", assignErr)
		} else if commitErr := tx.Commit(); commitErr != nil {
			slog.Error("member number: failed to commit", "application_id", id, "error", commitErr)
		}
	}

	// Send submission emails only on first submission (draft → submitted).
	if oldStatus == string(shared.StatusDraft) {
		entrypoint, epErr := s.entrypointRepo.GetByRCNumber(app.RCNumber)
		if epErr != nil {
			slog.Warn("mail: failed to load entrypoint", "rc", app.RCNumber, "error", epErr)
		} else {
			var attachment []byte
			if mandate := buildSEPAMandateData(app, entrypoint); mandate != nil {
				useCompany := entrypoint.UseCompanySEPAMandate &&
					(app.MemberType == shared.MemberTypeCompany || app.MemberType == shared.MemberTypeAssociation)
				// For B2B mandates, the debtor name must be the company name, not the contact person.
				if useCompany && app.CompanyName != nil && *app.CompanyName != "" {
					mandate.MemberName = *app.CompanyName
				}
				var pdfBytes []byte
				var pdfErr error
				if useCompany {
					pdfBytes, pdfErr = s.pdfGenerator.GenerateCompany(*mandate)
				} else {
					pdfBytes, pdfErr = s.pdfGenerator.Generate(*mandate)
				}
				if pdfErr != nil {
					slog.Warn("pdf: failed to generate SEPA mandate", "rc", app.RCNumber, "error", pdfErr)
				} else {
					attachment = pdfBytes
				}
			}
			var savedConsents []shared.DocumentConsent
			if s.consentRepo != nil {
				if sc, err := s.consentRepo.GetByApplicationID(id); err == nil {
					savedConsents = sc
				}
			}
			go func() {
				acquireMailSem()
				defer releaseMailSem()
				s.mailService.SendSubmissionEmails(app, meteringPoints, entrypoint, toStateMap(fieldConfig), attachment, savedConsents)
			}()
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

// generateReferenceNumber returns a collision-free reference number backed by a
// DB sequence. Format: MO-YYYY-NNNNNN (sequence is global, never resets per year).
// Falls back to a crypto/rand value only if the DB query fails unexpectedly.
func (s *ApplicationService) generateReferenceNumber() string {
	var seq int64
	if err := s.db.QueryRow("SELECT nextval('member_onboarding.application_reference_number_seq')").Scan(&seq); err != nil {
		slog.Error("generateReferenceNumber: sequence query failed, using fallback", "error", err)
		n, _ := rand.Int(rand.Reader, big.NewInt(900_000))
		return fmt.Sprintf("MO-%d-%06d", time.Now().Year(), n.Int64()+100_000)
	}
	return fmt.Sprintf("MO-%d-%06d", time.Now().Year(), seq)
}

// trimStringPtr trims whitespace from a *string, returning nil if the pointer is nil.
func trimStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	return &trimmed
}

// normalizeIBAN strips whitespace, uppercases, and reformats an IBAN into groups of 4.
func normalizeIBAN(iban string) string {
	compact := strings.ToUpper(strings.ReplaceAll(iban, " ", ""))
	var buf strings.Builder
	for i, ch := range compact {
		if i > 0 && i%4 == 0 {
			buf.WriteByte(' ')
		}
		buf.WriteRune(ch)
	}
	return buf.String()
}

// validateIBAN checks IBAN structure and MOD-97 checksum.
// Accepts both compact and space-formatted IBANs.
func validateIBAN(iban string) bool {
	iban = strings.ReplaceAll(iban, " ", "")
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

// applyAdminValues sets application fields from admin-configured default values for admin_only fields.
// Only fields that the caller left as nil (not provided) are overwritten.
func applyAdminValues(app *shared.Application, fieldConfig map[string]FieldConfigEntry) {
	apply := func(name string, setter func(string)) {
		entry, ok := fieldConfig[name]
		if !ok || entry.State != "admin_only" || entry.AdminValue == nil || *entry.AdminValue == "" {
			return
		}
		setter(*entry.AdminValue)
	}
	apply("membership_start_date", func(v string) {
		if app.MembershipStartDate == nil {
			if t, err := parseDateString(&v); err == nil {
				app.MembershipStartDate = t
			}
		}
	})
	apply("persons_in_household", func(v string) {
		if app.PersonsInHousehold == nil {
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
				app.PersonsInHousehold = &n
			}
		}
	})
	apply("consumption_previous_year", func(v string) {
		if app.ConsumptionPreviousYear == nil {
			var n int64
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
				app.ConsumptionPreviousYear = &n
			}
		}
	})
	apply("consumption_forecast", func(v string) {
		if app.ConsumptionForecast == nil {
			var n int64
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
				app.ConsumptionForecast = &n
			}
		}
	})
	apply("feed_in_forecast", func(v string) {
		if app.FeedInForecast == nil {
			var n int64
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
				app.FeedInForecast = &n
			}
		}
	})
	apply("pv_power_kwp", func(v string) {
		if app.PvPowerKwp == nil {
			var f float64
			if _, err := fmt.Sscanf(v, "%f", &f); err == nil {
				app.PvPowerKwp = &f
			}
		}
	})
	apply("heat_pump", func(v string) {
		if app.HeatPump == nil {
			b := v == "true"
			app.HeatPump = &b
		}
	})
	apply("electric_vehicle", func(v string) {
		if app.ElectricVehicle == nil {
			b := v == "true"
			app.ElectricVehicle = &b
		}
	})
	apply("electric_hot_water", func(v string) {
		if app.ElectricHotWater == nil {
			b := v == "true"
			app.ElectricHotWater = &b
		}
	})
}

// validateConfigurableRequiredFields checks application-level fields configured as "required".
func validateConfigurableRequiredFields(app *shared.Application, fieldConfig map[string]FieldConfigEntry) error {
	errs := map[string]string{}

	checkStr := func(name, jsonKey string, val *string, label string) {
		if effectiveState(fieldConfig, name) == "required" {
			if val == nil || strings.TrimSpace(*val) == "" {
				errs[jsonKey] = label + " ist erforderlich"
			}
		}
	}
	checkTime := func(name, jsonKey string, val *time.Time, label string) {
		if effectiveState(fieldConfig, name) == "required" && val == nil {
			errs[jsonKey] = label + " ist erforderlich"
		}
	}
	checkInt := func(name, jsonKey string, val *int, label string) {
		if effectiveState(fieldConfig, name) == "required" && val == nil {
			errs[jsonKey] = label + " ist erforderlich"
		}
	}
	checkInt64 := func(name, jsonKey string, val *int64, label string) {
		if effectiveState(fieldConfig, name) == "required" && val == nil {
			errs[jsonKey] = label + " ist erforderlich"
		}
	}
	checkFloat := func(name, jsonKey string, val *float64, label string) {
		if effectiveState(fieldConfig, name) == "required" && val == nil {
			errs[jsonKey] = label + " ist erforderlich"
		}
	}
	checkBool := func(name, jsonKey string, val *bool, label string) {
		if effectiveState(fieldConfig, name) == "required" && val == nil {
			errs[jsonKey] = label + " ist erforderlich"
		}
	}

	checkStr("phone", "phone", app.Phone, "Telefonnummer")
	checkTime("birth_date", "birthDate", app.BirthDate, "Geburtsdatum")
	checkStr("uid_number", "uidNumber", app.UIDNumber, "UID-Nummer")
	checkTime("membership_start_date", "membershipStartDate", app.MembershipStartDate, "Beitrittsdatum")
	checkInt("persons_in_household", "personsInHousehold", app.PersonsInHousehold, "Anzahl Personen im Haushalt")
	checkInt64("consumption_previous_year", "consumptionPreviousYear", app.ConsumptionPreviousYear, "Verbrauch Vorjahr")
	checkInt64("consumption_forecast", "consumptionForecast", app.ConsumptionForecast, "Verbrauch Prognose")
	checkInt64("feed_in_forecast", "feedInForecast", app.FeedInForecast, "Einspeisung Prognose")
	checkFloat("pv_power_kwp", "pvPowerKwp", app.PvPowerKwp, "PV-Leistung")
	checkBool("heat_pump", "heatPump", app.HeatPump, "Wärmepumpe vorhanden")
	checkBool("electric_vehicle", "electricVehicle", app.ElectricVehicle, "E-Auto vorhanden")
	checkBool("electric_hot_water", "electricHotWater", app.ElectricHotWater, "Warmwasser elektrisch")

	if len(errs) > 0 {
		return shared.NewValidationError("Validation failed", errs)
	}
	return nil
}

// validateConfigurableMeteringPointFields checks metering-point-level fields configured as "required".
func validateConfigurableMeteringPointFields(points []shared.MeteringPoint, fieldConfig map[string]FieldConfigEntry) error {
	for i, mp := range points {
		errs := map[string]string{}
		checkStr := func(name, label string, val *string) {
			if effectiveState(fieldConfig, name) == "required" {
				if val == nil || strings.TrimSpace(*val) == "" {
					errs[fmt.Sprintf("meteringPoints.%d.%s", i, name)] = label + " ist erforderlich"
				}
			}
		}
		checkStr("transformer", "Transformator", mp.Transformer)
		checkStr("installation_number", "Anlagen-Nr.", mp.InstallationNumber)
		checkStr("installation_name", "Anlagenname", mp.InstallationName)
		if len(errs) > 0 {
			return shared.NewValidationError("Validation failed", errs)
		}
	}
	return nil
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

// validateMemberTypeFields checks that structurally required fields for the member type are present.
// Birth date is no longer enforced here — it is a configurable field (PROJ-8).
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

// buildSEPAMandateData returns a SEPAMandateData struct when all required EEG fields
// are set and SEPA mandate sending is enabled. Returns nil otherwise.
func buildSEPAMandateData(app *shared.Application, ep *shared.RegistrationEntrypoint) *pdf.SEPAMandateData {
	if !ep.SEPAMandateEnabled ||
		ep.EEGName == nil || ep.EEGStreet == nil || ep.EEGStreetNumber == nil ||
		ep.EEGZip == nil || ep.EEGCity == nil || ep.CreditorID == nil {
		return nil
	}
	// The SEPA mandate must show the account holder — the person the bank account is
	// registered under — not the member's name. For companies, the company name is used.
	name := strings.TrimSpace(derefStr(app.AccountHolder))
	if name == "" && app.CompanyName != nil {
		name = *app.CompanyName
	}
	return &pdf.SEPAMandateData{
		EEGName:            *ep.EEGName,
		EEGStreet:          *ep.EEGStreet,
		EEGStreetNumber:    *ep.EEGStreetNumber,
		EEGZip:             *ep.EEGZip,
		EEGCity:            *ep.EEGCity,
		CreditorID:         *ep.CreditorID,
		MemberName:         name,
		MemberStreet:       app.ResidentStreet,
		MemberStreetNumber: app.ResidentStreetNumber,
		MemberZip:          app.ResidentZip,
		MemberCity:         app.ResidentCity,
		IBAN:               derefStr(app.IBAN),
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// toStateMap extracts only the state string from a FieldConfigEntry map.
// Used to pass a minimal representation to the mail service.
func toStateMap(fieldConfig map[string]FieldConfigEntry) map[string]string {
	m := make(map[string]string, len(fieldConfig))
	for k, v := range fieldConfig {
		m[k] = v.State
	}
	return m
}
