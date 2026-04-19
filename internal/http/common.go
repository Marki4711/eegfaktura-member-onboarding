package http

import (
	"encoding/json"
	"log"
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
	default:
		return http.StatusInternalServerError
	}
}

func (h *RegistrationHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func (h *RegistrationHandler) writeError(w http.ResponseWriter, errorResp shared.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusFor(errorResp.Code))
	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}

func (h *ApplicationHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func (h *ApplicationHandler) writeError(w http.ResponseWriter, errorResp shared.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusFor(errorResp.Code))
	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}
