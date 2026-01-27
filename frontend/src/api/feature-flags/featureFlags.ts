/**
 * Feature Flag API Service
 * Handles feature flag management API calls
 */
import { customInstance } from '../../services/axios-instance'
import type {
  ApiResponse,
  FeatureFlag,
  CreateFlagRequest,
  UpdateFlagRequest,
  FlagListQuery,
  FlagListResponse,
  MessageResponse,
  AuditLogListResponse,
  AuditLogQuery,
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
 * Feature Flag API service factory
 * @returns Feature Flag API methods
 */
export const getFeatureFlags = () => {
  /**
   * List feature flags with filtering and pagination
   */
  const listFlags = (query: FlagListQuery = {}): Promise<ApiResponse<FlagListResponse>> => {
    return customInstance<ApiResponse<FlagListResponse>>({
      url: `/feature-flags${buildQueryString(query)}`,
      method: 'GET',
    })
  }

  /**
   * Get a feature flag by key
   */
  const getFlag = (key: string): Promise<ApiResponse<FeatureFlag>> => {
    return customInstance<ApiResponse<FeatureFlag>>({
      url: `/feature-flags/${encodeURIComponent(key)}`,
      method: 'GET',
    })
  }

  /**
   * Create a new feature flag
   */
  const createFlag = (request: CreateFlagRequest): Promise<ApiResponse<FeatureFlag>> => {
    return customInstance<ApiResponse<FeatureFlag>>({
      url: '/feature-flags',
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Update a feature flag
   */
  const updateFlag = (
    key: string,
    request: UpdateFlagRequest
  ): Promise<ApiResponse<FeatureFlag>> => {
    return customInstance<ApiResponse<FeatureFlag>>({
      url: `/feature-flags/${encodeURIComponent(key)}`,
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      data: request,
    })
  }

  /**
   * Archive a feature flag (soft delete)
   */
  const archiveFlag = (key: string): Promise<ApiResponse<void>> => {
    return customInstance<ApiResponse<void>>({
      url: `/feature-flags/${encodeURIComponent(key)}`,
      method: 'DELETE',
    })
  }

  /**
   * Enable a feature flag
   */
  const enableFlag = (key: string): Promise<ApiResponse<MessageResponse>> => {
    return customInstance<ApiResponse<MessageResponse>>({
      url: `/feature-flags/${encodeURIComponent(key)}/enable`,
      method: 'POST',
    })
  }

  /**
   * Disable a feature flag
   */
  const disableFlag = (key: string): Promise<ApiResponse<MessageResponse>> => {
    return customInstance<ApiResponse<MessageResponse>>({
      url: `/feature-flags/${encodeURIComponent(key)}/disable`,
      method: 'POST',
    })
  }

  /**
   * Get audit logs for a feature flag
   */
  const getAuditLogs = (
    key: string,
    query: AuditLogQuery = {}
  ): Promise<ApiResponse<AuditLogListResponse>> => {
    return customInstance<ApiResponse<AuditLogListResponse>>({
      url: `/feature-flags/${encodeURIComponent(key)}/audit-logs${buildQueryString(query)}`,
      method: 'GET',
    })
  }

  return {
    listFlags,
    getFlag,
    createFlag,
    updateFlag,
    archiveFlag,
    enableFlag,
    disableFlag,
    getAuditLogs,
  }
}

// Export types for consumers
export type {
  ApiResponse,
  FeatureFlag,
  CreateFlagRequest,
  UpdateFlagRequest,
  FlagListQuery,
  FlagListResponse,
  FlagValue,
  FlagType,
  FlagStatus,
  TargetingRule,
  Condition,
  MessageResponse,
  AuditLog,
  AuditLogListResponse,
  AuditLogQuery,
} from './types'

// Create a singleton instance for direct usage
const featureFlagsApi = getFeatureFlags()

export default featureFlagsApi
