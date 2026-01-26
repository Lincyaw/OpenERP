package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appidentity "github.com/erp/backend/internal/application/identity"
	"github.com/erp/backend/internal/domain/identity"
	"github.com/erp/backend/internal/infrastructure/auth"
	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// testCookieConfig returns a default cookie config for tests
func testCookieConfig() config.CookieConfig {
	return config.CookieConfig{
		Domain:   "",
		Path:     "/",
		Secure:   false,
		SameSite: "lax",
	}
}

// testJWTConfig returns a default JWT config for tests
func testJWTConfig() config.JWTConfig {
	return config.JWTConfig{
		Secret:                 "test-secret-key-32-characters-long",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
}

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

func setupAuthRouter(handler *AuthHandler, jwtService *auth.JWTService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Auth routes (no JWT required for login/refresh)
	authGroup := r.Group("/api/v1/auth")
	{
		authGroup.POST("/login", handler.Login)
		authGroup.POST("/refresh", handler.RefreshToken)
	}

	// Protected auth routes (JWT required)
	protectedGroup := r.Group("/api/v1/auth")
	protectedGroup.Use(middleware.JWTAuthMiddleware(jwtService))
	{
		protectedGroup.POST("/logout", handler.Logout)
		protectedGroup.GET("/me", handler.GetCurrentUser)
		protectedGroup.PUT("/password", handler.ChangePassword)
	}

	return r
}

func createTestUserForHandler(tenantID uuid.UUID) *identity.User {
	user, _ := identity.NewActiveUser(tenantID, "testuser", "Password123")
	return user
}

func createTestRoleForHandler(tenantID uuid.UUID) *identity.Role {
	role, _ := identity.NewRole(tenantID, "TEST_ROLE", "Test Role")
	perm, _ := identity.NewPermission("product", "read")
	role.GrantPermission(*perm)
	return role
}

func createAuthServiceForHandler(userRepo *MockUserRepository, roleRepo *MockRoleRepository, jwtService *auth.JWTService) *appidentity.AuthService {
	logger := zap.NewNop()
	return appidentity.NewAuthService(
		userRepo,
		roleRepo,
		jwtService,
		appidentity.DefaultAuthServiceConfig(),
		logger,
	)
}

func TestAuthHandler_Login_Success(t *testing.T) {
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUserForHandler(tenantID)
	role := createTestRoleForHandler(tenantID)
	user.RoleIDs = []uuid.UUID{role.ID}

	userRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
	userRepo.On("LoadUserRoles", mock.Anything, user).Return(nil)
	userRepo.On("Update", mock.Anything, user).Return(nil)
	roleRepo.On("FindByIDs", mock.Anything, user.RoleIDs).Return([]*identity.Role{role}, nil)
	roleRepo.On("LoadPermissions", mock.Anything, role).Return(nil)

	jwtCfg := config.JWTConfig{
		Secret:                 "test-secret-key-32-characters-long",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	jwtService := auth.NewJWTService(jwtCfg)

	authService := createAuthServiceForHandler(userRepo, roleRepo, jwtService)
	handler := NewAuthHandler(authService, testCookieConfig(), testJWTConfig())
	router := setupAuthRouter(handler, jwtService)

	reqBody := LoginRequest{
		Username: "testuser",
		Password: "Password123",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	token := data["token"].(map[string]interface{})
	assert.NotEmpty(t, token["access_token"])
	// Refresh token is now in httpOnly cookie, not in response body (SEC-004)
	assert.Empty(t, token["refresh_token"], "refresh_token should be empty in response body (stored in httpOnly cookie)")
	assert.Equal(t, "Bearer", token["token_type"])

	// Verify refresh token is set as httpOnly cookie
	cookies := w.Result().Cookies()
	var refreshTokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			refreshTokenCookie = c
			break
		}
	}
	require.NotNil(t, refreshTokenCookie, "refresh_token cookie should be set")
	assert.NotEmpty(t, refreshTokenCookie.Value)
	assert.True(t, refreshTokenCookie.HttpOnly, "cookie should be httpOnly")

	userData := data["user"].(map[string]interface{})
	assert.Equal(t, "testuser", userData["username"])
}

func TestAuthHandler_Login_InvalidRequestBody(t *testing.T) {
	jwtCfg := config.JWTConfig{
		Secret:                 "test-secret-key-32-characters-long",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	jwtService := auth.NewJWTService(jwtCfg)

	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)
	authService := createAuthServiceForHandler(userRepo, roleRepo, jwtService)
	handler := NewAuthHandler(authService, testCookieConfig(), testJWTConfig())
	router := setupAuthRouter(handler, jwtService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUserForHandler(tenantID)
	role := createTestRoleForHandler(tenantID)
	user.RoleIDs = []uuid.UUID{role.ID}

	userRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
	userRepo.On("LoadUserRoles", mock.Anything, user).Return(nil)
	userRepo.On("Update", mock.Anything, user).Return(nil)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	roleRepo.On("FindByIDs", mock.Anything, user.RoleIDs).Return([]*identity.Role{role}, nil)
	roleRepo.On("LoadPermissions", mock.Anything, role).Return(nil)

	jwtCfg := config.JWTConfig{
		Secret:                 "test-secret-key-32-characters-long",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	jwtService := auth.NewJWTService(jwtCfg)

	authService := createAuthServiceForHandler(userRepo, roleRepo, jwtService)
	handler := NewAuthHandler(authService, testCookieConfig(), testJWTConfig())
	router := setupAuthRouter(handler, jwtService)

	// First login
	loginReq := LoginRequest{
		Username: "testuser",
		Password: "Password123",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHttpReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginHttpReq)
	require.Equal(t, http.StatusOK, loginW.Code)

	// Get refresh token from httpOnly cookie (SEC-004)
	var refreshTokenCookie *http.Cookie
	for _, c := range loginW.Result().Cookies() {
		if c.Name == "refresh_token" {
			refreshTokenCookie = c
			break
		}
	}
	require.NotNil(t, refreshTokenCookie, "refresh_token cookie should be set after login")

	// Now refresh - send cookie with the request (SEC-004)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(refreshTokenCookie) // Send refresh token via cookie

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	token := data["token"].(map[string]interface{})
	assert.NotEmpty(t, token["access_token"])
	// Refresh token is now in httpOnly cookie, not in response body
	assert.Empty(t, token["refresh_token"], "refresh_token should be empty in response body")

	// Verify new refresh token cookie is set
	var newRefreshTokenCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "refresh_token" {
			newRefreshTokenCookie = c
			break
		}
	}
	require.NotNil(t, newRefreshTokenCookie, "new refresh_token cookie should be set")
	assert.NotEmpty(t, newRefreshTokenCookie.Value)
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUserForHandler(tenantID)
	role := createTestRoleForHandler(tenantID)
	user.RoleIDs = []uuid.UUID{role.ID}

	userRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
	userRepo.On("LoadUserRoles", mock.Anything, user).Return(nil)
	userRepo.On("Update", mock.Anything, user).Return(nil)
	roleRepo.On("FindByIDs", mock.Anything, user.RoleIDs).Return([]*identity.Role{role}, nil)
	roleRepo.On("LoadPermissions", mock.Anything, role).Return(nil)

	jwtCfg := config.JWTConfig{
		Secret:                 "test-secret-key-32-characters-long",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	jwtService := auth.NewJWTService(jwtCfg)

	authService := createAuthServiceForHandler(userRepo, roleRepo, jwtService)
	handler := NewAuthHandler(authService, testCookieConfig(), testJWTConfig())
	router := setupAuthRouter(handler, jwtService)

	// First login
	loginReq := LoginRequest{
		Username: "testuser",
		Password: "Password123",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHttpReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginHttpReq)
	require.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	loginData := loginResponse["data"].(map[string]interface{})
	loginToken := loginData["token"].(map[string]interface{})
	accessToken := loginToken["access_token"].(string)

	// Now logout
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Logged out successfully", data["message"])
}

func TestAuthHandler_Logout_Unauthorized(t *testing.T) {
	jwtCfg := config.JWTConfig{
		Secret:                 "test-secret-key-32-characters-long",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	jwtService := auth.NewJWTService(jwtCfg)

	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)
	authService := createAuthServiceForHandler(userRepo, roleRepo, jwtService)
	handler := NewAuthHandler(authService, testCookieConfig(), testJWTConfig())
	router := setupAuthRouter(handler, jwtService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandler_GetCurrentUser_Success(t *testing.T) {
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUserForHandler(tenantID)
	role := createTestRoleForHandler(tenantID)
	user.RoleIDs = []uuid.UUID{role.ID}

	userRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
	userRepo.On("LoadUserRoles", mock.Anything, user).Return(nil)
	userRepo.On("Update", mock.Anything, user).Return(nil)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	roleRepo.On("FindByIDs", mock.Anything, user.RoleIDs).Return([]*identity.Role{role}, nil)
	roleRepo.On("LoadPermissions", mock.Anything, role).Return(nil)

	jwtCfg := config.JWTConfig{
		Secret:                 "test-secret-key-32-characters-long",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	jwtService := auth.NewJWTService(jwtCfg)

	authService := createAuthServiceForHandler(userRepo, roleRepo, jwtService)
	handler := NewAuthHandler(authService, testCookieConfig(), testJWTConfig())
	router := setupAuthRouter(handler, jwtService)

	// First login
	loginReq := LoginRequest{
		Username: "testuser",
		Password: "Password123",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHttpReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginHttpReq)
	require.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	loginData := loginResponse["data"].(map[string]interface{})
	loginToken := loginData["token"].(map[string]interface{})
	accessToken := loginToken["access_token"].(string)

	// Get current user
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	data := response["data"].(map[string]interface{})
	userData := data["user"].(map[string]interface{})
	assert.Equal(t, "testuser", userData["username"])
}

func TestAuthHandler_ChangePassword_Success(t *testing.T) {
	tenantID := uuid.New()
	userRepo := new(MockUserRepository)
	roleRepo := new(MockRoleRepository)

	user := createTestUserForHandler(tenantID)
	role := createTestRoleForHandler(tenantID)
	user.RoleIDs = []uuid.UUID{role.ID}

	userRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
	userRepo.On("LoadUserRoles", mock.Anything, user).Return(nil)
	userRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	userRepo.On("FindByID", mock.Anything, user.ID).Return(user, nil)
	roleRepo.On("FindByIDs", mock.Anything, user.RoleIDs).Return([]*identity.Role{role}, nil)
	roleRepo.On("LoadPermissions", mock.Anything, role).Return(nil)

	jwtCfg := config.JWTConfig{
		Secret:                 "test-secret-key-32-characters-long",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
		Issuer:                 "test-issuer",
		MaxRefreshCount:        10,
	}
	jwtService := auth.NewJWTService(jwtCfg)

	authService := createAuthServiceForHandler(userRepo, roleRepo, jwtService)
	handler := NewAuthHandler(authService, testCookieConfig(), testJWTConfig())
	router := setupAuthRouter(handler, jwtService)

	// First login
	loginReq := LoginRequest{
		Username: "testuser",
		Password: "Password123",
	}
	loginBody, _ := json.Marshal(loginReq)
	loginHttpReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
	loginHttpReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, loginHttpReq)
	require.Equal(t, http.StatusOK, loginW.Code)

	var loginResponse map[string]interface{}
	json.Unmarshal(loginW.Body.Bytes(), &loginResponse)
	loginData := loginResponse["data"].(map[string]interface{})
	loginToken := loginData["token"].(map[string]interface{})
	accessToken := loginToken["access_token"].(string)

	// Change password
	changeReq := ChangePasswordRequest{
		OldPassword: "Password123",
		NewPassword: "NewPassword456",
	}
	changeBody, _ := json.Marshal(changeReq)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/password", bytes.NewReader(changeBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
}
