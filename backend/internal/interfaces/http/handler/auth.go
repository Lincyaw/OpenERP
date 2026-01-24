package handler

import (
	"net/http"

	"github.com/erp/backend/internal/application/identity"
	"github.com/erp/backend/internal/interfaces/http/dto"
	"github.com/erp/backend/internal/interfaces/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	BaseHandler
	authService *identity.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *identity.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login godoc
// @Summary      User login
// @Description  Authenticate user with username and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200 {object} dto.Response{data=LoginResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
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

	response := LoginResponse{
		Token: TokenResponse{
			AccessToken:           result.AccessToken,
			RefreshToken:          result.RefreshToken,
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
// @Summary      Refresh access token
// @Description  Get new access token using refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshTokenRequest true "Refresh token"
// @Success      200 {object} dto.Response{data=RefreshTokenResponse}
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.BadRequest(c, "Invalid request body")
		return
	}

	// The auth service extracts user info from the refresh token itself
	result, err := h.authService.RefreshToken(c.Request.Context(), identity.RefreshTokenInput{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		h.HandleError(c, err)
		return
	}

	response := RefreshTokenResponse{
		Token: TokenResponse{
			AccessToken:           result.AccessToken,
			RefreshToken:          result.RefreshToken,
			AccessTokenExpiresAt:  result.AccessTokenExpiresAt,
			RefreshTokenExpiresAt: result.RefreshTokenExpiresAt,
			TokenType:             result.TokenType,
		},
	}

	h.Success(c, response)
}

// Logout godoc
// @Summary      User logout
// @Description  Logout and invalidate the current session
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200 {object} dto.Response{data=LogoutResponse}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
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

	h.Success(c, LogoutResponse{
		Message: "Logged out successfully",
	})
}

// GetCurrentUser godoc
// @Summary      Get current user
// @Description  Get the currently authenticated user's information
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      200 {object} dto.Response{data=CurrentUserResponse}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
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
// @Summary      Change password
// @Description  Change the current user's password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body ChangePasswordRequest true "Password change request"
// @Success      200 {object} dto.Response
// @Failure      400 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      401 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      422 {object} dto.Response{error=dto.ErrorInfo}
// @Failure      500 {object} dto.Response{error=dto.ErrorInfo}
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
