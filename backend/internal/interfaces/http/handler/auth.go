package handler

import (
	"net/http"
	"strings"

	"github.com/erp/backend/internal/application/identity"
	"github.com/erp/backend/internal/infrastructure/config"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RefreshTokenCookieName is the name of the httpOnly cookie for refresh token
const RefreshTokenCookieName = "refresh_token"

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	BaseHandler
	authService  *identity.AuthService
	cookieConfig config.CookieConfig
	jwtConfig    config.JWTConfig
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *identity.AuthService, cookieConfig config.CookieConfig, jwtConfig config.JWTConfig) *AuthHandler {
	return &AuthHandler{
		authService:  authService,
		cookieConfig: cookieConfig,
		jwtConfig:    jwtConfig,
	}
}

// getSameSite converts string to http.SameSite
func getSameSite(sameSite string) http.SameSite {
	switch strings.ToLower(sameSite) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

// setRefreshTokenCookie sets the refresh token as an httpOnly cookie
func (h *AuthHandler) setRefreshTokenCookie(c *gin.Context, refreshToken string, maxAge int) {
	c.SetSameSite(getSameSite(h.cookieConfig.SameSite))
	c.SetCookie(
		RefreshTokenCookieName,
		refreshToken,
		maxAge,
		h.cookieConfig.Path,
		h.cookieConfig.Domain,
		h.cookieConfig.Secure,
		true, // httpOnly = true (critical for security)
	)
}

// clearRefreshTokenCookie clears the refresh token cookie
func (h *AuthHandler) clearRefreshTokenCookie(c *gin.Context) {
	c.SetSameSite(getSameSite(h.cookieConfig.SameSite))
	c.SetCookie(
		RefreshTokenCookieName,
		"",
		-1, // Expire immediately
		h.cookieConfig.Path,
		h.cookieConfig.Domain,
		h.cookieConfig.Secure,
		true, // httpOnly = true
	)
}

// Login godoc
// @ID           loginAuth
// @Summary      User login
// @Description  Authenticate user with username and password. Returns access token in response body and sets refresh token as httpOnly cookie.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200 {object} APIResponse[LoginResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	// Get client IP for login tracking
	clientIP := c.ClientIP()

	result, err := h.authService.Login(c.Request.Context(), identity.LoginInput{
		Username: req.Username,
		Password: req.Password,
		IP:       clientIP,
	})
	if err != nil {
		h.HandleError(c, err)
		return
	}

	// Convert role IDs to strings
	roleIDStrings := make([]string, len(result.User.RoleIDs))
	for i, rid := range result.User.RoleIDs {
		roleIDStrings[i] = rid.String()
	}

	// Set refresh token as httpOnly cookie
	// Calculate max age in seconds from refresh token expiration
	maxAge := int(h.jwtConfig.RefreshTokenExpiration.Seconds())
	h.setRefreshTokenCookie(c, result.RefreshToken, maxAge)

	// Response includes access token (stored in memory on frontend)
	// Refresh token is NOT included in response body - it's in httpOnly cookie
	response := LoginResponse{
		Token: TokenResponse{
			AccessToken:           result.AccessToken,
			RefreshToken:          "", // Empty - sent via httpOnly cookie
			AccessTokenExpiresAt:  result.AccessTokenExpiresAt,
			RefreshTokenExpiresAt: result.RefreshTokenExpiresAt,
			TokenType:             result.TokenType,
		},
		User: AuthUserResponse{
			ID:          result.User.ID,
			TenantID:    result.User.TenantID,
			Username:    result.User.Username,
			DisplayName: result.User.DisplayName,
			Email:       result.User.Email,
			Phone:       result.User.Phone,
			Avatar:      result.User.Avatar,
			Permissions: result.User.Permissions,
			RoleIDs:     roleIDStrings,
		},
	}

	h.Success(c, response)
}

// RefreshToken godoc
// @ID           refreshTokenAuth
// @Summary      Refresh access token
// @Description  Get new access token using refresh token from httpOnly cookie. Falls back to request body for backward compatibility.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshTokenRequest false "Refresh token (optional, prefer cookie)"
// @Success      200 {object} APIResponse[RefreshTokenResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var refreshToken string

	// Try to get refresh token from httpOnly cookie first (preferred, secure method)
	cookieToken, err := c.Cookie(RefreshTokenCookieName)
	if err == nil && cookieToken != "" {
		refreshToken = cookieToken
	} else {
		// Fallback: try to get from request body (for backward compatibility)
		var req RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
			refreshToken = req.RefreshToken
		}
	}

	// No refresh token found in either source
	if refreshToken == "" {
		h.Unauthorized(c, "Refresh token required")
		return
	}

	// The auth service extracts user info from the refresh token itself
	result, err := h.authService.RefreshToken(c.Request.Context(), identity.RefreshTokenInput{
		RefreshToken: refreshToken,
	})
	if err != nil {
		// Clear invalid cookie on refresh failure
		h.clearRefreshTokenCookie(c)
		h.HandleError(c, err)
		return
	}

	// Update refresh token cookie with new token
	maxAge := int(h.jwtConfig.RefreshTokenExpiration.Seconds())
	h.setRefreshTokenCookie(c, result.RefreshToken, maxAge)

	// Response includes only access token - refresh token is in httpOnly cookie
	response := RefreshTokenResponse{
		Token: TokenResponse{
			AccessToken:           result.AccessToken,
			RefreshToken:          "", // Empty - sent via httpOnly cookie
			AccessTokenExpiresAt:  result.AccessTokenExpiresAt,
			RefreshTokenExpiresAt: result.RefreshTokenExpiresAt,
			TokenType:             result.TokenType,
		},
	}

	h.Success(c, response)
}

