package http

import (
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

// GetRegistrationConfig handles GET /api/public/registration/{rc_number}
func (h *RegistrationHandler) GetRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	rcNumber := chi.URLParam(r, "rc_number")
	if rcNumber == "" {
		h.writeError(w, shared.NewErrorResponse(shared.ErrNotFound))
		return
	}

	config, err := h.registrationService.GetRegistrationConfig(rcNumber)
	if err != nil {
		switch err {
		case shared.ErrNotFound:
			h.writeError(w, shared.NewErrorResponse(shared.ErrNotFound))
		case shared.ErrGone:
			h.writeError(w, shared.NewErrorResponse(shared.ErrGone))
		default:
			h.writeError(w, shared.NewErrorResponse(shared.ErrInternal))
		}
		return
	}

	h.writeJSON(w, http.StatusOK, config)
}
