import { create } from 'zustand'
import { devtools, persist } from 'zustand/middleware'
import type { AppState, AppActions, BreadcrumbItem } from './types'

const STORAGE_KEY = 'erp-app-settings'

/**
 * Initial app state
 */
const initialState: AppState = {
  sidebarCollapsed: false,
  theme: 'light',
  locale: 'zh-CN',
  breadcrumbs: [],
  pageTitle: 'ERP System',
}

/**
 * App store for managing global application state
 *
 * Features:
 * - Sidebar collapse state (persistent)
 * - Theme switching (persistent)
 * - Locale settings (persistent)
 * - Page title and breadcrumbs (non-persistent)
 * - Devtools integration for debugging
 *
 * @example
 * ```tsx
 * import { useAppStore } from '@/store'
 *
 * function Sidebar() {
 *   const { sidebarCollapsed, toggleSidebar } = useAppStore()
 *
 *   return (
 *     <nav className={sidebarCollapsed ? 'collapsed' : ''}>
 *       <button onClick={toggleSidebar}>Toggle</button>
 *     </nav>
 *   )
 * }
 * ```
 */
export const useAppStore = create<AppState & AppActions>()(
  devtools(
    persist(
      (set, get) => ({
        ...initialState,

        toggleSidebar: () => {
          set(
            (state) => ({ sidebarCollapsed: !state.sidebarCollapsed }),
            false,
            'app/toggleSidebar'
          )
        },

        setSidebarCollapsed: (collapsed: boolean) => {
          set({ sidebarCollapsed: collapsed }, false, 'app/setSidebarCollapsed')
        },

        setTheme: (theme: 'light' | 'dark') => {
          set({ theme }, false, 'app/setTheme')
          // Apply theme to document for Semi Design
          document.body.setAttribute('theme-mode', theme)
        },

        toggleTheme: () => {
          const newTheme = get().theme === 'light' ? 'dark' : 'light'
          set({ theme: newTheme }, false, 'app/toggleTheme')
          document.body.setAttribute('theme-mode', newTheme)
        },

        setLocale: (locale: string) => {
          set({ locale }, false, 'app/setLocale')
        },

        setBreadcrumbs: (breadcrumbs: BreadcrumbItem[]) => {
          set({ breadcrumbs }, false, 'app/setBreadcrumbs')
        },

        setPageTitle: (title: string) => {
          set({ pageTitle: title }, false, 'app/setPageTitle')
          // Update document title
          document.title = title ? `${title} - ERP System` : 'ERP System'
        },
      }),
      {
        name: STORAGE_KEY,
        // Only persist user preferences, not page-specific state
        partialize: (state) => ({
          sidebarCollapsed: state.sidebarCollapsed,
          theme: state.theme,
          locale: state.locale,
        }),
        // Initialize theme on rehydration
        onRehydrateStorage: () => (state) => {
          if (state?.theme) {
            document.body.setAttribute('theme-mode', state.theme)
          }
        },
      }
    ),
    { name: 'AppStore' }
  )
)

/**
 * Selector hooks for common app state access patterns
 */
export const useSidebarCollapsed = () => useAppStore((state) => state.sidebarCollapsed)
export const useTheme = () => useAppStore((state) => state.theme)
export const useLocale = () => useAppStore((state) => state.locale)
export const useBreadcrumbs = () => useAppStore((state) => state.breadcrumbs)
export const usePageTitle = () => useAppStore((state) => state.pageTitle)
