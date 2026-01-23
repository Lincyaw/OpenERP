package shared

import (
	"context"

	"github.com/google/uuid"
)

// Repository is the base interface for all repositories
type Repository[T any] interface {
	FindByID(ctx context.Context, id uuid.UUID) (*T, error)
	FindAll(ctx context.Context, filter Filter) ([]T, error)
	Save(ctx context.Context, entity *T) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context, filter Filter) (int64, error)
}

// TenantRepository is a repository scoped to a tenant
type TenantRepository[T any] interface {
	Repository[T]
	FindByIDForTenant(ctx context.Context, tenantID, id uuid.UUID) (*T, error)
	FindAllForTenant(ctx context.Context, tenantID uuid.UUID, filter Filter) ([]T, error)
}

// Filter represents query filter options
type Filter struct {
	Page     int
	PageSize int
	OrderBy  string
	OrderDir string
	Search   string
	Filters  map[string]interface{}
}

// DefaultFilter returns a filter with default values
func DefaultFilter() Filter {
	return Filter{
		Page:     1,
		PageSize: 20,
		OrderBy:  "created_at",
		OrderDir: "desc",
		Filters:  make(map[string]interface{}),
	}
}

// Paginated represents a paginated result
type Paginated[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalPages int   `json:"total_pages"`
}

// NewPaginated creates a new paginated result
func NewPaginated[T any](items []T, total int64, page, pageSize int) Paginated[T] {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return Paginated[T]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}
