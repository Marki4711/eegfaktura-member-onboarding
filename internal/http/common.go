package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

func httpStatusFor(code string) int {
	switch code {
	case "not_found":
		return http.StatusNotFound
	case "gone":
		return http.StatusGone
	case "validation_error":
		return http.StatusBadRequest
	case "conflict":
		return http.StatusConflict
	case "forbidden":
		return http.StatusForbidden
	case "turnstile_failed", "turnstile_missing":
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

func (h *RegistrationHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("error encoding JSON response", "error", err)
	}
}

func (h *RegistrationHandler) writeError(w http.ResponseWriter, errorResp shared.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusFor(errorResp.Code))
	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		slog.Error("error encoding error response", "error", err)
	}
}

func (h *ApplicationHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("error encoding JSON response", "error", err)
	}
}

func (h *ApplicationHandler) writeError(w http.ResponseWriter, errorResp shared.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusFor(errorResp.Code))
	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		slog.Error("error encoding error response", "error", err)
	}
}
