/**
 * Usage API hooks
 *
 * Manual API hooks for tenant usage statistics endpoints.
 * These endpoints are not yet in the auto-generated swagger docs.
 */
import { useQuery } from '@tanstack/react-query'
import type { QueryFunction, UseQueryOptions, UseQueryResult } from '@tanstack/react-query'

import { customInstance } from '../../services/axios-instance'

// ============================================================================
// Types
// ============================================================================

/**
 * Usage metric with current value and quota limit
 */
export interface UsageMetric {
  name: string
  display_name: string
  current: number
  limit: number
  percentage: number
  unit: string
}

/**
 * Current tenant usage summary with all metrics
 */
export interface UsageSummaryResponse {
  tenant_id: string
  tenant_name: string
  plan: string
  metrics: UsageMetric[]
  last_updated: string
}

/**
 * Single data point in usage history
 */
export interface UsageHistoryPoint {
  date: string
  users: number
  products: number
  warehouses: number
}

/**
 * Historical usage trends over time
 */
export interface UsageHistoryResponse {
  tenant_id: string
  period: 'daily' | 'weekly' | 'monthly'
  start_date: string
  end_date: string
  data_points: UsageHistoryPoint[]
}

/**
 * Quota item with current usage and remaining capacity
 */
export interface QuotaItem {
  resource: string
  display_name: string
  used: number
  limit: number
  remaining: number
  is_unlimited: boolean
}

/**
 * All quotas and remaining capacity for a tenant
 */
export interface QuotasResponse {
  tenant_id: string
  plan: string
  quotas: QuotaItem[]
}

/**
 * API response wrapper
 */
export interface APIResponse<T> {
  success: boolean
  data: T
  error?: string
}

/**
 * Error response
 */
export interface ErrorResponse {
  success: boolean
  error: string
  message?: string
}

// ============================================================================
// Query parameters
// ============================================================================

export interface GetUsageHistoryParams {
  period?: 'daily' | 'weekly' | 'monthly'
  start_date?: string
  end_date?: string
}

// ============================================================================
// Response types
// ============================================================================

export type GetCurrentUsageResponse200 = {
  data: APIResponse<UsageSummaryResponse>
  status: 200
}

export type GetCurrentUsageResponse401 = {
  data: ErrorResponse
  status: 401
}

export type GetCurrentUsageResponse404 = {
  data: ErrorResponse
  status: 404
}

export type GetCurrentUsageResponse500 = {
  data: ErrorResponse
  status: 500
}

export type GetCurrentUsageResponseSuccess = GetCurrentUsageResponse200 & {
  headers: Headers
}

export type GetCurrentUsageResponseError = (
  | GetCurrentUsageResponse401
  | GetCurrentUsageResponse404
  | GetCurrentUsageResponse500
) & {
  headers: Headers
}

export type GetCurrentUsageResponse = GetCurrentUsageResponseSuccess | GetCurrentUsageResponseError

export type GetUsageHistoryResponse200 = {
  data: APIResponse<UsageHistoryResponse>
  status: 200
}

export type GetUsageHistoryResponseSuccess = GetUsageHistoryResponse200 & {
  headers: Headers
}

export type GetUsageHistoryResponse = GetUsageHistoryResponseSuccess | GetCurrentUsageResponseError

export type GetQuotasResponse200 = {
  data: APIResponse<QuotasResponse>
  status: 200
}

export type GetQuotasResponseSuccess = GetQuotasResponse200 & {
  headers: Headers
}

export type GetQuotasResponse = GetQuotasResponseSuccess | GetCurrentUsageResponseError

// ============================================================================
// API Functions
// ============================================================================

/**
 * Get current tenant usage summary
 */
export const getCurrentUsageUrl = () => `/identity/tenants/current/usage`

export const getCurrentUsage = async (options?: RequestInit): Promise<GetCurrentUsageResponse> => {
  return customInstance<GetCurrentUsageResponse>(getCurrentUsageUrl(), {
    ...options,
    method: 'GET',
  })
}

/**
 * Get historical usage trends
 */
