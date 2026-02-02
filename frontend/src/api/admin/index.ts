/**
 * Admin API exports
 *
 * Provides platform-wide statistics and audit logging
 * functionality for super admin users.
 */
export {
  // Types
  type PlatformStatsResponse,
  type TenantGrowthPoint,
  type TenantGrowthTrendResponse,
  type AdminAuditLogEntry,
  type AdminAuditLogListResponse,
  type AdminAuditLogListParams,
  type TenantGrowthParams,
  type TenantUsageStatsResponse,
  // API Functions
  getPlatformStats,
  getTenantGrowthTrend,
  getTenantUsageStats,
  getAdminAuditLogs,
  // Query Keys
  adminQueryKeys,
  // React Query Hooks
  usePlatformStats,
  useTenantGrowthTrend,
  useTenantUsageStats,
  useAdminAuditLogs,
} from './admin'
