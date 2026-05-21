package shared

import (
	"time"

	"github.com/google/uuid"
)

// RegistrationEntrypoint maps an EEG RC number to its internal EEG ID.
// It is the sole source of truth for public registration lookup.
type RegistrationEntrypoint struct {
	ID           uuid.UUID `json:"id"           db:"id"`
	RCNumber     string    `json:"rcNumber"     db:"rc_number"`
	EegID        *string   `json:"eegId"        db:"eeg_id"`
	IsActive     bool      `json:"isActive"     db:"is_active"`
	ContactEmail       *string   `json:"contactEmail"       db:"contact_email"`
	IntroText          *string   `json:"introText"          db:"intro_text"`
	EEGName            *string   `json:"eegName"            db:"eeg_name"`
	EEGStreet          *string   `json:"eegStreet"          db:"eeg_street"`
	EEGStreetNumber    *string   `json:"eegStreetNumber"    db:"eeg_street_number"`
	EEGZip             *string   `json:"eegZip"             db:"eeg_zip"`
	EEGCity            *string   `json:"eegCity"            db:"eeg_city"`
	CreditorID         *string   `json:"creditorId"         db:"creditor_id"`
	SEPAMandateEnabled         bool      `json:"sepaMandateEnabled"         db:"sepa_mandate_enabled"`
	UseCompanySEPAMandate      bool      `json:"useCompanySEPAMandate"      db:"use_company_sepa_mandate"`
	// PROJ-48: Wenn TRUE, wird das SEPA-Mandat-PDF NICHT beim Submit,
	// sondern erst beim Import mit eingedruckter Mandatsreferenz
	// (= Mitgliedsnummer) versendet. Gilt für beide Mandat-Varianten
	// (Basis + B2B). Default FALSE = heutiges Verhalten (Mandat bei
	// Submit ohne Mandatsreferenz).
	SEPAMandateAtImport        bool      `json:"sepaMandateAtImport"        db:"sepa_mandate_at_import"`
	ShowCentralPolicy          bool      `json:"showCentralPolicy"          db:"show_central_policy"`
	MemberNumberStart          int       `json:"memberNumberStart"          db:"member_number_start"`
	RequireEmailConfirmation   bool      `json:"requireEmailConfirmation"   db:"require_email_confirmation"`
	// PROJ-52: pro Richtung konfigurierbarer Zählpunkt-Prefix. NULL = heutiges
	// Verhalten (nur "AT" ist fix). Wenn gesetzt, prüft das Backend beim
	// Submit, dass Zählpunkte der jeweiligen Richtung mit dem Prefix beginnen
	// (defense-in-depth zur Frontend-Mask). DB-CHECK-Constraint stellt das
	// Roh-Format sicher (^AT[0-9A-Z]{0,31}$), Service-Layer normalisiert
	// vor dem Speichern (Whitespace + Dots entfernen, uppercase).
	MeteringPointPrefixConsumption *string `json:"meteringPointPrefixConsumption,omitempty" db:"metering_point_prefix_consumption"`
	MeteringPointPrefixProduction  *string `json:"meteringPointPrefixProduction,omitempty"  db:"metering_point_prefix_production"`
	// PROJ-53: Aktivierungs-Modus. Steuert, woran der Activation-Check-Batch
	// erkennt, dass eine Anwendung von ready_for_activation auf activated
	// wechseln darf. Default 'participant_active' = heutige Lösung
	// (Core-Teilnehmer-Status ACTIVE). 'any_meter_registration_started' =
	// mindestens ein Zählpunkt im Core mit processState in
	// PENDING/APPROVED/ACTIVE — frühere Aktivierung sobald der
	// Netzbetreiber die Online-Registrierung bestätigt hat.
	ActivationMode string `json:"activationMode" db:"activation_mode"`
	// LastSyncedFromCoreAt is NULL until the admin has triggered the first
	// sync (PROJ-32). After that, every successful sync stamps it with NOW().
	LastSyncedFromCoreAt       *time.Time `json:"lastSyncedFromCoreAt,omitempty" db:"last_synced_from_core_at"`
	// EEGLogoSyncedAt is NULL until the first successful logo fetch from the
	// eegFaktura-billing service (PROJ-33). Set separately from
	// LastSyncedFromCoreAt because the logo sync is best-effort: master-data
	// sync can succeed while the logo sync skips or fails.
	EEGLogoSyncedAt            *time.Time `json:"eegLogoSyncedAt,omitempty" db:"eeg_logo_synced_at"`
	// PROJ-37: Genossenschaftsanteile. Three settings configure whether
	// new members must subscribe cooperative shares, how many are
	// mandatory, and the price per share. All three are NULL/false when
	// the feature is off for this EEG.
	CooperativeSharesEnabled       bool       `json:"cooperativeSharesEnabled"       db:"cooperative_shares_enabled"`
	CooperativeRequiredShares      *int       `json:"cooperativeRequiredShares,omitempty"      db:"cooperative_required_shares"`
	CooperativeShareAmountCents    *int64     `json:"cooperativeShareAmountCents,omitempty"    db:"cooperative_share_amount_cents"`
	CreatedAt                  time.Time `json:"createdAt"                  db:"created_at"`
	UpdatedAt          time.Time `json:"updatedAt"          db:"updated_at"`
}

