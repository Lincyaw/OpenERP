import { type Page, type BrowserContext } from '@playwright/test'
import { TEST_USERS, type TestUserType } from '../fixtures'

/**
 * Login helper - authenticates a user and stores session
 * Note: Since SEC-004, access_token is stored in memory only (not localStorage)
 * We check for user data in localStorage as indicator of successful login
 */
export async function login(page: Page, userType: TestUserType = 'admin'): Promise<void> {
  const user = TEST_USERS[userType]

  await page.goto('/login')
  // Use domcontentloaded instead of networkidle as app may have continuous polling
  await page.waitForLoadState('domcontentloaded')

  // Wait briefly for any redirect to complete
  await page.waitForTimeout(500)

  // Check if user is already logged in (app redirected from /login to dashboard)
  const currentUrl = page.url()
  console.log(`[login] Current URL after goto('/login'): ${currentUrl}`)

  if (!currentUrl.includes('/login')) {
    // Already logged in, verify auth state is present
    const isLoggedIn = await page.evaluate(() => {
      // Check for user in localStorage (primary indicator of logged-in state)
      const userStr = window.localStorage.getItem('user')
      if (!userStr) return false
      try {
        const user = JSON.parse(userStr)
        return user && typeof user.id === 'string' && user.id.length > 0
      } catch {
        return false
      }
    })
    console.log(`[login] Already redirected away from login. isLoggedIn: ${isLoggedIn}`)
    if (isLoggedIn) {
      return // Already authenticated, skip login process
    }
  }

  // Wait for login form to be visible
  await page.waitForSelector('input[type="password"], #password', {
    state: 'visible',
    timeout: 10000,
  })

  // Fill login form
  await page.fill('input[name="username"], input[placeholder*="用户名"], #username', user.username)
  await page.fill('input[name="password"], input[type="password"], #password', user.password)

  // Submit
  await page.click('button[type="submit"], .login-button, button:has-text("登录")')

  // Wait for navigation away from login page
  // Use a function that checks we're NOT on the login page
  await page
    .waitForFunction(() => !window.location.pathname.includes('/login'), { timeout: 15000 })
    .catch(() => {
      // Navigation might have failed - continue to check auth state
    })

  // CRITICAL: Wait for auth state to be persisted
  // SEC-004: access_token is kept in memory only (not localStorage)
  // refresh_token is stored as httpOnly cookie (handled by browser)
  // We only need to check for user data in localStorage
  await page.waitForFunction(
    () => {
      // Check for user in localStorage (user data is still persisted synchronously)
      // This is the authoritative indicator of successful login
      const userStr = window.localStorage.getItem('user')
      if (!userStr) return false

      try {
        const user = JSON.parse(userStr)
        // Verify user object has required fields
        return user && typeof user.id === 'string' && user.id.length > 0
      } catch {
        return false
      }
    },
    { timeout: 15000 }
  )

  // Brief wait to ensure httpOnly cookie is properly set by the browser
  // The refresh_token cookie is set by the backend response
  await page.waitForTimeout(200)
}

/**
 * Logout helper - logs out the current user
 */
export async function logout(page: Page): Promise<void> {
  // Click user menu/avatar
  await page.click('.user-menu, .semi-avatar, [data-testid="user-menu"]')

  // Click logout button
  await page.click('button:has-text("登出"), button:has-text("退出"), [data-testid="logout"]')

  // Wait for redirect to login
  await page.waitForURL('**/login**')
}

/**
 * Save authentication state to file for reuse
 */
export async function saveAuthState(
  context: BrowserContext,
  path: string = 'tests/e2e/.auth/user.json'
): Promise<void> {
  await context.storageState({ path })
}

/**
 * Check if user is authenticated
 * Checks both URL (not on login page) and localStorage for user data
 */
export async function isAuthenticated(page: Page): Promise<boolean> {
  const url = page.url()
  if (url.includes('/login')) {
    return false
  }

  // Also verify user data exists in localStorage
  const hasUser = await page.evaluate(() => {
    const userStr = window.localStorage.getItem('user')
    if (!userStr) return false
    try {
      const user = JSON.parse(userStr)
      return user && typeof user.id === 'string' && user.id.length > 0
    } catch {
      return false
    }
  })

  return hasUser
}

