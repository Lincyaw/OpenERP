package identity

import (
	"context"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// UserService handles user management operations
type UserService struct {
	userRepo identity.UserRepository
	roleRepo identity.RoleRepository
	logger   *zap.Logger
}

// NewUserService creates a new user service
func NewUserService(
	userRepo identity.UserRepository,
	roleRepo identity.RoleRepository,
	logger *zap.Logger,
) *UserService {
	return &UserService{
		userRepo: userRepo,
		roleRepo: roleRepo,
		logger:   logger,
	}
}

// CreateUserInput contains input for creating a user
type CreateUserInput struct {
	TenantID    uuid.UUID
	Username    string
	Password    string
	Email       string
	Phone       string
	DisplayName string
	Notes       string
	RoleIDs     []uuid.UUID
}

// UpdateUserInput contains input for updating a user
type UpdateUserInput struct {
	ID          uuid.UUID
	Email       *string
	Phone       *string
	DisplayName *string
	Notes       *string
}

// UserDTO represents user data transfer object
type UserDTO struct {
	ID          uuid.UUID   `json:"id"`
	TenantID    uuid.UUID   `json:"tenant_id"`
	Username    string      `json:"username"`
	Email       string      `json:"email,omitempty"`
	Phone       string      `json:"phone,omitempty"`
	DisplayName string      `json:"display_name"`
	Avatar      string      `json:"avatar,omitempty"`
	Status      string      `json:"status"`
	RoleIDs     []uuid.UUID `json:"role_ids"`
	LastLoginAt *time.Time  `json:"last_login_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// UserListResult represents paginated user list result
type UserListResult struct {
	Users      []UserDTO `json:"users"`
	Total      int64     `json:"total"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	TotalPages int       `json:"total_pages"`
}

// Create creates a new user
func (s *UserService) Create(ctx context.Context, input CreateUserInput) (*UserDTO, error) {
	s.logger.Info("Creating new user",
		zap.String("username", input.Username),
		zap.String("tenant_id", input.TenantID.String()))

	// Check if username already exists
	exists, err := s.userRepo.ExistsByUsername(ctx, input.Username)
	if err != nil {
		s.logger.Error("Failed to check username existence", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to check username availability")
	}
	if exists {
		return nil, shared.NewDomainError("USERNAME_EXISTS", "Username already exists")
	}

	// Check email uniqueness if provided
	if input.Email != "" {
		exists, err := s.userRepo.ExistsByEmail(ctx, input.Email)
		if err != nil {
			s.logger.Error("Failed to check email existence", zap.Error(err))
			return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to check email availability")
		}
		if exists {
			return nil, shared.NewDomainError("EMAIL_EXISTS", "Email already exists")
		}
	}

	// Validate that all role IDs exist
	if len(input.RoleIDs) > 0 {
		for _, roleID := range input.RoleIDs {
			exists, err := s.roleRepo.ExistsByID(ctx, roleID)
			if err != nil {
				s.logger.Error("Failed to check role existence", zap.Error(err))
				return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to validate roles")
			}
			if !exists {
				return nil, shared.NewDomainError("ROLE_NOT_FOUND", "Role not found: "+roleID.String())
			}
		}
	}

	// Create user - immediately active
	user, err := identity.NewActiveUser(input.TenantID, input.Username, input.Password)
	if err != nil {
		return nil, err
	}

	// Set optional fields
	if input.Email != "" {
		if err := user.SetEmail(input.Email); err != nil {
			return nil, err
		}
	}
	if input.Phone != "" {
		if err := user.SetPhone(input.Phone); err != nil {
			return nil, err
		}
	}
	if input.DisplayName != "" {
		if err := user.SetDisplayName(input.DisplayName); err != nil {
			return nil, err
		}
	}
	if input.Notes != "" {
		user.SetNotes(input.Notes)
	}

	// Assign roles
	for _, roleID := range input.RoleIDs {
		if err := user.AssignRole(roleID); err != nil {
			// Skip duplicate errors silently
			if domainErr, ok := err.(*shared.DomainError); ok && domainErr.Code == "ROLE_ALREADY_ASSIGNED" {
				continue
			}
			return nil, err
		}
	}

	// Save user
	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to create user")
	}

	// Save user roles
	if len(user.RoleIDs) > 0 {
		if err := s.userRepo.SaveUserRoles(ctx, user); err != nil {
			s.logger.Error("Failed to save user roles", zap.Error(err))
			// Delete the user if role assignment failed
			_ = s.userRepo.Delete(ctx, user.ID)
			return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to assign roles to user")
		}
	}

	s.logger.Info("User created successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username))

	return toUserDTO(user), nil
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(ctx context.Context, id uuid.UUID) (*UserDTO, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("USER_NOT_FOUND", "User not found")
		}
		s.logger.Error("Failed to find user", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find user")
	}

	// Load roles
	if err := s.userRepo.LoadUserRoles(ctx, user); err != nil {
		s.logger.Error("Failed to load user roles", zap.Error(err))
	}

	return toUserDTO(user), nil
}

// List retrieves a paginated list of users
func (s *UserService) List(ctx context.Context, filter identity.UserFilter) (*UserListResult, error) {
	users, total, err := s.userRepo.FindAll(ctx, filter)
	if err != nil {
		s.logger.Error("Failed to list users", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to list users")
	}

	// Load roles for each user
	for _, user := range users {
		if err := s.userRepo.LoadUserRoles(ctx, user); err != nil {
			s.logger.Error("Failed to load user roles",
				zap.String("user_id", user.ID.String()),
				zap.Error(err))
		}
	}

	// Calculate total pages
	pageSize := filter.Limit()
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	userDTOs := make([]UserDTO, len(users))
	for i, user := range users {
		userDTOs[i] = *toUserDTO(user)
	}

	return &UserListResult{
		Users:      userDTOs,
		Total:      total,
		Page:       filter.Page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// Update updates a user's information
func (s *UserService) Update(ctx context.Context, input UpdateUserInput) (*UserDTO, error) {
	user, err := s.userRepo.FindByID(ctx, input.ID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("USER_NOT_FOUND", "User not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find user")
	}

	// Update fields
	if input.Email != nil {
		if *input.Email != "" && *input.Email != user.Email {
			// Check email uniqueness
			exists, err := s.userRepo.ExistsByEmail(ctx, *input.Email)
			if err != nil {
				return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to check email availability")
			}
			if exists {
				return nil, shared.NewDomainError("EMAIL_EXISTS", "Email already exists")
			}
		}
		if err := user.SetEmail(*input.Email); err != nil {
			return nil, err
		}
	}

	if input.Phone != nil {
		if err := user.SetPhone(*input.Phone); err != nil {
			return nil, err
		}
	}

	if input.DisplayName != nil {
		if err := user.SetDisplayName(*input.DisplayName); err != nil {
			return nil, err
		}
	}

	if input.Notes != nil {
		user.SetNotes(*input.Notes)
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update user", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to update user")
	}

	// Load roles
	if err := s.userRepo.LoadUserRoles(ctx, user); err != nil {
		s.logger.Error("Failed to load user roles", zap.Error(err))
	}

	s.logger.Info("User updated", zap.String("user_id", input.ID.String()))

	return toUserDTO(user), nil
}

// Delete deletes a user
func (s *UserService) Delete(ctx context.Context, id uuid.UUID) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return shared.NewDomainError("USER_NOT_FOUND", "User not found")
		}
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to find user")
	}

	// Prevent deletion of certain users (e.g., system users)
	// You might want to add a flag like IsSystemUser to the domain model

	if err := s.userRepo.Delete(ctx, user.ID); err != nil {
		s.logger.Error("Failed to delete user", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to delete user")
	}

	s.logger.Info("User deleted", zap.String("user_id", id.String()))

	return nil
}

// Activate activates a user
func (s *UserService) Activate(ctx context.Context, id uuid.UUID) (*UserDTO, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("USER_NOT_FOUND", "User not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find user")
	}

	if err := user.Activate(); err != nil {
		return nil, err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to activate user", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to activate user")
	}

	// Load roles
	if err := s.userRepo.LoadUserRoles(ctx, user); err != nil {
		s.logger.Error("Failed to load user roles", zap.Error(err))
	}

	s.logger.Info("User activated", zap.String("user_id", id.String()))

	return toUserDTO(user), nil
}

