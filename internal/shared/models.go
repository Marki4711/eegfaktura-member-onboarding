package shared

import (
	"time"

	"github.com/google/uuid"
)

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
	EEGID                *string           `json:"eegId,omitempty" db:"eeg_id"`
	RegistrationSlug     string            `json:"registrationSlug" db:"registration_slug"`
	Status               ApplicationStatus `json:"status" db:"status"`
	StartedAt            *time.Time        `json:"startedAt,omitempty" db:"started_at"`
	SubmittedAt          *time.Time        `json:"submittedAt,omitempty" db:"submitted_at"`
	ApprovedAt           *time.Time        `json:"approvedAt,omitempty" db:"approved_at"`
	RejectedAt           *time.Time        `json:"rejectedAt,omitempty" db:"rejected_at"`
	ImportedAt           *time.Time        `json:"importedAt,omitempty" db:"imported_at"`
	Firstname            string            `json:"firstname" db:"firstname"`
	Lastname             string            `json:"lastname" db:"lastname"`
	BirthDate            *time.Time        `json:"birthDate,omitempty" db:"birth_date"`
	Email                string            `json:"email" db:"email"`
	Phone                *string           `json:"phone,omitempty" db:"phone"`
	ResidentStreet       string            `json:"residentStreet" db:"resident_street"`
	ResidentStreetNumber string            `json:"residentStreetNumber" db:"resident_street_number"`
	ResidentZip          string            `json:"residentZip" db:"resident_zip"`
	ResidentCity         string            `json:"residentCity" db:"resident_city"`
	ResidentCountry      string            `json:"residentCountry" db:"resident_country"`
	PrivacyAccepted      bool              `json:"privacyAccepted" db:"privacy_accepted"`
	PrivacyVersion       *string           `json:"privacyVersion,omitempty" db:"privacy_version"`
	PrivacyAcceptedAt    *time.Time        `json:"privacyAcceptedAt,omitempty" db:"privacy_accepted_at"`
	AccuracyConfirmed    bool              `json:"accuracyConfirmed" db:"accuracy_confirmed"`
	CommunicationConsent bool              `json:"communicationConsent" db:"communication_consent"`
	ReviewedByUserID     *string           `json:"reviewedByUserId,omitempty" db:"reviewed_by_user_id"`
	AdminNote            *string           `json:"adminNote,omitempty" db:"admin_note"`
	NeedsInfoReason      *string           `json:"needsInfoReason,omitempty" db:"needs_info_reason"`
	TargetParticipantID  *string           `json:"targetParticipantId,omitempty" db:"target_participant_id"`
	ImportStartedAt      *time.Time        `json:"importStartedAt,omitempty" db:"import_started_at"`
	ImportFinishedAt     *time.Time        `json:"importFinishedAt,omitempty" db:"import_finished_at"`
	ImportErrorMessage   *string           `json:"importErrorMessage,omitempty" db:"import_error_message"`
	CreatedAt            time.Time         `json:"createdAt" db:"created_at"`
	UpdatedAt            time.Time         `json:"updatedAt" db:"updated_at"`
}

// MeteringPoint represents a metering point entity
type MeteringPoint struct {
	ID            uuid.UUID      `json:"id" db:"id"`
	ApplicationID uuid.UUID      `json:"applicationId" db:"application_id"`
	MeteringPoint string         `json:"meteringPoint" db:"metering_point"`
	Direction     MeterDirection `json:"direction" db:"direction"`
	CreatedAt     time.Time      `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time      `json:"updatedAt" db:"updated_at"`
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