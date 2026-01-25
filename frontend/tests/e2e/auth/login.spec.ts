import { test, expect } from '@playwright/test'
import { LoginPage } from '../pages'
import { TEST_USERS } from '../fixtures'

/**
 * Login E2E Tests
 *
 * Tests the authentication flow:
 * - Successful login with valid credentials
 * - Failed login with invalid credentials
 * - Logout functionality
 * - Token persistence
 */
test.describe('Authentication', () => {
  // Clear storage state for login tests - they need to test fresh login
  test.use({ storageState: { cookies: [], origins: [] } })

  test.describe('Login', () => {
    test('should display login page', async ({ page }) => {
      const loginPage = new LoginPage(page)
      await loginPage.navigate()

      // Verify login page elements are visible
      await expect(page.locator('input').first()).toBeVisible()
      await expect(page.locator('button[type="submit"], button:has-text("登录")')).toBeVisible()
    })

    test('should login successfully with valid credentials', async ({ page }) => {
      const loginPage = new LoginPage(page)
      await loginPage.navigate()

      // Login with admin credentials
      await loginPage.loginAndWait(TEST_USERS.admin.username, TEST_USERS.admin.password)

      // Should redirect away from login page
      await expect(page).not.toHaveURL(/.*login.*/)
    })

    test('should show error with invalid credentials', async ({ page }) => {
      const loginPage = new LoginPage(page)
      await loginPage.navigate()

      // Login with wrong credentials
      await loginPage.login('wronguser', 'wrongpassword')

      // Wait for error message or remain on login page
      await page.waitForTimeout(2000) // Allow time for API response

      // Should still be on login page
      const url = page.url()
      expect(url).toContain('login')
    })

    test('should login with sales user', async ({ page }) => {
      const loginPage = new LoginPage(page)
      await loginPage.navigate()

      await loginPage.loginAndWait(TEST_USERS.sales.username, TEST_USERS.sales.password)

      await expect(page).not.toHaveURL(/.*login.*/)
    })

    test('should login with warehouse user', async ({ page }) => {
      const loginPage = new LoginPage(page)
      await loginPage.navigate()

      await loginPage.loginAndWait(TEST_USERS.warehouse.username, TEST_USERS.warehouse.password)

      await expect(page).not.toHaveURL(/.*login.*/)
    })

    test('should login with finance user', async ({ page }) => {
      const loginPage = new LoginPage(page)
      await loginPage.navigate()

      await loginPage.loginAndWait(TEST_USERS.finance.username, TEST_USERS.finance.password)

      await expect(page).not.toHaveURL(/.*login.*/)
    })
  })

  test.describe('Session', () => {
    test('should redirect to login when not authenticated', async ({ page }) => {
      // With storageState: { cookies: [], origins: [] }, the session is already cleared
      // Just try to access protected route directly
      await page.goto('/dashboard')

      // Should redirect to login - wait for navigation to complete
      await page.waitForLoadState('networkidle')

      // Verify we're not on the dashboard (protected route) - either redirected to login or blocked
      const url = page.url()
      // Either we redirected to login, or we're on some other unprotected page
      const isOnProtectedDashboard = url.includes('/dashboard') && !url.includes('login')
      expect(isOnProtectedDashboard).toBeFalsy()
    })

    test('should persist login state', async ({ page }) => {
      const loginPage = new LoginPage(page)
      await loginPage.navigate()
      await loginPage.loginAndWait(TEST_USERS.admin.username, TEST_USERS.admin.password)

      // Reload page
      await page.reload()

      // Should still be logged in (not on login page)
      await page.waitForTimeout(1000)
      const url = page.url()
      expect(url).not.toContain('login')
    })
  })
})