// Deactivate deactivates a user
func (s *UserService) Deactivate(ctx context.Context, id uuid.UUID) (*UserDTO, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("USER_NOT_FOUND", "User not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find user")
	}

	if err := user.Deactivate(); err != nil {
		return nil, err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to deactivate user", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to deactivate user")
	}

	// Load roles
	if err := s.userRepo.LoadUserRoles(ctx, user); err != nil {
		s.logger.Error("Failed to load user roles", zap.Error(err))
	}

	s.logger.Info("User deactivated", zap.String("user_id", id.String()))

	return toUserDTO(user), nil
}

// Lock locks a user account
func (s *UserService) Lock(ctx context.Context, id uuid.UUID, duration time.Duration) (*UserDTO, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("USER_NOT_FOUND", "User not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find user")
	}

	if err := user.Lock(duration); err != nil {
		return nil, err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to lock user", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to lock user")
	}

	// Load roles
	if err := s.userRepo.LoadUserRoles(ctx, user); err != nil {
		s.logger.Error("Failed to load user roles", zap.Error(err))
	}

	s.logger.Info("User locked", zap.String("user_id", id.String()))

	return toUserDTO(user), nil
}

// Unlock unlocks a user account
func (s *UserService) Unlock(ctx context.Context, id uuid.UUID) (*UserDTO, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("USER_NOT_FOUND", "User not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find user")
	}

	if err := user.Unlock(); err != nil {
		return nil, err
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to unlock user", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to unlock user")
	}

	// Load roles
	if err := s.userRepo.LoadUserRoles(ctx, user); err != nil {
		s.logger.Error("Failed to load user roles", zap.Error(err))
	}

	s.logger.Info("User unlocked", zap.String("user_id", id.String()))

	return toUserDTO(user), nil
}

