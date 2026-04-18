package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// RegistrationHandler handles registration-related HTTP requests
type RegistrationHandler struct {
	registrationService *application.RegistrationService
}

// NewRegistrationHandler creates a new registration handler
func NewRegistrationHandler(registrationService *application.RegistrationService) *RegistrationHandler {
	return &RegistrationHandler{
		registrationService: registrationService,
	}
}

// GetRegistrationConfig handles GET /api/public/registration/{registration_slug}
func (h *RegistrationHandler) GetRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "registration_slug")
	if slug == "" {
		h.writeError(w, shared.NewErrorResponse(shared.ErrNotFound))
		return
	}

	config, err := h.registrationService.GetRegistrationConfig(slug)
	if err != nil {
		if err == shared.ErrNotFound {
			h.writeError(w, shared.NewErrorResponse(shared.ErrNotFound))
			return
		}
		h.writeError(w, shared.NewErrorResponse(shared.ErrInternal))
		return
	}

	h.writeJSON(w, http.StatusOK, config)
}