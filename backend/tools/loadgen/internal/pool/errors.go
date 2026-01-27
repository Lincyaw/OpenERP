package pool

import "errors"

// Common errors returned by parameter pools.
var (
	// ErrPoolClosed is returned when an operation is attempted on a closed pool.
	ErrPoolClosed = errors.New("parameter pool is closed")

	// ErrValueNotFound is returned when a requested value is not found.
	ErrValueNotFound = errors.New("value not found in pool")

	// ErrInvalidSemanticType is returned when an invalid semantic type is provided.
	ErrInvalidSemanticType = errors.New("invalid semantic type")
)
