/**
 * Admin API Functions
 *
 * Manual API functions for admin-specific endpoints that provide
 * platform-wide statistics and audit logging functionality.
 *
 * These functions supplement the auto-generated tenant API
 * with admin-only operations.
 */
import { useQuery } from '@tanstack/react-query'
import type {
  UseQueryOptions,
  QueryFunction,
  DataTag,
  QueryKey,
  UseQueryResult,
  QueryClient,
} from '@tanstack/react-query'
import { customInstance } from '../../services/axios-instance'

// ============================================================================
// Types
// ============================================================================

/**
 * Platform-wide statistics response
 */
export interface PlatformStatsResponse {
  total_tenants: number
  active_tenants: number
  trial_tenants: number
  suspended_tenants: number
  inactive_tenants: number
  tenants_by_plan: Record<string, number>
  total_users: number
  total_products: number
  total_orders: number
  total_storage_usage: number
  last_updated: string
}

/**
 * Tenant growth data point
 */
export interface TenantGrowthPoint {
  date: string
  total_tenants: number
  new_tenants: number
  churned_count: number
}

/**
 * Tenant growth trend response
 */
export interface TenantGrowthTrendResponse {
  period: string
  start_date: string
  end_date: string
  data_points: TenantGrowthPoint[]
}

/**
 * Admin audit log entry
 */
export interface AdminAuditLogEntry {
  id: string
  admin_user_id: string
  action: string
  target_type: string
  target_id?: string
  old_value?: Record<string, unknown>
  new_value?: Record<string, unknown>
  ip_address?: string
  user_agent?: string
  created_at: string
}

/**
 * Admin audit log list response
 */
export interface AdminAuditLogListResponse {
  logs: AdminAuditLogEntry[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

/**
 * Query parameters for listing audit logs
 */
export interface AdminAuditLogListParams {
  page?: number
  page_size?: number
  admin_user_id?: string
  action?: string
  target_type?: string
  target_id?: string
  start_time?: string
  end_time?: string
  sort_by?: 'created_at' | 'action' | 'target_type'
  sort_order?: 'asc' | 'desc'
}

/**
 * Query parameters for tenant growth trend
 */
export interface TenantGrowthParams {
  period?: 'daily' | 'weekly' | 'monthly'
  days?: number
}

/**
 * Tenant usage statistics response
 */
export interface TenantUsageStatsResponse {
  tenant_id: string
  tenant_name: string
  tenant_code: string
  plan: string
  status: string
  user_count: number
  product_count: number
  order_count: number
  warehouse_count: number
  storage_usage: number
  api_call_count: number
  max_users: number
  max_products: number
  max_warehouses: number
  last_updated: string
}

/**
 * Generic API response wrapper
 */
interface APIResponse<T> {
  success: boolean
  data: T
  error?: {
    code: string
    message: string
  }
}

// ============================================================================
// API Functions
// ============================================================================

/**
 * Get platform-wide statistics
 */
export const getPlatformStats = async (
  options?: RequestInit
): Promise<{ data: APIResponse<PlatformStatsResponse>; status: number; headers: Headers }> => {
  return customInstance<{
    data: APIResponse<PlatformStatsResponse>
    status: number
    headers: Headers
  }>('/admin/stats', {
    ...options,
    method: 'GET',
  })
}

/**
 * Get tenant growth trend
 */
export const getTenantGrowthTrend = async (
  params?: TenantGrowthParams,
  options?: RequestInit
): Promise<{ data: APIResponse<TenantGrowthTrendResponse>; status: number; headers: Headers }> => {
  const searchParams = new URLSearchParams()
  if (params?.period) searchParams.append('period', params.period)
  if (params?.days) searchParams.append('days', params.days.toString())

  const query = searchParams.toString()
  const url = query ? `/admin/stats/growth?${query}` : '/admin/stats/growth'

  return customInstance<{
    data: APIResponse<TenantGrowthTrendResponse>
    status: number
    headers: Headers
  }>(url, {
    ...options,
    method: 'GET',
  })
}

/**
 * Get tenant usage statistics
 */
export const getTenantUsageStats = async (
  tenantId: string,
  options?: RequestInit
): Promise<{ data: APIResponse<TenantUsageStatsResponse>; status: number; headers: Headers }> => {
  return customInstance<{
    data: APIResponse<TenantUsageStatsResponse>
    status: number
    headers: Headers
  }>(`/admin/tenants/${tenantId}/stats`, {
    ...options,
    method: 'GET',
  })
}

/**
 * Get admin audit logs
 */
export const getAdminAuditLogs = async (
  params?: AdminAuditLogListParams,
  options?: RequestInit
): Promise<{ data: APIResponse<AdminAuditLogListResponse>; status: number; headers: Headers }> => {
  const searchParams = new URLSearchParams()
  if (params?.page) searchParams.append('page', params.page.toString())
  if (params?.page_size) searchParams.append('page_size', params.page_size.toString())
  if (params?.admin_user_id) searchParams.append('admin_user_id', params.admin_user_id)
  if (params?.action) searchParams.append('action', params.action)
  if (params?.target_type) searchParams.append('target_type', params.target_type)
  if (params?.target_id) searchParams.append('target_id', params.target_id)
  if (params?.start_time) searchParams.append('start_time', params.start_time)
  if (params?.end_time) searchParams.append('end_time', params.end_time)
  if (params?.sort_by) searchParams.append('sort_by', params.sort_by)
  if (params?.sort_order) searchParams.append('sort_order', params.sort_order)

  const query = searchParams.toString()
  const url = query ? `/admin/audit-logs?${query}` : '/admin/audit-logs'

  return customInstance<{
    data: APIResponse<AdminAuditLogListResponse>
    status: number
    headers: Headers
  }>(url, {
    ...options,
    method: 'GET',
  })
}

// ============================================================================
// React Query Hooks
// ============================================================================

// Query Keys
export const adminQueryKeys = {
  all: ['admin'] as const,
  platformStats: () => [...adminQueryKeys.all, 'platformStats'] as const,
  growthTrend: (params?: TenantGrowthParams) =>
    [...adminQueryKeys.all, 'growthTrend', params] as const,
  tenantStats: (tenantId: string) => [...adminQueryKeys.all, 'tenantStats', tenantId] as const,
  auditLogs: (params?: AdminAuditLogListParams) =>
    [...adminQueryKeys.all, 'auditLogs', params] as const,
}

/**
 * Hook to fetch platform statistics
 */
export function usePlatformStats<
  TData = Awaited<ReturnType<typeof getPlatformStats>>,
  TError = Error,
>(
  options?: {
    query?: Partial<UseQueryOptions<Awaited<ReturnType<typeof getPlatformStats>>, TError, TData>>
  },
  queryClient?: QueryClient
): UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryKey = adminQueryKeys.platformStats()

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getPlatformStats>>> = ({ signal }) =>
    getPlatformStats({ signal })

  const queryOptions = {
    queryKey,
    queryFn,
    staleTime: 30000, // Stats can be stale for 30 seconds
    ...options?.query,
  } as UseQueryOptions<Awaited<ReturnType<typeof getPlatformStats>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>
  }