/**
 * Wait for API call to complete
 */
export async function waitForApi(
  page: Page,
  urlPattern: string | RegExp,
  options?: { timeout?: number }
): Promise<void> {
  await page.waitForResponse(urlPattern, { timeout: options?.timeout || 30000 })
}

/**
 * Get authentication token from storage
 *
 * SEC-004: Access tokens are now stored in memory only, not in localStorage.
 * This function cannot retrieve the token from memory state.
 * Use getApiToken() instead if you need to make authenticated API calls in tests.
 *
 * @deprecated Use getApiToken() for API authentication or verify login state via user data
 */
export async function getAuthToken(page: Page): Promise<string | null> {
  // SEC-004: Token is no longer stored in localStorage, it's kept in memory
  // We check for user data as indicator of successful login instead
  const userData = await page.evaluate(() => window.localStorage.getItem('user'))
  if (userData) {
    // User is logged in, but we can't access the token from memory
    // Return a placeholder to indicate authenticated state for legacy tests
    // Tests should be updated to use getApiToken() or check user data directly
    return '[token-in-memory]'
  }
  return null
}

/**
 * Clear all authentication data
 */
export async function clearAuth(page: Page): Promise<void> {
  try {
    await page.evaluate(() => {
      window.localStorage.clear()
      window.sessionStorage.clear()
    })
  } catch {
    // SecurityError may occur if page is on a different origin (e.g., about:blank)
    // Navigate to the app first, then clear storage
    try {
      await page.goto('/', { waitUntil: 'domcontentloaded', timeout: 5000 })
      await page.evaluate(() => {
        window.localStorage.clear()
        window.sessionStorage.clear()
      })
    } catch {
      // If still failing, just continue - storage might be empty anyway
    }
  }
}

/**
 * Wait for page to be ready (more robust than networkidle)
 * This handles apps with persistent connections like SSE or WebSocket
 */
export async function waitForPageReady(page: Page, options?: { timeout?: number }): Promise<void> {
  const timeout = options?.timeout || 10000

  // Wait for DOM to be loaded
  await page.waitForLoadState('domcontentloaded')

  // Wait for main content area to be visible (Semi Design layout)
  await page
    .locator('.semi-layout-content, main, [role="main"], .page-content')
    .first()
    .waitFor({ state: 'visible', timeout })
    .catch(() => {
      // Content might already be visible or use different structure
    })

  // Wait for any loading spinners to disappear
  await page
    .locator('.semi-spin-spinning, .loading-spinner, [data-testid="loading"]')
    .first()
    .waitFor({ state: 'hidden', timeout: 5000 })
    .catch(() => {
      // Spinner might not exist or already hidden
    })
}

/**
 * Reload page and wait for it to be ready
 */
export async function reloadAndWait(page: Page): Promise<void> {
  await page.reload()
  await waitForPageReady(page)
}

/**
 * Get the API base URL based on environment
 * Works in both local (localhost:8080) and Docker (erp-backend:8080) environments
 */
export function getApiBaseUrl(): string {
  const frontendUrl = process.env.E2E_BASE_URL || 'http://localhost:3000'

  if (frontendUrl.includes('erp-frontend')) {
    // Docker environment: frontend is erp-frontend:80, backend is erp-backend:8080
    return 'http://erp-backend:8080'
  } else if (frontendUrl.includes('localhost:3000')) {
    // Local development: backend is on localhost:8080
    return 'http://localhost:8080'
  } else {
    // Custom environment: try to derive backend URL
    return frontendUrl.replace(':3000', ':8080').replace(':80', ':8080')
  }
}

/**
 * Get actual JWT token for API testing via direct login API call
 */
export async function getApiToken(
  page: Page,
  userType: TestUserType = 'admin'
): Promise<string | null> {
  const user = TEST_USERS[userType]
  const apiBaseUrl = getApiBaseUrl()

  try {
    const response = await page.request.post(`${apiBaseUrl}/api/v1/auth/login`, {
      data: {
        username: user.username,
        password: user.password,
      },
    })

    if (response.ok()) {
      const data = await response.json()
      return data?.data?.token?.access_token || null
    }
    return null
  } catch (error) {
    console.error('Failed to get API token:', error)
    return null
  }
}
