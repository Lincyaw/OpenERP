/**
 * Tests for Feature Store
 *
 * @module store/featureStore.test
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import {
  useFeatureStore,
  useTenantPlan,
  useFeaturesReady,
  useHasFeature,
  useFeatureLimit,
  useRequiredPlan,
  isPlanHigherOrEqual,
  getNextPlan,
  getPlanDisplayName,
  getAllFeatureKeys,
} from './featureStore'

describe('featureStore', () => {
  beforeEach(() => {
    // Reset store before each test
    act(() => {
      useFeatureStore.getState().reset()
    })
  })

  describe('initialization', () => {
    it('should start with free plan and not ready', () => {
      const state = useFeatureStore.getState()
      expect(state.plan).toBe('free')
      expect(state.isReady).toBe(false)
    })

    it('should initialize with given plan', () => {
      act(() => {
        useFeatureStore.getState().initialize('pro', 'tenant-123')
      })

      const state = useFeatureStore.getState()
      expect(state.plan).toBe('pro')
      expect(state.isReady).toBe(true)
      expect(state.tenantId).toBe('tenant-123')
    })

    it('should update features when plan changes', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      // Free plan should not have advanced_reporting
      expect(useFeatureStore.getState().hasFeature('advanced_reporting')).toBe(false)

      act(() => {
        useFeatureStore.getState().setPlan('pro')
      })

      // Pro plan should have advanced_reporting
      expect(useFeatureStore.getState().hasFeature('advanced_reporting')).toBe(true)
    })
  })

  describe('hasFeature', () => {
    it('should return correct feature availability for free plan', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { hasFeature } = useFeatureStore.getState()

      // Free plan features
      expect(hasFeature('sales_orders')).toBe(true)
      expect(hasFeature('purchase_orders')).toBe(true)
      expect(hasFeature('data_export')).toBe(true)

      // Not available on free
      expect(hasFeature('multi_warehouse')).toBe(false)
      expect(hasFeature('api_access')).toBe(false)
      expect(hasFeature('advanced_reporting')).toBe(false)
    })

    it('should return correct feature availability for basic plan', () => {
      act(() => {
        useFeatureStore.getState().initialize('basic', 'tenant-123')
      })

      const { hasFeature } = useFeatureStore.getState()

      // Basic plan features
      expect(hasFeature('multi_warehouse')).toBe(true)
      expect(hasFeature('batch_management')).toBe(true)
      expect(hasFeature('audit_log')).toBe(true)

      // Not available on basic
      expect(hasFeature('api_access')).toBe(false)
      expect(hasFeature('advanced_reporting')).toBe(false)
    })

    it('should return correct feature availability for pro plan', () => {
      act(() => {
        useFeatureStore.getState().initialize('pro', 'tenant-123')
      })

      const { hasFeature } = useFeatureStore.getState()

      // Pro plan features
      expect(hasFeature('api_access')).toBe(true)
      expect(hasFeature('advanced_reporting')).toBe(true)
      expect(hasFeature('integrations')).toBe(true)

      // Not available on pro
      expect(hasFeature('white_labeling')).toBe(false)
      expect(hasFeature('dedicated_support')).toBe(false)
    })

    it('should return correct feature availability for enterprise plan', () => {
      act(() => {
        useFeatureStore.getState().initialize('enterprise', 'tenant-123')
      })

      const { hasFeature } = useFeatureStore.getState()

      // Enterprise has all features
      expect(hasFeature('white_labeling')).toBe(true)
      expect(hasFeature('dedicated_support')).toBe(true)
      expect(hasFeature('sla')).toBe(true)
    })
  })

  describe('getFeatureLimit', () => {
    it('should return correct limits for data_import', () => {
      const { initialize, setPlan } = useFeatureStore.getState()

      act(() => {
        initialize('free', 'tenant-123')
      })
      expect(useFeatureStore.getState().getFeatureLimit('data_import')).toBe(100)

      act(() => {
        setPlan('basic')
      })
      expect(useFeatureStore.getState().getFeatureLimit('data_import')).toBe(1000)

      act(() => {
        setPlan('pro')
      })
      expect(useFeatureStore.getState().getFeatureLimit('data_import')).toBe(10000)

      act(() => {
        setPlan('enterprise')
      })
      expect(useFeatureStore.getState().getFeatureLimit('data_import')).toBe(null) // unlimited
    })
  })

  describe('getEnabledFeatures and getDisabledFeatures', () => {
    it('should return correct enabled/disabled features', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const enabled = useFeatureStore.getState().getEnabledFeatures()
      const disabled = useFeatureStore.getState().getDisabledFeatures()

      // Free plan has some enabled features
      expect(enabled.length).toBeGreaterThan(0)
      expect(enabled.some((f) => f.key === 'sales_orders')).toBe(true)

      // Free plan has many disabled features
      expect(disabled.length).toBeGreaterThan(0)
      expect(disabled.some((f) => f.key === 'api_access')).toBe(true)
    })
  })

  describe('getRequiredPlan', () => {
    it('should return minimum plan required for features', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { getRequiredPlan } = useFeatureStore.getState()

      // Features available on free
      expect(getRequiredPlan('sales_orders')).toBe('free')
      expect(getRequiredPlan('data_export')).toBe('free')

      // Features requiring basic
      expect(getRequiredPlan('multi_warehouse')).toBe('basic')
      expect(getRequiredPlan('batch_management')).toBe('basic')

      // Features requiring pro
      expect(getRequiredPlan('api_access')).toBe('pro')
      expect(getRequiredPlan('advanced_reporting')).toBe('pro')

      // Features requiring enterprise
      expect(getRequiredPlan('white_labeling')).toBe('enterprise')
      expect(getRequiredPlan('dedicated_support')).toBe('enterprise')
    })
  })

  describe('selector hooks', () => {
    it('useTenantPlan should return current plan', () => {
      act(() => {
        useFeatureStore.getState().initialize('pro', 'tenant-123')
      })

      const { result } = renderHook(() => useTenantPlan())
      expect(result.current).toBe('pro')
    })

    it('useFeaturesReady should return ready state', () => {
      const { result, rerender } = renderHook(() => useFeaturesReady())
      expect(result.current).toBe(false)

      act(() => {
        useFeatureStore.getState().initialize('pro', 'tenant-123')
      })

      rerender()
      expect(result.current).toBe(true)
    })

    it('useHasFeature should return feature availability', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { result: hasApiAccess } = renderHook(() => useHasFeature('api_access'))
      const { result: hasSalesOrders } = renderHook(() => useHasFeature('sales_orders'))

      expect(hasApiAccess.current).toBe(false)
      expect(hasSalesOrders.current).toBe(true)
    })

    it('useFeatureLimit should return feature limit', () => {
      act(() => {
        useFeatureStore.getState().initialize('basic', 'tenant-123')
      })

      const { result } = renderHook(() => useFeatureLimit('data_import'))
      expect(result.current).toBe(1000)
    })

    it('useRequiredPlan should return required plan', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { result } = renderHook(() => useRequiredPlan('api_access'))
      expect(result.current).toBe('pro')
    })
  })
})

describe('utility functions', () => {
  describe('isPlanHigherOrEqual', () => {
    it('should correctly compare plans', () => {
      expect(isPlanHigherOrEqual('free', 'free')).toBe(true)
      expect(isPlanHigherOrEqual('basic', 'free')).toBe(true)
      expect(isPlanHigherOrEqual('pro', 'basic')).toBe(true)
      expect(isPlanHigherOrEqual('enterprise', 'pro')).toBe(true)

      expect(isPlanHigherOrEqual('free', 'basic')).toBe(false)
      expect(isPlanHigherOrEqual('basic', 'pro')).toBe(false)
      expect(isPlanHigherOrEqual('pro', 'enterprise')).toBe(false)
    })
  })

  describe('getNextPlan', () => {
    it('should return next plan in hierarchy', () => {
      expect(getNextPlan('free')).toBe('basic')
      expect(getNextPlan('basic')).toBe('pro')
      expect(getNextPlan('pro')).toBe('enterprise')
      expect(getNextPlan('enterprise')).toBe(null)
    })
  })

  describe('getPlanDisplayName', () => {
    it('should return human-readable plan names', () => {
      expect(getPlanDisplayName('free')).toBe('Free')
      expect(getPlanDisplayName('basic')).toBe('Basic')
      expect(getPlanDisplayName('pro')).toBe('Professional')
      expect(getPlanDisplayName('enterprise')).toBe('Enterprise')
    })
  })

  describe('getAllFeatureKeys', () => {
    it('should return all feature keys', () => {
      const keys = getAllFeatureKeys()
      expect(keys.length).toBeGreaterThan(20)
      expect(keys).toContain('multi_warehouse')
      expect(keys).toContain('api_access')
      expect(keys).toContain('sales_orders')
    })
  })
})
