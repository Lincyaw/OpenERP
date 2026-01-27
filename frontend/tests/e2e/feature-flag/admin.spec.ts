import { test, expect } from '@playwright/test'
import { getApiToken } from '../utils/auth'
import { API_BASE_URL } from './api-utils'

/**
 * FF-INT-001: Feature Flag Admin E2E Tests
 *
 * Tests the Feature Flag management API functionality:
 * - Create Boolean Flag -> Verify via API
 * - Enable Flag -> Verify status change
 * - Update Flag -> Verify save success
 * - Archive Flag -> Verify flag is archived
 *
 * Note: These tests use direct API calls since admin UI (FF-ADMIN-001/002/003)
 * is not yet implemented. Tests will be updated to use UI once available.
 */

test.describe('FF-INT-001: Feature Flag Admin (API)', () => {
  let authToken: string | null = null
  const testFlagKey = `test_flag_${Date.now()}`

  test.beforeAll(async ({ browser }) => {
    // Get auth token for API calls
    const context = await browser.newContext()
    const page = await context.newPage()
    authToken = await getApiToken(page, 'admin')
    await context.close()
  })

  test.describe('Flag Management Flow', () => {
    test('should create a Boolean feature flag via API', async ({ request }) => {
      const response = await request.post(`${API_BASE_URL}/api/v1/feature-flags`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
          'Content-Type': 'application/json',
        },
        data: {
          key: testFlagKey,
          name: 'Test Boolean Flag',
          description: 'A test flag for E2E testing',
          type: 'boolean',
          default_value: {
            enabled: false,
          },
          tags: ['test', 'e2e'],
        },
      })

      expect(response.status()).toBe(201)
      const body = await response.json()
      expect(body.success).toBe(true)
      expect(body.data.key).toBe(testFlagKey)
      expect(body.data.type).toBe('boolean')
      expect(body.data.status).toBe('disabled')
    })

    test('should list feature flags and find created flag', async ({ request }) => {
      const response = await request.get(`${API_BASE_URL}/api/v1/feature-flags`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      })

      expect(response.status()).toBe(200)
      const body = await response.json()
      expect(body.success).toBe(true)
      expect(body.data.items).toBeDefined()

      const createdFlag = body.data.items.find((f: { key: string }) => f.key === testFlagKey)
      expect(createdFlag).toBeDefined()
      expect(createdFlag.name).toBe('Test Boolean Flag')
    })

    test('should enable the feature flag', async ({ request }) => {
      const response = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/enable`,
        {
          headers: {
            Authorization: `Bearer ${authToken}`,
          },
        }
      )

      expect(response.status()).toBe(200)
      const body = await response.json()
      expect(body.success).toBe(true)

      // Verify flag is now enabled
      const getResponse = await request.get(`${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      })
      const getBody = await getResponse.json()
      expect(getBody.data.status).toBe('enabled')
    })

    test('should update the feature flag', async ({ request }) => {
      const response = await request.put(`${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
          'Content-Type': 'application/json',
        },
        data: {
          name: 'Updated Test Flag',
          description: 'Updated description for E2E testing',
          tags: ['test', 'e2e', 'updated'],
        },
      })

      expect(response.status()).toBe(200)
      const body = await response.json()
      expect(body.success).toBe(true)
      expect(body.data.name).toBe('Updated Test Flag')
      expect(body.data.description).toBe('Updated description for E2E testing')
    })

    test('should disable the feature flag', async ({ request }) => {
      const response = await request.post(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/disable`,
        {
          headers: {
            Authorization: `Bearer ${authToken}`,
          },
        }
      )

      expect(response.status()).toBe(200)

      // Verify flag is now disabled
      const getResponse = await request.get(`${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      })
      const getBody = await getResponse.json()
      expect(getBody.data.status).toBe('disabled')
    })

    test('should archive the feature flag', async ({ request }) => {
      const response = await request.delete(`${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      })

      expect(response.status()).toBe(204)

      // Verify flag is archived (GET should still work but status is archived)
      const getResponse = await request.get(`${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      })
      const getBody = await getResponse.json()
      expect(getBody.data.status).toBe('archived')
    })

    test('should get audit logs for flag operations', async ({ request }) => {
      const response = await request.get(
        `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/audit-logs`,
        {
          headers: {
            Authorization: `Bearer ${authToken}`,
          },
        }
      )

      expect(response.status()).toBe(200)
      const body = await response.json()
      expect(body.success).toBe(true)
      expect(body.data.items).toBeDefined()

      // Should have audit logs for: created, enabled, updated, disabled, archived
      const actions = body.data.items.map((log: { action: string }) => log.action)
      expect(actions).toContain('created')
    })
  })

  test.describe('Flag Type Validation', () => {
    test('should create a percentage type flag', async ({ request }) => {
      const percentageFlagKey = `test_percentage_${Date.now()}`

      const response = await request.post(`${API_BASE_URL}/api/v1/feature-flags`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
          'Content-Type': 'application/json',
        },
        data: {
          key: percentageFlagKey,
          name: 'Test Percentage Flag',
          description: 'A percentage rollout flag',
          type: 'percentage',
          default_value: {
            enabled: false,
            metadata: {
              percentage: 50,
            },
          },
          tags: ['test', 'percentage'],
        },
      })

      expect(response.status()).toBe(201)
      const body = await response.json()
      expect(body.data.type).toBe('percentage')

      // Cleanup
      await request.delete(`${API_BASE_URL}/api/v1/feature-flags/${percentageFlagKey}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      })
    })

    test('should create a variant type flag', async ({ request }) => {
      const variantFlagKey = `test_variant_${Date.now()}`

      const response = await request.post(`${API_BASE_URL}/api/v1/feature-flags`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
          'Content-Type': 'application/json',
        },
        data: {
          key: variantFlagKey,
          name: 'Test Variant Flag',
          description: 'A/B/C variant flag',
          type: 'variant',
          default_value: {
            enabled: false,
            variant: 'A',
            metadata: {
              variants: ['A', 'B', 'C'],
            },
          },
          tags: ['test', 'variant'],
        },
      })

      expect(response.status()).toBe(201)
      const body = await response.json()
      expect(body.data.type).toBe('variant')

      // Cleanup
      await request.delete(`${API_BASE_URL}/api/v1/feature-flags/${variantFlagKey}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      })
    })

    test('should reject invalid flag type', async ({ request }) => {
      const response = await request.post(`${API_BASE_URL}/api/v1/feature-flags`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
          'Content-Type': 'application/json',
        },
        data: {
          key: 'invalid_type_flag',
          name: 'Invalid Type Flag',
          type: 'invalid_type',
          default_value: { enabled: false },
        },
      })

      expect(response.status()).toBe(400)
    })

    test('should reject duplicate flag key', async ({ request }) => {
      const duplicateKey = `duplicate_test_${Date.now()}`

      // Create first flag
      await request.post(`${API_BASE_URL}/api/v1/feature-flags`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
          'Content-Type': 'application/json',
        },
        data: {
          key: duplicateKey,
          name: 'First Flag',
          type: 'boolean',
          default_value: { enabled: false },
        },
      })

      // Try to create duplicate
      const response = await request.post(`${API_BASE_URL}/api/v1/feature-flags`, {
        headers: {
          Authorization: `Bearer ${authToken}`,
          'Content-Type': 'application/json',
        },
        data: {
          key: duplicateKey,
          name: 'Duplicate Flag',
          type: 'boolean',
          default_value: { enabled: false },
        },
      })

      expect(response.status()).toBe(409)

      // Cleanup
      await request.delete(`${API_BASE_URL}/api/v1/feature-flags/${duplicateKey}`, {
        headers: { Authorization: `Bearer ${authToken}` },
      })
    })
  })
})
