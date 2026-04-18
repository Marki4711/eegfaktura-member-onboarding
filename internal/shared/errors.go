package shared

import "errors"

// Common errors
var (
	ErrNotFound      = errors.New("resource not found")
	ErrConflict      = errors.New("resource conflict")
	ErrValidation    = errors.New("validation failed")
	ErrInternal      = errors.New("internal server error")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information
type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string][]string `json:"details,omitempty"`
}

// ValidationError represents a validation error with field details
type ValidationError struct {
	Message string
	Details map[string][]string
}

func (e ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a new validation error
func NewValidationError(message string, details map[string][]string) ValidationError {
	return ValidationError{
		Message: message,
		Details: details,
	}
}

// NewErrorResponse creates an error response from an error
func NewErrorResponse(err error) ErrorResponse {
	detail := ErrorDetail{
		Code:    "INTERNAL_ERROR",
		Message: "An internal error occurred",
	}

	switch e := err.(type) {
	case ValidationError:
		detail.Code = "VALIDATION_ERROR"
		detail.Message = e.Message
		detail.Details = e.Details
	case *ValidationError:
		detail.Code = "VALIDATION_ERROR"
		detail.Message = e.Message
		detail.Details = e.Details
	default:
		if errors.Is(err, ErrNotFound) {
			detail.Code = "NOT_FOUND"
			detail.Message = "Resource not found"
		} else if errors.Is(err, ErrConflict) {
			detail.Code = "CONFLICT"
			detail.Message = "Resource conflict"
		}
	}

	return ErrorResponse{Error: detail}
}