// ApplicationStatus represents the status of an application
type ApplicationStatus string

const (
	StatusDraft                     ApplicationStatus = "draft"
	StatusSubmitted                 ApplicationStatus = "submitted"
	StatusEmailConfirmed            ApplicationStatus = "email_confirmed"
	StatusUnderReview               ApplicationStatus = "under_review"
	StatusNeedsInfo                 ApplicationStatus = "needs_info"
	StatusApproved                  ApplicationStatus = "approved"
	StatusRejected                  ApplicationStatus = "rejected"
	StatusImported                  ApplicationStatus = "imported"
	StatusImportFailed              ApplicationStatus = "import_failed"
	// PROJ-46: Stati für die Nachbereitung nach erfolgreichem Import.
	StatusAwaitingBankConfirmation  ApplicationStatus = "awaiting_bank_confirmation"
	StatusReadyForActivation        ApplicationStatus = "ready_for_activation"
	StatusActivated                 ApplicationStatus = "activated"
)

// ActivationMode (PROJ-53) selects how the activation-check decides
// when an application moves from ready_for_activation to activated.
const (
	ActivationModeParticipantActive          = "participant_active"
	ActivationModeAnyMeterRegistrationStarted = "any_meter_registration_started"
)

// IsValidActivationMode returns true for the two known modes.
func IsValidActivationMode(s string) bool {
	return s == ActivationModeParticipantActive || s == ActivationModeAnyMeterRegistrationStarted
}

// MemberType represents the type of EEG member
type MemberType string

const (
	MemberTypePrivate        MemberType = "private"
	MemberTypeSoleProprietor MemberType = "sole_proprietor"
	MemberTypeFarmer         MemberType = "farmer"
	MemberTypeMunicipality   MemberType = "municipality"
	MemberTypeCompany        MemberType = "company"
	MemberTypeAssociation    MemberType = "association"
)

// MeterDirection represents the direction of a metering point
type MeterDirection string

const (
	DirectionConsumption MeterDirection = "CONSUMPTION"
	DirectionProduction  MeterDirection = "PRODUCTION"
)

