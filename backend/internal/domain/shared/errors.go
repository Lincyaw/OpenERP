package shared

// DomainError represents a domain-level error
type DomainError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface
func (e *DomainError) Error() string {
	return e.Message
}

// NewDomainError creates a new domain error
func NewDomainError(code, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
	}
}

// Common domain errors
var (
	ErrNotFound            = NewDomainError("NOT_FOUND", "Resource not found")
	ErrAlreadyExists       = NewDomainError("ALREADY_EXISTS", "Resource already exists")
	ErrInvalidInput        = NewDomainError("INVALID_INPUT", "Invalid input provided")
	ErrConcurrencyConflict = NewDomainError("CONCURRENCY_CONFLICT", "Resource was modified by another process")
	ErrUnauthorized        = NewDomainError("UNAUTHORIZED", "Not authorized to perform this action")
	ErrForbidden           = NewDomainError("FORBIDDEN", "Access to this resource is forbidden")
	ErrInvalidState        = NewDomainError("INVALID_STATE", "Operation not allowed in current state")
	ErrInsufficientStock   = NewDomainError("INSUFFICIENT_STOCK", "Insufficient stock available")
	ErrInsufficientBalance = NewDomainError("INSUFFICIENT_BALANCE", "Insufficient balance available")
)
