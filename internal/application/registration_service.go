package application

import (
	"database/sql"
	"fmt"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// RegistrationService handles business logic for registration
type RegistrationService struct {
	appRepo *ApplicationRepository
}

// NewRegistrationService creates a new registration service
func NewRegistrationService(appRepo *ApplicationRepository) *RegistrationService {
	return &RegistrationService{appRepo: appRepo}
}

// GetRegistrationConfig gets the configuration for a registration slug
func (s *RegistrationService) GetRegistrationConfig(slug string) (*shared.RegistrationConfig, error) {
	// For now, we check if the slug exists in the database
	// In a real implementation, there might be a separate table for registration configurations
	exists, err := s.appRepo.CheckRegistrationSlugExists(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check registration slug: %w", err)
	}

	if !exists {
		return nil, shared.ErrNotFound
	}

	// Mock configuration - in production, this would come from a database table
	config := &shared.RegistrationConfig{
		RegistrationSlug: slug,
		EEGID:            "mock-eeg-id", // This would be looked up based on slug
		Title:            "Mitglied werden",
		Active:           true,
	}

	return config, nil
}