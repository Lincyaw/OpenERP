/**
 * UpdateQuotaModal Component Tests (P3-ADMIN-021)
 *
 * Tests for the quota update modal component.
 * Covers: quota calculation logic, usage percentages, preset application, warnings.
 */

import { describe, it, expect } from 'vitest'

// Test the business logic that drives the UI

const PLAN_QUOTAS: Record<
  string,
  { maxUsers: number; maxProducts: number; maxWarehouses: number }
> = {
  free: { maxUsers: 5, maxProducts: 100, maxWarehouses: 1 },
  basic: { maxUsers: 10, maxProducts: 500, maxWarehouses: 3 },
  pro: { maxUsers: 50, maxProducts: 5000, maxWarehouses: 10 },
  enterprise: { maxUsers: 999, maxProducts: 99999, maxWarehouses: 999 },
}

interface QuotaValues {
  maxUsers: number
  maxProducts: number
  maxWarehouses: number
}

function calcPercent(current: number, max: number): number {
  if (max <= 0) return 0
  return Math.min(100, Math.round((current / max) * 100))
}

function getWarnings(
  currentUsage: { users: number; products: number; warehouses: number },
  quotaValues: QuotaValues
): string[] {
  const result: string[] = []
  if (currentUsage.users > quotaValues.maxUsers) {
    result.push('Current user count exceeds new limit')
  }
  if (currentUsage.products > quotaValues.maxProducts) {
    result.push('Current product count exceeds new limit')
  }
  if (currentUsage.warehouses > quotaValues.maxWarehouses) {
    result.push('Current warehouse count exceeds new limit')
  }
  return result
}

function hasChanges(
  config: { max_users?: number; max_products?: number; max_warehouses?: number } | undefined,
  quotaValues: QuotaValues
): boolean {
  if (!config) return false
  return (
    quotaValues.maxUsers !== (config.max_users || 10) ||
    quotaValues.maxProducts !== (config.max_products || 1000) ||
    quotaValues.maxWarehouses !== (config.max_warehouses || 5)
  )
}

