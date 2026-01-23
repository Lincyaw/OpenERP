/**
 * Standard API response format matching backend
 */
export interface ApiResponse<T> {
  success: boolean
  data?: T
  error?: ApiError
  meta?: PaginationMeta
}

/**
 * API error structure
 */
export interface ApiError {
  code: string
  message: string
  request_id?: string
  timestamp?: string
  details?: ValidationError[]
  help?: string
}

/**
 * Validation error detail
 */
export interface ValidationError {
  field: string
  message: string
}

/**
 * Pagination metadata
 */
export interface PaginationMeta {
  total: number
  page: number
  page_size: number
  total_pages: number
}

/**
 * Pagination parameters for requests
 */
export interface PaginationParams {
  page?: number
  page_size?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

/**
 * Base entity with common fields
 */
export interface BaseEntity {
  id: string
  created_at: string
  updated_at: string
}

/**
 * Tenant-aware entity
 */
export interface TenantEntity extends BaseEntity {
  tenant_id: string
}
