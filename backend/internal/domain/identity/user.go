package identity

import (
	"regexp"
	"strings"
	"time"

	"github.com/erp/backend/internal/domain/shared"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// UserStatus represents the status of a user
type UserStatus string

const (
	UserStatusPending     UserStatus = "pending"     // Awaiting activation
	UserStatusActive      UserStatus = "active"      // Normal active status
	UserStatusLocked      UserStatus = "locked"      // Locked due to failed attempts/security
	UserStatusDeactivated UserStatus = "deactivated" // Manually deactivated
)

// Password cost for bcrypt
const bcryptCost = 12

// User represents a user in the system
// It is the aggregate root for user-related operations
type User struct {
	shared.TenantAggregateRoot
	Username           string
	Email              string
	Phone              string
	PasswordHash       string
	DisplayName        string
	Avatar             string
	Status             UserStatus
	DepartmentID       *uuid.UUID  // Primary department the user belongs to
	RoleIDs            []uuid.UUID // Stored in separate table, loaded by repository
	LastLoginAt        *time.Time
	LastLoginIP        string // IPv6 max length
	FailedAttempts     int
	LockedUntil        *time.Time
	PasswordChangedAt  *time.Time
	MustChangePassword bool
	Notes              string
}

// UserRole represents the many-to-many relationship between users and roles
type UserRole struct {
	UserID    uuid.UUID
	RoleID    uuid.UUID
	TenantID  uuid.UUID
	CreatedAt time.Time
}

// NewUser creates a new user with required fields
func NewUser(tenantID uuid.UUID, username, password string) (*User, error) {
	if err := validateUsername(username); err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, shared.NewDomainError("PASSWORD_HASH_ERROR", "Failed to hash password")
	}

	now := time.Now()
	user := &User{
		TenantAggregateRoot: shared.NewTenantAggregateRoot(tenantID),
		Username:            strings.ToLower(strings.TrimSpace(username)),
		PasswordHash:        passwordHash,
		Status:              UserStatusPending,
		RoleIDs:             make([]uuid.UUID, 0),
		PasswordChangedAt:   &now,
	}

	user.AddDomainEvent(NewUserCreatedEvent(user))

	return user, nil
}

// NewActiveUser creates a new user that is immediately active
func NewActiveUser(tenantID uuid.UUID, username, password string) (*User, error) {
	user, err := NewUser(tenantID, username, password)
	if err != nil {
		return nil, err
	}

	user.Status = UserStatusActive
	return user, nil
}

// SetEmail sets the user's email
func (u *User) SetEmail(email string) error {
	if email != "" {
		if err := validateEmail(email); err != nil {
			return err
		}
		email = strings.ToLower(strings.TrimSpace(email))
	}

	u.Email = email
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	return nil
}

// SetPhone sets the user's phone number
func (u *User) SetPhone(phone string) error {
	if phone != "" && len(phone) > 50 {
		return shared.NewDomainError("INVALID_PHONE", "Phone cannot exceed 50 characters")
	}

	u.Phone = strings.TrimSpace(phone)
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	return nil
}

// SetDisplayName sets the user's display name
func (u *User) SetDisplayName(displayName string) error {
	if displayName != "" && len(displayName) > 200 {
		return shared.NewDomainError("INVALID_DISPLAY_NAME", "Display name cannot exceed 200 characters")
	}

	u.DisplayName = strings.TrimSpace(displayName)
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	return nil
}

// SetAvatar sets the user's avatar URL
func (u *User) SetAvatar(avatar string) error {
	if avatar != "" && len(avatar) > 500 {
		return shared.NewDomainError("INVALID_AVATAR", "Avatar URL cannot exceed 500 characters")
	}

	u.Avatar = avatar
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	return nil
}

// SetNotes sets the user's notes
func (u *User) SetNotes(notes string) {
	u.Notes = notes
	u.UpdatedAt = time.Now()
	u.IncrementVersion()
}

// SetDepartment sets the user's department
func (u *User) SetDepartment(departmentID *uuid.UUID) {
	u.DepartmentID = departmentID
	u.UpdatedAt = time.Now()
	u.IncrementVersion()
}