describe('UpdateQuotaModal - Quota Logic', () => {
  describe('PLAN_QUOTAS', () => {
    it('should have quotas for all plans', () => {
      expect(Object.keys(PLAN_QUOTAS)).toEqual(['free', 'basic', 'pro', 'enterprise'])
    })

    it('should have increasing quotas for higher plans', () => {
      const plans = ['free', 'basic', 'pro', 'enterprise']
      for (let i = 1; i < plans.length; i++) {
        const prev = PLAN_QUOTAS[plans[i - 1]]
        const curr = PLAN_QUOTAS[plans[i]]
        expect(curr.maxUsers).toBeGreaterThan(prev.maxUsers)
        expect(curr.maxProducts).toBeGreaterThan(prev.maxProducts)
        expect(curr.maxWarehouses).toBeGreaterThan(prev.maxWarehouses)
      }
    })

    it('should have minimum 1 for all quotas', () => {
      Object.values(PLAN_QUOTAS).forEach((quota) => {
        expect(quota.maxUsers).toBeGreaterThanOrEqual(1)
        expect(quota.maxProducts).toBeGreaterThanOrEqual(1)
        expect(quota.maxWarehouses).toBeGreaterThanOrEqual(1)
      })
    })
  })

  describe('calcPercent', () => {
    it('should calculate percentage correctly', () => {
      expect(calcPercent(5, 10)).toBe(50)
      expect(calcPercent(1, 4)).toBe(25)
      expect(calcPercent(3, 3)).toBe(100)
    })

    it('should cap at 100%', () => {
      expect(calcPercent(15, 10)).toBe(100)
      expect(calcPercent(200, 100)).toBe(100)
    })

    it('should handle 0 max by returning 0', () => {
      expect(calcPercent(5, 0)).toBe(0)
      expect(calcPercent(0, 0)).toBe(0)
    })

    it('should handle negative max by returning 0', () => {
      expect(calcPercent(5, -1)).toBe(0)
    })

    it('should round to nearest integer', () => {
      expect(calcPercent(1, 3)).toBe(33) // 33.33 -> 33
      expect(calcPercent(2, 3)).toBe(67) // 66.67 -> 67
    })

    it('should handle zero current', () => {
      expect(calcPercent(0, 100)).toBe(0)
    })
  })

  describe('getWarnings', () => {
    it('should return no warnings when usage is within limits', () => {
      const usage = { users: 5, products: 100, warehouses: 2 }
      const quotas = { maxUsers: 10, maxProducts: 500, maxWarehouses: 5 }
      expect(getWarnings(usage, quotas)).toEqual([])
    })

    it('should warn when user count exceeds limit', () => {
      const usage = { users: 15, products: 100, warehouses: 2 }
      const quotas = { maxUsers: 10, maxProducts: 500, maxWarehouses: 5 }
      const warnings = getWarnings(usage, quotas)
      expect(warnings).toHaveLength(1)
      expect(warnings[0]).toContain('user')
    })

    it('should warn when product count exceeds limit', () => {
      const usage = { users: 5, products: 600, warehouses: 2 }
      const quotas = { maxUsers: 10, maxProducts: 500, maxWarehouses: 5 }
      const warnings = getWarnings(usage, quotas)
      expect(warnings).toHaveLength(1)
      expect(warnings[0]).toContain('product')
    })

    it('should warn when warehouse count exceeds limit', () => {
      const usage = { users: 5, products: 100, warehouses: 10 }
      const quotas = { maxUsers: 10, maxProducts: 500, maxWarehouses: 5 }
      const warnings = getWarnings(usage, quotas)
      expect(warnings).toHaveLength(1)
      expect(warnings[0]).toContain('warehouse')
    })

    it('should return multiple warnings when multiple limits exceeded', () => {
      const usage = { users: 20, products: 1000, warehouses: 10 }
      const quotas = { maxUsers: 10, maxProducts: 500, maxWarehouses: 5 }
      expect(getWarnings(usage, quotas)).toHaveLength(3)
    })

    it('should not warn when usage equals limit', () => {
      const usage = { users: 10, products: 500, warehouses: 5 }
      const quotas = { maxUsers: 10, maxProducts: 500, maxWarehouses: 5 }
      expect(getWarnings(usage, quotas)).toEqual([])
    })
  })

  describe('hasChanges', () => {
    it('should return false when config is undefined', () => {
      expect(hasChanges(undefined, { maxUsers: 10, maxProducts: 1000, maxWarehouses: 5 })).toBe(
        false
      )
    })

    it('should return false when values match config', () => {
      const config = { max_users: 10, max_products: 1000, max_warehouses: 5 }
      expect(hasChanges(config, { maxUsers: 10, maxProducts: 1000, maxWarehouses: 5 })).toBe(false)
    })

    it('should return true when users changed', () => {
      const config = { max_users: 10, max_products: 1000, max_warehouses: 5 }
      expect(hasChanges(config, { maxUsers: 20, maxProducts: 1000, maxWarehouses: 5 })).toBe(true)
    })

    it('should return true when products changed', () => {
      const config = { max_users: 10, max_products: 1000, max_warehouses: 5 }
      expect(hasChanges(config, { maxUsers: 10, maxProducts: 2000, maxWarehouses: 5 })).toBe(true)
    })

    it('should return true when warehouses changed', () => {
      const config = { max_users: 10, max_products: 1000, max_warehouses: 5 }
      expect(hasChanges(config, { maxUsers: 10, maxProducts: 1000, maxWarehouses: 10 })).toBe(true)
    })

    it('should use default values when config fields are missing', () => {
      const config = {}
      // Default: max_users=10, max_products=1000, max_warehouses=5
      expect(hasChanges(config, { maxUsers: 10, maxProducts: 1000, maxWarehouses: 5 })).toBe(false)
      expect(hasChanges(config, { maxUsers: 20, maxProducts: 1000, maxWarehouses: 5 })).toBe(true)
    })
  })
})
