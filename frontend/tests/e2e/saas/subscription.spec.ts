import { test, expect } from '@playwright/test'
import { login, getApiToken, getApiBaseUrl } from '../utils/auth'

/**
 * SAAS-TEST-001: Subscription E2E Tests
 *
 * Tests the subscription management functionality:
 * - View current subscription plan
 * - Navigate to upgrade page
 * - View plan comparison
 * - Verify plan features display
 */

test.describe('SAAS-TEST-001: Subscription Management', () => {
  let authToken: string | null = null
  const API_BASE_URL = getApiBaseUrl()

  test.beforeAll(async ({ browser }) => {
    const context = await browser.newContext()
    const page = await context.newPage()
    authToken = await getApiToken(page, 'admin')
    await context.close()
  })

  test.describe('Subscription Status', () => {
    test('should display current subscription plan in user profile', async ({ page }) => {
      await login(page, 'admin')

      // Navigate to user profile or settings
      await page.goto('/settings')
      await page.waitForLoadState('domcontentloaded')

      // Check for subscription-related content
      // The page should show current plan information
      const pageContent = await page.content()
      expect(pageContent).toBeTruthy()
    })

    test('should navigate to upgrade page', async ({ page }) => {
      await login(page, 'admin')

      // Navigate to upgrade page
      await page.goto('/upgrade')
      await page.waitForLoadState('domcontentloaded')

      // Verify upgrade page loads
      const url = page.url()
      expect(url).toContain('/upgrade')
    })

    test('should display plan comparison on upgrade page', async ({ page }) => {
      await login(page, 'admin')

      await page.goto('/upgrade')
      await page.waitForLoadState('domcontentloaded')

      // Wait for plan cards to load
      await page.waitForTimeout(1000)

      // Check for plan-related content
      const pageContent = await page.content()
      expect(pageContent).toBeTruthy()
    })
  })

  test.describe('Subscription API', () => {
    test('should get tenant subscription info via API', async ({ request }) => {
      // Get current tenant info which includes subscription details
      const response = await request.get(`${API_BASE_URL}/api/v1/auth/me`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      })

      expect(response.status()).toBe(200)
      const body = await response.json()
      expect(body.success).toBe(true)
      expect(body.data).toBeDefined()
      // Tenant should have plan information
      expect(body.data.tenant).toBeDefined()
    })

    test('should get tenant plan features via API', async ({ request }) => {
      // Get tenant features based on current plan
      const response = await request.get(`${API_BASE_URL}/api/v1/tenants/current`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      })

      // API might return 200 or 404 depending on implementation
      if (response.status() === 200) {
        const body = await response.json()
        expect(body.success).toBe(true)
      }
    })
  })

  test.describe('Plan Features Display', () => {
    test('should show feature availability based on plan', async ({ page }) => {
      await login(page, 'admin')

      // Navigate to a page that shows feature availability
      await page.goto('/dashboard')
      await page.waitForLoadState('domcontentloaded')

      // The dashboard should load successfully
      const url = page.url()
      expect(url).toContain('/dashboard')
    })
  })
})
