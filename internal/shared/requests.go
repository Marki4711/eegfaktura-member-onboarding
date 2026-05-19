package shared

import (
	"time"

	"github.com/google/uuid"
)

// Request models

// CreateApplicationRequest represents the request to create a new application
type CreateApplicationRequest struct {
	RCNumber             string                      `json:"rcNumber" validate:"required"`
	MemberType           string                      `json:"memberType" validate:"required,oneof=private sole_proprietor farmer municipality company association"`
	Titel                *string                     `json:"titel,omitempty" validate:"omitempty,max=50"`
	TitelNach            *string                     `json:"titelNach,omitempty" validate:"omitempty,max=50"`
	Firstname            *string                     `json:"firstname,omitempty" validate:"omitempty,min=1,max=100"`
	Lastname             *string                     `json:"lastname,omitempty" validate:"omitempty,min=1,max=100"`
	BirthDate            *string                     `json:"birthDate,omitempty"  validate:"omitempty,len=10"`
	CompanyName          *string                     `json:"companyName,omitempty" validate:"omitempty,min=1,max=150"`
	UIDNumber            *string                     `json:"uidNumber,omitempty" validate:"omitempty,max=50"`
	RegisterNumber       *string                     `json:"registerNumber,omitempty" validate:"omitempty,max=50"`
	Email                string                      `json:"email" validate:"required,email"`
	Phone                *string                     `json:"phone,omitempty"      validate:"omitempty,max=50"`
	ResidentStreet       string                      `json:"residentStreet" validate:"required,min=1,max=100"`
	ResidentStreetNumber string                      `json:"residentStreetNumber" validate:"required,min=1,max=50"`
	ResidentZip          string                      `json:"residentZip" validate:"required,min=1,max=20"`
	ResidentCity         string                      `json:"residentCity" validate:"required,min=1,max=100"`
	PrivacyAccepted      bool                        `json:"privacyAccepted" validate:"required"`
	PrivacyVersion       string                      `json:"privacyVersion" validate:"required"`
	AccuracyConfirmed    bool                        `json:"accuracyConfirmed" validate:"required"`
	IBAN                 string                      `json:"iban" validate:"required,min=15,max=50"`
	AccountHolder        string                      `json:"accountHolder" validate:"required,min=1,max=150"`
	BankName             *string                     `json:"bankName,omitempty" validate:"omitempty,max=255"`
	SepaMandateAccepted  bool                        `json:"sepaMandateAccepted" validate:"required"`
	MeteringPoints       []CreateMeteringPointRequest `json:"meteringPoints" validate:"required,min=1,max=10,dive"`
	// Configurable application-level fields (PROJ-8). PROJ-49 moved
	// consumption/feed_in/pv_power fields to MeteringPoint sub-struct.
	MembershipStartDate     *string  `json:"membershipStartDate,omitempty" validate:"omitempty,len=10"`
	PersonsInHousehold      *int     `json:"personsInHousehold,omitempty" validate:"omitempty,min=0"`
	HeatPump                *bool    `json:"heatPump,omitempty"`
	ElectricVehicle         *bool    `json:"electricVehicle,omitempty"`
	// PROJ-42: Details zur E-Fahrzeug-Erfassung. Werden serverseitig
	// auf NULL gesetzt wenn electric_vehicle != true.
	ElectricVehicleCount    *int     `json:"electricVehicleCount,omitempty" validate:"omitempty,min=1"`
	ElectricVehicleAnnualKm *int     `json:"electricVehicleAnnualKm,omitempty" validate:"omitempty,min=0"`
	ElectricHotWater        *bool    `json:"electricHotWater,omitempty"`
	// PROJ-37: Anzahl der gezeichneten Genossenschaftsanteile. Pflicht bei
	// EEGs mit aktivierter Anteils-Erfassung, sonst optional (wird dann
	// serverseitig ignoriert).
	CooperativeSharesCount *int `json:"cooperativeSharesCount,omitempty" validate:"omitempty,min=1"`
	// PROJ-44: Netzbetreiber-Vollmacht. Nur relevant wenn EEG das Feld als
	// optional/required konfiguriert hat.
	NetworkOperatorAuthorization *bool `json:"networkOperatorAuthorization,omitempty"`
	// Cloudflare Turnstile token (PROJ-16) — optional, verified server-side when TURNSTILE_SECRET_KEY is set
	TurnstileToken *string `json:"turnstileToken,omitempty"`
}

