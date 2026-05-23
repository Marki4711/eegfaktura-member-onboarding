package application

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"log/slog"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/mail"
	"github.com/your-org/eegfaktura-member-onboarding/internal/metrics"
	"github.com/your-org/eegfaktura-member-onboarding/internal/pdf"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ApplicationService handles business logic for applications
type ApplicationService struct {
	db                *sql.DB
	appRepo           *ApplicationRepository
	meteringRepo      *MeteringPointRepository
	statusLogRepo     *StatusLogRepository
	entrypointRepo    *RegistrationEntrypointRepository
	fieldConfigRepo   *FieldConfigRepository
	consentRepo       *DocumentConsentRepository
	legalDocumentRepo *LegalDocumentRepository
	mailService       mail.MailService
	pdfGenerator      pdf.SEPAMandateGenerator
	publicBaseURL     string
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
	legalDocumentRepo *LegalDocumentRepository,
	mailService mail.MailService,
	pdfGenerator pdf.SEPAMandateGenerator,
	publicBaseURL string,
) *ApplicationService {
	return &ApplicationService{
		db:                db,
		appRepo:           appRepo,
		meteringRepo:      meteringRepo,
		statusLogRepo:     statusLogRepo,
		entrypointRepo:    entrypointRepo,
		fieldConfigRepo:   fieldConfigRepo,
		consentRepo:       consentRepo,
		legalDocumentRepo: legalDocumentRepo,
		mailService:       mailService,
		pdfGenerator:      pdfGenerator,
		publicBaseURL:     publicBaseURL,
	}
}

