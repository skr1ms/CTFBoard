package apperr

import "fmt"

// ValidationError is a domain-level validation error without HTTP semantics.
// Mapped to 400 / "VALIDATION_ERROR" at the transport layer.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func NewValidationErrorf(format string, args ...any) *ValidationError {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}