// Application represents the application entity
type Application struct {
	ID                   uuid.UUID         `json:"id" db:"id"`
	ReferenceNumber      string            `json:"referenceNumber" db:"reference_number"`
	RCNumber             string            `json:"rcNumber"        db:"rc_number"`
	Status               ApplicationStatus `json:"status" db:"status"`
	StartedAt            *time.Time        `json:"startedAt,omitempty" db:"started_at"`
	SubmittedAt          *time.Time        `json:"submittedAt,omitempty" db:"submitted_at"`
	ApprovedAt           *time.Time        `json:"approvedAt,omitempty" db:"approved_at"`
	RejectedAt           *time.Time        `json:"rejectedAt,omitempty" db:"rejected_at"`
	ImportedAt           *time.Time        `json:"importedAt,omitempty" db:"imported_at"`
	// PROJ-46: Audit-Timestamps für die Stati nach dem Import.
	// BankConfirmedAt wird gesetzt beim Übergang awaiting_bank_confirmation →
	// ready_for_activation (Admin manuell). ActivatedAt beim Übergang
	// ready_for_activation → activated (Admin manuell oder Activation-Check).
	BankConfirmedAt      *time.Time        `json:"bankConfirmedAt,omitempty" db:"bank_confirmed_at"`
	ActivatedAt          *time.Time        `json:"activatedAt,omitempty"     db:"activated_at"`
	// PROJ-53: Zeitpunkt des Versands der Beitrittsbestätigungs-Mail
	// (Move von 'imported' auf 'activated'). NULL = noch nicht versandt.
	// Verhindert doppelten Versand bei mehrfachem Statuswechsel und
	// markiert Bestandsanträge per Migration als "schon versandt".
	ActivationNotificationSentAt *time.Time `json:"activationNotificationSentAt,omitempty" db:"activation_notification_sent_at"`
	MemberType           MemberType        `json:"memberType" db:"member_type"`
	Titel                *string           `json:"titel,omitempty" db:"titel"`
	TitelNach            *string           `json:"titelNach,omitempty" db:"titel_nach"`
	Firstname            *string           `json:"firstname,omitempty" db:"firstname"`
	Lastname             *string           `json:"lastname,omitempty" db:"lastname"`
	BirthDate            *time.Time        `json:"birthDate,omitempty" db:"birth_date"`
	CompanyName          *string           `json:"companyName,omitempty" db:"company_name"`
	UIDNumber            *string           `json:"uidNumber,omitempty" db:"uid_number"`
	RegisterNumber       *string           `json:"registerNumber,omitempty" db:"register_number"`
	Email                string            `json:"email" db:"email"`
	Phone                *string           `json:"phone,omitempty" db:"phone"`
	ResidentStreet       string            `json:"residentStreet" db:"resident_street"`
	ResidentStreetNumber string            `json:"residentStreetNumber" db:"resident_street_number"`
	ResidentZip          string            `json:"residentZip" db:"resident_zip"`
	ResidentCity         string            `json:"residentCity" db:"resident_city"`
	PrivacyAccepted      bool              `json:"privacyAccepted" db:"privacy_accepted"`
	PrivacyVersion       *string           `json:"privacyVersion,omitempty" db:"privacy_version"`
	PrivacyAcceptedAt    *time.Time        `json:"privacyAcceptedAt,omitempty" db:"privacy_accepted_at"`
	AccuracyConfirmed    bool              `json:"accuracyConfirmed" db:"accuracy_confirmed"`
	IBAN                    *string           `json:"iban,omitempty" db:"iban"`
	AccountHolder           *string           `json:"accountHolder,omitempty" db:"account_holder"`
	SepaMandateAccepted     bool              `json:"sepaMandateAccepted" db:"sepa_mandate_accepted"`
	SepaMandateAcceptedAt   *time.Time        `json:"sepaMandateAcceptedAt,omitempty" db:"sepa_mandate_accepted_at"`
	ReviewedByUserID        *string           `json:"reviewedByUserId,omitempty" db:"reviewed_by_user_id"`
	AdminNote            *string           `json:"adminNote,omitempty" db:"admin_note"`
	Einzugsart           string            `json:"einzugsart" db:"einzugsart"`
	BankName             *string           `json:"bankName,omitempty" db:"bank_name"`
	MandateReference     *string           `json:"mandateReference,omitempty" db:"mandate_reference"`
	MandateDate          *time.Time        `json:"mandateDate,omitempty" db:"mandate_date"`
	NeedsInfoReason      *string           `json:"needsInfoReason,omitempty" db:"needs_info_reason"`
	TargetParticipantID  *string           `json:"targetParticipantId,omitempty" db:"target_participant_id"`
	ImportStartedAt      *time.Time        `json:"importStartedAt,omitempty" db:"import_started_at"`
	ImportFinishedAt     *time.Time        `json:"importFinishedAt,omitempty" db:"import_finished_at"`
	ImportErrorMessage   *string           `json:"importErrorMessage,omitempty" db:"import_error_message"`
	CreatedAt            time.Time         `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time         `json:"updatedAt" db:"updated_at"`
	// Configurable application-level fields (PROJ-8). Note: PROJ-49 moved
	// consumption/feed_in/pv_power fields to metering_point — they now live
	// on MeteringPoint, not here.
	MembershipStartDate     *time.Time `json:"membershipStartDate,omitempty" db:"membership_start_date"`
	PersonsInHousehold      *int       `json:"personsInHousehold,omitempty" db:"persons_in_household"`
	HeatPump                *bool      `json:"heatPump,omitempty" db:"heat_pump"`
	ElectricVehicle         *bool      `json:"electricVehicle,omitempty" db:"electric_vehicle"`
	// PROJ-42: Detail-Erfassung — nur relevant wenn ElectricVehicle == true.
	// Service-Layer clearet beide auf NULL falls electric_vehicle nicht gesetzt.
	ElectricVehicleCount    *int       `json:"electricVehicleCount,omitempty" db:"electric_vehicle_count"`
	ElectricVehicleAnnualKm *int       `json:"electricVehicleAnnualKm,omitempty" db:"electric_vehicle_annual_km"`
	ElectricHotWater        *bool      `json:"electricHotWater,omitempty" db:"electric_hot_water"`
	MemberNumber            *string    `json:"memberNumber,omitempty" db:"member_number"`
	// PROJ-37: Anzahl gezeichneter Genossenschaftsanteile. NULL bei
	// EEGs ohne aktivierte Anteils-Erfassung; sonst > 0 (Submit-Validierung
	// erzwingt >= entrypoint.cooperative_required_shares).
	CooperativeSharesCount  *int       `json:"cooperativeSharesCount,omitempty" db:"cooperative_shares_count"`
	// PROJ-44: Netzbetreiber-Vollmacht. Default FALSE; per-EEG via
	// field_config konfigurierbar (Standard hidden). `_at` wird vom
	// Service auf NOW() gesetzt, wenn der Wert von FALSE auf TRUE wechselt.
	NetworkOperatorAuthorization   bool       `json:"networkOperatorAuthorization" db:"network_operator_authorization"`
	NetworkOperatorAuthorizationAt *time.Time `json:"networkOperatorAuthorizationAt,omitempty" db:"network_operator_authorization_at"`
	// PROJ-56: Zwei optionale Netzbetreiber-Info-Felder, die im Public-
	// Formular sichtbar werden, sobald die Vollmacht-Checkbox aktiv ist.
	// Werden serverseitig auf NULL geclearted, wenn die Vollmacht nicht
	// erteilt wurde — egal was der Frontend-Submit sendet.
	NetworkOperatorCustomerNumber *string `json:"networkOperatorCustomerNumber,omitempty" db:"network_operator_customer_number"`
	MeterInventoryNumber          *string `json:"meterInventoryNumber,omitempty"          db:"meter_inventory_number"`
	// PROJ-57: Ansprechperson für Org-Mitgliedstypen (company, association,
	// municipality). Toggle + drei TEXT-Felder. Service-Layer cleart die
	// drei Felder auf NULL, wenn HasContactPerson=false oder der Mitgliedstyp
	// nicht in der Org-Liste liegt.
	HasContactPerson    bool    `json:"hasContactPerson"               db:"has_contact_person"`
	ContactPersonName   *string `json:"contactPersonName,omitempty"   db:"contact_person_name"`
	ContactPersonEmail  *string `json:"contactPersonEmail,omitempty"  db:"contact_person_email"`
	ContactPersonPhone  *string `json:"contactPersonPhone,omitempty"  db:"contact_person_phone"`
	// PROJ-58: Abweichende Rechnungs-E-Mail für Org-Mitgliedstypen.
	// Toggle + Email. Service-Layer cleart auf NULL, wenn Toggle aus
	// oder Mitgliedstyp nicht in der Org-Liste.
	HasBillingEmail bool    `json:"hasBillingEmail"            db:"has_billing_email"`
	BillingEmail    *string `json:"billingEmail,omitempty"     db:"billing_email"`
	// E-Mail-Bestätigung (PROJ-31). Token-Hash + Expiry sind interne Felder
	// und werden nicht in API-Responses serialisiert (JSON-Tag "-").
	EmailConfirmedAt                 *time.Time `json:"emailConfirmedAt,omitempty"      db:"email_confirmed_at"`
	EmailConfirmationUsedAt          *time.Time `json:"-"                               db:"email_confirmation_used_at"`
	EmailConfirmationTokenHash       *string    `json:"-"                               db:"email_confirmation_token_hash"`
	EmailConfirmationTokenExpiresAt  *time.Time `json:"-"                               db:"email_confirmation_token_expires_at"`
	// EmailConfirmationPending is a derived, non-persistent flag set by the
	// repository after load: true when a confirmation token is still active
	// and the member has not yet clicked it. Lets the admin UI render the
	// "⏳ unbestätigt" badge without re-deriving from internal columns.
	EmailConfirmationPending bool `json:"emailConfirmationPending,omitempty" db:"-"`
}

// MeteringPoint represents a metering point entity
type MeteringPoint struct {
	ID                  uuid.UUID      `json:"id" db:"id"`
	ApplicationID       uuid.UUID      `json:"applicationId" db:"application_id"`
	MeteringPoint       string         `json:"meteringPoint" db:"metering_point"`
	Direction           MeterDirection `json:"direction" db:"direction"`
	ParticipationFactor int            `json:"participationFactor" db:"participation_factor"`
	CreatedAt           time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt           time.Time      `json:"updatedAt" db:"updated_at"`
	// Configurable metering-point-level fields (PROJ-8)
	Transformer        *string `json:"transformer,omitempty" db:"transformer"`
	InstallationNumber *string `json:"installationNumber,omitempty" db:"installation_number"`
	InstallationName   *string `json:"installationName,omitempty" db:"installation_name"`
	// Abweichende Adresse je Zählpunkt (PROJ-39). Wenn alle vier NULL,
	// gilt die Mitgliederadresse. Wenn ≥1 gesetzt, müssen alle vier
	// gesetzt sein (Validierung im Service-Layer).
	AddressStreet       *string `json:"addressStreet,omitempty" db:"address_street"`
	AddressStreetNumber *string `json:"addressStreetNumber,omitempty" db:"address_street_number"`
	AddressZip          *string `json:"addressZip,omitempty" db:"address_zip"`
	AddressCity         *string `json:"addressCity,omitempty" db:"address_city"`
	// PROJ-45: Erzeugungsform + Batterie. GenerationType ist Pflicht für
	// PRODUCTION (DB-Check), NULL für CONSUMPTION. BatterySizeKwh +
	// InverterManufacturer sind optional und nur befüllt wenn
	// GenerationType='pv' — Service-Layer cleart sonst.
	GenerationType       *string  `json:"generationType,omitempty" db:"generation_type"`
	BatterySizeKwh       *float64 `json:"batterySizeKwh,omitempty" db:"battery_size_kwh"`
	InverterManufacturer *string  `json:"inverterManufacturer,omitempty" db:"inverter_manufacturer"`
	// InverterPowerKw (Migration 000046): Nennleistung des PV-Wechselrichters
	// in kW. Nur sinnvoll bei PRODUCTION + GenerationType='pv'; Service-Layer
	// cleart sonst. Per PROJ-8 konfigurierbar (knownConfigurableFields).
	InverterPowerKw *float64 `json:"inverterPowerKw,omitempty" db:"inverter_power_kw"`
	// PROJ-49: Energie-Felder pro Zählpunkt. Service-Layer-Regeln:
	//   - ConsumptionPreviousYear / ConsumptionForecast: nur CONSUMPTION,
	//     sonst auf NULL gesetzt.
	//   - FeedInForecast: nur PRODUCTION, sonst NULL.
	//   - PvPowerKwp / FeedInLimitPresent / FeedInLimitKw: nur PRODUCTION
	//     mit GenerationType='pv', sonst NULL.
	//   - FeedInLimitKw nur wenn FeedInLimitPresent=true, sonst NULL.
	ConsumptionPreviousYear *int64   `json:"consumptionPreviousYear,omitempty" db:"consumption_previous_year"`
	ConsumptionForecast     *int64   `json:"consumptionForecast,omitempty" db:"consumption_forecast"`
	FeedInForecast          *int64   `json:"feedInForecast,omitempty" db:"feed_in_forecast"`
	PvPowerKwp              *float64 `json:"pvPowerKwp,omitempty" db:"pv_power_kwp"`
	FeedInLimitPresent      *bool    `json:"feedInLimitPresent,omitempty" db:"feed_in_limit_present"`
	FeedInLimitKw           *float64 `json:"feedInLimitKw,omitempty" db:"feed_in_limit_kw"`
	// PROJ-49 follow-up: Mitglied-Antwort auf „Speichersteuerung über die
	// EEG vorstellbar?". Nur sinnvoll bei PRODUCTION + generation_type='pv'
	// UND wenn das Mitglied Batterie-Parameter angegeben hat (Service cleart
	// in allen anderen Fällen).
	BatteryControlAcceptable *bool `json:"batteryControlAcceptable,omitempty" db:"battery_control_acceptable"`
}

// HasDeviatingAddress returns true if this metering point has a different
// address from the member's primary residence. Helper used by mail/PDF
// rendering to decide whether to print the address line.
func (mp *MeteringPoint) HasDeviatingAddress() bool {
	return mp.AddressStreet != nil && *mp.AddressStreet != ""
}

// LegalDocument is a legal document configured per EEG.
type LegalDocument struct {
	ID        uuid.UUID `json:"id"        db:"id"`
	RCNumber  string    `json:"rcNumber"  db:"rc_number"`
	Title     string    `json:"title"     db:"title"`
	URL       string    `json:"url"       db:"url"`
	Required  bool      `json:"required"  db:"required"`
	SortOrder int       `json:"sortOrder" db:"sort_order"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// DocumentConsent is an immutable consent snapshot stored at application submit time.
type DocumentConsent struct {
	ID              uuid.UUID `json:"id"              db:"id"`
	ApplicationID   uuid.UUID `json:"applicationId"   db:"application_id"`
	Title           string    `json:"title"           db:"title"`
	URL             string    `json:"url"             db:"url"`
	IsCentralPolicy bool      `json:"isCentralPolicy" db:"is_central_policy"`
	ConsentedAt     time.Time `json:"consentedAt"     db:"consented_at"`
	// ConsentType (PROJ-36) is `explicit` for documents the member actively
	// checked at submit, or `informational` for documents that were merely
	// shown on the form as info-only (no checkbox). Pre-PROJ-36 entries are
	// all `explicit` via the DB column default.
	ConsentType ConsentType `json:"consentType" db:"consent_type"`
}

// ConsentType distinguishes active acceptance from informational
// acknowledgement (PROJ-36).
type ConsentType string

const (
	ConsentTypeExplicit      ConsentType = "explicit"
	ConsentTypeInformational ConsentType = "informational"
)

// StatusLogEntry represents a status log entry
type StatusLogEntry struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	ApplicationID    uuid.UUID  `json:"applicationId" db:"application_id"`
	FromStatus       *string    `json:"fromStatus,omitempty" db:"from_status"`
	ToStatus         string     `json:"toStatus" db:"to_status"`
	ChangedByUserID  *string    `json:"changedByUserId,omitempty" db:"changed_by_user_id"`
	Reason           *string    `json:"reason,omitempty" db:"reason"`
	CreatedAt        time.Time  `json:"createdAt" db:"created_at"`
}