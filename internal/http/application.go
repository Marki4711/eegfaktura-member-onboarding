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
	applicationService  *application.ApplicationService
	validate            *validator.Validate
	turnstileSecretKey  string
}

// NewApplicationHandler creates a new application handler
func NewApplicationHandler(applicationService *application.ApplicationService, turnstileSecretKey string) *ApplicationHandler {
	return &ApplicationHandler{
		applicationService: applicationService,
		validate:           validator.New(),
		turnstileSecretKey: turnstileSecretKey,
	}
}

// CreateApplication handles POST /api/public/applications
//
// @Summary      Create application draft
// @Description  Creates a new member application in status `draft`. Rate-limited (10 req/10 min per IP). Protected by Cloudflare Turnstile when configured.
// @Tags         Public
// @Accept       json
// @Produce      json
// @Param        body  body     shared.CreateApplicationRequest  true  "Application data"
// @Success      201   {object} shared.ApplicationResponse
// @Failure      400   {object} shared.ErrorResponse  "Validation error"
// @Failure      409   {object} shared.ErrorResponse  "Duplicate metering point"
// @Failure      410   {object} shared.ErrorResponse  "Registration deactivated"
// @Failure      422   {object} shared.ErrorResponse  "Turnstile verification failed"
// @Failure      500   {object} shared.ErrorResponse
// @Router       /api/public/applications [post]
func (h *ApplicationHandler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var req shared.CreateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid JSON", nil)))
		return
	}

	// Verify Cloudflare Turnstile token (skipped when secret key not configured)
	token := ""
	if req.TurnstileToken != nil {
		token = *req.TurnstileToken
	}
	if errCode, err := verifyTurnstileToken(h.turnstileSecretKey, token); err != nil {
		slog.Warn("turnstile verification failed", "code", errCode, "error", err)
		h.writeError(w, shared.ErrorResponse{Code: errCode, Message: err.Error()})
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
//
// @Summary      Update application draft
// @Description  Updates a member application in status `draft`. Only drafts can be updated; submitted applications return 409.
// @Tags         Public
// @Accept       json
// @Produce      json
// @Param        id    path     string                           true  "Application UUID"
// @Param        body  body     shared.UpdateApplicationRequest  true  "Updated application data"
// @Success      200   {object} shared.ApplicationResponse
// @Failure      400   {object} shared.ErrorResponse  "Validation error or invalid UUID"
// @Failure      404   {object} shared.ErrorResponse  "Application not found"
// @Failure      409   {object} shared.ErrorResponse  "Application already submitted"
// @Failure      500   {object} shared.ErrorResponse
// @Router       /api/public/applications/{id} [put]
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
//
// @Summary      Submit application
// @Description  Transitions the application from `draft` to `submitted`. Triggers confirmation email to the member. Consents for legal documents can be passed in the request body.
// @Tags         Public
// @Accept       json
// @Produce      json
// @Param        id    path     string               true   "Application UUID"
// @Param        body  body     shared.SubmitRequest false  "Optional legal document consents"
// @Success      200   {object} shared.SubmitResponse
// @Failure      400   {object} shared.ErrorResponse  "Invalid UUID"
// @Failure      404   {object} shared.ErrorResponse  "Application not found"
// @Failure      409   {object} shared.ErrorResponse  "Invalid status transition"
// @Failure      500   {object} shared.ErrorResponse
// @Router       /api/public/applications/{id}/submit [post]
func (h *ApplicationHandler) SubmitApplication(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.writeError(w, shared.NewErrorResponse(shared.NewValidationError("Invalid application ID", nil)))
		return
	}

	var req shared.SubmitRequest
	// Body is optional — ignore decode errors (empty body is valid)
	json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck

	response, err := h.applicationService.SubmitApplication(id, req.Consents)
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