// ChangePassword changes the user's password
func (u *User) ChangePassword(oldPassword, newPassword string) error {
	// Verify old password
	if !u.VerifyPassword(oldPassword) {
		return shared.NewDomainError("INVALID_PASSWORD", "Current password is incorrect")
	}

	return u.SetPassword(newPassword)
}

// SetPassword sets a new password (admin reset, no old password check)
func (u *User) SetPassword(newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}

	passwordHash, err := hashPassword(newPassword)
	if err != nil {
		return shared.NewDomainError("PASSWORD_HASH_ERROR", "Failed to hash password")
	}

	u.PasswordHash = passwordHash
	now := time.Now()
	u.PasswordChangedAt = &now
	u.MustChangePassword = false
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	u.AddDomainEvent(NewUserPasswordChangedEvent(u))

	return nil
}

// ForcePasswordChange marks that user must change password on next login
func (u *User) ForcePasswordChange() {
	u.MustChangePassword = true
	u.UpdatedAt = time.Now()
	u.IncrementVersion()
}

// VerifyPassword verifies if the provided password matches
func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// AssignRole assigns a role to the user
func (u *User) AssignRole(roleID uuid.UUID) error {
	if roleID == uuid.Nil {
		return shared.NewDomainError("INVALID_ROLE_ID", "Role ID cannot be empty")
	}

	// Check if already has the role
	for _, rid := range u.RoleIDs {
		if rid == roleID {
			return shared.NewDomainError("ROLE_ALREADY_ASSIGNED", "User already has this role")
		}
	}

	u.RoleIDs = append(u.RoleIDs, roleID)
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	u.AddDomainEvent(NewUserRoleAssignedEvent(u, roleID))

	return nil
}

// RemoveRole removes a role from the user
func (u *User) RemoveRole(roleID uuid.UUID) error {
	if roleID == uuid.Nil {
		return shared.NewDomainError("INVALID_ROLE_ID", "Role ID cannot be empty")
	}

	found := false
	newRoleIDs := make([]uuid.UUID, 0, len(u.RoleIDs))
	for _, rid := range u.RoleIDs {
		if rid != roleID {
			newRoleIDs = append(newRoleIDs, rid)
		} else {
			found = true
		}
	}

	if !found {
		return shared.NewDomainError("ROLE_NOT_ASSIGNED", "User does not have this role")
	}

	u.RoleIDs = newRoleIDs
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	u.AddDomainEvent(NewUserRoleRemovedEvent(u, roleID))

	return nil
}

// SetRoles sets all roles for the user (replaces existing roles)
func (u *User) SetRoles(roleIDs []uuid.UUID) error {
	for _, rid := range roleIDs {
		if rid == uuid.Nil {
			return shared.NewDomainError("INVALID_ROLE_ID", "Role ID cannot be empty")
		}
	}

	// Deduplicate
	seen := make(map[uuid.UUID]bool)
	uniqueRoles := make([]uuid.UUID, 0, len(roleIDs))
	for _, rid := range roleIDs {
		if !seen[rid] {
			seen[rid] = true
			uniqueRoles = append(uniqueRoles, rid)
		}
	}

	u.RoleIDs = uniqueRoles
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	return nil
}

// HasRole checks if user has a specific role
func (u *User) HasRole(roleID uuid.UUID) bool {
	for _, rid := range u.RoleIDs {
		if rid == roleID {
			return true
		}
	}
	return false
}

// Activate activates the user
func (u *User) Activate() error {
	if u.Status == UserStatusActive {
		return shared.NewDomainError("ALREADY_ACTIVE", "User is already active")
	}

	oldStatus := u.Status
	u.Status = UserStatusActive
	u.FailedAttempts = 0
	u.LockedUntil = nil
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	u.AddDomainEvent(NewUserStatusChangedEvent(u, oldStatus, UserStatusActive))

	return nil
}

// Deactivate deactivates the user
func (u *User) Deactivate() error {
	if u.Status == UserStatusDeactivated {
		return shared.NewDomainError("ALREADY_DEACTIVATED", "User is already deactivated")
	}

	oldStatus := u.Status
	u.Status = UserStatusDeactivated
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	u.AddDomainEvent(NewUserDeactivatedEvent(u))
	u.AddDomainEvent(NewUserStatusChangedEvent(u, oldStatus, UserStatusDeactivated))

	return nil
}

