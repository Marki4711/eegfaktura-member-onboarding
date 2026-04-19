package shared

import "errors"

// Common errors
var (
	ErrNotFound     = errors.New("resource not found")
	ErrGone         = errors.New("resource no longer available")
	ErrConflict     = errors.New("resource conflict")
	ErrValidation   = errors.New("validation failed")
	ErrInternal     = errors.New("internal server error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// ErrorResponse is the flat error envelope returned by all API endpoints.
// Shape matches docs/api-spec.md §7.
type ErrorResponse struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

// ValidationError represents a validation error with per-field messages.
type ValidationError struct {
	Message string
	Fields  map[string]string
}

func (e ValidationError) Error() string {
	return e.Message
}

// NewValidationError creates a ValidationError with per-field messages.
func NewValidationError(message string, fields map[string]string) ValidationError {
	return ValidationError{
		Message: message,
		Fields:  fields,
	}
}

// ConflictError represents a business rule conflict with a descriptive message.
type ConflictError struct {
	Message string
}

func (e ConflictError) Error() string { return e.Message }

// NewConflictError creates a ConflictError with the given message.
func NewConflictError(message string) ConflictError {
	return ConflictError{Message: message}
}

// NewErrorResponse maps an error to the canonical ErrorResponse.
func NewErrorResponse(err error) ErrorResponse {
	resp := ErrorResponse{
		Code:    "internal_error",
		Message: "An internal error occurred",
	}

	switch e := err.(type) {
	case ValidationError:
		resp.Code = "validation_error"
		resp.Message = e.Message
		resp.Fields = e.Fields
	case *ValidationError:
		resp.Code = "validation_error"
		resp.Message = e.Message
		resp.Fields = e.Fields
	case ConflictError:
		resp.Code = "conflict"
		resp.Message = e.Message
	default:
		switch {
		case errors.Is(err, ErrNotFound):
			resp.Code = "not_found"
			resp.Message = "Resource not found"
		case errors.Is(err, ErrGone):
			resp.Code = "gone"
			resp.Message = "Registration is no longer active"
		case errors.Is(err, ErrConflict):
			resp.Code = "conflict"
			resp.Message = "Resource conflict"
		case errors.Is(err, ErrForbidden):
			resp.Code = "forbidden"
			resp.Message = "Forbidden"
		}
	}

	return resp
}
