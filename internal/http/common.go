package http

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/your-org/eegfaktura-member-onboarding/internal/shared"
)

// writeJSON writes a JSON response
func (h *RegistrationHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// writeError writes an error response
func (h *RegistrationHandler) writeError(w http.ResponseWriter, errorResp shared.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")

	status := http.StatusInternalServerError
	switch errorResp.Error.Code {
	case "NOT_FOUND":
		status = http.StatusNotFound
	case "VALIDATION_ERROR":
		status = http.StatusBadRequest
	case "CONFLICT":
		status = http.StatusConflict
	}

	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}

// writeJSON for ApplicationHandler
func (h *ApplicationHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// writeError for ApplicationHandler
func (h *ApplicationHandler) writeError(w http.ResponseWriter, errorResp shared.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")

	status := http.StatusInternalServerError
	switch errorResp.Error.Code {
	case "NOT_FOUND":
		status = http.StatusNotFound
	case "VALIDATION_ERROR":
		status = http.StatusBadRequest
	case "CONFLICT":
		status = http.StatusConflict
	}

	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}