  const query = useQuery(queryOptions, queryClient) as UseQueryResult<TData, TError> & {
    queryKey: DataTag<QueryKey, TData, TError>
  }

  return { ...query, queryKey: queryOptions.queryKey }
}

/**
 * Hook to fetch tenant growth trend
 */
export function useTenantGrowthTrend<
  TData = Awaited<ReturnType<typeof getTenantGrowthTrend>>,
  TError = Error,
>(
  params?: TenantGrowthParams,
  options?: {
    query?: Partial<
      UseQueryOptions<Awaited<ReturnType<typeof getTenantGrowthTrend>>, TError, TData>
    >
  },
  queryClient?: QueryClient
): UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryKey = adminQueryKeys.growthTrend(params)

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getTenantGrowthTrend>>> = ({ signal }) =>
    getTenantGrowthTrend(params, { signal })

  const queryOptions = {
    queryKey,
    queryFn,
    staleTime: 60000, // Growth data can be stale for 1 minute
    ...options?.query,
  } as UseQueryOptions<Awaited<ReturnType<typeof getTenantGrowthTrend>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>
  }

  const query = useQuery(queryOptions, queryClient) as UseQueryResult<TData, TError> & {
    queryKey: DataTag<QueryKey, TData, TError>
  }

  return { ...query, queryKey: queryOptions.queryKey }
}

/**
 * Hook to fetch tenant usage statistics
 */
export function useTenantUsageStats<
  TData = Awaited<ReturnType<typeof getTenantUsageStats>>,
  TError = Error,
>(
  tenantId: string,
  options?: {
    query?: Partial<UseQueryOptions<Awaited<ReturnType<typeof getTenantUsageStats>>, TError, TData>>
  },
  queryClient?: QueryClient
): UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryKey = adminQueryKeys.tenantStats(tenantId)

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getTenantUsageStats>>> = ({ signal }) =>
    getTenantUsageStats(tenantId, { signal })

  const queryOptions = {
    queryKey,
    queryFn,
    enabled: !!tenantId,
    staleTime: 30000,
    ...options?.query,
  } as UseQueryOptions<Awaited<ReturnType<typeof getTenantUsageStats>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>
  }

  const query = useQuery(queryOptions, queryClient) as UseQueryResult<TData, TError> & {
    queryKey: DataTag<QueryKey, TData, TError>
  }

  return { ...query, queryKey: queryOptions.queryKey }
}

/**
 * Hook to fetch admin audit logs
 */
export function useAdminAuditLogs<
  TData = Awaited<ReturnType<typeof getAdminAuditLogs>>,
  TError = Error,
>(
  params?: AdminAuditLogListParams,
  options?: {
    query?: Partial<UseQueryOptions<Awaited<ReturnType<typeof getAdminAuditLogs>>, TError, TData>>
  },
  queryClient?: QueryClient
): UseQueryResult<TData, TError> & { queryKey: DataTag<QueryKey, TData, TError> } {
  const queryKey = adminQueryKeys.auditLogs(params)

  const queryFn: QueryFunction<Awaited<ReturnType<typeof getAdminAuditLogs>>> = ({ signal }) =>
    getAdminAuditLogs(params, { signal })

  const queryOptions = {
    queryKey,
    queryFn,
    staleTime: 10000, // Audit logs should be relatively fresh
    ...options?.query,
  } as UseQueryOptions<Awaited<ReturnType<typeof getAdminAuditLogs>>, TError, TData> & {
    queryKey: DataTag<QueryKey, TData, TError>
  }

  const query = useQuery(queryOptions, queryClient) as UseQueryResult<TData, TError> & {
    queryKey: DataTag<QueryKey, TData, TError>
  }

  return { ...query, queryKey: queryOptions.queryKey }
}
