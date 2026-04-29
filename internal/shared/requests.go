package shared

import (
	"time"

	"github.com/google/uuid"
)

// Request models

// CreateApplicationRequest represents the request to create a new application
type CreateApplicationRequest struct {
	RCNumber             string                      `json:"rcNumber" validate:"required"`
	MemberType           string                      `json:"memberType" validate:"required,oneof=private farmer municipality company association"`
	Titel                *string                     `json:"titel,omitempty" validate:"omitempty,max=50"`
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
	SepaMandateAccepted  bool                        `json:"sepaMandateAccepted" validate:"required"`
	MeteringPoints       []CreateMeteringPointRequest `json:"meteringPoints" validate:"required,min=1,max=10,dive"`
	// Configurable application-level fields (PROJ-8)
	MembershipStartDate     *string  `json:"membershipStartDate,omitempty" validate:"omitempty,len=10"`
	PersonsInHousehold      *int     `json:"personsInHousehold,omitempty" validate:"omitempty,min=0"`
	ConsumptionPreviousYear *int64   `json:"consumptionPreviousYear,omitempty" validate:"omitempty,min=0"`
	ConsumptionForecast     *int64   `json:"consumptionForecast,omitempty" validate:"omitempty,min=0"`
	FeedInForecast          *int64   `json:"feedInForecast,omitempty" validate:"omitempty,min=0"`
	PvPowerKwp              *float64 `json:"pvPowerKwp,omitempty" validate:"omitempty,min=0"`
	HeatPump                *bool    `json:"heatPump,omitempty"`
	ElectricVehicle         *bool    `json:"electricVehicle,omitempty"`
	ElectricHotWater        *bool    `json:"electricHotWater,omitempty"`
	// Cloudflare Turnstile token (PROJ-16) — optional, verified server-side when TURNSTILE_SECRET_KEY is set
	TurnstileToken *string `json:"turnstileToken,omitempty"`
}

// CreateMeteringPointRequest represents a metering point in create request
type CreateMeteringPointRequest struct {
	MeteringPoint       string  `json:"meteringPoint" validate:"required,max=33"`
	Direction           string  `json:"direction" validate:"required,oneof=CONSUMPTION PRODUCTION"`
	ParticipationFactor int     `json:"participationFactor" validate:"required,min=1,max=100"`
	// Configurable metering-point-level fields (PROJ-8)
	Transformer        *string `json:"transformer,omitempty" validate:"omitempty,max=100"`
	InstallationNumber *string `json:"installationNumber,omitempty" validate:"omitempty,max=50"`
	InstallationName   *string `json:"installationName,omitempty" validate:"omitempty,max=100"`
}

// UpdateApplicationRequest represents the request to update an application
type UpdateApplicationRequest struct {
	MemberType           *string                     `json:"memberType,omitempty" validate:"omitempty,oneof=private farmer municipality company association"`
	Titel                *string                     `json:"titel,omitempty" validate:"omitempty,max=50"`
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
	SepaMandateAccepted  *bool                       `json:"sepaMandateAccepted,omitempty"`
	MeteringPoints       []CreateMeteringPointRequest `json:"meteringPoints,omitempty" validate:"omitempty,min=1,max=10,dive"`
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
	ShowCentralPolicy  bool                `json:"showCentralPolicy"`
	LegalDocuments     []LegalDocumentItem `json:"legalDocuments"`
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

// ---------- Admin request / response models ----------

// AdminUpdateApplicationRequest is the admin partial-update payload.
// Unlike the public update it exposes AdminNote and omits consent fields
// (privacyAccepted, accuracyConfirmed, etc.) which only the public user sets.
type AdminUpdateApplicationRequest struct {
	MemberType           *string                      `json:"memberType,omitempty" validate:"omitempty,oneof=private farmer municipality company association"`
	Titel                *string                      `json:"titel,omitempty" validate:"omitempty,max=50"`
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
	MemberNumber         *int                          `json:"memberNumber,omitempty" validate:"omitempty,min=1"`
}

// ChangeStatusRequest is the admin status-transition payload.
type ChangeStatusRequest struct {
	ToStatus string `json:"toStatus" validate:"required"`
	Reason   string `json:"reason"`
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
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	URL             string    `json:"url"`
	IsCentralPolicy bool      `json:"isCentralPolicy"`
	ConsentedAt     time.Time `json:"consentedAt"`
}

// AdminApplicationDetailResponse is the full admin detail view: application
// record plus its metering points and complete status history.
type AdminApplicationDetailResponse struct {
	Application
	MeteringPoints []MeteringPoint       `json:"meteringPoints"`
	StatusLog      []StatusLogEntry      `json:"statusLog"`
	Consents       []DocumentConsentView `json:"consents"`
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