// CreateMeteringPointRequest represents a metering point in create request
type CreateMeteringPointRequest struct {
	MeteringPoint       string  `json:"meteringPoint" validate:"required,len=33,startswith=AT"`
	Direction           string  `json:"direction" validate:"required,oneof=CONSUMPTION PRODUCTION"`
	// ParticipationFactor (Teilnahmefaktor in Prozent).
	// Seit der PROJ-8-Erweiterung am 2026-05-19 ist das Feld per EEG via
	// `field_config` ein-/ausblendbar. Wenn das Frontend es ausblendet
	// oder das Mitglied es nicht angegeben hat, kommt der Wert als `0`
	// hier an — der Service-Layer setzt dann Default 100. Validate
	// `min=0,max=100` lässt 0 explizit zu (Service normalisiert).
	ParticipationFactor int     `json:"participationFactor" validate:"min=0,max=100"`
	// Configurable metering-point-level fields (PROJ-8)
	Transformer        *string `json:"transformer,omitempty" validate:"omitempty,max=100"`
	InstallationNumber *string `json:"installationNumber,omitempty" validate:"omitempty,max=50"`
	InstallationName   *string `json:"installationName,omitempty" validate:"omitempty,max=100"`
	// Abweichende Adresse je Zählpunkt (PROJ-39). Entweder alle vier
	// leer/NULL → Mitgliederadresse gilt, oder alle vier gesetzt →
	// abweichende Adresse. Service-Layer prüft die All-or-Nothing-Regel.
	AddressStreet       *string `json:"addressStreet,omitempty" validate:"omitempty,max=255"`
	AddressStreetNumber *string `json:"addressStreetNumber,omitempty" validate:"omitempty,max=50"`
	AddressZip          *string `json:"addressZip,omitempty" validate:"omitempty,max=20"`
	AddressCity         *string `json:"addressCity,omitempty" validate:"omitempty,max=255"`
	// PROJ-45: Erzeugungsform + Batterie. GenerationType wird vom Service
	// auf "pv" defaultet, wenn Direction=PRODUCTION und leer übermittelt.
	GenerationType       *string  `json:"generationType,omitempty" validate:"omitempty,oneof=pv hydro wind biomass"`
	BatterySizeKwh       *float64 `json:"batterySizeKwh,omitempty" validate:"omitempty,min=0"`
	InverterManufacturer *string  `json:"inverterManufacturer,omitempty" validate:"omitempty,max=100"`
	// Migration 000046: Nennleistung PV-Wechselrichter in kW. Nur sinnvoll
	// bei PRODUCTION + GenerationType='pv'; Service-Layer cleart sonst.
	InverterPowerKw *float64 `json:"inverterPowerKw,omitempty" validate:"omitempty,min=0"`
	// PROJ-49: Energie-Felder pro Zählpunkt. Sichtbarkeit + Validierung
	// per Direction/GenerationType im Service-Layer (siehe MeteringPoint).
	ConsumptionPreviousYear *int64   `json:"consumptionPreviousYear,omitempty" validate:"omitempty,min=0"`
	ConsumptionForecast     *int64   `json:"consumptionForecast,omitempty"     validate:"omitempty,min=0"`
	FeedInForecast          *int64   `json:"feedInForecast,omitempty"          validate:"omitempty,min=0"`
	PvPowerKwp              *float64 `json:"pvPowerKwp,omitempty"              validate:"omitempty,min=0"`
	FeedInLimitPresent      *bool    `json:"feedInLimitPresent,omitempty"`
	FeedInLimitKw           *float64 `json:"feedInLimitKw,omitempty"           validate:"omitempty,min=0"`
	// PROJ-49 follow-up: „Speichersteuerung über die EEG vorstellbar?"
	// Nur sinnvoll bei PV + vorhandenem Speicher (Service cleart sonst).
	BatteryControlAcceptable *bool `json:"batteryControlAcceptable,omitempty"`
}

