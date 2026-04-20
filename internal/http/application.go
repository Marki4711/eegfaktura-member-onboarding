package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"github.com/your-org/eegfaktura-member-onboarding/internal/application"
	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// ApplicationHandler handles application-related HTTP requests
type ApplicationHandler struct {
	applicationService *application.ApplicationService
	validate           *validator.Validate
}

// NewApplicationHandler creates a new application handler
func NewApplicationHandler(applicationService *application.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{
		applicationService: applicationService,
		validate:           validator.New(),
	}
}

// CreateApplication handles POST /api/public/applications
func (h *ApplicationHandler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var req shared.CreateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}

	// Validate request
	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	// Additional validation for metering points
	if len(req.MeteringPoints) == 0 {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", map[string]string{
			"meteringPoints": "At least one metering point is required",
		})))
		return
	}

	response, err := h.applicationService.CreateApplication(req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusCreated, response)
}

// UpdateApplication handles PUT /api/public/applications/{id}
func (h *ApplicationHandler) UpdateApplication(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid application ID", nil)))
		return
	}

	var req shared.UpdateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}

	// Validate request
	if err := h.validate.Struct(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	response, err := h.applicationService.UpdateApplication(id, req)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, response)
}

// SubmitApplication handles POST /api/public/applications/{id}/submit
func (h *ApplicationHandler) SubmitApplication(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid application ID", nil)))
		return
	}

	response, err := h.applicationService.SubmitApplication(id)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Helper methods

func (h *ApplicationHandler) writeValidationError(w http.ResponseWriter, err error) {
	fields := make(map[string]string)
	for _, verr := range err.(validator.ValidationErrors) {
		field := verr.Field()
		if _, exists := fields[field]; !exists {
			fields[field] = h.getValidationMessage(verr)
		}
	}
	h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Validation failed", fields)))
}

func (h *ApplicationHandler) getValidationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return "Value is too short"
	case "max":
		return "Value is too long"
	default:
		return "Invalid value"
	}
}

func (h *ApplicationHandler) handleServiceError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case shared.ValidationError:
		h.writeError(w, shared.NewErrorResponse(e))
	default:
		switch err {
		case shared.ErrNotFound:
			h.writeError(w, shared.NewErrorResponse(shared.ErrNotFound))
		case shared.ErrGone:
			h.writeError(w, shared.NewErrorResponse(shared.ErrGone))
		case shared.ErrConflict:
			h.writeError(w, shared.NewErrorResponse(shared.ErrConflict))
		default:
			slog.Error("internal error", "error", err)
			h.writeError(w, shared.NewErrorResponse(shared.ErrInternal))
		}
	}
}