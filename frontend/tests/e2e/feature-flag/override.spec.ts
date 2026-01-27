import { test, expect } from '@playwright/test'
import { getApiToken } from '../utils/auth'
import { API_BASE_URL } from './api-utils'

/**
 * FF-INT-001: Feature Flag Override E2E Tests
 *
 * Tests the Feature Flag override functionality:
 * - Create user-level Override
 * - Verify the user sees the override value
 * - Verify other users see the default value
 * - Delete Override and verify default is restored
 *
 * Note: These tests use direct API calls since admin UI (FF-ADMIN-003)
 * is not yet implemented.
 */

test.describe('FF-INT-001: Feature Flag Overrides', () => {
  let adminToken: string | null = null
  let salesToken: string | null = null
  const testFlagKey = `override_test_flag_${Date.now()}`

  // Test user IDs from seed data
  const ADMIN_USER_ID = '00000000-0000-0000-0000-000000000002'
  const SALES_USER_ID = '00000000-0000-0000-0000-000000000003'

  test.beforeAll(async ({ browser }) => {
    const context = await browser.newContext()
    const page = await context.newPage()

    // Get tokens for both admin and sales users
    adminToken = await getApiToken(page, 'admin')
    salesToken = await getApiToken(page, 'sales')

    // Create a test flag for override tests
    await page.request.post(`${API_BASE_URL}/api/v1/feature-flags`, {
      headers: {
        Authorization: `Bearer ${adminToken}`,
        'Content-Type': 'application/json',
      },
      data: {
        key: testFlagKey,
        name: 'Override Test Flag',
        description: 'Flag for testing overrides',
        type: 'boolean',
        default_value: {
          enabled: false, // Default is disabled
        },
        tags: ['test', 'override'],
      },
    })

    // Enable the flag but with default value false
    await page.request.post(`${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/enable`, {
      headers: {
        Authorization: `Bearer ${adminToken}`,
      },
    })

    await context.close()
  })

  test.afterAll(async ({ browser }) => {
    // Cleanup: Archive the test flag
    const context = await browser.newContext()
    const page = await context.newPage()
    const token = await getApiToken(page, 'admin')

    await page.request.delete(`${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })

    await context.close()
  })

  test.describe('User-Level Override Flow', () => {
    let overrideId: string | null = null

    test('should evaluate flag as false for all users initially', async ({ request }) => {
      // Admin user evaluation
      const adminResponse = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/evaluate`,
        {
          headers: {
            Authorization: `Bearer ${adminToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            context: {
              user_id: ADMIN_USER_ID,
            },
          },
        }
      )

      expect(adminResponse.status()).toBe(200)
      const adminBody = await adminResponse.json()
      expect(adminBody.data.enabled).toBe(false)

      // Sales user evaluation
      const salesResponse = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/evaluate`,
        {
          headers: {
            Authorization: `Bearer ${salesToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            context: {
              user_id: SALES_USER_ID,
            },
          },
        }
      )

      expect(salesResponse.status()).toBe(200)
      const salesBody = await salesResponse.json()
      expect(salesBody.data.enabled).toBe(false)
    })

    test('should create a user-level override for sales user', async ({ request }) => {
      const response = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/overrides`,
        {
          headers: {
            Authorization: `Bearer ${adminToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            target_type: 'user',
            target_id: SALES_USER_ID,
            value: {
              enabled: true, // Override to enabled for sales user
            },
            reason: 'E2E test: Enable flag for sales user only',
          },
        }
      )

      expect(response.status()).toBe(201)
      const body = await response.json()
      expect(body.success).toBe(true)
      expect(body.data.target_type).toBe('user')
      expect(body.data.target_id).toBe(SALES_USER_ID)

      overrideId = body.data.id
    })

    test('should list overrides for the flag', async ({ request }) => {
      const response = await request.get(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/overrides`,
        {
          headers: {
            Authorization: `Bearer ${adminToken}`,
          },
        }
      )

      expect(response.status()).toBe(200)
      const body = await response.json()
      expect(body.success).toBe(true)
      expect(body.data.items).toBeDefined()
      expect(body.data.items.length).toBeGreaterThan(0)

      const override = body.data.items.find(
        (o: { target_id: string }) => o.target_id === SALES_USER_ID
      )
      expect(override).toBeDefined()
    })

    test('sales user should see override value (enabled)', async ({ request }) => {
      const response = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/evaluate`,
        {
          headers: {
            Authorization: `Bearer ${salesToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            context: {
              user_id: SALES_USER_ID,
            },
          },
        }
      )

      expect(response.status()).toBe(200)
      const body = await response.json()
      expect(body.data.enabled).toBe(true) // Override value
      expect(body.data.source).toBe('override') // Should indicate source is override
    })

    test('admin user should still see default value (disabled)', async ({ request }) => {
      const response = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/evaluate`,
        {
          headers: {
            Authorization: `Bearer ${adminToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            context: {
              user_id: ADMIN_USER_ID,
            },
          },
        }
      )

      expect(response.status()).toBe(200)
      const body = await response.json()
      expect(body.data.enabled).toBe(false) // Default value (no override)
    })

    test('should delete the override', async ({ request }) => {
      expect(overrideId).not.toBeNull()

      const response = await request.delete(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/overrides/${overrideId}`,
        {
          headers: {
            Authorization: `Bearer ${adminToken}`,
          },
        }
      )

      expect(response.status()).toBe(204)
    })

    test('sales user should see default value after override deletion', async ({ request }) => {
      const response = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/evaluate`,
        {
          headers: {
            Authorization: `Bearer ${salesToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            context: {
              user_id: SALES_USER_ID,
            },
          },
        }
      )

      expect(response.status()).toBe(200)
      const body = await response.json()
      expect(body.data.enabled).toBe(false) // Back to default value
    })
  })

  test.describe('Tenant-Level Override', () => {
    const TENANT_ID = '00000000-0000-0000-0000-000000000001'
    let tenantOverrideId: string | null = null

    test('should create a tenant-level override', async ({ request }) => {
      const response = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/overrides`,
        {
          headers: {
            Authorization: `Bearer ${adminToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            target_type: 'tenant',
            target_id: TENANT_ID,
            value: {
              enabled: true,
            },
            reason: 'E2E test: Enable flag for entire tenant',
          },
        }
      )

      expect(response.status()).toBe(201)
      const body = await response.json()
      expect(body.data.target_type).toBe('tenant')

      tenantOverrideId = body.data.id
    })

    test('all users in tenant should see override value', async ({ request }) => {
      // Both admin and sales are in the same tenant
      const adminResponse = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/evaluate`,
        {
          headers: {
            Authorization: `Bearer ${adminToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            context: {
              tenant_id: TENANT_ID,
            },
          },
        }
      )

      const adminBody = await adminResponse.json()
      expect(adminBody.data.enabled).toBe(true)

      const salesResponse = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/evaluate`,
        {
          headers: {
            Authorization: `Bearer ${salesToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            context: {
              tenant_id: TENANT_ID,
            },
          },
        }
      )

      const salesBody = await salesResponse.json()
      expect(salesBody.data.enabled).toBe(true)
    })

    test('should delete tenant override', async ({ request }) => {
      expect(tenantOverrideId).not.toBeNull()

      const response = await request.delete(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/overrides/${tenantOverrideId}`,
        {
          headers: {
            Authorization: `Bearer ${adminToken}`,
          },
        }
      )

      expect(response.status()).toBe(204)
    })
  })

  test.describe('Override with Expiration', () => {
    test('should create override with expiration time', async ({ request }) => {
      // Set expiration to 1 hour from now
      const expiresAt = new Date(Date.now() + 60 * 60 * 1000).toISOString()

      const response = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/overrides`,
        {
          headers: {
            Authorization: `Bearer ${adminToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            target_type: 'user',
            target_id: SALES_USER_ID,
            value: {
              enabled: true,
            },
            reason: 'E2E test: Temporary override with expiration',
            expires_at: expiresAt,
          },
        }
      )

      expect(response.status()).toBe(201)
      const body = await response.json()
      expect(body.data.expires_at).toBeDefined()

      // Cleanup
      await request.delete(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/overrides/${body.data.id}`,
        {
          headers: { Authorization: `Bearer ${adminToken}` },
        }
      )
    })
  })

  test.describe('Override Conflict Handling', () => {
    test('should reject duplicate override for same target', async ({ request }) => {
      // Create first override
      const firstResponse = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/overrides`,
        {
          headers: {
            Authorization: `Bearer ${adminToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            target_type: 'user',
            target_id: SALES_USER_ID,
            value: { enabled: true },
            reason: 'First override',
          },
        }
      )

      expect(firstResponse.status()).toBe(201)
      const firstBody = await firstResponse.json()

      // Try to create duplicate
      const duplicateResponse = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/overrides`,
        {
          headers: {
            Authorization: `Bearer ${adminToken}`,
            'Content-Type': 'application/json',
          },
          data: {
            target_type: 'user',
            target_id: SALES_USER_ID,
            value: { enabled: false },
            reason: 'Duplicate override',
          },
        }
      )

      expect(duplicateResponse.status()).toBe(409)

      // Cleanup
      await request.delete(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/overrides/${firstBody.data.id}`,
        {
          headers: { Authorization: `Bearer ${adminToken}` },
        }
      )
    })
  })
})