// UpdateApplicationRequest represents the request to update an application
type UpdateApplicationRequest struct {
	MemberType           *string                     `json:"memberType,omitempty" validate:"omitempty,oneof=private sole_proprietor farmer municipality company association"`
	Titel                *string                     `json:"titel,omitempty" validate:"omitempty,max=50"`
	TitelNach            *string                     `json:"titelNach,omitempty" validate:"omitempty,max=50"`
	Firstname            *string                     `json:"firstname,omitempty" validate:"omitempty,min=1,max=100"`
	Lastname             *string                     `json:"lastname,omitempty" validate:"omitempty,min=1,max=100"`
	BirthDate            *string                     `json:"birthDate,omitempty"  validate:"omitempty,len=10"`
	CompanyName          *string                     `json:"companyName,omitempty" validate:"omitempty,min=1,max=150"`
	UIDNumber            *string                     `json:"uidNumber,omitempty" validate:"omitempty,max=50"`
	RegisterNumber       *string                     `json:"registerNumber,omitempty" validate:"omitempty,max=50"`
	Email                *string                     `json:"email,omitempty" validate:"omitempty,email"`
	Phone                *string                     `json:"phone,omitempty"      validate:"omitempty,max=50"`
	ResidentStreet       *string                     `json:"residentStreet,omitempty" validate:"omitempty,min=1,max=100"`
	ResidentStreetNumber *string                     `json:"residentStreetNumber,omitempty" validate:"omitempty,min=1,max=50"`
	ResidentZip          *string                     `json:"residentZip,omitempty" validate:"omitempty,min=1,max=20"`
	ResidentCity         *string                     `json:"residentCity,omitempty" validate:"omitempty,min=1,max=100"`
	PrivacyAccepted      *bool                       `json:"privacyAccepted,omitempty"`
	PrivacyVersion       *string                     `json:"privacyVersion,omitempty"`
	AccuracyConfirmed    *bool                       `json:"accuracyConfirmed,omitempty"`
	IBAN                 *string                     `json:"iban,omitempty" validate:"omitempty,min=15,max=50"`
	AccountHolder        *string                     `json:"accountHolder,omitempty" validate:"omitempty,min=1,max=150"`
	BankName             *string                     `json:"bankName,omitempty" validate:"omitempty,max=255"`
	SepaMandateAccepted  *bool                       `json:"sepaMandateAccepted,omitempty"`
	MeteringPoints       []CreateMeteringPointRequest `json:"meteringPoints,omitempty" validate:"omitempty,min=1,max=10,dive"`
	// PROJ-37: Admin/Member-Updates der gezeichneten Anteils-Anzahl. Bei
	// EEGs ohne Anteils-Erfassung serverseitig ignoriert.
	CooperativeSharesCount *int `json:"cooperativeSharesCount,omitempty" validate:"omitempty,min=1"`
	// PROJ-44: Netzbetreiber-Vollmacht (Update durch Member im needs_info-Flow).
	NetworkOperatorAuthorization *bool `json:"networkOperatorAuthorization,omitempty"`
}

// Response models

// LegalDocumentItem is a single legal document as returned in the public registration config.
type LegalDocumentItem struct {
	ID               uuid.UUID `json:"id"`
	Title            string    `json:"title"`
	URL              string    `json:"url"`
	Required         bool      `json:"required"`
	SortOrder        int       `json:"sortOrder"`
	IsCentralPolicy  bool      `json:"isCentralPolicy"`
}

// ConsentInput is one consent entry sent by the frontend at submit time.
// All entries from the frontend represent an active checkbox tick by the
// member and are treated as `explicit`. Informational consents for
// non-required documents are written server-side from legal_document and
// are not sent in this payload.
type ConsentInput struct {
	Title           string `json:"title"`
	URL             string `json:"url"`
	IsCentralPolicy bool   `json:"isCentralPolicy"`
}

// SubmitRequest is the optional body sent with POST /api/public/applications/{id}/submit.
type SubmitRequest struct {
	Consents []ConsentInput `json:"consents"`
}