// Logout godoc
// @ID           logoutAuth
// @Summary      User logout
// @Description  Logout and invalidate the current session. Clears refresh token cookie.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200 {object} APIResponse[LogoutResponse]
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		h.Unauthorized(c, "Authentication required")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		h.BadRequest(c, "Invalid user ID in token")
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID in token")
		return
	}

	err = h.authService.Logout(c.Request.Context(), identity.LogoutInput{
		UserID:   userID,
		TenantID: tenantID,
		TokenJTI: claims.ID,
	})
	if err != nil {
		h.HandleError(c, err)
		return
	}

	// Clear refresh token cookie
	h.clearRefreshTokenCookie(c)

	h.Success(c, LogoutResponse{
		Message: "Logged out successfully",
	})
}

// GetCurrentUser godoc
// @ID           getAuthCurrentUser
// @Summary      Get current user
// @Description  Get the currently authenticated user's information
// @Tags         auth
// @Produce      json
// @Success      200 {object} APIResponse[CurrentUserResponse]
// @Failure      401 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /auth/me [get]
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		h.Unauthorized(c, "Authentication required")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		h.BadRequest(c, "Invalid user ID in token")
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID in token")
		return
	}

	result, err := h.authService.GetCurrentUser(c.Request.Context(), identity.GetCurrentUserInput{
		UserID:   userID,
		TenantID: tenantID,
	})
	if err != nil {
		h.HandleError(c, err)
		return
	}

	// Convert role IDs to strings
	roleIDStrings := make([]string, len(result.User.RoleIDs))
	for i, rid := range result.User.RoleIDs {
		roleIDStrings[i] = rid.String()
	}

	response := CurrentUserResponse{
		User: AuthUserResponse{
			ID:          result.User.ID,
			TenantID:    result.User.TenantID,
			Username:    result.User.Username,
			DisplayName: result.User.DisplayName,
			Email:       result.User.Email,
			Phone:       result.User.Phone,
			Avatar:      result.User.Avatar,
			Permissions: result.User.Permissions,
			RoleIDs:     roleIDStrings,
		},
		Permissions: result.Permissions,
	}

	h.Success(c, response)
}

// ChangePassword godoc
// @ID           changePasswordAuth
// @Summary      Change password
// @Description  Change the current user's password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ChangePasswordRequest true "Password change request"
// @Success      200 {object} SuccessResponse
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      422 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /auth/password [put]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		h.Unauthorized(c, "Authentication required")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		h.BadRequest(c, "Invalid user ID in token")
		return
	}

	err = h.authService.ChangePassword(c.Request.Context(), identity.ChangePasswordInput{
		UserID:      userID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		h.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(gin.H{
		"message": "Password changed successfully",
	}))
}

// ForceLogout godoc
// @ID           forceLogoutAuth
// @Summary      Force logout user (Admin)
// @Description  Invalidate all sessions for a specific user. Requires user:force_logout permission.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ForceLogoutRequest true "Force logout request"
// @Success      200 {object} APIResponse[ForceLogoutResponse]
// @Failure      400 {object} dto.ErrorResponse
// @Failure      401 {object} dto.ErrorResponse
// @Failure      403 {object} dto.ErrorResponse
// @Failure      404 {object} dto.ErrorResponse
// @Failure      500 {object} dto.ErrorResponse
// @Security     BearerAuth
// @Router       /auth/force-logout [post]
func (h *AuthHandler) ForceLogout(c *gin.Context) {
	claims := middleware.GetJWTClaims(c)
	if claims == nil {
		h.Unauthorized(c, "Authentication required")
		return
	}

	var req ForceLogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	adminUserID, err := uuid.Parse(claims.UserID)
	if err != nil {
		h.BadRequest(c, "Invalid admin user ID in token")
		return
	}

	targetUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		h.BadRequest(c, "Invalid target user ID")
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		h.BadRequest(c, "Invalid tenant ID in token")
		return
	}

	result, err := h.authService.ForceLogout(c.Request.Context(), identity.ForceLogoutInput{
		AdminUserID:  adminUserID,
		TargetUserID: targetUserID,
		TenantID:     tenantID,
		Reason:       req.Reason,
	})
	if err != nil {
		h.HandleError(c, err)
		return
	}

	h.Success(c, ForceLogoutResponse{
		Message: result.Message,
	})
}
