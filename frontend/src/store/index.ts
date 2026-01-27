// Store types
export type { User, AuthState, AuthActions, AppState, AppActions, BreadcrumbItem } from './types'
export type { FlagValue, FeatureFlagState, FeatureFlagActions } from './featureFlagStore'

// Auth store
export { useAuthStore, useUser, useIsAuthenticated, useAuthLoading } from './authStore'

// App store
export {
  useAppStore,
  useSidebarCollapsed,
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

// Store utilities
export { createSelectors, createStoreWithSelectors } from './createStore'