// RegistrationConfig represents the response for the registration entry point endpoint
type RegistrationConfig struct {
	RCNumber           string              `json:"rcNumber"`
	Title              string              `json:"title"`
	Active             bool                `json:"active"`
	FieldConfig        map[string]string   `json:"fieldConfig"`
	IntroText          *string             `json:"introText"`
	SEPAMandateEnabled bool                `json:"sepaMandateEnabled"`
	// SEPAMandateAtImport (PROJ-48) wird an die Public-Form weitergereicht,
	// damit der Hinweistext „Mandat kommt jetzt vs. nach Import" korrekt
	// gerendert werden kann. False = heutiges Verhalten (Mandat als Anhang
	// in der Eingangsbestätigung); True = Mandat kommt erst beim Import
	// mit Mitgliedsnummer als Mandatsreferenz.
	SEPAMandateAtImport bool               `json:"sepaMandateAtImport"`
	ShowCentralPolicy  bool                `json:"showCentralPolicy"`
	// RequireEmailConfirmation (PROJ-31) wird in die Public-Form
	// weitergereicht, damit die Erfolgsmeldung nach dem Einreichen den
	// richtigen Hinweis zeigt („Bitte E-Mail bestätigen" statt
	// „wird nun von unserem Team geprüft").
	RequireEmailConfirmation bool          `json:"requireEmailConfirmation"`
	// PROJ-52: pro Richtung konfigurierbarer Zählpunkt-Prefix. NULL =
	// keine EEG-spezifische Vorbelegung (Mask zeigt nur "AT" als fix).
	// Wenn gesetzt, baut die Public-Form die Zählpunkt-Mask dynamisch
	// je nach gewählter Richtung; Service-Layer validiert Match beim Submit.
	MeteringPointPrefixConsumption *string `json:"meteringPointPrefixConsumption,omitempty"`
	MeteringPointPrefixProduction  *string `json:"meteringPointPrefixProduction,omitempty"`
	LegalDocuments     []LegalDocumentItem `json:"legalDocuments"`
	// PROJ-37: only set when CooperativeSharesEnabled=true on the EEG.
	// Both inner values are then non-nil and > 0.
	CooperativeSharesEnabled    bool   `json:"cooperativeSharesEnabled"`
	CooperativeRequiredShares   *int   `json:"cooperativeRequiredShares,omitempty"`
	CooperativeShareAmountCents *int64 `json:"cooperativeShareAmountCents,omitempty"`
}

