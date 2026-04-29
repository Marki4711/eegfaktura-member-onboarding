package http

import (
	"net/http"
	"strings"

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
//
// @Summary      Get registration config
// @Description  Returns EEG-specific configuration for the public registration form: field visibility, legal documents, intro text, SEPA settings.
// @Tags         Public
// @Produce      json
// @Param        rc_number  path     string  true  "EEG registration code (e.g. RC123456)"
// @Success      200        {object} shared.RegistrationConfig
// @Failure      404        {object} shared.ErrorResponse  "RC number unknown"
// @Failure      410        {object} shared.ErrorResponse  "Registration deactivated for this EEG"
// @Failure      500        {object} shared.ErrorResponse
// @Router       /api/public/registration/{rc_number} [get]
func (h *RegistrationHandler) GetRegistrationConfig(w http.ResponseWriter, r *http.Request) {
	rcNumber := strings.ToUpper(chi.URLParam(r, "rc_number"))
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