// emailConfirmationTokenLifetime is how long a freshly-issued e-mail
// confirmation token stays valid. After this period the auto-reject job
// (Stage E) transitions the application to `rejected` if the member never
// clicked. 30 days is the spec-recommended default (PROJ-31).
const emailConfirmationTokenLifetime = 30 * 24 * time.Hour

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
		normalized := strings.ToUpper(strings.ReplaceAll(mpReq.MeteringPoint, " ", ""))
		if !validateMeteringPointFormat(normalized) {
			return nil, shared.NewValidationError("Validation failed", map[string]string{
				"meteringPoints": fmt.Sprintf("Zählpunkt %q muss mit AT beginnen und 31 Ziffern enthalten (33 Zeichen gesamt)", mpReq.MeteringPoint),
			})
		}
		meteringPoints = append(meteringPoints, shared.MeteringPoint{
			MeteringPoint:       normalized,
			Direction:           shared.MeterDirection(mpReq.Direction),
			ParticipationFactor: defaultParticipationFactor(mpReq.ParticipationFactor),
			Transformer:          trimStringPtr(mpReq.Transformer),
			InstallationNumber:   trimStringPtr(mpReq.InstallationNumber),
			InstallationName:     trimStringPtr(mpReq.InstallationName),
			AddressStreet:        trimStringPtr(mpReq.AddressStreet),
			AddressStreetNumber:  trimStringPtr(mpReq.AddressStreetNumber),
			AddressZip:           trimStringPtr(mpReq.AddressZip),
			AddressCity:          trimStringPtr(mpReq.AddressCity),
			GenerationType:       trimStringPtr(mpReq.GenerationType),
			BatterySizeKwh:       mpReq.BatterySizeKwh,
			InverterManufacturer: trimStringPtr(mpReq.InverterManufacturer),
			InverterPowerKw:      mpReq.InverterPowerKw,
			// PROJ-49: Energie-Felder pro Zählpunkt.
			ConsumptionPreviousYear: mpReq.ConsumptionPreviousYear,
			ConsumptionForecast:     mpReq.ConsumptionForecast,
			FeedInForecast:          mpReq.FeedInForecast,
			PvPowerKwp:              mpReq.PvPowerKwp,
			FeedInLimitPresent:      mpReq.FeedInLimitPresent,
			FeedInLimitKw:           mpReq.FeedInLimitKw,
			// PROJ-49 follow-up: Speichersteuerung-Frage.
			BatteryControlAcceptable: mpReq.BatteryControlAcceptable,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		})
	}
	if err = validateMeteringPointAddresses(meteringPoints); err != nil {
		return nil, err
	}
	normalizeMeteringPointGeneration(meteringPoints)
	clearMeteringPointEnergyByType(meteringPoints)
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
		ReferenceNumber:         s.generateReferenceNumber(req.RCNumber),
		RCNumber:                strings.ToUpper(strings.TrimSpace(req.RCNumber)),
		Status:                  shared.StatusDraft,
		StartedAt:               &now,
		MemberType:              shared.MemberType(strings.TrimSpace(req.MemberType)),
		Titel:                   trimStringPtr(req.Titel),
		TitelNach:               trimStringPtr(req.TitelNach),
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
		BankName:                trimStringPtr(req.BankName),
		SepaMandateAccepted:     req.SepaMandateAccepted,
		SepaMandateAcceptedAt:   sepaMandateAcceptedAt,
		Einzugsart:              "core",
		CreatedAt:               now,
		UpdatedAt:               now,
		MembershipStartDate:     membershipStartDate,
		PersonsInHousehold:      req.PersonsInHousehold,
		HeatPump:                req.HeatPump,
		ElectricVehicle:         req.ElectricVehicle,
		ElectricVehicleCount:    req.ElectricVehicleCount,
		ElectricVehicleAnnualKm: req.ElectricVehicleAnnualKm,
		ElectricHotWater:        req.ElectricHotWater,
		CooperativeSharesCount:  req.CooperativeSharesCount,
	}
	if req.NetworkOperatorAuthorization != nil && *req.NetworkOperatorAuthorization {
		app.NetworkOperatorAuthorization = true
		app.NetworkOperatorAuthorizationAt = &now
	}
	// PROJ-56: Netzbetreiber-Info-Felder. Werden nur übernommen, wenn das
	// Mitglied die Vollmacht aktiv erteilt hat — sonst stillschweigend
	// ignoriert, auch wenn ein forged client Werte senden würde.
	app.NetworkOperatorCustomerNumber = req.NetworkOperatorCustomerNumber
	app.MeterInventoryNumber = req.MeterInventoryNumber
	// PROJ-57: Ansprechperson. Toggle + drei Felder. Service-Layer cleart
	// die drei Felder auf NULL, wenn der Toggle false ist oder der
	// Mitgliedstyp nicht in der Org-Liste liegt (clearContactPersonIfDisabled).
	if req.HasContactPerson != nil {
		app.HasContactPerson = *req.HasContactPerson
	}
	app.ContactPersonName = req.ContactPersonName
	app.ContactPersonEmail = req.ContactPersonEmail
	app.ContactPersonPhone = req.ContactPersonPhone
	// PROJ-58: Abweichende Rechnungs-E-Mail. Toggle + Email. Service-
	// Layer cleart, wenn Toggle aus oder Mitgliedstyp nicht in Org-Liste.
	if req.HasBillingEmail != nil {
		app.HasBillingEmail = *req.HasBillingEmail
	}
	app.BillingEmail = req.BillingEmail
	applyAdminValues(app, fieldConfig)
	clearMemberTypeFields(app)
	clearEVDetailsIfDisabled(app)
	clearNetworkAuthIfHidden(app, fieldConfig)
	clearContactPersonIfDisabled(app, fieldConfig)
	clearBillingEmailIfDisabled(app, fieldConfig)
	if err = validateMemberTypeFields(app); err != nil {
		return nil, err
	}
	clearAppFieldsByMpTypes(app, meteringPoints)
	if err = validateConfigurableRequiredFields(app, fieldConfig, meteringPoints); err != nil {
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
	if req.BankName != nil {
		app.BankName = trimStringPtr(req.BankName)
	}
	if req.SepaMandateAccepted != nil {
		app.SepaMandateAccepted = *req.SepaMandateAccepted
		if *req.SepaMandateAccepted && app.SepaMandateAcceptedAt == nil {
			now := time.Now()
			app.SepaMandateAcceptedAt = &now
		}
	}
	if req.NetworkOperatorAuthorization != nil {
		// PROJ-44: only allow setting true; the timestamp records first grant.
		// We don't expose a revoke path through this endpoint (out of V1 scope).
		if *req.NetworkOperatorAuthorization && !app.NetworkOperatorAuthorization {
			now := time.Now()
			app.NetworkOperatorAuthorization = true
			app.NetworkOperatorAuthorizationAt = &now
		}
	}
	// PROJ-56: Netzbetreiber-Info-Felder werden über das Update-Path durchgereicht.
	// Sentinel-Logik wie bei den anderen Pointer-Feldern: wenn der Client das
	// Feld weglässt, bleibt der bestehende Wert; explicit "" überschreibt mit
	// NULL via trimStringPtr (analog zu BankName etc.).
	if req.NetworkOperatorCustomerNumber != nil {
		app.NetworkOperatorCustomerNumber = trimStringPtr(req.NetworkOperatorCustomerNumber)
	}
	if req.MeterInventoryNumber != nil {
		app.MeterInventoryNumber = trimStringPtr(req.MeterInventoryNumber)
	}
	// PROJ-57: Ansprechperson-Felder im Update-Path. Sentinel-Logik wie oben.
	if req.HasContactPerson != nil {
		app.HasContactPerson = *req.HasContactPerson
	}
	if req.ContactPersonName != nil {
		app.ContactPersonName = trimStringPtr(req.ContactPersonName)
	}
	if req.ContactPersonEmail != nil {
		app.ContactPersonEmail = trimStringPtr(req.ContactPersonEmail)
	}
	if req.ContactPersonPhone != nil {
		app.ContactPersonPhone = trimStringPtr(req.ContactPersonPhone)
	}
	// PROJ-58: Rechnungs-E-Mail-Felder im Update-Path.
	if req.HasBillingEmail != nil {
		app.HasBillingEmail = *req.HasBillingEmail
	}
	if req.BillingEmail != nil {
		app.BillingEmail = trimStringPtr(req.BillingEmail)
	}

	fieldConfig, fcErr := s.fieldConfigRepo.Get(strings.ToUpper(app.RCNumber))
	if fcErr != nil {
		slog.Warn("failed to load field config for update", "rc", app.RCNumber, "error", fcErr)
		fieldConfig = map[string]FieldConfigEntry{}
	}
	clearMemberTypeFields(app)
	clearEVDetailsIfDisabled(app)
	clearNetworkAuthIfHidden(app, fieldConfig)
	clearContactPersonIfDisabled(app, fieldConfig)
	clearBillingEmailIfDisabled(app, fieldConfig)
	if err = validateMemberTypeFields(app); err != nil {
		return nil, err
	}

	var meteringPoints []shared.MeteringPoint
	if req.MeteringPoints != nil {
		for _, mpReq := range req.MeteringPoints {
			normalized := strings.ToUpper(strings.ReplaceAll(mpReq.MeteringPoint, " ", ""))
			if !validateMeteringPointFormat(normalized) {
				return nil, shared.NewValidationError("Validation failed", map[string]string{
					"meteringPoints": fmt.Sprintf("Zählpunkt %q muss mit AT beginnen und 31 Ziffern enthalten (33 Zeichen gesamt)", mpReq.MeteringPoint),
				})
			}
			meteringPoints = append(meteringPoints, shared.MeteringPoint{
				ApplicationID:        id,
				MeteringPoint:        normalized,
				Direction:            shared.MeterDirection(mpReq.Direction),
				ParticipationFactor:  defaultParticipationFactor(mpReq.ParticipationFactor),
				Transformer:          trimStringPtr(mpReq.Transformer),
				InstallationNumber:   trimStringPtr(mpReq.InstallationNumber),
				InstallationName:     trimStringPtr(mpReq.InstallationName),
				AddressStreet:        trimStringPtr(mpReq.AddressStreet),
				AddressStreetNumber:  trimStringPtr(mpReq.AddressStreetNumber),
				AddressZip:           trimStringPtr(mpReq.AddressZip),
				AddressCity:          trimStringPtr(mpReq.AddressCity),
				GenerationType:       trimStringPtr(mpReq.GenerationType),
				BatterySizeKwh:       mpReq.BatterySizeKwh,
				InverterManufacturer: trimStringPtr(mpReq.InverterManufacturer),
				InverterPowerKw:      mpReq.InverterPowerKw,
				CreatedAt:            time.Now(),
				UpdatedAt:            time.Now(),
			})
		}
		if err = validateMeteringPointAddresses(meteringPoints); err != nil {
			return nil, err
		}
		normalizeMeteringPointGeneration(meteringPoints)
	clearMeteringPointEnergyByType(meteringPoints)

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
	clearAppFieldsByMpTypes(app, meteringPoints)
	if err = validateConfigurableRequiredFields(app, fieldConfig, meteringPoints); err != nil {
		return nil, err
	}
	if err = validateConfigurableMeteringPointFields(meteringPoints, fieldConfig); err != nil {
		return nil, err
	}

	// Load the entrypoint up-front: we need to know whether the EEG requires
	// e-mail confirmation BEFORE the transaction starts, so we can generate the
	// token in the same transaction as the status transition.
	entrypoint, epErr := s.entrypointRepo.GetByRCNumber(app.RCNumber)
	if epErr != nil {
		return nil, fmt.Errorf("failed to load entrypoint for submit: %w", epErr)
	}

	// PROJ-52: defense-in-depth prefix-match. If the EEG has configured a
	// metering-point-prefix for a direction, every metering point of that
	// direction must start with the configured prefix. NULL prefix = no
	// check for that direction (Fallback 2a: andere Richtung fällt auf
	// das reine "AT"-Pattern zurück).
	if err := validateMeteringPointPrefixMatch(meteringPoints, entrypoint); err != nil {
		return nil, err
	}

	// PROJ-37: cooperative-shares validation. When the EEG has activated
	// the feature, the application must carry a count >= required_shares.
	// Members fill in the count when the form is configured to display
	// the shares card; CreateApplication stores it on the row.
	if entrypoint.CooperativeSharesEnabled {
		minRequired := 1
		if entrypoint.CooperativeRequiredShares != nil {
			minRequired = *entrypoint.CooperativeRequiredShares
		}
		if app.CooperativeSharesCount == nil {
			return nil, shared.NewValidationError("Validation failed", map[string]string{
				"cooperativeSharesCount": "Anzahl der Genossenschaftsanteile ist erforderlich",
			})
		}
		if *app.CooperativeSharesCount < minRequired {
			return nil, shared.NewValidationError("Validation failed", map[string]string{
				"cooperativeSharesCount": fmt.Sprintf(
					"Mindestens %d Pflichtanteil(e) müssen gezeichnet werden", minRequired,
				),
			})
		}
	}

	now := time.Now()
	oldStatus := string(app.Status)

	// PROJ-31: when the EEG opt-in is on AND the public base URL is configured,
	// mint a fresh token, persist the SHA-256 hash, and ship the plaintext into
	// the outgoing mail. Without a public base URL the link can't be built — log
	// loudly and fall back to the legacy flow (better than blocking the submit).
	var emailConfirmationURL string
	var emailConfirmationTokenHash string
	if entrypoint.RequireEmailConfirmation {
		if s.publicBaseURL == "" {
			slog.Warn("email-confirmation: PUBLIC_BASE_URL unset — falling back to legacy flow", "rc", app.RCNumber)
		} else {
			plaintext, hash, tokErr := GenerateEmailConfirmationToken()
			if tokErr != nil {
				return nil, fmt.Errorf("token generation: %w", tokErr)
			}
			emailConfirmationTokenHash = hash
			emailConfirmationURL = BuildEmailConfirmationURL(s.publicBaseURL, plaintext)
		}
	}

	// All DB mutations (status, status log, consents, member number) run in one transaction
	// so a partial failure cannot leave the application in an inconsistent state.
	//
	// PROJ-36: two-stage consent collection. Frontend-supplied entries (the
	// boxes the member actively ticked) are written as `explicit`. We then
	// load the EEG's legal_documents and write an `informational` entry for
	// every non-required document the member did NOT also tick — those are
	// the "displayed for information" docs that don't get a checkbox in the
	// new UI but still need an audit-trail row.
	var consentRows []shared.DocumentConsent
	if s.consentRepo != nil {
		consentRows = make([]shared.DocumentConsent, 0, len(consents))
		seenURLs := make(map[string]struct{}, len(consents))
		for _, c := range consents {
			consentRows = append(consentRows, shared.DocumentConsent{
				ID:              uuid.New(),
				ApplicationID:   id,
				Title:           c.Title,
				URL:             c.URL,
				IsCentralPolicy: c.IsCentralPolicy,
				ConsentedAt:     now,
				ConsentType:     shared.ConsentTypeExplicit,
			})
			seenURLs[c.URL] = struct{}{}
		}
		if s.legalDocumentRepo != nil {
			docs, docErr := s.legalDocumentRepo.GetByRCNumber(app.RCNumber)
			if docErr != nil {
				slog.Warn("submit: failed to load legal documents for informational consents — skipping",
					"application_id", id, "error", docErr)
			} else {
				for _, d := range docs {
					if d.Required {
						continue
					}
					if _, dup := seenURLs[d.URL]; dup {
						continue
					}
					consentRows = append(consentRows, shared.DocumentConsent{
						ID:              uuid.New(),
						ApplicationID:   id,
						Title:           d.Title,
						URL:             d.URL,
						IsCentralPolicy: false,
						ConsentedAt:     now,
						ConsentType:     shared.ConsentTypeInformational,
					})
				}
			}
		}
	}

	tx, txErr := s.db.Begin()
	if txErr != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", txErr)
	}
	defer tx.Rollback()

	if err = s.appRepo.UpdateStatusTx(tx, id, shared.StatusSubmitted, &now); err != nil {
		return nil, err
	}

	// PROJ-52 Mini-Lücke 3: wenn der Mandat-PDF beim Submit verschickt
	// wird (Default-Pfad: sepa_mandate_at_import=false UND EEG hat die
	// nötigen Felder UND Mitglied hat zugestimmt), persistieren wir das
	// Mandatsdatum hier im selben Commit. Im Import-Pfad
	// (sepa_mandate_at_import=true oder B2B) setzt SendPostImportNotification
	// das Feld später nochmal.
	if !entrypoint.SEPAMandateAtImport && app.SepaMandateAccepted &&
		buildSEPAMandateData(app, entrypoint) != nil {
		if err = s.appRepo.SetMandateDateTx(tx, id, now); err != nil {
			return nil, err
		}
		app.MandateDate = &now
	}

	statusLog := &shared.StatusLogEntry{
		ApplicationID: id,
		FromStatus:    &oldStatus,
		ToStatus:      string(shared.StatusSubmitted),
		CreatedAt:     now,
	}
	if err = s.statusLogRepo.CreateTx(tx, statusLog); err != nil {
		return nil, fmt.Errorf("failed to create status log: %w", err)
	}

	if len(consentRows) > 0 {
		if err = s.consentRepo.CreateBulkTx(tx, consentRows); err != nil {
			return nil, fmt.Errorf("failed to save consents: %w", err)
		}
	}

	if emailConfirmationTokenHash != "" {
		expiresAt := now.Add(emailConfirmationTokenLifetime)
		if err = s.appRepo.AssignEmailConfirmationTokenTx(tx, id, emailConfirmationTokenHash, expiresAt); err != nil {
			return nil, err
		}
	}

	// Member number is no longer auto-assigned at submit time. The admin
	// picks it at import time in the tariff dialog (pre-filled from the
	// core's max+1 suggestion). application.member_number stays NULL until
	// the import succeeds — the approval PDF tolerates that.

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Send submission emails only on first submission (draft → submitted).
	if oldStatus == string(shared.StatusDraft) {
		metrics.ApplicationsSubmittedTotal.Inc()
		{
			var attachment []byte
			// PROJ-48: bei sepa_mandate_at_import=TRUE wird das Mandat NICHT
			// beim Submit angehängt — kommt erst beim Import mit ausgefüllter
			// Mandatsreferenz = Mitgliedsnummer (siehe SendPostImportNotification).
			//
			// PROJ-48 außerdem: die alte Auto-Logik "Firmenlastschrift bei
			// company/association" entfällt. Submit-Mail enthält IMMER die
			// Basis-Variante (Submit setzt einzugsart=core; B2B kommt erst
			// per Admin-Edit nach der Prüfung).
			if !entrypoint.SEPAMandateAtImport {
				if mandate := buildSEPAMandateData(app, entrypoint); mandate != nil {
					// PROJ-33: pull the cached EEG logo for the PDF embed.
					if logoBytes, logoMime, logoErr := s.entrypointRepo.GetLogo(app.RCNumber); logoErr == nil && len(logoBytes) > 0 {
						mandate.LogoBytes = logoBytes
						mandate.LogoMIME = logoMime
					}
					pdfBytes, pdfErr := s.pdfGenerator.Generate(*mandate)
					if pdfErr != nil {
						slog.Warn("pdf: failed to generate SEPA mandate", "rc", app.RCNumber, "error", pdfErr)
					} else {
						attachment = pdfBytes
					}
				}
			}
			var savedConsents []shared.DocumentConsent
			if s.consentRepo != nil {
				if sc, err := s.consentRepo.GetByApplicationID(id); err == nil {
					savedConsents = sc
				}
			}
			confirmationURL := emailConfirmationURL
			if confirmationURL != "" {
				metrics.EmailConfirmationsTotal.WithLabelValues("sent").Inc()
			}
			go func() {
				acquireMailSem()
				defer releaseMailSem()
				s.mailService.SendSubmissionEmails(app, meteringPoints, entrypoint, toStateMap(fieldConfig), attachment, savedConsents, confirmationURL)
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

// ConfirmEmail validates a member-supplied confirmation token, transitions the
// application from `submitted` to `email_confirmed`, writes the status_log
// entry, and triggers the deferred EEG-notification mail.
//
// Idempotent on re-clicks: when the application has already been confirmed
// (and the token row was already consumed) the call returns AlreadyConfirmed
// without an error so the success page renders cleanly the second time.
func (s *ApplicationService) ConfirmEmail(plaintext string) (*shared.ConfirmEmailResponse, error) {
	if strings.TrimSpace(plaintext) == "" {
		return nil, shared.NewValidationError("Validation failed", map[string]string{
			"token": "Token ist erforderlich",
		})
	}
	hash := HashEmailConfirmationToken(plaintext)

	app, err := s.appRepo.FindByEmailConfirmationTokenHash(hash)
	if err != nil {
		// Token doesn't (or no longer) matches any pending confirmation.
		// We surface ErrNotFound — the handler maps that to the generic
		// "ungültig oder abgelaufen" error message so an attacker can't
		// distinguish "wrong token" from "expired token".
		return nil, shared.ErrNotFound
	}

	// Idempotent re-click: the token-hash row is kept after consumption
	// (PROJ-31 Q5) so a member clicking the link twice gets a friendly
	// "already confirmed" page instead of a generic error.
	if app.EmailConfirmationUsedAt != nil {
		resp := &shared.ConfirmEmailResponse{AlreadyConfirmed: true}
		if entrypoint, epErr := s.entrypointRepo.GetByRCNumber(app.RCNumber); epErr == nil {
			if entrypoint.EEGName != nil {
				resp.EEGName = *entrypoint.EEGName
			}
			if entrypoint.ContactEmail != nil {
				resp.EEGContactEmail = *entrypoint.ContactEmail
			}
		}
		return resp, nil
	}

	// Expiry check
	if app.EmailConfirmationTokenExpiresAt == nil || app.EmailConfirmationTokenExpiresAt.Before(time.Now()) {
		return nil, shared.ErrNotFound
	}

	// Status must be submitted. If something else is going on (admin
	// rejected first, race with auto-reject job), treat as conflict.
	if app.Status != shared.StatusSubmitted {
		return nil, shared.NewConflictError("application is not waiting for e-mail confirmation")
	}

	now := time.Now()
	oldStatus := string(shared.StatusSubmitted)

	tx, txErr := s.db.Begin()
	if txErr != nil {
		return nil, fmt.Errorf("begin tx: %w", txErr)
	}
	defer tx.Rollback()

	if err := s.appRepo.MarkEmailConfirmedTx(tx, app.ID, now); err != nil {
		return nil, err
	}

	statusLog := &shared.StatusLogEntry{
		ApplicationID: app.ID,
		FromStatus:    &oldStatus,
		ToStatus:      string(shared.StatusEmailConfirmed),
		CreatedAt:     now,
	}
	reason := "E-Mail-Adresse über Bestätigungs-Link bestätigt"
	statusLog.Reason = &reason
	memberActor := "member"
	statusLog.ChangedByUserID = &memberActor

	if err := s.statusLogRepo.CreateTx(tx, statusLog); err != nil {
		return nil, fmt.Errorf("status log: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	metrics.EmailConfirmationsTotal.WithLabelValues("confirmed").Inc()

	// Trigger the deferred EEG-notification mail. Best-effort — log on
	// failure but don't fail the member's confirmation.
	entrypoint, epErr := s.entrypointRepo.GetByRCNumber(app.RCNumber)
	if epErr == nil {
		meteringPoints, mpErr := s.meteringRepo.GetByApplicationID(app.ID)
		if mpErr == nil {
			fieldConfig, fcErr := s.fieldConfigRepo.Get(strings.ToUpper(app.RCNumber))
			if fcErr != nil {
				fieldConfig = map[string]FieldConfigEntry{}
			}
			go func() {
				acquireMailSem()
				defer releaseMailSem()
				s.mailService.SendEEGNotification(app, meteringPoints, entrypoint, toStateMap(fieldConfig))
			}()
		} else {
			slog.Warn("confirm-email: failed to load metering points for EEG notification", "application_id", app.ID, "error", mpErr)
		}
	} else {
		slog.Warn("confirm-email: failed to load entrypoint for EEG notification", "application_id", app.ID, "error", epErr)
	}

	resp := &shared.ConfirmEmailResponse{}
	if entrypoint != nil {
		if entrypoint.EEGName != nil {
			resp.EEGName = *entrypoint.EEGName
		}
		if entrypoint.ContactEmail != nil {
			resp.EEGContactEmail = *entrypoint.ContactEmail
		}
	}
	return resp, nil
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

// generateReferenceNumber returns the per-EEG, per-year reference number
// (PROJ-35) of the form `<rc>-<year>-<NNNN>`, e.g. `RC105720-2026-0001`.
// Counter resets each year and runs independently per EEG. Falls back to a
// random suffix on DB failure so the submit path never blocks on this.
//
// Old applications (created before PROJ-35) keep their `MO-YYYY-NNNNNN`
// refs unchanged; the obsolete sequence from migration 25 stays in place
// as a historical artefact.
func (s *ApplicationService) generateReferenceNumber(rcNumber string) string {
	rcNumber = strings.ToUpper(strings.TrimSpace(rcNumber))
	year := time.Now().Year()
	ref, err := s.appRepo.NextReferenceNumber(rcNumber, year)
	if err != nil {
		slog.Error("generateReferenceNumber: counter query failed, using fallback",
			"rc_number", rcNumber, "year", year, "error", err)
		n, _ := rand.Int(rand.Reader, big.NewInt(9000))
		return fmt.Sprintf("%s-%d-FB%04d", rcNumber, year, n.Int64()+1000)
	}
	return ref
}

// trimStringPtr trims whitespace from a *string, returning nil if the pointer is nil.
func trimStringPtr(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	return &trimmed
}

// defaultParticipationFactor (Teilnahmefaktor, Erweiterung 2026-05-19) maps
// ParticipationFactor=0 (hidden/admin_only field, or member-submitted 0) to
// the historical default of 100. Upper bound is enforced by validate:"max=100"
// at the request layer.
func defaultParticipationFactor(v int) int {
	if v <= 0 {
		return 100
	}
	return v
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

// meteringPointRegex enforces the Austrian Zählpunkt-Nummer format nach
// E-Control / MeteringCode (PROJ-52): "AT" + 11 Ziffern (Netzbetreiber-
// nummer + PLZ) + 20 alphanumerische Stellen (Zählpunkt-Kennung).
// Pre-compiled at package init.
//
// Davor (Pre-PROJ-52): `^AT[0-9]{31}$` — die letzten 20 Stellen waren auf
// Ziffern beschränkt. In der österreichischen Praxis sind die Zählpunkte
// fast immer numerisch, die offizielle Spec erlaubt aber A-Z0-9. Bestands-
// daten bleiben gültig (Ziffern sind eine Teilmenge von [A-Z0-9]).
var meteringPointRegex = regexp.MustCompile(`^AT[0-9]{11}[A-Z0-9]{20}$`)

// meteringPointPrefixRegex matches the per-direction configurable prefix
// (PROJ-52). Must start with "AT", length 2–33, only digits + uppercase
// letters after the "AT". Matches the DB CHECK on registration_entrypoint.
var meteringPointPrefixRegex = regexp.MustCompile(`^AT[0-9A-Z]{0,31}$`)

// validateMeteringPointFormat returns true when mp matches the Austrian
// Zählpunkt format. Whitespace and case are NOT normalised here — callers
// must pass the canonical (uppercase, no spaces) form, which is what both
// the frontend Zod transform and the public-form mask deliver.
func validateMeteringPointFormat(mp string) bool {
	return meteringPointRegex.MatchString(mp)
}

// NormalizeMeteringPointPrefix strips whitespace and dots, uppercases the
// result, and returns it. Returns nil when the input is nil or yields an
// empty string after normalisation (so "  " stores as NULL, not empty).
func NormalizeMeteringPointPrefix(in *string) *string {
	if in == nil {
		return nil
	}
	s := *in
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ToUpper(s)
	if s == "" {
		return nil
	}
	return &s
}

// ValidateMeteringPointPrefix returns nil when in matches the configurable
// prefix format (PROJ-52) or is nil/empty. Callers should normalize first.
func ValidateMeteringPointPrefix(in *string) error {
	if in == nil || *in == "" {
		return nil
	}
	if !meteringPointPrefixRegex.MatchString(*in) {
		return fmt.Errorf("Prefix muss mit AT beginnen und darf nur Ziffern + A-Z enthalten (max 33 Stellen)")
	}
	return nil
}

// validateMeteringPointPrefixMatch enforces the per-direction Zählpunkt-
// Prefix-Konfiguration aus dem EEG-Entrypoint (PROJ-52). NULL prefix für
// eine Richtung ⇒ keine Prüfung; sonst muss jeder Zählpunkt dieser
// Richtung mit dem konfigurierten Prefix beginnen. Liefert genau einen
// Fehler pro betroffener Richtung (kompakt; das Frontend sieht ohnehin
// alle Verstöße über die dynamische Mask).
func validateMeteringPointPrefixMatch(points []shared.MeteringPoint, ep *shared.RegistrationEntrypoint) error {
	if ep == nil {
		return nil
	}
	errs := map[string]string{}
	for i, mp := range points {
		var prefix *string
		switch mp.Direction {
		case shared.DirectionConsumption:
			prefix = ep.MeteringPointPrefixConsumption
		case shared.DirectionProduction:
			prefix = ep.MeteringPointPrefixProduction
		}
		if prefix == nil || *prefix == "" {
			continue
		}
		if !strings.HasPrefix(mp.MeteringPoint, *prefix) {
			errs[fmt.Sprintf("meteringPoints.%d.meteringPoint", i)] = fmt.Sprintf(
				"Zählpunkt muss mit %s beginnen (vom EEG-Admin konfigurierter Prefix für diese Richtung)",
				*prefix,
			)
		}
	}
	if len(errs) > 0 {
		return shared.NewValidationError("Validation failed", errs)
	}
	return nil
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
	// PROJ-49: consumption_previous_year, consumption_forecast,
	// feed_in_forecast, pv_power_kwp leben jetzt pro Zählpunkt — siehe
	// applyAdminValuesToMeteringPoint. Hier nichts mehr zu tun.
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
	apply("electric_vehicle_count", func(v string) {
		if app.ElectricVehicleCount == nil {
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
				app.ElectricVehicleCount = &n
			}
		}
	})
	apply("electric_vehicle_annual_km", func(v string) {
		if app.ElectricVehicleAnnualKm == nil {
			var n int
			if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
				app.ElectricVehicleAnnualKm = &n
			}
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
//
// `mps` enables PROJ-45 typabhängige Sichtbarkeit: when provided, consumption-
// related fields (heat_pump, electric_vehicle, …) only fail the required-check
// when at least one CONSUMPTION metering point exists, and production-related
// fields (pv_power_kwp, feed_in_forecast) only when a PRODUCTION point exists.
// Pass nil to disable the gating (legacy callers and unit tests).
func validateConfigurableRequiredFields(app *shared.Application, fieldConfig map[string]FieldConfigEntry, mps []shared.MeteringPoint) error {
	errs := map[string]string{}

	// PROJ-45: when mps is provided, gate type-specific required-checks
	// on the presence of a matching meter direction. nil ⇒ no gating
	// (legacy + test callers). PROJ-49 removed the production-only
	// application-level fields (they live on the metering point now), so
	// only the consumption gate remains here.
	hasConsumption := true
	if mps != nil {
		hasConsumption = false
		for _, mp := range mps {
			if mp.Direction == shared.DirectionConsumption {
				hasConsumption = true
			}
		}
	}

	requiredIfMissing := func(name, jsonKey, label string, missing bool) {
		if effectiveState(fieldConfig, name) == "required" && missing {
			errs[jsonKey] = label + " ist erforderlich"
		}
	}
	requiredIfMissingTyped := func(name, jsonKey, label string, missing bool, gate bool) {
		if !gate {
			return
		}
		requiredIfMissing(name, jsonKey, label, missing)
	}
	missingStr := func(v *string) bool { return v == nil || strings.TrimSpace(*v) == "" }

	requiredIfMissing("phone", "phone", "Telefonnummer", missingStr(app.Phone))
	requiredIfMissing("birth_date", "birthDate", "Geburtsdatum", app.BirthDate == nil)
	requiredIfMissing("bank_name", "bankName", "Bankname", missingStr(app.BankName))
	requiredIfMissing("uid_number", "uidNumber", "UID-Nummer", missingStr(app.UIDNumber))
	requiredIfMissing("membership_start_date", "membershipStartDate", "Beitrittsdatum", app.MembershipStartDate == nil)
	// "Personen im Haushalt" gilt nur für natürliche Personen (Privatperson,
	// Landwirt). Bei Org-Mitgliedstypen wäre die Required-Validierung ein
	// Submit-Hänger (Feld wird FE nicht gezeigt, Field-State zeigt es aber
	// als required). Gate hier doppelt: Mitgliedstyp UND hasConsumption.
	isNaturalPerson := app.MemberType == shared.MemberTypePrivate || app.MemberType == shared.MemberTypeFarmer
	requiredIfMissingTyped("persons_in_household", "personsInHousehold", "Anzahl Personen im Haushalt", app.PersonsInHousehold == nil, hasConsumption && isNaturalPerson)
	// PROJ-49: consumption_previous_year, consumption_forecast,
	// feed_in_forecast, pv_power_kwp werden jetzt pro Zählpunkt validiert
	// (validateConfigurableMeteringPointFields), nicht mehr hier.
	requiredIfMissingTyped("heat_pump", "heatPump", "Wärmepumpe vorhanden", app.HeatPump == nil, hasConsumption)
	requiredIfMissingTyped("electric_vehicle", "electricVehicle", "E-Auto vorhanden", app.ElectricVehicle == nil, hasConsumption)
	// PROJ-42: die Detail-Felder sind nur sinnvoll wenn EV=true. Wenn der
	// Bewerber EV=Nein angegeben hat, gelten Count + Jahres-km als „nicht
	// anwendbar" — auch wenn die EEG sie als required konfiguriert hat
	// (sonst würde ein Nein-Bewerber an „Anzahl E-Fahrzeuge ist erforderlich"
	// scheitern). Required greift also nur wenn EV=true UND count/km fehlt.
	evIsTrue := app.ElectricVehicle != nil && *app.ElectricVehicle
	requiredIfMissingTyped("electric_vehicle_count", "electricVehicleCount", "Anzahl E-Fahrzeuge", evIsTrue && app.ElectricVehicleCount == nil, hasConsumption)
	requiredIfMissingTyped("electric_vehicle_annual_km", "electricVehicleAnnualKm", "Jahres-Kilometer (E-Fahrzeuge)", evIsTrue && app.ElectricVehicleAnnualKm == nil, hasConsumption)
	requiredIfMissingTyped("electric_hot_water", "electricHotWater", "Warmwasser elektrisch", app.ElectricHotWater == nil, hasConsumption)
	// PROJ-44: required ⇒ Häkchen muss gesetzt sein. Bool default FALSE
	// reicht nicht — die Vollmacht muss explizit erteilt werden.
	requiredIfMissing("network_operator_authorization", "networkOperatorAuthorization", "Netzbetreiber-Vollmacht", !app.NetworkOperatorAuthorization)
	// PROJ-56: Required gilt nur, wenn die Vollmacht erteilt wurde —
	// sonst sind die Felder konzeptuell nicht anwendbar (analog zur
	// EV-Count/AnnualKm-Logik oben). Sichtbar im UI sind sie nur dann,
	// also wäre eine Required-Validierung ohne Vollmacht ein stiller
	// Submit-Hänger (vgl. PROJ-56-Spec / Frontend-Bug-Pattern).
	requiredIfMissing("network_operator_customer_number", "networkOperatorCustomerNumber", "Netzbetreiber Kundennummer", app.NetworkOperatorAuthorization && missingStr(app.NetworkOperatorCustomerNumber))
	requiredIfMissing("meter_inventory_number", "meterInventoryNumber", "Inventarnummer eines Zählers", app.NetworkOperatorAuthorization && missingStr(app.MeterInventoryNumber))

	// PROJ-57 v3: Ansprechperson — alle drei Felder werden seit dieser
	// Version einzeln per field_config gesteuert (hidden/optional/required).
	// Wenn der Toggle aktiv ist, wird jedes Feld geprüft, aber required
	// gilt nur bei state=required. Bei state=optional ist das Feld
	// sichtbar aber leer absendbar; bei state=hidden ist es weder
	// sichtbar noch validiert (clearContactPersonIfDisabled cleart zudem).
	if app.HasContactPerson {
		if effectiveState(fieldConfig, "contact_person_name") == "required" && missingStr(app.ContactPersonName) {
			errs["contactPersonName"] = "Name der Ansprechperson ist erforderlich"
		}
		if effectiveState(fieldConfig, "contact_person_email") == "required" && missingStr(app.ContactPersonEmail) {
			errs["contactPersonEmail"] = "E-Mail der Ansprechperson ist erforderlich"
		}
		if effectiveState(fieldConfig, "contact_person_phone") == "required" && missingStr(app.ContactPersonPhone) {
			errs["contactPersonPhone"] = "Telefon der Ansprechperson ist erforderlich"
		}
	}

	// PROJ-58: Rechnungs-E-Mail. Wenn Toggle aktiv, Email Pflicht
	// (Format-Validation läuft via validate-Tag im Request-Schema).
	if app.HasBillingEmail && missingStr(app.BillingEmail) {
		errs["billingEmail"] = "Rechnungs-E-Mail ist erforderlich"
	}

	if len(errs) > 0 {
		return shared.NewValidationError("Validation failed", errs)
	}
	return nil
}

// validateConfigurableMeteringPointFields checks metering-point-level fields configured as "required".
// validateMeteringPointAddresses enforces PROJ-39's all-or-nothing rule:
// per metering point either all four address fields are empty (the member's
// primary address is used) or all four are set (deviating address).
func validateMeteringPointAddresses(points []shared.MeteringPoint) error {
	for i, mp := range points {
		fields := map[string]*string{
			"addressStreet":       mp.AddressStreet,
			"addressStreetNumber": mp.AddressStreetNumber,
			"addressZip":          mp.AddressZip,
			"addressCity":         mp.AddressCity,
		}
		filled := 0
		for _, v := range fields {
			if v != nil && strings.TrimSpace(*v) != "" {
				filled++
			}
		}
		if filled == 0 || filled == 4 {
			continue
		}
		errs := map[string]string{}
		for name, v := range fields {
			if v == nil || strings.TrimSpace(*v) == "" {
				errs[fmt.Sprintf("meteringPoints.%d.%s", i, name)] = "Adressfeld ist erforderlich wenn abweichende Adresse aktiv"
			}
		}
		return shared.NewValidationError("Validation failed", errs)
	}
	return nil
}

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
		// PROJ-49: Verbrauchs-Felder nur bei CONSUMPTION-Zählpunkten prüfen.
		if mp.Direction == shared.DirectionConsumption {
			if effectiveState(fieldConfig, "consumption_previous_year") == "required" && mp.ConsumptionPreviousYear == nil {
				errs[fmt.Sprintf("meteringPoints.%d.consumptionPreviousYear", i)] = "Verbrauch Vorjahr ist erforderlich"
			}
			if effectiveState(fieldConfig, "consumption_forecast") == "required" && mp.ConsumptionForecast == nil {
				errs[fmt.Sprintf("meteringPoints.%d.consumptionForecast", i)] = "Verbrauch Prognose ist erforderlich"
			}
		}
		// PROJ-45 + PROJ-49: Einspeise-Felder nur bei PRODUCTION; PV-Leistung
		// + Batterie/Wechselrichter + Einspeiselimit nur bei PRODUCTION + PV.
		isPv := mp.Direction == shared.DirectionProduction &&
			mp.GenerationType != nil && *mp.GenerationType == "pv"
		if mp.Direction == shared.DirectionProduction {
			if effectiveState(fieldConfig, "feed_in_forecast") == "required" && mp.FeedInForecast == nil {
				errs[fmt.Sprintf("meteringPoints.%d.feedInForecast", i)] = "Einspeisung Prognose ist erforderlich"
			}
		}
		if isPv {
			if effectiveState(fieldConfig, "battery_size_kwh") == "required" && mp.BatterySizeKwh == nil {
				errs[fmt.Sprintf("meteringPoints.%d.batterySizeKwh", i)] = "Größe Batterie ist erforderlich"
			}
			if effectiveState(fieldConfig, "inverter_manufacturer") == "required" &&
				(mp.InverterManufacturer == nil || strings.TrimSpace(*mp.InverterManufacturer) == "") {
				errs[fmt.Sprintf("meteringPoints.%d.inverterManufacturer", i)] = "Hersteller Wechselrichter ist erforderlich"
			}
			if effectiveState(fieldConfig, "pv_power_kwp") == "required" && mp.PvPowerKwp == nil {
				errs[fmt.Sprintf("meteringPoints.%d.pvPowerKwp", i)] = "PV-Leistung ist erforderlich"
			}
			if effectiveState(fieldConfig, "inverter_power_kw") == "required" && mp.InverterPowerKw == nil {
				errs[fmt.Sprintf("meteringPoints.%d.inverterPowerKw", i)] = "Leistung PV-Wechselrichter ist erforderlich"
			}
			// feed_in_limit_kw ist nur Pflicht wenn FeedInLimitPresent=true.
			if effectiveState(fieldConfig, "feed_in_limit_kw") == "required" &&
				mp.FeedInLimitPresent != nil && *mp.FeedInLimitPresent && mp.FeedInLimitKw == nil {
				errs[fmt.Sprintf("meteringPoints.%d.feedInLimitKw", i)] = "Einspeiselimit (kW) ist erforderlich"
			}
			// PROJ-49 follow-up: Speichersteuerung-Frage ist nur Pflicht,
			// wenn der Zählpunkt PV ist UND Batterie-Daten gesetzt sind.
			hasBattery := mp.BatterySizeKwh != nil ||
				(mp.InverterManufacturer != nil && strings.TrimSpace(*mp.InverterManufacturer) != "")
			if effectiveState(fieldConfig, "battery_control_acceptable") == "required" &&
				hasBattery && mp.BatteryControlAcceptable == nil {
				errs[fmt.Sprintf("meteringPoints.%d.batteryControlAcceptable", i)] = "Speichersteuerung im Sinne der EEG ist erforderlich"
			}
		}
		if len(errs) > 0 {
			return shared.NewValidationError("Validation failed", errs)
		}
	}
	return nil
}

// normalizeMeteringPointGeneration enforces PROJ-45 invariants on each MP:
//   - CONSUMPTION ⇒ generation_type/battery/inverter all NULL
//   - PRODUCTION without explicit generation_type defaults to 'pv'
//   - PRODUCTION with non-pv generation_type ⇒ battery/inverter NULL
//
// Called before Insert/Update so the DB-CHECK doesn't reject valid client
// payloads that omitted the default, and so forged clients can't smuggle
// battery values for wind/hydro/biomass plants.
func normalizeMeteringPointGeneration(points []shared.MeteringPoint) {
	for i := range points {
		mp := &points[i]
		if mp.Direction != shared.DirectionProduction {
			mp.GenerationType = nil
			mp.BatterySizeKwh = nil
			mp.InverterManufacturer = nil
			mp.InverterPowerKw = nil
			continue
		}
		if mp.GenerationType == nil || strings.TrimSpace(*mp.GenerationType) == "" {
			pv := "pv"
			mp.GenerationType = &pv
		}
		if *mp.GenerationType != "pv" {
			mp.BatterySizeKwh = nil
			mp.InverterManufacturer = nil
			mp.InverterPowerKw = nil
		}
	}
}

// clearEVDetailsIfDisabled drops PROJ-42 details when ElectricVehicle is
// not actively set to true. Service-level gate (no DB constraint) so the
// row never carries Count/Km values that don't match the "ja/nein"-flag.
func clearEVDetailsIfDisabled(app *shared.Application) {
	if app.ElectricVehicle == nil || !*app.ElectricVehicle {
		app.ElectricVehicleCount = nil
		app.ElectricVehicleAnnualKm = nil
	}
}

// clearAppFieldsByMpTypes implements PROJ-45 typabhängige Sichtbarkeit at the
// service layer: when the application carries no CONSUMPTION meter, all
// consumption-related application-level fields (Wärmepumpe, E-Auto,
// Warmwasser, Personen) are nilled. PROJ-49 moved the energy values
// (Verbrauch, PV-Leistung, Einspeisung Prognose) to the metering point —
// they are gated per-MP by clearMeteringPointEnergyByType.
func clearAppFieldsByMpTypes(app *shared.Application, mps []shared.MeteringPoint) {
	if mps == nil {
		return
	}
	hasConsumption := false
	for _, mp := range mps {
		if mp.Direction == shared.DirectionConsumption {
			hasConsumption = true
		}
	}
	if !hasConsumption {
		app.PersonsInHousehold = nil
		app.HeatPump = nil
		app.ElectricVehicle = nil
		app.ElectricVehicleCount = nil
		app.ElectricVehicleAnnualKm = nil
		app.ElectricHotWater = nil
	}
}

// clearMeteringPointEnergyByType enforces PROJ-49 invariants per metering
// point:
//   - CONSUMPTION ⇒ feed_in_*, pv_power_kwp, feed_in_limit_* NULL
//   - PRODUCTION ⇒ consumption_* NULL
//   - PRODUCTION + GenerationType != "pv" ⇒ pv_power_kwp + feed_in_limit_* NULL
//   - FeedInLimitPresent != true ⇒ FeedInLimitKw NULL
//
// Run before persist so forged clients can't smuggle values that don't match
// the meter's direction/generation type.
func clearMeteringPointEnergyByType(points []shared.MeteringPoint) {
	for i := range points {
		mp := &points[i]
		if mp.Direction == shared.DirectionConsumption {
			mp.FeedInForecast = nil
			mp.PvPowerKwp = nil
			mp.FeedInLimitPresent = nil
			mp.FeedInLimitKw = nil
			mp.BatteryControlAcceptable = nil
			continue
		}
		// PRODUCTION
		mp.ConsumptionPreviousYear = nil
		mp.ConsumptionForecast = nil
		isPv := mp.GenerationType != nil && *mp.GenerationType == "pv"
		if !isPv {
			mp.PvPowerKwp = nil
			mp.FeedInLimitPresent = nil
			mp.FeedInLimitKw = nil
			mp.BatteryControlAcceptable = nil
			mp.InverterPowerKw = nil
			continue
		}
		if mp.FeedInLimitPresent == nil || !*mp.FeedInLimitPresent {
			mp.FeedInLimitKw = nil
		}
		// PROJ-49 follow-up: Speichersteuerung-Frage ist nur sinnvoll, wenn
		// das Mitglied einen Batteriespeicher angegeben hat (Größe oder
		// Hersteller). Wenn beide leer sind, gibt es nichts zu steuern —
		// Antwort wird genullt, damit kein "Phantom-Consent" persistiert.
		hasBattery := mp.BatterySizeKwh != nil ||
			(mp.InverterManufacturer != nil && *mp.InverterManufacturer != "")
		if !hasBattery {
			mp.BatteryControlAcceptable = nil
		}
	}
}

// contactPersonEnabled liefert true, wenn die EEG mindestens eines der drei
// Ansprechperson-Felder (Name, Email, Telefon) nicht auf hidden gestellt
// hat. Ersetzt seit PROJ-57 v3 den ehemaligen Master-Switch `contact_person`:
// die Sichtbarkeit des Blocks wird aus den drei Sub-Field-States abgeleitet.
// Mindestens ein Feld != hidden → Checkbox im Public-Form sichtbar.
func contactPersonEnabled(fieldConfig map[string]FieldConfigEntry) bool {
	return effectiveState(fieldConfig, "contact_person_name") != "hidden" ||
		effectiveState(fieldConfig, "contact_person_email") != "hidden" ||
		effectiveState(fieldConfig, "contact_person_phone") != "hidden"
}

// clearContactPersonIfDisabled handles the PROJ-57 contact-person fields.
// Cleared zu false/NULL, wenn:
//   - alle drei Sub-Field-States auf "hidden" stehen (kein Block sichtbar)
//   - der Mitgliedstyp nicht in der Org-Liste liegt
//   - HasContactPerson=false ist
// Plus: einzelne Sub-Felder werden geclearted, wenn ihr State hidden ist —
// Schutz gegen forged Clients.
func clearContactPersonIfDisabled(app *shared.Application, fieldConfig map[string]FieldConfigEntry) {
	disabled := !contactPersonEnabled(fieldConfig) ||
		!isOrgMemberType(app.MemberType)
	if disabled {
		app.HasContactPerson = false
	}
	if !app.HasContactPerson {
		app.ContactPersonName = nil
		app.ContactPersonEmail = nil
		app.ContactPersonPhone = nil
		return
	}
	if effectiveState(fieldConfig, "contact_person_name") == "hidden" {
		app.ContactPersonName = nil
	}
	if effectiveState(fieldConfig, "contact_person_email") == "hidden" {
		app.ContactPersonEmail = nil
	}
	if effectiveState(fieldConfig, "contact_person_phone") == "hidden" {
		app.ContactPersonPhone = nil
	}
}

// isOrgMemberType returns true für die drei Mitgliedstypen, bei denen
// die Unterscheidung zwischen Org-Konto und konkreter Ansprechperson
// sinnvoll ist (PROJ-57). sole_proprietor (Kleinunternehmer) ist bewusst
// nicht dabei — dort ist der Inhaber der Ansprechpartner.
func isOrgMemberType(mt shared.MemberType) bool {
	return mt == shared.MemberTypeCompany ||
		mt == shared.MemberTypeAssociation ||
		mt == shared.MemberTypeMunicipality
}

// clearBillingEmailIfDisabled handles the PROJ-58 billing-email fields.
// Analog zu clearContactPersonIfDisabled: Toggle + Email gehören nur zum
// Org-Mitgliedstyp + aktivem Toggle + nicht-hidden field_config.
func clearBillingEmailIfDisabled(app *shared.Application, fieldConfig map[string]FieldConfigEntry) {
	disabled := effectiveState(fieldConfig, "billing_email") == "hidden" ||
		!isOrgMemberType(app.MemberType)
	if disabled {
		app.HasBillingEmail = false
	}
	if !app.HasBillingEmail {
		app.BillingEmail = nil
	}
}

// clearNetworkAuthIfHidden resets the PROJ-44 authorization when the EEG has
// the field set to "hidden". Prevents a forged client from setting the flag
// for an EEG that doesn't collect it. Auch PROJ-56-Felder werden hier
// geclearted: zum einen wenn die EEG sie versteckt hat, zum anderen wenn
// die Vollmacht selbst nicht erteilt wurde (die Felder gehören semantisch
// nur zum Vollmachts-Kontext).
func clearNetworkAuthIfHidden(app *shared.Application, fieldConfig map[string]FieldConfigEntry) {
	if effectiveState(fieldConfig, "network_operator_authorization") == "hidden" {
		app.NetworkOperatorAuthorization = false
		app.NetworkOperatorAuthorizationAt = nil
	}
	// PROJ-56: ohne erteilte Vollmacht keine Netzbetreiber-Info-Felder.
	if !app.NetworkOperatorAuthorization {
		app.NetworkOperatorCustomerNumber = nil
		app.MeterInventoryNumber = nil
	}
	// Zusätzlich: wenn die EEG die Info-Felder versteckt hat, ebenfalls
	// auf NULL setzen — verhindert dass ein forged client sie an einer
	// EEG ohne entsprechende Konfiguration einschleust.
	if effectiveState(fieldConfig, "network_operator_customer_number") == "hidden" {
		app.NetworkOperatorCustomerNumber = nil
	}
	if effectiveState(fieldConfig, "meter_inventory_number") == "hidden" {
		app.MeterInventoryNumber = nil
	}
}

// clearMemberTypeFields nils out fields not applicable to the current member type.
func clearMemberTypeFields(app *shared.Application) {
	switch app.MemberType {
	case shared.MemberTypePrivate, shared.MemberTypeFarmer:
		app.CompanyName = nil
		app.UIDNumber = nil
		app.RegisterNumber = nil
	case shared.MemberTypeSoleProprietor:
		// PROJ-28: only company_name is collected; everything else is wiped.
		app.Firstname = nil
		app.Lastname = nil
		app.BirthDate = nil
		app.UIDNumber = nil
		app.RegisterNumber = nil
		// "Personen im Haushalt" ist nur bei natürlichen Personen
		// (private/farmer) konzeptuell sinnvoll — defence-in-depth gegen
		// forged Clients und gegen Required-Validierung, die sonst bei
		// EEG-config "required" einen Submit-Hänger erzeugt.
		app.PersonsInHousehold = nil
	case shared.MemberTypeMunicipality, shared.MemberTypeCompany, shared.MemberTypeAssociation:
		app.Firstname = nil
		app.Lastname = nil
		app.BirthDate = nil
		app.PersonsInHousehold = nil
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
	case shared.MemberTypeSoleProprietor:
		if app.CompanyName == nil || strings.TrimSpace(*app.CompanyName) == "" {
			return shared.NewValidationError("Validation failed", map[string]string{
				"companyName": "Firmenname ist erforderlich",
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
		// Firmenbuchnummer ist optional — manche Firmen (z. B. nicht
		// firmenbuchpflichtige Einzelunternehmer mit Firmenbezeichnung)
		// haben keine. Vereinsnummer für `association` bleibt Pflicht
		// (ZVR ist für Vereine in AT verpflichtend).
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
	data := &pdf.SEPAMandateData{
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
	// PROJ-52 Mini-Lücke 3: Tag der Mandat-Übermittlung im Unterschriftsfeld
	// vorbefüllen. app.MandateDate ist beim Submit/Import vom Service-Layer
	// auf time.Now() gesetzt worden (siehe SubmitApplication +
	// SendPostImportNotification) und entspricht damit dem tatsächlichen
	// Versanddatum des PDFs.
	if app.MandateDate != nil {
		data.MandateDate = *app.MandateDate
	}
	return data
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