// ApplicationResponse represents the response for application operations
type ApplicationResponse struct {
	ID             uuid.UUID `json:"id"`
	ReferenceNumber string    `json:"referenceNumber"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// SubmitResponse represents the response for submit operation
type SubmitResponse struct {
	ID              uuid.UUID         `json:"id"`
	ReferenceNumber string            `json:"referenceNumber"`
	Status          ApplicationStatus `json:"status"`
	SubmittedAt     time.Time         `json:"submittedAt"`
}

// EEGSettingsFieldDiff describes one field where the local Onboarding value
// differs from what the eegFaktura core currently has. Both string-typed
// values are pre-dereffed (empty string for NULL) so the frontend can render
// them as-is. Field is a stable identifier (`eegName`, `creditorId`, …);
// Label is the German human-readable label for the admin UI banner.
type EEGSettingsFieldDiff struct {
	Field      string `json:"field"`
	Label      string `json:"label"`
	LocalValue string `json:"localValue"`
	CoreValue  string `json:"coreValue"`
}

// EEGSettingsComparisonResponse is what GET /core-comparison and
// POST /sync both return — same shape, so the frontend can use one render
// path. inSync==true ⇒ differingFields is empty.
//
// coreReachable=false signals the comparison couldn't be performed (Core
// down / not configured); the frontend then renders a neutral "Core nicht
// erreichbar — letzter Stand: lastSyncedAt" banner instead of either a
// drift warning or a synchron-OK.
type EEGSettingsComparisonResponse struct {
	CoreReachable        bool                   `json:"coreReachable"`
	CoreUnreachableError string                 `json:"coreUnreachableError,omitempty"`
	InSync               bool                   `json:"inSync"`
	DifferingFields      []EEGSettingsFieldDiff `json:"differingFields,omitempty"`
	LastSyncedAt         *time.Time             `json:"lastSyncedAt,omitempty"`
	// LogoSyncWarning is set by POST /sync when the master-data sync
	// succeeded but the follow-up logo fetch did not — e.g. the core's
	// logo is larger than the 256 KB cap, or has an unsupported MIME.
	// Empty on the comparison endpoint and on a clean sync.
	LogoSyncWarning string `json:"logoSyncWarning,omitempty"`
	// LogoSyncedAt mirrors registration_entrypoint.eeg_logo_synced_at —
	// the admin UI uses it to render a "Logo zuletzt synchronisiert"
	// label next to the preview. NULL until the first successful logo
	// sync.
	LogoSyncedAt *time.Time `json:"logoSyncedAt,omitempty"`
}

// ConfirmEmailRequest carries the plaintext token sent in the confirmation
// e-mail. Used by POST /api/public/applications/confirm-email (PROJ-31).
type ConfirmEmailRequest struct {
	Token string `json:"token" validate:"required,min=10,max=200"`
}

// ConfirmEmailResponse is intentionally minimal — it carries only what the
// success page needs to render. Notably it does NOT echo back the application
// id, the member's name, e-mail, or any other PII.
type ConfirmEmailResponse struct {
	EEGName             string `json:"eegName,omitempty"`
	EEGContactEmail     string `json:"eegContactEmail,omitempty"`
	AlreadyConfirmed    bool   `json:"alreadyConfirmed,omitempty"`
}

// ---------- Admin request / response models ----------

// AdminUpdateApplicationRequest is the admin partial-update payload.
// Unlike the public update it exposes AdminNote and omits consent fields
// (privacyAccepted, accuracyConfirmed, etc.) which only the public user sets.
type AdminUpdateApplicationRequest struct {
	MemberType           *string                      `json:"memberType,omitempty" validate:"omitempty,oneof=private sole_proprietor farmer municipality company association"`
	Titel                *string                      `json:"titel,omitempty" validate:"omitempty,max=50"`
	TitelNach            *string                      `json:"titelNach,omitempty" validate:"omitempty,max=50"`
	Firstname            *string                      `json:"firstname,omitempty" validate:"omitempty,min=1,max=100"`
	Lastname             *string                      `json:"lastname,omitempty" validate:"omitempty,min=1,max=100"`
	BirthDate            *string                      `json:"birthDate,omitempty"  validate:"omitempty,len=10"`
	CompanyName          *string                      `json:"companyName,omitempty" validate:"omitempty,min=1,max=150"`
	UIDNumber            *string                      `json:"uidNumber,omitempty" validate:"omitempty,max=50"`
	RegisterNumber       *string                      `json:"registerNumber,omitempty" validate:"omitempty,max=50"`
	Email                *string                      `json:"email,omitempty" validate:"omitempty,email"`
	Phone                *string                      `json:"phone,omitempty"      validate:"omitempty,max=50"`
	ResidentStreet       *string                      `json:"residentStreet,omitempty" validate:"omitempty,min=1,max=100"`
	ResidentStreetNumber *string                      `json:"residentStreetNumber,omitempty" validate:"omitempty,min=1,max=50"`
	ResidentZip          *string                      `json:"residentZip,omitempty" validate:"omitempty,min=1,max=20"`
	ResidentCity         *string                      `json:"residentCity,omitempty" validate:"omitempty,min=1,max=100"`
	AdminNote            *string                      `json:"adminNote,omitempty"`
	IBAN                 *string                      `json:"iban,omitempty" validate:"omitempty,min=15,max=50"`
	AccountHolder        *string                      `json:"accountHolder,omitempty" validate:"omitempty,min=1,max=150"`
	Einzugsart           *string                      `json:"einzugsart,omitempty" validate:"omitempty,oneof=kein_sepa b2b core"`
	BankName             *string                      `json:"bankName,omitempty" validate:"omitempty,max=255"`
	MandateReference     *string                      `json:"mandateReference,omitempty" validate:"omitempty,max=255"`
	MandateDate          *string                      `json:"mandateDate,omitempty" validate:"omitempty,len=10"`
	MeteringPoints       []CreateMeteringPointRequest  `json:"meteringPoints,omitempty" validate:"omitempty,min=1,max=10,dive"`
	// MemberNumber is no longer admin-editable: the import dialog assigns it
	// at import time. The struct field is intentionally removed; any value
	// in the JSON body is silently ignored.
}

// ChangeStatusRequest is the admin status-transition payload.
type ChangeStatusRequest struct {
	ToStatus string `json:"toStatus" validate:"required"`
	Reason   string `json:"reason"`
}

// ResetImportRequest is the admin reset-imported-to-approved payload (PROJ-30).
// Reason is mandatory and bounded so the resulting status_log entry stays
// auditable without becoming an unbounded text field.
type ResetImportRequest struct {
	Reason string `json:"reason" validate:"required,min=5,max=500"`
}

// MarkImportedManuallyRequest is the admin orphan-recovery payload (PROJ-34).
// Used when the core created a participant but the onboarding bookkeeping
// transaction failed — the admin reads the participant UUID + member-number
// from eegFaktura and submits them here to close the loop.
type MarkImportedManuallyRequest struct {
	TargetParticipantID string `json:"targetParticipantId" validate:"required,uuid"`
	MemberNumber        string `json:"memberNumber"        validate:"required,min=1,max=50"`
	Reason              string `json:"reason"              validate:"max=500"`
}

// ClearImportLockRequest is the admin "give up — retry" payload (PROJ-34).
// Reason is mandatory because clearing the lock can lead to a duplicate
// participant in the core; we want the operator's intent in the audit log.
type ClearImportLockRequest struct {
	Reason string `json:"reason" validate:"required,min=5,max=500"`
}

// MarkActivatedRequest (PROJ-53) is the admin's manual-skip payload for
// the rare case where the member already exists in the eegFaktura core
// (Faktura cannot delete members) and was manually overwritten there with
// the onboarding data. The admin supplies the Mitgliedsnummer; the import
// path is skipped entirely.
type MarkActivatedRequest struct {
	MemberNumber string `json:"memberNumber" validate:"required,min=1,max=50"`
}

// ReassignEEGRequest is the admin EEG-reassign payload (PROJ-40).
// targetRcNumber must be an existing, active EEG entrypoint; the admin
// must be authorized for both source and target (or be a superuser).
type ReassignEEGRequest struct {
	TargetRCNumber string `json:"targetRcNumber" validate:"required,min=1,max=50"`
	Reason         string `json:"reason"         validate:"required,min=5,max=500"`
}

// UpdateAdminNoteRequest is the body for PATCH /api/admin/applications/{id}/admin-note.
// Replaces only the admin_note column — never touches any other field so the
// editor cannot accidentally reset participation factors or membertype on save.
// An empty string clears the note (column becomes NULL).
type UpdateAdminNoteRequest struct {
	Note string `json:"note" validate:"max=2000"`
}

// ImportApplicationRequest is the PROJ-27 import-time payload. All fields are
// optional: empty body = legacy import (no tariffs). The TariffID applies to
// the participant (mapped via a follow-up PUT /participant/v2/{id} call); the
// MeterTariffs map (metering_point → tariff UUID) is merged into the POST
// /participant body's meters[].tariff_id.
type ImportApplicationRequest struct {
	TariffID     string            `json:"tariffId,omitempty"`
	MeterTariffs map[string]string `json:"meterTariffs,omitempty"`
	// MemberNumber is required: since the onboarding stopped auto-assigning
	// numbers, the import dialog passes the admin's chosen number (pre-filled
	// from the core's pattern-aware suggestion). The backend verifies the
	// number is not already taken in the core before sending POST /participant.
	// Stored as string because the core's participantNumber column is VARCHAR
	// and may contain letters (e.g. "A005", "M-12").
	MemberNumber *string `json:"memberNumber" validate:"required,min=1,max=50"`
}

// ApplicationListItem is one summary row in the admin list response.
type ApplicationListItem struct {
	ID              uuid.UUID  `json:"id"`
	ReferenceNumber string     `json:"referenceNumber"`
	RCNumber        string     `json:"rcNumber"`
	Status          string     `json:"status"`
	MemberType      string     `json:"memberType"`
	Firstname       *string    `json:"firstname,omitempty"`
	Lastname        *string    `json:"lastname,omitempty"`
	CompanyName     *string    `json:"companyName,omitempty"`
	Email           string     `json:"email"`
	SubmittedAt     *time.Time `json:"submittedAt"`
}

// ApplicationListResponse wraps a paginated list of applications.
type ApplicationListResponse struct {
	Items    []ApplicationListItem `json:"items"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
	Total    int                   `json:"total"`
}