// ResetPassword resets a user's password (admin action)
func (s *UserService) ResetPassword(ctx context.Context, userID uuid.UUID, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if err == shared.ErrNotFound {
			return shared.NewDomainError("USER_NOT_FOUND", "User not found")
		}
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to find user")
	}

	if err := user.SetPassword(newPassword); err != nil {
		return err
	}

	// Force password change on next login
	user.ForcePasswordChange()

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to reset password", zap.Error(err))
		return shared.NewDomainError("INTERNAL_ERROR", "Failed to reset password")
	}

	s.logger.Info("User password reset", zap.String("user_id", userID.String()))

	return nil
}

// AssignRoles assigns roles to a user (replaces existing roles)
func (s *UserService) AssignRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) (*UserDTO, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		if err == shared.ErrNotFound {
			return nil, shared.NewDomainError("USER_NOT_FOUND", "User not found")
		}
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to find user")
	}

	// Validate that all role IDs exist
	for _, roleID := range roleIDs {
		exists, err := s.roleRepo.ExistsByID(ctx, roleID)
		if err != nil {
			s.logger.Error("Failed to check role existence", zap.Error(err))
			return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to validate roles")
		}
		if !exists {
			return nil, shared.NewDomainError("ROLE_NOT_FOUND", "Role not found: "+roleID.String())
		}
	}

	// Set new roles
	if err := user.SetRoles(roleIDs); err != nil {
		return nil, err
	}

	// Save user roles
	if err := s.userRepo.SaveUserRoles(ctx, user); err != nil {
		s.logger.Error("Failed to save user roles", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to assign roles")
	}

	// Update user version
	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update user", zap.Error(err))
		return nil, shared.NewDomainError("INTERNAL_ERROR", "Failed to update user")
	}

	s.logger.Info("User roles assigned",
		zap.String("user_id", userID.String()),
		zap.Int("role_count", len(roleIDs)))

	return toUserDTO(user), nil
}

// Count returns the total number of users
func (s *UserService) Count(ctx context.Context) (int64, error) {
	return s.userRepo.Count(ctx)
}

// toUserDTO converts domain User to UserDTO
func toUserDTO(user *identity.User) *UserDTO {
	return &UserDTO{
		ID:          user.ID,
		TenantID:    user.TenantID,
		Username:    user.Username,
		Email:       user.Email,
		Phone:       user.Phone,
		DisplayName: user.GetDisplayNameOrUsername(),
		Avatar:      user.Avatar,
		Status:      string(user.Status),
		RoleIDs:     user.RoleIDs,
		LastLoginAt: user.LastLoginAt,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}
