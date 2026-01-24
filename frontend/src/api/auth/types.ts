/**
 * Auth API Types
 * These types match the backend auth DTOs from auth_dto.go
 */

// =====================
// Request Types
// =====================

export interface LoginRequest {
  username: string
  password: string
}

export interface RefreshTokenRequest {
  refresh_token: string
}

export interface ChangePasswordRequest {
  old_password: string
  new_password: string
}

// =====================
// Response Types
// =====================

export interface TokenResponse {
  access_token: string
  refresh_token: string
  access_token_expires_at: string
  refresh_token_expires_at: string
  token_type: string
}

export interface AuthUserResponse {
  id: string
  tenant_id: string
  username: string
  display_name: string
  email?: string
  phone?: string
  avatar?: string
  permissions: string[]
  role_ids: string[]
}

export interface LoginResponse {
  token: TokenResponse
  user: AuthUserResponse
}

export interface RefreshTokenResponse {
  token: TokenResponse
}

export interface CurrentUserResponse {
  user: AuthUserResponse
  permissions: string[]
}

export interface LogoutResponse {
  message: string
}

// =====================
// API Response Wrapper
// =====================

export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
    details?: string
  }
}