// DocumentConsentView is one consent entry in the admin detail response.
type DocumentConsentView struct {
	ID              uuid.UUID   `json:"id"`
	Title           string      `json:"title"`
	URL             string      `json:"url"`
	IsCentralPolicy bool        `json:"isCentralPolicy"`
	ConsentedAt     time.Time   `json:"consentedAt"`
	// ConsentType distinguishes `explicit` (member ticked a checkbox)
	// from `informational` (document was shown as info-only). PROJ-36.
	ConsentType ConsentType `json:"consentType"`
}

// AdminApplicationDetailResponse is the full admin detail view: application
// record plus its metering points and complete status history.
type AdminApplicationDetailResponse struct {
	Application
	MeteringPoints []MeteringPoint       `json:"meteringPoints"`
	StatusLog      []StatusLogEntry      `json:"statusLog"`
	Consents       []DocumentConsentView `json:"consents"`
	// ImportStuck (PROJ-34) is true when the application is in
	// status='approved' with import_started_at set > ImportStuckThreshold
	// ago and no import_finished_at. The admin UI renders the unstuck
	// banner only when this flag is true. Computed by the handler from
	// the underlying timestamps; not persisted.
	ImportStuck bool `json:"importStuck"`
	// CooperativeSharesEnabled (PROJ-37) mirrors the EEG-level toggle so
	// the admin detail can render the shares block when relevant. The
	// matching two value fields (required / amount-per-share) come from
	// the entrypoint and are joined in at detail-build time.
	CooperativeSharesEnabled    bool   `json:"cooperativeSharesEnabled,omitempty"`
	CooperativeRequiredShares   *int   `json:"cooperativeRequiredShares,omitempty"`
	CooperativeShareAmountCents *int64 `json:"cooperativeShareAmountCents,omitempty"`
}

