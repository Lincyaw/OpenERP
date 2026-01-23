/**
 * User information stored in auth state
 */
export interface User {
  id: string
  username: string
  email?: string
  displayName?: string
  avatar?: string
  tenantId?: string
  permissions?: string[]
  roles?: string[]
}

/**
 * Authentication state
 */
export interface AuthState {
  /** Current authenticated user */
  user: User | null
  /** JWT access token */
  accessToken: string | null
  /** JWT refresh token */
  refreshToken: string | null
  /** Whether authentication is being checked */
  isLoading: boolean
  /** Whether user is authenticated */
  isAuthenticated: boolean
}

/**
 * Authentication actions
 */
export interface AuthActions {
  /** Set user after successful login */
  setUser: (user: User) => void
  /** Set tokens after successful login */
  setTokens: (accessToken: string, refreshToken?: string) => void
  /** Login with credentials (sets user and tokens) */
  login: (user: User, accessToken: string, refreshToken?: string) => void
  /** Logout and clear all auth state */
  logout: () => void
  /** Update user information */
  updateUser: (updates: Partial<User>) => void
  /** Set loading state */
  setLoading: (isLoading: boolean) => void
  /** Check if user has specific permission */
  hasPermission: (permission: string) => boolean
  /** Check if user has any of the specified permissions */
  hasAnyPermission: (permissions: string[]) => boolean
  /** Check if user has all specified permissions */
  hasAllPermissions: (permissions: string[]) => boolean
  /** Initialize auth state from localStorage */
  initialize: () => void
}

/**
 * App settings state
 */
export interface AppState {
  /** Whether sidebar is collapsed */
  sidebarCollapsed: boolean
  /** Current theme (light/dark) */
  theme: 'light' | 'dark'
  /** Current locale */
  locale: string
  /** Breadcrumb items */
  breadcrumbs: BreadcrumbItem[]
  /** Page title */
  pageTitle: string
}

/**
 * App settings actions
 */
export interface AppActions {
  /** Toggle sidebar collapsed state */
  toggleSidebar: () => void
  /** Set sidebar collapsed state */
  setSidebarCollapsed: (collapsed: boolean) => void
  /** Set theme */
  setTheme: (theme: 'light' | 'dark') => void
  /** Toggle theme between light and dark */
  toggleTheme: () => void
  /** Set locale */
  setLocale: (locale: string) => void
  /** Set breadcrumbs */
  setBreadcrumbs: (breadcrumbs: BreadcrumbItem[]) => void
  /** Set page title */
  setPageTitle: (title: string) => void
}

/**
 * Breadcrumb item
 */
export interface BreadcrumbItem {
  /** Display title */
  title: string
  /** Route path (optional - no link if not provided) */
  path?: string
  /** Icon component (optional) */
  icon?: React.ReactNode
}
