package shared

import (
	"time"

	"github.com/google/uuid"
)

// Request models

// CreateApplicationRequest represents the request to create a new application
type CreateApplicationRequest struct {
	RCNumber             string                      `json:"rcNumber" validate:"required"`
	Firstname            string                      `json:"firstname" validate:"required,min=1,max=255"`
	Lastname             string                      `json:"lastname" validate:"required,min=1,max=255"`
	BirthDate            *string                     `json:"birthDate,omitempty"`
	Email                string                      `json:"email" validate:"required,email"`
	Phone                *string                     `json:"phone,omitempty"`
	ResidentStreet       string                      `json:"residentStreet" validate:"required,min=1,max=255"`
	ResidentStreetNumber string                      `json:"residentStreetNumber" validate:"required,min=1,max=50"`
	ResidentZip          string                      `json:"residentZip" validate:"required,min=1,max=20"`
	ResidentCity         string                      `json:"residentCity" validate:"required,min=1,max=255"`
	ResidentCountry      string                      `json:"residentCountry" validate:"required,len=2"`
	PrivacyAccepted      bool                        `json:"privacyAccepted" validate:"required"`
	PrivacyVersion       string                      `json:"privacyVersion" validate:"required"`
	AccuracyConfirmed    bool                        `json:"accuracyConfirmed" validate:"required"`
	CommunicationConsent bool                        `json:"communicationConsent"`
	MeteringPoints       []CreateMeteringPointRequest `json:"meteringPoints" validate:"required,min=1,max=10,dive"`
}

// CreateMeteringPointRequest represents a metering point in create request
type CreateMeteringPointRequest struct {
	MeteringPoint string `json:"meteringPoint" validate:"required"`
	Direction     string `json:"direction" validate:"required,oneof=CONSUMPTION PRODUCTION"`
}

// UpdateApplicationRequest represents the request to update an application
type UpdateApplicationRequest struct {
	Firstname            *string                     `json:"firstname,omitempty" validate:"omitempty,min=1,max=255"`
	Lastname             *string                     `json:"lastname,omitempty" validate:"omitempty,min=1,max=255"`
	BirthDate            *string                     `json:"birthDate,omitempty"`
	Email                *string                     `json:"email,omitempty" validate:"omitempty,email"`
	Phone                *string                     `json:"phone,omitempty"`
	ResidentStreet       *string                     `json:"residentStreet,omitempty" validate:"omitempty,min=1,max=255"`
	ResidentStreetNumber *string                     `json:"residentStreetNumber,omitempty" validate:"omitempty,min=1,max=50"`
	ResidentZip          *string                     `json:"residentZip,omitempty" validate:"omitempty,min=1,max=20"`
	ResidentCity         *string                     `json:"residentCity,omitempty" validate:"omitempty,min=1,max=255"`
	ResidentCountry      *string                     `json:"residentCountry,omitempty" validate:"omitempty,len=2"`
	PrivacyAccepted      *bool                       `json:"privacyAccepted,omitempty"`
	PrivacyVersion       *string                     `json:"privacyVersion,omitempty"`
	AccuracyConfirmed    *bool                       `json:"accuracyConfirmed,omitempty"`
	CommunicationConsent *bool                       `json:"communicationConsent,omitempty"`
	MeteringPoints       []CreateMeteringPointRequest `json:"meteringPoints,omitempty" validate:"omitempty,min=1,max=10,dive"`
}

// Response models

// RegistrationConfig represents the response for the registration entry point endpoint
type RegistrationConfig struct {
	RCNumber string `json:"rcNumber"`
	EEGID    string `json:"eegId"`
	Title    string `json:"title"`
	Active   bool   `json:"active"`
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
	Firstname            *string                      `json:"firstname,omitempty" validate:"omitempty,min=1,max=255"`
	Lastname             *string                      `json:"lastname,omitempty" validate:"omitempty,min=1,max=255"`
	BirthDate            *string                      `json:"birthDate,omitempty"`
	Email                *string                      `json:"email,omitempty" validate:"omitempty,email"`
	Phone                *string                      `json:"phone,omitempty"`
	ResidentStreet       *string                      `json:"residentStreet,omitempty" validate:"omitempty,min=1,max=255"`
	ResidentStreetNumber *string                      `json:"residentStreetNumber,omitempty" validate:"omitempty,min=1,max=50"`
	ResidentZip          *string                      `json:"residentZip,omitempty" validate:"omitempty,min=1,max=20"`
	ResidentCity         *string                      `json:"residentCity,omitempty" validate:"omitempty,min=1,max=255"`
	ResidentCountry      *string                      `json:"residentCountry,omitempty" validate:"omitempty,len=2"`
	AdminNote            *string                      `json:"adminNote,omitempty"`
	MeteringPoints       []CreateMeteringPointRequest  `json:"meteringPoints,omitempty" validate:"omitempty,min=1,max=10,dive"`
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
	EEGID           *string    `json:"eegId"`
	RCNumber        string     `json:"rcNumber"`
	Status          string     `json:"status"`
	Firstname       string     `json:"firstname"`
	Lastname        string     `json:"lastname"`
	Email           string     `json:"email"`
	SubmittedAt     *time.Time `json:"submittedAt"`
	MeteringPoints  []string   `json:"meteringPoints"`
}

// ApplicationListResponse wraps a paginated list of applications.
type ApplicationListResponse struct {
	Items    []ApplicationListItem `json:"items"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
	Total    int                   `json:"total"`
}

// AdminApplicationDetailResponse is the full admin detail view: application
// record plus its metering points and complete status history.
type AdminApplicationDetailResponse struct {
	Application
	MeteringPoints []MeteringPoint  `json:"meteringPoints"`
	StatusLog      []StatusLogEntry `json:"statusLog"`
}

// ChangeStatusResponse is returned after a successful status transition.
type ChangeStatusResponse struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}