export const getUsageHistoryUrl = (params?: GetUsageHistoryParams) => {
  const normalizedParams = new URLSearchParams()

  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined) {
      normalizedParams.append(key, value === null ? 'null' : value.toString())
    }
  })

  const stringifiedParams = normalizedParams.toString()

  return stringifiedParams.length > 0
    ? `/identity/tenants/current/usage/history?${stringifiedParams}`
    : `/identity/tenants/current/usage/history`
}

export const getUsageHistory = async (
  params?: GetUsageHistoryParams,
  options?: RequestInit
): Promise<GetUsageHistoryResponse> => {
  return customInstance<GetUsageHistoryResponse>(getUsageHistoryUrl(params), {
    ...options,
    method: 'GET',
  })
}

/**
 * Get tenant quotas and remaining capacity
 */
export const getQuotasUrl = () => `/identity/tenants/current/quotas`

export const getQuotas = async (options?: RequestInit): Promise<GetQuotasResponse> => {
  return customInstance<GetQuotasResponse>(getQuotasUrl(), {
    ...options,
    method: 'GET',
  })
}

// ============================================================================
// React Query Hooks
// ============================================================================

export const getGetCurrentUsageQueryKey = () => {
  return ['getCurrentUsage'] as const
}

export type GetCurrentUsageQueryResult = NonNullable<Awaited<ReturnType<typeof getCurrentUsage>>>
export type GetCurrentUsageQueryError = GetCurrentUsageResponseError

/**
 * Hook to get current tenant usage summary
 */
export const useGetCurrentUsage = <
  TData = Awaited<ReturnType<typeof getCurrentUsage>>,
  TError = GetCurrentUsageResponseError,
>(options?: {
  query?: Partial<UseQueryOptions<Awaited<ReturnType<typeof getCurrentUsage>>, TError, TData>>
}): UseQueryResult<TData, TError> => {
  const { query: queryOptions } = options ?? {}

  const queryKey = queryOptions?.queryKey ?? getGetCurrentUsageQueryKey()

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getCurrentUsage>>> = ({ signal }) =>
    getCurrentUsage({ signal })

  return useQuery({
    queryKey,
    queryFn,
    ...queryOptions,
  })
}

export const getGetUsageHistoryQueryKey = (params?: GetUsageHistoryParams) => {
  return ['getUsageHistory', ...(params ? [params] : [])] as const
}

export type GetUsageHistoryQueryResult = NonNullable<Awaited<ReturnType<typeof getUsageHistory>>>
export type GetUsageHistoryQueryError = GetCurrentUsageResponseError

/**
 * Hook to get historical usage trends
 */
export const useGetUsageHistory = <
  TData = Awaited<ReturnType<typeof getUsageHistory>>,
  TError = GetCurrentUsageResponseError,
>(
  params?: GetUsageHistoryParams,
  options?: {
    query?: Partial<UseQueryOptions<Awaited<ReturnType<typeof getUsageHistory>>, TError, TData>>
  }
): UseQueryResult<TData, TError> => {
  const { query: queryOptions } = options ?? {}

  const queryKey = queryOptions?.queryKey ?? getGetUsageHistoryQueryKey(params)

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getUsageHistory>>> = ({ signal }) =>
    getUsageHistory(params, { signal })

  return useQuery({
    queryKey,
    queryFn,
    ...queryOptions,
  })
}

export const getGetQuotasQueryKey = () => {
  return ['getQuotas'] as const
}

export type GetQuotasQueryResult = NonNullable<Awaited<ReturnType<typeof getQuotas>>>
export type GetQuotasQueryError = GetCurrentUsageResponseError

/**
 * Hook to get tenant quotas and remaining capacity
 */
export const useGetQuotas = <
  TData = Awaited<ReturnType<typeof getQuotas>>,
  TError = GetCurrentUsageResponseError,
>(options?: {
  query?: Partial<UseQueryOptions<Awaited<ReturnType<typeof getQuotas>>, TError, TData>>
}): UseQueryResult<TData, TError> => {
  const { query: queryOptions } = options ?? {}

  const queryKey = queryOptions?.queryKey ?? getGetQuotasQueryKey()

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getQuotas>>> = ({ signal }) =>
    getQuotas({ signal })

  return useQuery({
    queryKey,
    queryFn,
    ...queryOptions,
  })
}
