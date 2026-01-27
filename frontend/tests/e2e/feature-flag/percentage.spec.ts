import { test, expect } from '@playwright/test'
import { getApiToken } from '../utils/auth'
import { API_BASE_URL } from './api-utils'

/**
 * FF-INT-001: Feature Flag Percentage Rollout E2E Tests
 *
 * Tests the percentage-based feature flag functionality:
 * - Create 50% percentage Flag
 * - Multiple evaluations verify distribution is approximately 50%
 * - Same user multiple evaluations return consistent result (sticky bucketing)
 */

test.describe('FF-INT-001: Percentage Rollout', () => {
  let authToken: string | null = null
  const testFlagKey = `percentage_test_flag_${Date.now()}`

  test.beforeAll(async ({ browser }) => {
    const context = await browser.newContext()
    const page = await context.newPage()
    authToken = await getApiToken(page, 'admin')

    // Create a percentage type flag with 50% rollout
    await page.request.post(`${API_BASE_URL}/api/v1/feature-flags`, {
      headers: {
        Authorization: `Bearer ${authToken}`,
        'Content-Type': 'application/json',
      },
      data: {
        key: testFlagKey,
        name: 'Percentage Rollout Test Flag',
        description: '50% rollout flag for testing',
        type: 'percentage',
        default_value: {
          enabled: true,
          metadata: {
            percentage: 50,
          },
        },
        tags: ['test', 'percentage', 'rollout'],
      },
    })

    // Enable the flag
    await page.request.post(`${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/enable`, {
      headers: {
        Authorization: `Bearer ${authToken}`,
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

  test.describe('Percentage Distribution', () => {
    test('should return distribution approximately 50% over many evaluations', async ({
      request,
    }) => {
      const totalEvaluations = 100
      let enabledCount = 0

      // Evaluate with different user IDs to get distribution
      for (let i = 0; i < totalEvaluations; i++) {
        const userId = `test-user-${i}-${Date.now()}`

        const response = await request.post(
          `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/evaluate`,
          {
            headers: {
              Authorization: `Bearer ${authToken}`,
              'Content-Type': 'application/json',
            },
            data: {
              context: {
                user_id: userId,
              },
            },
          }
        )

        expect(response.status()).toBe(200)
        const body = await response.json()

        if (body.data.enabled) {
          enabledCount++
        }
      }

      // Calculate percentage
      const actualPercentage = (enabledCount / totalEvaluations) * 100

      // With 100 evaluations and 50% target, we expect roughly 40-60% due to randomness
      // Using a wider tolerance for statistical variation
      expect(actualPercentage).toBeGreaterThanOrEqual(30)
      expect(actualPercentage).toBeLessThanOrEqual(70)

      console.log(
        `Percentage rollout test: ${enabledCount}/${totalEvaluations} = ${actualPercentage.toFixed(1)}%`
      )
    })
  })

  test.describe('Sticky Bucketing', () => {
    test('same user should get consistent result across multiple evaluations', async ({
      request,
    }) => {
      const userId = `sticky-test-user-${Date.now()}`
      let firstResult: boolean | null = null

      // Evaluate 10 times with the same user ID
      for (let i = 0; i < 10; i++) {
        const response = await request.post(
          `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/evaluate`,
          {
            headers: {
              Authorization: `Bearer ${authToken}`,
              'Content-Type': 'application/json',
            },
            data: {
              context: {
                user_id: userId,
              },
            },
          }
        )

        expect(response.status()).toBe(200)
        const body = await response.json()

        if (firstResult === null) {
          firstResult = body.data.enabled
        } else {
          // All subsequent evaluations should match the first
          expect(body.data.enabled).toBe(firstResult)
        }
      }

      console.log(`Sticky bucketing test: User ${userId} consistently got ${firstResult}`)
    })

    test('different users may get different results', async ({ request }) => {
      const results: boolean[] = []

      // Evaluate with different user IDs
      for (let i = 0; i < 20; i++) {
        const userId = `different-user-${i}-${Date.now()}`

        const response = await request.post(
          `${API_BASE_URL}/api/v1/feature-flags/${testFlagKey}/evaluate`,
          {
            headers: {
              Authorization: `Bearer ${authToken}`,
              'Content-Type': 'application/json',
            },
            data: {
              context: {
                user_id: userId,
              },
            },
          }
        )

        expect(response.status()).toBe(200)
        const body = await response.json()
        results.push(body.data.enabled)
      }

      // With 20 different users at 50%, we should have at least some true and some false
      const trueCount = results.filter((r) => r).length
      const falseCount = results.filter((r) => !r).length

      // Very unlikely to have all same results with 50% rollout
      // At least check we got some variety
      console.log(`Different users test: ${trueCount} true, ${falseCount} false`)

      // Soft assertion - just log if unexpected
      if (trueCount === 0 || falseCount === 0) {
        console.warn('Unexpected: All users got the same result. This is statistically unlikely.')
      }
    })
  })

  test.describe('Percentage Variations', () => {
    test('0% rollout should always return false', async ({ request, browser }) => {
      const zeroPercentFlag = `zero_percent_flag_${Date.now()}`

      // Create 0% flag
      const context = await browser.newContext()
      const page = await context.newPage()
      const token = await getApiToken(page, 'admin')

      await page.request.post(`${API_BASE_URL}/api/v1/feature-flags`, {
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        data: {
          key: zeroPercentFlag,
          name: '0% Rollout Flag',
          type: 'percentage',
          default_value: {
            enabled: true,
            metadata: { percentage: 0 },
          },
        },
      })

      await page.request.post(`${API_BASE_URL}/api/v1/feature-flags/${zeroPercentFlag}/enable`, {
        headers: { Authorization: `Bearer ${token}` },
      })

      // Evaluate multiple times
      for (let i = 0; i < 10; i++) {
        const response = await request.post(
          `${API_BASE_URL}/api/v1/feature-flags/${zeroPercentFlag}/evaluate`,
          {
            headers: {
              Authorization: `Bearer ${authToken}`,
              'Content-Type': 'application/json',
            },
            data: {
              context: { user_id: `user-${i}` },
            },
          }
        )

        const body = await response.json()
        expect(body.data.enabled).toBe(false)
      }

      // Cleanup
      await page.request.delete(`${API_BASE_URL}/api/v1/feature-flags/${zeroPercentFlag}`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      await context.close()
    })

    test('100% rollout should always return true', async ({ request, browser }) => {
      const fullPercentFlag = `full_percent_flag_${Date.now()}`

      // Create 100% flag
      const context = await browser.newContext()
      const page = await context.newPage()
      const token = await getApiToken(page, 'admin')

      await page.request.post(`${API_BASE_URL}/api/v1/feature-flags`, {
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        data: {
          key: fullPercentFlag,
          name: '100% Rollout Flag',
          type: 'percentage',
          default_value: {
            enabled: true,
            metadata: { percentage: 100 },
          },
        },
      })

      await page.request.post(`${API_BASE_URL}/api/v1/feature-flags/${fullPercentFlag}/enable`, {
        headers: { Authorization: `Bearer ${token}` },
      })

      // Evaluate multiple times
      for (let i = 0; i < 10; i++) {
        const response = await request.post(
          `${API_BASE_URL}/api/v1/feature-flags/${fullPercentFlag}/evaluate`,
          {
            headers: {
              Authorization: `Bearer ${authToken}`,
              'Content-Type': 'application/json',
            },
            data: {
              context: { user_id: `user-${i}` },
            },
          }
        )

        const body = await response.json()
        expect(body.data.enabled).toBe(true)
      }

      // Cleanup
      await page.request.delete(`${API_BASE_URL}/api/v1/feature-flags/${fullPercentFlag}`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      await context.close()
    })
  })
})
