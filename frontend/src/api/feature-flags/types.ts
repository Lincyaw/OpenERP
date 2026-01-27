/**
 * Feature Flag API Types
 * These types match the backend dto/flag_dto.go
 */

// =====================
// Common Types
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

// =====================
// Flag Value Types
// =====================

export interface FlagValue {
  enabled: boolean
  variant?: string
  metadata?: Record<string, unknown>
}

// =====================
// Targeting Rule Types
// =====================

export interface Condition {
  attribute: string
  operator: string
  values: string[]
}

export interface TargetingRule {
  rule_id: string
  priority: number
  conditions: Condition[]
  value: FlagValue
  percentage: number
}

// =====================
// Flag Types
// =====================

export type FlagType = 'boolean' | 'percentage' | 'variant' | 'user_segment'
export type FlagStatus = 'enabled' | 'disabled' | 'archived'

export interface FeatureFlag {
  id: string
  key: string
  name: string
  description?: string
  type: FlagType
  status: FlagStatus
  default_value: FlagValue
  rules?: TargetingRule[]
  tags?: string[]
  version: number
  created_by?: string
  updated_by?: string
  created_at: string
  updated_at: string
}

// =====================
// Request Types
// =====================

export interface CreateFlagRequest {
  key: string
  name: string
  description?: string
  type: FlagType
  default_value: FlagValue
  rules?: TargetingRule[]
  tags?: string[]
}

export interface UpdateFlagRequest {
  name?: string
  description?: string
  default_value?: FlagValue
  rules?: TargetingRule[]
  tags?: string[]
  version?: number // For optimistic locking
}

export interface FlagListQuery {
  page?: number
  page_size?: number
  status?: FlagStatus
  type?: FlagType
  tags?: string
  search?: string
}

// =====================
// Response Types
// =====================

export interface FlagListResponse {
  flags: FeatureFlag[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface MessageResponse {
  message: string
}

// =====================
// Audit Log Types
// =====================

export interface AuditLog {
  id: string
  flag_key: string
  action: string
  user_id: string
  user_name?: string
  changes?: Record<string, unknown>
  created_at: string
}

export interface AuditLogListResponse {
  logs: AuditLog[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface AuditLogQuery {
  page?: number
  page_size?: number
  action?: string
}

// =====================
// Override Types
// =====================

export type OverrideTargetType = 'user' | 'tenant'

export interface Override {
  id: string
  flag_key: string
  target_type: OverrideTargetType
  target_id: string
  target_name?: string
  value: FlagValue
  reason?: string
  expires_at?: string
  created_by?: string
  created_by_name?: string
  created_at: string
  updated_at: string
}

export interface CreateOverrideRequest {
  target_type: OverrideTargetType
  target_id: string
  value: FlagValue
  reason?: string
  expires_at?: string
}

export interface UpdateOverrideRequest {
  value?: FlagValue
  reason?: string
  expires_at?: string
}

export interface OverrideListResponse {
  overrides: Override[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface OverrideListQuery {
  page?: number
  page_size?: number
  target_type?: OverrideTargetType
}
