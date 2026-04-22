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
	IsActive     bool      `json:"isActive"     db:"is_active"`
	ContactEmail *string   `json:"contactEmail" db:"contact_email"`
	CreatedAt    time.Time `json:"createdAt"    db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt"    db:"updated_at"`
}

// ApplicationStatus represents the status of an application
type ApplicationStatus string

const (
	StatusDraft       ApplicationStatus = "draft"
	StatusSubmitted   ApplicationStatus = "submitted"
	StatusUnderReview ApplicationStatus = "under_review"
	StatusNeedsInfo   ApplicationStatus = "needs_info"
	StatusApproved    ApplicationStatus = "approved"
	StatusRejected    ApplicationStatus = "rejected"
	StatusImported    ApplicationStatus = "imported"
	StatusImportFailed ApplicationStatus = "import_failed"
)

// MemberType represents the type of EEG member
type MemberType string

const (
	MemberTypePrivate      MemberType = "private"
	MemberTypeFarmer       MemberType = "farmer"
	MemberTypeMunicipality MemberType = "municipality"
	MemberTypeCompany      MemberType = "company"
	MemberTypeAssociation  MemberType = "association"
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
	MemberType           MemberType        `json:"memberType" db:"member_type"`
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
	NeedsInfoReason      *string           `json:"needsInfoReason,omitempty" db:"needs_info_reason"`
	TargetParticipantID  *string           `json:"targetParticipantId,omitempty" db:"target_participant_id"`
	ImportStartedAt      *time.Time        `json:"importStartedAt,omitempty" db:"import_started_at"`
	ImportFinishedAt     *time.Time        `json:"importFinishedAt,omitempty" db:"import_finished_at"`
	ImportErrorMessage   *string           `json:"importErrorMessage,omitempty" db:"import_error_message"`
	CreatedAt            time.Time         `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time         `json:"updatedAt" db:"updated_at"`
	// Configurable application-level fields (PROJ-8)
	MembershipStartDate     *time.Time `json:"membershipStartDate,omitempty" db:"membership_start_date"`
	PersonsInHousehold      *int       `json:"personsInHousehold,omitempty" db:"persons_in_household"`
	ConsumptionPreviousYear *int       `json:"consumptionPreviousYear,omitempty" db:"consumption_previous_year"`
	ConsumptionForecast     *int       `json:"consumptionForecast,omitempty" db:"consumption_forecast"`
	FeedInForecast          *int       `json:"feedInForecast,omitempty" db:"feed_in_forecast"`
	PvPowerKwp              *float64   `json:"pvPowerKwp,omitempty" db:"pv_power_kwp"`
	HeatPump                *bool      `json:"heatPump,omitempty" db:"heat_pump"`
	ElectricVehicle         *bool      `json:"electricVehicle,omitempty" db:"electric_vehicle"`
	ElectricHotWater        *bool      `json:"electricHotWater,omitempty" db:"electric_hot_water"`
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
}

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