// ImportStuckThreshold is the age past which an in-flight import is treated
// as abandoned (so the admin UI surfaces the unstuck banner). The Core call
// itself has a 2-minute hard timeout; a slot older than that is guaranteed
// not in flight anymore.
const ImportStuckThreshold = 2 * time.Minute

// IsImportStuck reports whether the given application is in the stuck state
// described by ImportStuckThreshold. Lives on shared so both the detail
// handler and the unstuck-endpoint validator can reuse it.
func IsImportStuck(app *Application, now time.Time) bool {
	if app == nil || app.Status != StatusApproved {
		return false
	}
	if app.ImportStartedAt == nil || app.ImportFinishedAt != nil {
		return false
	}
	return now.Sub(*app.ImportStartedAt) > ImportStuckThreshold
}

// ChangeStatusResponse is returned after a successful status transition.
type ChangeStatusResponse struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

// BulkActionRequest is the payload for POST /api/admin/applications/bulk-action.
// IDs must contain between 1 and 200 application UUIDs.
// For action "reject", Reason is required.
type BulkActionRequest struct {
	Action string   `json:"action" validate:"required,oneof=approve reject under_review"`
	IDs    []string `json:"ids"    validate:"required,min=1,max=200"`
	Reason string   `json:"reason" validate:"max=2000"`
}

// BulkActionResponse is returned after POST /api/admin/applications/bulk-action.
type BulkActionResponse struct {
	Succeeded []string `json:"succeeded"`
	Skipped   []string `json:"skipped"`
}
