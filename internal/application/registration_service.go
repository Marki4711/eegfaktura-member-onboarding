package application

import (
	"fmt"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// RegistrationService handles business logic for public registration lookups.
type RegistrationService struct {
	entrypointRepo  *RegistrationEntrypointRepository
	fieldConfigRepo *FieldConfigRepository
}

// NewRegistrationService creates a new RegistrationService.
func NewRegistrationService(
	entrypointRepo *RegistrationEntrypointRepository,
	fieldConfigRepo *FieldConfigRepository,
) *RegistrationService {
	return &RegistrationService{
		entrypointRepo:  entrypointRepo,
		fieldConfigRepo: fieldConfigRepo,
	}
}

// GetRegistrationConfig resolves an RC number via the local registration_entrypoint
// table and returns the public configuration including the EEG's field config.
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

	rawConfig, err := s.fieldConfigRepo.Get(rcNumber)
	if err != nil {
		// Non-fatal: log and fall back to empty config (frontend uses defaults)
		fmt.Printf("warning: failed to load field config for rc=%s: %v\n", rcNumber, err)
		rawConfig = map[string]FieldConfigEntry{}
	}

	// Map to public format: admin_only fields are hidden from members.
	fieldConfig := make(map[string]string, len(rawConfig))
	for name, entry := range rawConfig {
		if entry.State == "admin_only" {
			fieldConfig[name] = "hidden"
		} else {
			fieldConfig[name] = entry.State
		}
	}

	return &shared.RegistrationConfig{
		RCNumber:           ep.RCNumber,
		Title:              "Mitglied werden",
		Active:             ep.IsActive,
		FieldConfig:        fieldConfig,
		IntroText:          ep.IntroText,
		SEPAMandateEnabled: ep.SEPAMandateEnabled,
	}, nil
}