// Lock locks the user account
func (u *User) Lock(duration time.Duration) error {
	if u.Status == UserStatusDeactivated {
		return shared.NewDomainError("USER_DEACTIVATED", "Cannot lock a deactivated user")
	}

	oldStatus := u.Status
	u.Status = UserStatusLocked
	if duration > 0 {
		lockedUntil := time.Now().Add(duration)
		u.LockedUntil = &lockedUntil
	}
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	u.AddDomainEvent(NewUserStatusChangedEvent(u, oldStatus, UserStatusLocked))

	return nil
}

// Unlock unlocks the user account
func (u *User) Unlock() error {
	if u.Status != UserStatusLocked {
		return shared.NewDomainError("NOT_LOCKED", "User is not locked")
	}

	u.Status = UserStatusActive
	u.FailedAttempts = 0
	u.LockedUntil = nil
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	u.AddDomainEvent(NewUserStatusChangedEvent(u, UserStatusLocked, UserStatusActive))

	return nil
}

// RecordLoginSuccess records a successful login
func (u *User) RecordLoginSuccess(ip string) {
	now := time.Now()
	u.LastLoginAt = &now
	u.LastLoginIP = ip
	u.FailedAttempts = 0
	u.UpdatedAt = time.Now()
	u.IncrementVersion()
}

// RecordLoginFailure records a failed login attempt
// Returns true if account should be locked
func (u *User) RecordLoginFailure(maxAttempts int, lockDuration time.Duration) bool {
	u.FailedAttempts++
	u.UpdatedAt = time.Now()
	u.IncrementVersion()

	if u.FailedAttempts >= maxAttempts {
		_ = u.Lock(lockDuration)
		return true
	}

	return false
}

// IsActive returns true if user is active
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsLocked returns true if user is locked
func (u *User) IsLocked() bool {
	if u.Status != UserStatusLocked {
		return false
	}

	// Check if lock has expired
	if u.LockedUntil != nil && time.Now().After(*u.LockedUntil) {
		return false
	}

	return true
}

// IsDeactivated returns true if user is deactivated
func (u *User) IsDeactivated() bool {
	return u.Status == UserStatusDeactivated
}

// IsPending returns true if user is pending activation
func (u *User) IsPending() bool {
	return u.Status == UserStatusPending
}

// CanLogin returns true if user can login
func (u *User) CanLogin() bool {
	if u.Status == UserStatusDeactivated {
		return false
	}
	if u.Status == UserStatusPending {
		return false
	}
	if u.IsLocked() {
		return false
	}
	return true
}

// GetDisplayNameOrUsername returns display name if set, otherwise username
func (u *User) GetDisplayNameOrUsername() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	return u.Username
}

// Validation functions

func validateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return shared.NewDomainError("INVALID_USERNAME", "Username cannot be empty")
	}
	if len(username) < 3 {
		return shared.NewDomainError("INVALID_USERNAME", "Username must be at least 3 characters")
	}
	if len(username) > 100 {
		return shared.NewDomainError("INVALID_USERNAME", "Username cannot exceed 100 characters")
	}

	// Allow alphanumeric, underscore, hyphen, and dot
	usernameRegex := regexp.MustCompile(`^[a-zA-Z0-9_\-.]+$`)
	if !usernameRegex.MatchString(username) {
		return shared.NewDomainError("INVALID_USERNAME", "Username can only contain letters, numbers, underscores, hyphens, and dots")
	}

	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return shared.NewDomainError("INVALID_PASSWORD", "Password cannot be empty")
	}
	if len(password) < 8 {
		return shared.NewDomainError("INVALID_PASSWORD", "Password must be at least 8 characters")
	}
	if len(password) > 128 {
		return shared.NewDomainError("INVALID_PASSWORD", "Password cannot exceed 128 characters")
	}

	// Check for at least one letter and one number
	hasLetter := regexp.MustCompile(`[a-zA-Z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	if !hasLetter || !hasNumber {
		return shared.NewDomainError("INVALID_PASSWORD", "Password must contain at least one letter and one number")
	}

	return nil
}

func validateEmail(email string) error {
	if len(email) > 200 {
		return shared.NewDomainError("INVALID_EMAIL", "Email cannot exceed 200 characters")
	}

	// Basic email validation
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return shared.NewDomainError("INVALID_EMAIL", "Invalid email format")
	}

	return nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
