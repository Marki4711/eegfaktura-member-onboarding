package application

import (
	"log/slog"

	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// RegistrationService handles business logic for public registration lookups.
type RegistrationService struct {
	entrypointRepo    *RegistrationEntrypointRepository
	fieldConfigRepo   *FieldConfigRepository
	legalDocumentRepo *LegalDocumentRepository
	centralPolicy     shared.LegalDocumentItem
}

// NewRegistrationService creates a new RegistrationService.
func NewRegistrationService(
	entrypointRepo *RegistrationEntrypointRepository,
	fieldConfigRepo *FieldConfigRepository,
	legalDocumentRepo *LegalDocumentRepository,
	centralPolicyTitle, centralPolicyURL string,
) *RegistrationService {
	return &RegistrationService{
		entrypointRepo:    entrypointRepo,
		fieldConfigRepo:   fieldConfigRepo,
		legalDocumentRepo: legalDocumentRepo,
		centralPolicy: shared.LegalDocumentItem{
			ID:              uuid.Nil,
			Title:           centralPolicyTitle,
			URL:             centralPolicyURL,
			Required:        true,
			SortOrder:       9999,
			IsCentralPolicy: true,
		},
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
		slog.Warn("failed to load field config", "rc", rcNumber, "error", err)
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

	docs, err := s.legalDocumentRepo.GetByRCNumber(rcNumber)
	if err != nil {
		slog.Warn("failed to load legal documents", "rc", rcNumber, "error", err)
		docs = nil
	}

	legalDocuments := make([]shared.LegalDocumentItem, 0, len(docs)+1)
	for _, d := range docs {
		legalDocuments = append(legalDocuments, shared.LegalDocumentItem{
			ID:              d.ID,
			Title:           d.Title,
			URL:             d.URL,
			Required:        d.Required,
			SortOrder:       d.SortOrder,
			IsCentralPolicy: false,
		})
	}
	if ep.ShowCentralPolicy && s.centralPolicy.URL != "" {
		legalDocuments = append(legalDocuments, s.centralPolicy)
	}

	cfg := &shared.RegistrationConfig{
		RCNumber:           ep.RCNumber,
		Title:              "Mitglied werden",
		Active:             ep.IsActive,
		FieldConfig:        fieldConfig,
		IntroText:          ep.IntroText,
		SEPAMandateEnabled: ep.SEPAMandateEnabled,
		ShowCentralPolicy:  ep.ShowCentralPolicy,
		LegalDocuments:     legalDocuments,
		// PROJ-37: only ship the two value fields when the feature is on.
		CooperativeSharesEnabled: ep.CooperativeSharesEnabled,
	}
	if ep.CooperativeSharesEnabled {
		cfg.CooperativeRequiredShares = ep.CooperativeRequiredShares
		cfg.CooperativeShareAmountCents = ep.CooperativeShareAmountCents
	}
	return cfg, nil
}
