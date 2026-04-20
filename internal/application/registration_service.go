package application

import (
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// RegistrationService handles business logic for public registration lookups.
type RegistrationService struct {
	entrypointRepo *RegistrationEntrypointRepository
}

// NewRegistrationService creates a new RegistrationService.
func NewRegistrationService(entrypointRepo *RegistrationEntrypointRepository) *RegistrationService {
	return &RegistrationService{entrypointRepo: entrypointRepo}
}

// GetRegistrationConfig resolves an RC number via the local registration_entrypoint
// table and returns the public configuration for the registration form.
// Returns shared.ErrNotFound when the RC number is unknown.
// Returns shared.ErrGone when the entry point exists but is_active = false.
func (s *RegistrationService) GetRegistrationConfig(rcNumber string) (*shared.RegistrationConfig, error) {
	ep, err := s.entrypointRepo.GetByRCNumber(rcNumber)
	if err != nil {
		return nil, err
	}
	if !ep.IsActive {
		return nil, shared.ErrGone
	}
	return &shared.RegistrationConfig{
		RCNumber: ep.RCNumber,
		Title:    "Mitglied werden",
		Active:   ep.IsActive,
	}, nil
}
