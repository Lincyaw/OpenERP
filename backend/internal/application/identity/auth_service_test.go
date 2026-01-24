package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/domain/shared"
	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockUserRepository is a mock implementation of identity.UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *identity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *identity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) FindByUsername(ctx context.Context, username string) (*identity.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*identity.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) FindByPhone(ctx context.Context, phone string) (*identity.User, error) {
	args := m.Called(ctx, phone)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.User), args.Error(1)
}

func (m *MockUserRepository) FindAll(ctx context.Context, filter identity.UserFilter) ([]*identity.User, int64, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).([]*identity.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserRepository) FindByRoleID(ctx context.Context, roleID uuid.UUID) ([]*identity.User, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).([]*identity.User), args.Error(1)
}

func (m *MockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserRepository) SaveUserRoles(ctx context.Context, user *identity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) LoadUserRoles(ctx context.Context, user *identity.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Count(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// MockRoleRepository is a mock implementation of identity.RoleRepository
type MockRoleRepository struct {
	mock.Mock
}

func (m *MockRoleRepository) Create(ctx context.Context, role *identity.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) Update(ctx context.Context, role *identity.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRoleRepository) FindByID(ctx context.Context, id uuid.UUID) (*identity.Role, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Role), args.Error(1)
}

func (m *MockRoleRepository) FindByCode(ctx context.Context, tenantID uuid.UUID, code string) (*identity.Role, error) {
	args := m.Called(ctx, tenantID, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*identity.Role), args.Error(1)
}

func (m *MockRoleRepository) FindAll(ctx context.Context, tenantID uuid.UUID, filter *identity.RoleFilter) ([]*identity.Role, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).([]*identity.Role), args.Error(1)
}

func (m *MockRoleRepository) Count(ctx context.Context, tenantID uuid.UUID, filter *identity.RoleFilter) (int64, error) {
	args := m.Called(ctx, tenantID, filter)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRoleRepository) ExistsByCode(ctx context.Context, tenantID uuid.UUID, code string) (bool, error) {
	args := m.Called(ctx, tenantID, code)
	return args.Bool(0), args.Error(1)
}

func (m *MockRoleRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

func (m *MockRoleRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*identity.Role, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*identity.Role), args.Error(1)
}

func (m *MockRoleRepository) FindSystemRoles(ctx context.Context, tenantID uuid.UUID) ([]*identity.Role, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*identity.Role), args.Error(1)
}

func (m *MockRoleRepository) SavePermissions(ctx context.Context, role *identity.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) LoadPermissions(ctx context.Context, role *identity.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) SaveDataScopes(ctx context.Context, role *identity.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) LoadDataScopes(ctx context.Context, role *identity.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) LoadPermissionsAndDataScopes(ctx context.Context, role *identity.Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *MockRoleRepository) FindUsersWithRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockRoleRepository) CountUsersWithRole(ctx context.Context, roleID uuid.UUID) (int64, error) {
	args := m.Called(ctx, roleID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRoleRepository) FindRolesWithPermission(ctx context.Context, tenantID uuid.UUID, permissionCode string) ([]*identity.Role, error) {
	args := m.Called(ctx, tenantID, permissionCode)
	return args.Get(0).([]*identity.Role), args.Error(1)
}

func (m *MockRoleRepository) GetAllPermissionCodes(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]string), args.Error(1)
}

// Helper function to create a test user
func createTestUser(tenantID uuid.UUID) *identity.User {
	user, _ := identity.NewActiveUser(tenantID, "testuser", "Password123")
	return user
}

// Helper function to create a test role
func createTestRole(tenantID uuid.UUID) *identity.Role {
	role, _ := identity.NewRole(tenantID, "TEST_ROLE", "Test Role")
	perm, _ := identity.NewPermission("product", "read")
	role.GrantPermission(*perm)
	return role
}

