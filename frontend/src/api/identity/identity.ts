/**
 * Identity API Service
 * Handles user and role management API calls
 */
import { customInstance } from '../../services/axios-instance'
import type {
  ApiResponse,
  User,
  CreateUserRequest,
  UpdateUserRequest,
  ResetPasswordRequest,
  AssignRolesRequest,
  LockUserRequest,
  UserListQuery,
  UserListResponse,
  Role,
  CreateRoleRequest,
  UpdateRoleRequest,
  SetPermissionsRequest,
  RoleListQuery,
  RoleListResponse,
  CountResponse,
} from './types'

/**
 * Build query string from params
 */
function buildQueryString<T extends object>(params: T): string {
  const searchParams = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      searchParams.append(key, String(value))
    }
  }
  const query = searchParams.toString()
  return query ? `?${query}` : ''
}

/**
 * Identity API service factory
 * @returns Identity API methods for users and roles
 */
export const getIdentity = () => {
  // =====================
  // User API Methods
  // =====================

  /**
   * Create a new user
   */
  const createUser = (request: CreateUserRequest): Promise<ApiResponse<User>> => {
    return customInstance<ApiResponse<User>>({
      url: '/identity/users',
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Get a user by ID
   */
  const getUser = (id: string): Promise<ApiResponse<User>> => {
    return customInstance<ApiResponse<User>>({
      url: `/identity/users/${id}`,
      method: 'GET',
    })
  }

  /**
   * List users with filtering and pagination
   */
  const listUsers = (query: UserListQuery = {}): Promise<ApiResponse<UserListResponse>> => {
    return customInstance<ApiResponse<UserListResponse>>({
      url: `/identity/users${buildQueryString(query)}`,
      method: 'GET',
    })
  }

  /**
   * Update a user
   */
  const updateUser = (id: string, request: UpdateUserRequest): Promise<ApiResponse<User>> => {
    return customInstance<ApiResponse<User>>({
      url: `/identity/users/${id}`,
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Delete a user
   */
  const deleteUser = (id: string): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: `/identity/users/${id}`,
      method: 'DELETE',
    })
  }

  /**
   * Activate a user
   */
  const activateUser = (id: string): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: `/identity/users/${id}/activate`,
      method: 'POST',
    })
  }

  /**
   * Deactivate a user
   */
  const deactivateUser = (id: string): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: `/identity/users/${id}/deactivate`,
      method: 'POST',
    })
  }

  /**
   * Lock a user account
   */
  const lockUser = (
    id: string,
    request?: LockUserRequest
  ): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: `/identity/users/${id}/lock`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request || {},
    })
  }

  /**
   * Unlock a user account
   */
  const unlockUser = (id: string): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: `/identity/users/${id}/unlock`,
      method: 'POST',
    })
  }

  /**
   * Reset a user's password
   */
  const resetPassword = (
    id: string,
    request: ResetPasswordRequest
  ): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: `/identity/users/${id}/reset-password`,
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Assign roles to a user
   */
  const assignRoles = (
    id: string,
    request: AssignRolesRequest
  ): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: `/identity/users/${id}/roles`,
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Get user count statistics
   */
  const getUserCount = (): Promise<ApiResponse<CountResponse>> => {
    return customInstance<ApiResponse<CountResponse>>({
      url: '/identity/users/stats/count',
      method: 'GET',
    })
  }

  // =====================
  // Role API Methods
  // =====================

  /**
   * Create a new role
   */
  const createRole = (request: CreateRoleRequest): Promise<ApiResponse<Role>> => {
    return customInstance<ApiResponse<Role>>({
      url: '/identity/roles',
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Get a role by ID
   */
  const getRole = (id: string): Promise<ApiResponse<Role>> => {
    return customInstance<ApiResponse<Role>>({
      url: `/identity/roles/${id}`,
      method: 'GET',
    })
  }

  /**
   * Get a role by code
   */
  const getRoleByCode = (code: string): Promise<ApiResponse<Role>> => {
    return customInstance<ApiResponse<Role>>({
      url: `/identity/roles/code/${code}`,
      method: 'GET',
    })
  }

  /**
   * List roles with filtering and pagination
   */
  const listRoles = (query: RoleListQuery = {}): Promise<ApiResponse<RoleListResponse>> => {
    return customInstance<ApiResponse<RoleListResponse>>({
      url: `/identity/roles${buildQueryString(query)}`,
      method: 'GET',
    })
  }

  /**
   * Update a role
   */
  const updateRole = (id: string, request: UpdateRoleRequest): Promise<ApiResponse<Role>> => {
    return customInstance<ApiResponse<Role>>({
      url: `/identity/roles/${id}`,
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Delete a role
   */
  const deleteRole = (id: string): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: `/identity/roles/${id}`,
      method: 'DELETE',
    })
  }

  /**
   * Enable a role
   */
  const enableRole = (id: string): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: `/identity/roles/${id}/enable`,
      method: 'POST',
    })
  }

  /**
   * Disable a role
   */
  const disableRole = (id: string): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: `/identity/roles/${id}/disable`,
      method: 'POST',
    })
  }

  /**
   * Set permissions for a role
   */
  const setRolePermissions = (
    id: string,
    request: SetPermissionsRequest
  ): Promise<ApiResponse<{ message: string }>> => {
    return customInstance<ApiResponse<{ message: string }>>({
      url: `/identity/roles/${id}/permissions`,
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Get system roles
   */
  const getSystemRoles = (): Promise<ApiResponse<Role[]>> => {
    return customInstance<ApiResponse<Role[]>>({
      url: '/identity/roles/system',
      method: 'GET',
    })
  }

  /**
   * Get role count statistics
   */
  const getRoleCount = (): Promise<ApiResponse<CountResponse>> => {
    return customInstance<ApiResponse<CountResponse>>({
      url: '/identity/roles/stats/count',
      method: 'GET',
    })
  }

  /**
   * Get all available permissions
   */
  const getAllPermissions = (): Promise<ApiResponse<{ permissions: string[] }>> => {
    return customInstance<ApiResponse<{ permissions: string[] }>>({
      url: '/identity/permissions',
      method: 'GET',
    })
  }

  return {
    // User methods
    createUser,
    getUser,
    listUsers,
    updateUser,
    deleteUser,
    activateUser,
    deactivateUser,
    lockUser,
    unlockUser,
    resetPassword,
    assignRoles,
    getUserCount,
    // Role methods
    createRole,
    getRole,
    getRoleByCode,
    listRoles,
    updateRole,
    deleteRole,
    enableRole,
    disableRole,
    setRolePermissions,
    getSystemRoles,
    getRoleCount,
    getAllPermissions,
  }
}

// Export types for consumers
export type {
  User,
  CreateUserRequest,
  UpdateUserRequest,
  ResetPasswordRequest,
  AssignRolesRequest,
  LockUserRequest,
  UserListQuery,
  UserListResponse,
  UserStatus,
  Role,
  CreateRoleRequest,
  UpdateRoleRequest,
  SetPermissionsRequest,
  RoleListQuery,
  RoleListResponse,
  Permission,
  CountResponse,
  ApiResponse,
} from './types'

// Create a singleton instance for direct usage
const identityApi = getIdentity()

export default identityApi
