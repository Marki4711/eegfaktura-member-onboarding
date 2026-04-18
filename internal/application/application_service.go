package application

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ApplicationService handles business logic for applications
type ApplicationService struct {
	appRepo        *ApplicationRepository
	meteringRepo   *MeteringPointRepository
	statusLogRepo  *StatusLogRepository
}

// NewApplicationService creates a new application service
func NewApplicationService(
	appRepo *ApplicationRepository,
	meteringRepo *MeteringPointRepository,
	statusLogRepo *StatusLogRepository,
) *ApplicationService {
	return &ApplicationService{
		appRepo:       appRepo,
		meteringRepo:  meteringRepo,
		statusLogRepo: statusLogRepo,
	}
}

// CreateApplication creates a new application
func (s *ApplicationService) CreateApplication(req shared.CreateApplicationRequest) (*shared.ApplicationResponse, error) {
	// Validate registration slug exists
	exists, err := s.appRepo.CheckRegistrationSlugExists(req.RegistrationSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to validate registration slug: %w", err)
	}
	if !exists {
		return nil, shared.ErrNotFound
	}

	// Validate metering points
	var meteringPoints []shared.MeteringPoint
	for _, mpReq := range req.MeteringPoints {
		point := shared.MeteringPoint{
			MeteringPoint: mpReq.MeteringPoint,
			Direction:     shared.MeterDirection(mpReq.Direction),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		meteringPoints = append(meteringPoints, point)
	}

	// Check for duplicate metering points
	err = s.meteringRepo.ValidateUniqueMeteringPoints(uuid.Nil, meteringPoints)
	if err != nil {
		return nil, shared.NewValidationError("Validation failed", map[string][]string{
			"meteringPoints": {err.Error()},
		})
	}

	// Generate reference number
	referenceNumber := s.generateReferenceNumber()

	// Create application
	now := time.Now()
	privacyAcceptedAt := now

	app := &shared.Application{
		ReferenceNumber:      referenceNumber,
		RegistrationSlug:     req.RegistrationSlug,
		Status:               shared.StatusDraft,
		StartedAt:            &now,
		Firstname:            req.Firstname,
		Lastname:             req.Lastname,
		BirthDate:            req.BirthDate,
		Email:                req.Email,
		Phone:                req.Phone,
		ResidentStreet:       req.ResidentStreet,
		ResidentStreetNumber: req.ResidentStreetNumber,
		ResidentZip:          req.ResidentZip,
		ResidentCity:         req.ResidentCity,
		ResidentCountry:      req.ResidentCountry,
		PrivacyAccepted:      req.PrivacyAccepted,
		PrivacyVersion:       &req.PrivacyVersion,
		PrivacyAcceptedAt:    &privacyAcceptedAt,
		AccuracyConfirmed:    req.AccuracyConfirmed,
		CommunicationConsent: req.CommunicationConsent,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	err = s.appRepo.Create(app)
	if err != nil {
		return nil, fmt.Errorf("failed to create application: %w", err)
	}

	// Set application ID for metering points
	for i := range meteringPoints {
		meteringPoints[i].ApplicationID = app.ID
	}

	// Create metering points
	err = s.meteringRepo.CreateBulk(app.ID, meteringPoints)
	if err != nil {
		return nil, fmt.Errorf("failed to create metering points: %w", err)
	}

	return &shared.ApplicationResponse{
		ID:             app.ID,
		ReferenceNumber: app.ReferenceNumber,
		Status:         string(app.Status),
		CreatedAt:      app.CreatedAt,
		UpdatedAt:      app.UpdatedAt,
	}, nil
}

// UpdateApplication updates an existing application
func (s *ApplicationService) UpdateApplication(id uuid.UUID, req shared.UpdateApplicationRequest) (*shared.ApplicationResponse, error) {
	// Get existing application
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check if application can be updated
	if app.Status != shared.StatusDraft && app.Status != shared.StatusNeedsInfo {
		return nil, shared.ErrConflict
	}

	// Apply updates
	if req.Firstname != nil {
		app.Firstname = *req.Firstname
	}
	if req.Lastname != nil {
		app.Lastname = *req.Lastname
	}
	if req.BirthDate != nil {
		app.BirthDate = req.BirthDate
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
	if req.ResidentCountry != nil {
		app.ResidentCountry = *req.ResidentCountry
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
	if req.CommunicationConsent != nil {
		app.CommunicationConsent = *req.CommunicationConsent
	}

	// Handle metering points update
	if req.MeteringPoints != nil {
		var meteringPoints []shared.MeteringPoint
		for _, mpReq := range req.MeteringPoints {
			point := shared.MeteringPoint{
				ApplicationID: id,
				MeteringPoint: mpReq.MeteringPoint,
				Direction:     shared.MeterDirection(mpReq.Direction),
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			}
			meteringPoints = append(meteringPoints, point)
		}

		// Validate metering points
		err = s.meteringRepo.ValidateUniqueMeteringPoints(id, meteringPoints)
		if err != nil {
			return nil, shared.NewValidationError("Validation failed", map[string][]string{
				"meteringPoints": {err.Error()},
			})
		}

		// Update metering points
		err = s.meteringRepo.CreateBulk(id, meteringPoints)
		if err != nil {
			return nil, fmt.Errorf("failed to update metering points: %w", err)
		}
	}

	// Update application
	err = s.appRepo.Update(app)
	if err != nil {
		return nil, err
	}

	return &shared.ApplicationResponse{
		ID:             app.ID,
		ReferenceNumber: app.ReferenceNumber,
		Status:         string(app.Status),
		CreatedAt:      app.CreatedAt,
		UpdatedAt:      app.UpdatedAt,
	}, nil
}

// SubmitApplication submits an application
func (s *ApplicationService) SubmitApplication(id uuid.UUID) (*shared.SubmitResponse, error) {
	// Get application
	app, err := s.appRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Check if application can be submitted
	if app.Status != shared.StatusDraft && app.Status != shared.StatusNeedsInfo {
		return nil, shared.ErrConflict
	}

	// Validate required fields for submission
	if !app.PrivacyAccepted || app.PrivacyVersion == nil || !app.AccuracyConfirmed {
		return nil, shared.NewValidationError("Validation failed", map[string][]string{
			"general": {"Privacy consent and accuracy confirmation required for submission"},
		})
	}

	// Check if metering points exist
	meteringPoints, err := s.meteringRepo.GetByApplicationID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get metering points: %w", err)
	}
	if len(meteringPoints) == 0 {
		return nil, shared.NewValidationError("Validation failed", map[string][]string{
			"meteringPoints": {"At least one metering point is required"},
		})
	}

	// Update status
	now := time.Now()
	oldStatus := string(app.Status)
	err = s.appRepo.UpdateStatus(id, shared.StatusSubmitted, &now)
	if err != nil {
		return nil, err
	}

	// Log status change
	statusLog := &shared.StatusLogEntry{
		ApplicationID: id,
		FromStatus:    &oldStatus,
		ToStatus:      string(shared.StatusSubmitted),
		CreatedAt:     now,
	}
	err = s.statusLogRepo.Create(statusLog)
	if err != nil {
		// Log error but don't fail the submission
		fmt.Printf("Failed to create status log: %v\n", err)
	}

	return &shared.SubmitResponse{
		ID:             id,
		ReferenceNumber: app.ReferenceNumber,
		Status:         shared.StatusSubmitted,
		SubmittedAt:    now,
	}, nil
}

// generateReferenceNumber generates a unique reference number
func (s *ApplicationService) generateReferenceNumber() string {
	// Simple implementation - in production, this would be more sophisticated
	now := time.Now()
	return fmt.Sprintf("MO-%s-%06d", now.Format("2006"), now.Unix()%1000000)
}