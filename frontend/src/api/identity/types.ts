/**
 * Identity API Types
 * These types match the backend user_role_dto.go
 */

// =====================
// Common Types
// =====================

export interface PaginationMeta {
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: {
    code: string
    message: string
    details?: string
  }
}

// =====================
// User Types
// =====================

export type UserStatus = 'pending' | 'active' | 'locked' | 'deactivated'

export interface User {
  id: string
  tenant_id: string
  username: string
  email?: string
  phone?: string
  display_name: string
  avatar?: string
  status: UserStatus
  role_ids: string[]
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface CreateUserRequest {
  username: string
  password: string
  email?: string
  phone?: string
  display_name?: string
  notes?: string
  role_ids?: string[]
}

export interface UpdateUserRequest {
  email?: string
  phone?: string
  display_name?: string
  notes?: string
}

export interface ResetPasswordRequest {
  new_password: string
}

export interface AssignRolesRequest {
  role_ids: string[]
}

export interface LockUserRequest {
  duration_minutes?: number
}

export interface UserListQuery {
  keyword?: string
  status?: UserStatus
  role_id?: string
  page?: number
  page_size?: number
  sort_by?: 'username' | 'email' | 'display_name' | 'created_at' | 'updated_at' | 'last_login_at'
  sort_dir?: 'asc' | 'desc'
}

export interface UserListResponse {
  users: User[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

// =====================
// Role Types
// =====================

export interface Role {
  id: string
  tenant_id: string
  code: string
  name: string
  description?: string
  is_system_role: boolean
  is_enabled: boolean
  sort_order: number
  permissions: string[]
  user_count?: number
  created_at: string
  updated_at: string
}

export interface CreateRoleRequest {
  code: string
  name: string
  description?: string
  permissions?: string[]
  sort_order?: number
}

export interface UpdateRoleRequest {
  name?: string
  description?: string
  sort_order?: number
}

export interface SetPermissionsRequest {
  permissions: string[]
}

export interface RoleListQuery {
  keyword?: string
  is_enabled?: boolean
  is_system_role?: boolean
  page?: number
  page_size?: number
}

export interface RoleListResponse {
  roles: Role[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

// =====================
// Permission Types
// =====================

export interface Permission {
  code: string
  resource: string
  action: string
  description?: string
}

export interface PermissionListResponse {
  permissions: string[]
}

// =====================
// Stats Types
// =====================

export interface CountResponse {
  count: number
}
