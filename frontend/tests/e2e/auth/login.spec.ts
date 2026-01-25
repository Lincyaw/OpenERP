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
      // Clear any existing session
      await page.context().clearCookies()
      await page.evaluate(() => {
        window.localStorage.clear()
        window.sessionStorage.clear()
      })

      // Try to access protected route
      await page.goto('/dashboard')

      // Should redirect to login
      await expect(page).toHaveURL(/.*login.*/)
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