// Helper function to create auth service
func createAuthService(userRepo *MockUserRepository, roleRepo *MockRoleRepository) *AuthService {
	jwtCfg := config.JWTConfig{
		Secret:                 "test-secret-key-32-characters-long",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	jwtService := auth.NewJWTService(jwtCfg)
	logger := zap.NewNop()

	return NewAuthService(
		userRepo,
		roleRepo,
		jwtService,
		DefaultAuthServiceConfig(),
		logger,
	)
}

func TestAuthService_Login_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUser(tenantID)
	role := createTestRole(tenantID)
	user.RoleIDs = []uuid.UUID{role.ID}

	userRepo.On("FindByUsername", ctx, "testuser").Return(user, nil)
	userRepo.On("LoadUserRoles", ctx, user).Return(nil)
	userRepo.On("Update", ctx, user).Return(nil)
	roleRepo.On("FindByIDs", ctx, user.RoleIDs).Return([]*identity.Role{role}, nil)
	roleRepo.On("LoadPermissions", ctx, role).Return(nil)

	authService := createAuthService(userRepo, roleRepo)

	result, err := authService.Login(ctx, LoginInput{
		Username: "testuser",
		Password: "Password123",
		IP:       "127.0.0.1",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.AccessToken)
	assert.NotEmpty(t, result.RefreshToken)
	assert.Equal(t, "testuser", result.User.Username)
	assert.Equal(t, tenantID, result.User.TenantID)
	assert.Equal(t, "Bearer", result.TokenType)

	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
}

func TestAuthService_Login_InvalidCredentials(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUser(tenantID)

	userRepo.On("FindByUsername", ctx, "testuser").Return(user, nil)
	userRepo.On("Update", ctx, user).Return(nil)

	authService := createAuthService(userRepo, roleRepo)

	result, err := authService.Login(ctx, LoginInput{
		Username: "testuser",
		Password: "wrongpassword",
		IP:       "127.0.0.1",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	require.True(t, errors.As(err, &domainErr))
	assert.Equal(t, "INVALID_CREDENTIALS", domainErr.Code)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	ctx := context.Background()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	userRepo.On("FindByUsername", ctx, "nonexistent").Return(nil, errors.New("user not found"))

	authService := createAuthService(userRepo, roleRepo)

	result, err := authService.Login(ctx, LoginInput{
		Username: "nonexistent",
		Password: "Password123",
		IP:       "127.0.0.1",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	require.True(t, errors.As(err, &domainErr))
	assert.Equal(t, "INVALID_CREDENTIALS", domainErr.Code)
}

func TestAuthService_Login_LockedAccount(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUser(tenantID)
	user.Lock(1 * time.Hour)

	userRepo.On("FindByUsername", ctx, "testuser").Return(user, nil)

	authService := createAuthService(userRepo, roleRepo)

	result, err := authService.Login(ctx, LoginInput{
		Username: "testuser",
		Password: "Password123",
		IP:       "127.0.0.1",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	require.True(t, errors.As(err, &domainErr))
	assert.Equal(t, "ACCOUNT_LOCKED", domainErr.Code)
}

func TestAuthService_Login_DeactivatedAccount(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUser(tenantID)
	user.Deactivate()

	userRepo.On("FindByUsername", ctx, "testuser").Return(user, nil)

	authService := createAuthService(userRepo, roleRepo)

	result, err := authService.Login(ctx, LoginInput{
		Username: "testuser",
		Password: "Password123",
		IP:       "127.0.0.1",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	require.True(t, errors.As(err, &domainErr))
	assert.Equal(t, "ACCOUNT_DEACTIVATED", domainErr.Code)
}

func TestAuthService_RefreshToken_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUser(tenantID)
	role := createTestRole(tenantID)
	user.RoleIDs = []uuid.UUID{role.ID}

	// First login to get a refresh token
	userRepo.On("FindByUsername", ctx, "testuser").Return(user, nil)
	userRepo.On("LoadUserRoles", ctx, user).Return(nil)
	userRepo.On("Update", ctx, user).Return(nil)
	roleRepo.On("FindByIDs", ctx, user.RoleIDs).Return([]*identity.Role{role}, nil)
	roleRepo.On("LoadPermissions", ctx, role).Return(nil)

	authService := createAuthService(userRepo, roleRepo)

	loginResult, err := authService.Login(ctx, LoginInput{
		Username: "testuser",
		Password: "Password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)

	// Now refresh the token
	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)

	refreshResult, err := authService.RefreshToken(ctx, RefreshTokenInput{
		RefreshToken: loginResult.RefreshToken,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, refreshResult.AccessToken)
	assert.NotEmpty(t, refreshResult.RefreshToken)
	assert.Equal(t, "Bearer", refreshResult.TokenType)
	// New tokens should be different
	assert.NotEqual(t, loginResult.AccessToken, refreshResult.AccessToken)
}

func TestAuthService_RefreshToken_InvalidToken(t *testing.T) {
	ctx := context.Background()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	authService := createAuthService(userRepo, roleRepo)

	result, err := authService.RefreshToken(ctx, RefreshTokenInput{
		RefreshToken: "invalid-token",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	require.True(t, errors.As(err, &domainErr))
	assert.Equal(t, "TOKEN_INVALID", domainErr.Code)
}

func TestAuthService_RefreshToken_UserNotFound(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUser(tenantID)
	role := createTestRole(tenantID)
	user.RoleIDs = []uuid.UUID{role.ID}

	// First login to get a refresh token
	userRepo.On("FindByUsername", ctx, "testuser").Return(user, nil)
	userRepo.On("LoadUserRoles", ctx, user).Return(nil)
	userRepo.On("Update", ctx, user).Return(nil)
	roleRepo.On("FindByIDs", ctx, user.RoleIDs).Return([]*identity.Role{role}, nil)
	roleRepo.On("LoadPermissions", ctx, role).Return(nil)

	authService := createAuthService(userRepo, roleRepo)

	loginResult, err := authService.Login(ctx, LoginInput{
		Username: "testuser",
		Password: "Password123",
		IP:       "127.0.0.1",
	})
	require.NoError(t, err)

	// User deleted
	userRepo.On("FindByID", ctx, user.ID).Return(nil, errors.New("user not found"))

	result, err := authService.RefreshToken(ctx, RefreshTokenInput{
		RefreshToken: loginResult.RefreshToken,
	})

	require.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	require.True(t, errors.As(err, &domainErr))
	assert.Equal(t, "USER_NOT_FOUND", domainErr.Code)
}

func TestAuthService_GetCurrentUser_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUser(tenantID)
	role := createTestRole(tenantID)
	user.RoleIDs = []uuid.UUID{role.ID}

	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)
	userRepo.On("LoadUserRoles", ctx, user).Return(nil)
	roleRepo.On("FindByIDs", ctx, user.RoleIDs).Return([]*identity.Role{role}, nil)
	roleRepo.On("LoadPermissions", ctx, role).Return(nil)

	authService := createAuthService(userRepo, roleRepo)

	result, err := authService.GetCurrentUser(ctx, GetCurrentUserInput{
		UserID:   user.ID,
		TenantID: tenantID,
	})

	require.NoError(t, err)
	assert.Equal(t, user.ID, result.User.ID)
	assert.Equal(t, user.Username, result.User.Username)
	assert.NotEmpty(t, result.Permissions)

	userRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
}

func TestAuthService_ChangePassword_Success(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUser(tenantID)

	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)
	userRepo.On("Update", ctx, mock.Anything).Return(nil)

	authService := createAuthService(userRepo, roleRepo)

	err := authService.ChangePassword(ctx, ChangePasswordInput{
		UserID:      user.ID,
		OldPassword: "Password123",
		NewPassword: "NewPassword456",
	})

	require.NoError(t, err)
	userRepo.AssertExpectations(t)
}

func TestAuthService_ChangePassword_WrongOldPassword(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUser(tenantID)

	userRepo.On("FindByID", ctx, user.ID).Return(user, nil)

	authService := createAuthService(userRepo, roleRepo)

	err := authService.ChangePassword(ctx, ChangePasswordInput{
		UserID:      user.ID,
		OldPassword: "wrongpassword",
		NewPassword: "NewPassword456",
	})

	require.Error(t, err)
	var domainErr *shared.DomainError
	require.True(t, errors.As(err, &domainErr))
	assert.Equal(t, "INVALID_PASSWORD", domainErr.Code)
}

func TestAuthService_Logout_Success(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	authService := createAuthService(userRepo, roleRepo)

	err := authService.Logout(ctx, LogoutInput{
		UserID:   userID,
		TenantID: tenantID,
		TokenJTI: "some-jti",
	})

	require.NoError(t, err)
}

func TestAuthService_Login_AccountLocksAfterMaxAttempts(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUser(tenantID)
	user.FailedAttempts = 4 // One more failure will lock

	userRepo.On("FindByUsername", ctx, "testuser").Return(user, nil)
	userRepo.On("Update", ctx, mock.Anything).Return(nil)

	authService := createAuthService(userRepo, roleRepo)

	result, err := authService.Login(ctx, LoginInput{
		Username: "testuser",
		Password: "wrongpassword",
		IP:       "127.0.0.1",
	})

	require.Error(t, err)
	assert.Nil(t, result)
	var domainErr *shared.DomainError
	require.True(t, errors.As(err, &domainErr))
	assert.Equal(t, "ACCOUNT_LOCKED", domainErr.Code)
}
