// Store types
export type { User, AuthState, AuthActions, AppState, AppActions, BreadcrumbItem } from './types'
export { SYSTEM_TENANT_ID, SUPER_ADMIN_ROLE_ID } from './types'
export type { FlagValue, FeatureFlagState, FeatureFlagActions } from './featureFlagStore'

// Auth store
export {
  useAuthStore,
  useUser,
  useIsAuthenticated,
  useAuthLoading,
  useIsSuperAdmin,
} from './authStore'

// App store
export {
  useAppStore,
  useSidebarCollapsed,
  useMobileSidebarOpen,
  useTheme,
  useLocale,
  useBreadcrumbs,
  usePageTitle,
} from './appStore'

// Feature flag store
// Note: Selector hooks (useFeatureFlag, useFeatureVariant, etc.) are provided
// by @/hooks/useFeatureFlag for type-safe, documented API.
// Only export the store itself here for advanced use cases.
export { useFeatureFlagStore } from './featureFlagStore'

// Feature store (SaaS plan-based features)
// Note: Selector hooks (useHasFeature, useFeatureLimit, etc.) are provided
// by @/hooks/useFeature for type-safe, documented API.
export type {
  TenantPlan,
  FeatureKey,
  FeatureDefinition,
  FeatureState,
  FeatureActions,
} from './featureStore'
export {
  useFeatureStore,
  useTenantPlan,
  useFeaturesReady,
  useHasFeature,
  useFeatureLimit,
  useRequiredPlan,
  isPlanHigherOrEqual,
  getNextPlan,
  getPlanDisplayName,
  getAllFeatureKeys,
} from './featureStore'

// Store utilities
export { createSelectors, createStoreWithSelectors } from './createStore'

// Admin store
export type { AdminState, AdminActions } from './adminStore'
export {
  useAdminStore,
  useSelectedTenantId,
  useSelectedTenantName,
  useOperationLoading,
  useAdminError,
  // Enhanced hooks with optimistic updates
  useAdminTenants,
  useAdminTenant,
  useAdminTenantStats,
  useAdminPlatformStats,
  useAdminGrowthTrend,
  useAdminTenantUsage,
  useAdminAuditLogs,
  useCreateAdminTenant,
  useUpdateAdminTenant,
  useDeleteAdminTenant,
  useSuspendAdminTenant,
  useActivateAdminTenant,
  useChangePlanAdminTenant,
  useUpdateQuotaAdminTenant,
  // Combined operations hook
  useAdminOperations,
} from './adminStore'
