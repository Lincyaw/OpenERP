package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormUserRepository implements UserRepository using GORM
type GormUserRepository struct {
	db *gorm.DB
}

// NewGormUserRepository creates a new GormUserRepository
func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

// Create creates a new user
func (r *GormUserRepository) Create(ctx context.Context, user *identity.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update updates an existing user
func (r *GormUserRepository) Update(ctx context.Context, user *identity.User) error {
	result := r.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// Delete deletes a user by ID
func (r *GormUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Delete user roles first
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", id).
		Delete(&identity.UserRole{}).Error; err != nil {
		return err
	}

	result := r.db.WithContext(ctx).Delete(&identity.User{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return shared.ErrNotFound
	}
	return nil
}

// FindByID finds a user by ID
func (r *GormUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.User, error) {
	var user identity.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByUsername finds a user by username within the tenant
func (r *GormUserRepository) FindByUsername(ctx context.Context, username string) (*identity.User, error) {
	var user identity.User
	if err := r.db.WithContext(ctx).
		Where("LOWER(username) = ?", strings.ToLower(username)).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail finds a user by email within the tenant
func (r *GormUserRepository) FindByEmail(ctx context.Context, email string) (*identity.User, error) {
	if email == "" {
		return nil, shared.ErrNotFound
	}
	var user identity.User
	if err := r.db.WithContext(ctx).
		Where("LOWER(email) = ?", strings.ToLower(email)).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindByPhone finds a user by phone within the tenant
func (r *GormUserRepository) FindByPhone(ctx context.Context, phone string) (*identity.User, error) {
	if phone == "" {
		return nil, shared.ErrNotFound
	}
	var user identity.User
	if err := r.db.WithContext(ctx).
		Where("phone = ?", phone).
		First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, shared.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindAll returns all users for the current tenant with pagination
func (r *GormUserRepository) FindAll(ctx context.Context, filter identity.UserFilter) ([]*identity.User, int64, error) {
	var users []*identity.User
	var total int64

	query := r.db.WithContext(ctx).Model(&identity.User{})

	// Apply filters
	query = r.applyFilter(query, filter)

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := filter.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}
	query = query.Order(sortBy + " " + sortOrder)

	// Apply pagination
	offset := filter.Offset()
	limit := filter.Limit()
	query = query.Offset(offset).Limit(limit)

	if err := query.Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// FindByRoleID finds all users with a specific role
func (r *GormUserRepository) FindByRoleID(ctx context.Context, roleID uuid.UUID) ([]*identity.User, error) {
	var users []*identity.User
	if err := r.db.WithContext(ctx).
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Where("user_roles.role_id = ?", roleID).
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// ExistsByUsername checks if a username already exists
func (r *GormUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&identity.User{}).
		Where("LOWER(username) = ?", strings.ToLower(username)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ExistsByEmail checks if an email already exists
func (r *GormUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if email == "" {
		return false, nil
	}
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&identity.User{}).
		Where("LOWER(email) = ?", strings.ToLower(email)).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// SaveUserRoles saves the user's roles (replaces existing)
func (r *GormUserRepository) SaveUserRoles(ctx context.Context, user *identity.User) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete existing roles
		if err := tx.Where("user_id = ?", user.ID).Delete(&identity.UserRole{}).Error; err != nil {
			return err
		}

		// Insert new roles
		if len(user.RoleIDs) > 0 {
			userRoles := make([]identity.UserRole, len(user.RoleIDs))
			for i, roleID := range user.RoleIDs {
				userRoles[i] = identity.UserRole{
					UserID:    user.ID,
					RoleID:    roleID,
					TenantID:  user.TenantID,
					CreatedAt: time.Now(),
				}
			}
			if err := tx.Create(&userRoles).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// LoadUserRoles loads the user's roles from the database
func (r *GormUserRepository) LoadUserRoles(ctx context.Context, user *identity.User) error {
	var userRoles []identity.UserRole
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", user.ID).
		Find(&userRoles).Error; err != nil {
		return err
	}

	roleIDs := make([]uuid.UUID, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}
	user.RoleIDs = roleIDs

	return nil
}

// Count returns the total number of users for the tenant
func (r *GormUserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&identity.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// applyFilter applies filter options to the query
func (r *GormUserRepository) applyFilter(query *gorm.DB, filter identity.UserFilter) *gorm.DB {
	// Apply keyword search
	if filter.Keyword != "" {
		searchPattern := "%" + filter.Keyword + "%"
		query = query.Where(
			"username ILIKE ? OR email ILIKE ? OR display_name ILIKE ? OR phone ILIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern,
		)
	}

	// Apply status filter
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}

	// Apply role filter
	if filter.RoleID != nil {
		query = query.Joins("JOIN user_roles ON users.id = user_roles.user_id").
			Where("user_roles.role_id = ?", *filter.RoleID)
	}

	return query
}

// Ensure GormUserRepository implements UserRepository
var _ identity.UserRepository = (*GormUserRepository)(nil)
