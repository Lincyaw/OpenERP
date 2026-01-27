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
export {
  useFeatureFlagStore,
  useIsFeatureEnabled,
  useFeatureVariant,
  useFeatureFlag,
  useFeatureFlagsLoading,
  useFeatureFlagsReady,
  useFeatureFlagsError,
} from './featureFlagStore'

// Store utilities
export { createSelectors, createStoreWithSelectors } from './createStore'
