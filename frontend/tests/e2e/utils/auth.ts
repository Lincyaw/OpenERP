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
  await page.waitForLoadState('networkidle')

  // Fill login form
  await page.fill('input[name="username"], input[placeholder*="用户名"], #username', user.username)
  await page.fill('input[name="password"], input[type="password"], #password', user.password)

  // Submit
  await page.click('button[type="submit"], .login-button, button:has-text("登录")')

  // Wait for navigation away from login page
  // Use a function that checks we're NOT on the login page
  await page.waitForFunction(
    () => !window.location.pathname.includes('/login'),
    { timeout: 15000 }
  ).catch(() => {
    // Navigation might have failed - continue to check auth state
  })

  // CRITICAL: Wait for auth state to be persisted
  // Note: Since SEC-004, access_token is kept in memory only (not localStorage)
  // We only check for user data in localStorage and erp-auth Zustand state
  await page.waitForFunction(
    () => {
      // Check for user in localStorage (user data is still persisted)
      const user = window.localStorage.getItem('user')
      if (!user) return false

      // Also check for erp-auth Zustand persisted state
      const erpAuth = window.localStorage.getItem('erp-auth')
      if (!erpAuth) return false

      try {
        const parsed = JSON.parse(erpAuth)
        // Check if user is stored in Zustand state
        return parsed?.state?.user !== null && parsed?.state?.user !== undefined
      } catch {
        return false
      }
    },
    { timeout: 10000 }
  )
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
 */
export async function isAuthenticated(page: Page): Promise<boolean> {
  const url = page.url()
  return !url.includes('/login')
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
  await page.evaluate(() => {
    window.localStorage.clear()
    window.sessionStorage.clear()
  })
}

/**
 * Reload page and wait for network idle
 */
export async function reloadAndWait(page: Page): Promise<void> {
  await page.reload()
  await page.waitForLoadState('networkidle')
}

/**
 * Get actual JWT token for API testing via direct login API call
 */
export async function getApiToken(
  page: Page,
  userType: TestUserType = 'admin'
): Promise<string | null> {
  const user = TEST_USERS[userType]
  // Determine the API base URL based on environment
  // E2E_BASE_URL is the frontend URL, we need the backend URL
  const frontendUrl = process.env.E2E_BASE_URL || 'http://localhost:3000'
  let apiBaseUrl: string

  if (frontendUrl.includes('erp-frontend')) {
    // Docker environment: frontend is erp-frontend:80, backend is erp-backend:8080
    apiBaseUrl = 'http://erp-backend:8080'
  } else if (frontendUrl.includes('localhost:3000')) {
    // Local development: backend is on localhost:8080
    apiBaseUrl = 'http://localhost:8080'
  } else {
    // Custom environment: try to derive backend URL
    apiBaseUrl = frontendUrl.replace(':3000', ':8080').replace(':80', ':8080')
  }

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
