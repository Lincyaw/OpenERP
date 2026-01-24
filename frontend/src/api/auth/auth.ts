/**
 * Auth API Service
 * Handles authentication-related API calls
 */
import { customInstance } from '../../services/axios-instance'
import type {
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  RefreshTokenResponse,
  ChangePasswordRequest,
  CurrentUserResponse,
  LogoutResponse,
  ApiResponse,
} from './types'

/**
 * Auth API service factory
 * @returns Auth API methods
 */
export const getAuth = () => {
  /**
   * User login
   * @summary Authenticate user with username and password
   */
  const postAuthLogin = (
    loginRequest: LoginRequest
  ): Promise<ApiResponse<LoginResponse>> => {
    return customInstance<ApiResponse<LoginResponse>>({
      url: '/auth/login',
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: loginRequest,
    })
  }

  /**
   * Refresh access token
   * @summary Get new access token using refresh token
   */
  const postAuthRefresh = (
    refreshTokenRequest: RefreshTokenRequest
  ): Promise<ApiResponse<RefreshTokenResponse>> => {
    return customInstance<ApiResponse<RefreshTokenResponse>>({
      url: '/auth/refresh',
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: refreshTokenRequest,
    })
  }

  /**
   * User logout
   * @summary Logout and invalidate the current session
   */
  const postAuthLogout = (): Promise<ApiResponse<LogoutResponse>> => {
    return customInstance<ApiResponse<LogoutResponse>>({
      url: '/auth/logout',
      method: 'POST',
    })
  }

  /**
   * Get current user
   * @summary Get the currently authenticated user's information
   */
  const getAuthMe = (): Promise<ApiResponse<CurrentUserResponse>> => {
    return customInstance<ApiResponse<CurrentUserResponse>>({
      url: '/auth/me',
      method: 'GET',
    })
  }

  /**
   * Change password
   * @summary Change the current user's password
   */
  const putAuthPassword = (
    changePasswordRequest: ChangePasswordRequest
  ): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: '/auth/password',
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      data: changePasswordRequest,
    })
  }

  return {
    postAuthLogin,
    postAuthRefresh,
    postAuthLogout,
    getAuthMe,
    putAuthPassword,
  }
}

// Export types for consumers
export type {
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  RefreshTokenResponse,
  ChangePasswordRequest,
  CurrentUserResponse,
  LogoutResponse,
  ApiResponse,
  AuthUserResponse,
  TokenResponse,
} from './types'

// Create a singleton instance for direct usage
const authApi = getAuth()

export default authApi
