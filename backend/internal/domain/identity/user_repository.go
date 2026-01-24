package identity

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository defines the interface for user persistence
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *User) error

	// Update updates an existing user
	Update(ctx context.Context, user *User) error

	// Delete deletes a user by ID
	Delete(ctx context.Context, id uuid.UUID) error

	// FindByID finds a user by ID
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)

	// FindByUsername finds a user by username within the tenant
	FindByUsername(ctx context.Context, username string) (*User, error)

	// FindByEmail finds a user by email within the tenant
	FindByEmail(ctx context.Context, email string) (*User, error)

	// FindByPhone finds a user by phone within the tenant
	FindByPhone(ctx context.Context, phone string) (*User, error)

	// FindAll returns all users for the current tenant with pagination
	FindAll(ctx context.Context, filter UserFilter) ([]*User, int64, error)

	// FindByRoleID finds all users with a specific role
	FindByRoleID(ctx context.Context, roleID uuid.UUID) ([]*User, error)

	// ExistsByUsername checks if a username already exists
	ExistsByUsername(ctx context.Context, username string) (bool, error)

	// ExistsByEmail checks if an email already exists
	ExistsByEmail(ctx context.Context, email string) (bool, error)

	// SaveUserRoles saves the user's roles (replaces existing)
	SaveUserRoles(ctx context.Context, user *User) error

	// LoadUserRoles loads the user's roles from the database
	LoadUserRoles(ctx context.Context, user *User) error

	// Count returns the total number of users for the tenant
	Count(ctx context.Context) (int64, error)
}

// UserFilter contains filter options for querying users
type UserFilter struct {
	// Search keyword for username, email, or display name
	Keyword string

	// Filter by status
	Status *UserStatus

	// Filter by role ID
	RoleID *uuid.UUID

	// Pagination
	Page     int
	PageSize int

	// Sorting
	SortBy    string
	SortOrder string // "asc" or "desc"
}

// NewUserFilter creates a new UserFilter with default values
func NewUserFilter() UserFilter {
	return UserFilter{
		Page:      1,
		PageSize:  20,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
}

// WithKeyword sets the search keyword
func (f UserFilter) WithKeyword(keyword string) UserFilter {
	f.Keyword = keyword
	return f
}

// WithStatus sets the status filter
func (f UserFilter) WithStatus(status UserStatus) UserFilter {
	f.Status = &status
	return f
}

// WithRoleID sets the role ID filter
func (f UserFilter) WithRoleID(roleID uuid.UUID) UserFilter {
	f.RoleID = &roleID
	return f
}

// WithPagination sets pagination parameters
func (f UserFilter) WithPagination(page, pageSize int) UserFilter {
	f.Page = page
	f.PageSize = pageSize
	return f
}

// WithSorting sets sorting parameters
func (f UserFilter) WithSorting(sortBy, sortOrder string) UserFilter {
	f.SortBy = sortBy
	f.SortOrder = sortOrder
	return f
}

// Offset returns the offset for pagination
func (f UserFilter) Offset() int {
	if f.Page <= 0 {
		return 0
	}
	return (f.Page - 1) * f.PageSize
}

// Limit returns the limit for pagination
func (f UserFilter) Limit() int {
	if f.PageSize <= 0 {
		return 20
	}
	if f.PageSize > 100 {
		return 100
	}
	return f.PageSize
}
