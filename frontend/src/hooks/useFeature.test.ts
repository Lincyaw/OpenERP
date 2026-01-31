/**
 * Tests for useFeature hooks
 *
 * @module hooks/useFeature.test
 */

import { describe, it, expect, beforeEach } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useFeatureStore } from '@/store'
import {
  useFeature,
  useFeatures,
  useDisabledFeatures,
  useEnabledFeatures,
  usePlan,
  useFeatureReady,
} from './useFeature'

describe('useFeature hooks', () => {
  beforeEach(() => {
    // Reset store before each test
    act(() => {
      useFeatureStore.getState().reset()
    })
  })

  describe('useFeature', () => {
    it('should return feature info when not ready', () => {
      const { result } = renderHook(() => useFeature('api_access'))

      expect(result.current.isReady).toBe(false)
      expect(result.current.enabled).toBe(false)
    })

    it('should return correct feature info for free plan', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { result } = renderHook(() => useFeature('api_access'))

      expect(result.current.enabled).toBe(false)
      expect(result.current.needsUpgrade).toBe(true)
      expect(result.current.requiredPlan).toBe('pro')
      expect(result.current.requiredPlanName).toBe('Professional')
      expect(result.current.currentPlan).toBe('free')
      expect(result.current.currentPlanName).toBe('Free')
      expect(result.current.upgradePlan).toBe('basic')
      expect(result.current.upgradePlanName).toBe('Basic')
      expect(result.current.isReady).toBe(true)
    })

    it('should return correct feature info for pro plan', () => {
      act(() => {
        useFeatureStore.getState().initialize('pro', 'tenant-123')
      })

      const { result } = renderHook(() => useFeature('api_access'))

      expect(result.current.enabled).toBe(true)
      expect(result.current.needsUpgrade).toBe(false)
      expect(result.current.currentPlan).toBe('pro')
      expect(result.current.upgradePlan).toBe('enterprise')
    })

    it('should return feature limit', () => {
      act(() => {
        useFeatureStore.getState().initialize('basic', 'tenant-123')
      })

      const { result } = renderHook(() => useFeature('data_import'))

      expect(result.current.enabled).toBe(true)
      expect(result.current.limit).toBe(1000)
    })

    it('should return null limit for unlimited features', () => {
      act(() => {
        useFeatureStore.getState().initialize('enterprise', 'tenant-123')
      })

      const { result } = renderHook(() => useFeature('data_import'))

      expect(result.current.enabled).toBe(true)
      expect(result.current.limit).toBe(null)
    })

    it('should update when plan changes', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { result, rerender } = renderHook(() => useFeature('multi_warehouse'))

      expect(result.current.enabled).toBe(false)

      act(() => {
        useFeatureStore.getState().setPlan('basic')
      })

      rerender()
      expect(result.current.enabled).toBe(true)
    })
  })

  describe('useFeatures', () => {
    it('should return multiple feature states', () => {
      act(() => {
        useFeatureStore.getState().initialize('basic', 'tenant-123')
      })

      const { result } = renderHook(() =>
        useFeatures(['multi_warehouse', 'api_access', 'sales_orders'])
      )

      expect(result.current.multi_warehouse).toBe(true)
      expect(result.current.api_access).toBe(false)
      expect(result.current.sales_orders).toBe(true)
    })

    it('should update when plan changes', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { result, rerender } = renderHook(() => useFeatures(['multi_warehouse', 'api_access']))

      expect(result.current.multi_warehouse).toBe(false)
      expect(result.current.api_access).toBe(false)

      act(() => {
        useFeatureStore.getState().setPlan('pro')
      })

      rerender()
      expect(result.current.multi_warehouse).toBe(true)
      expect(result.current.api_access).toBe(true)
    })
  })

  describe('useDisabledFeatures', () => {
    it('should return disabled features for free plan', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { result } = renderHook(() => useDisabledFeatures())

      expect(result.current.length).toBeGreaterThan(0)
      expect(result.current.some((f) => f.key === 'api_access')).toBe(true)
      expect(result.current.some((f) => f.key === 'multi_warehouse')).toBe(true)
    })

    it('should return fewer disabled features for higher plans', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { result: freeResult } = renderHook(() => useDisabledFeatures())
      const freeDisabledCount = freeResult.current.length

      act(() => {
        useFeatureStore.getState().setPlan('pro')
      })

      const { result: proResult } = renderHook(() => useDisabledFeatures())
      expect(proResult.current.length).toBeLessThan(freeDisabledCount)
    })
  })

  describe('useEnabledFeatures', () => {
    it('should return enabled features', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { result } = renderHook(() => useEnabledFeatures())

      expect(result.current.length).toBeGreaterThan(0)
      expect(result.current.some((f) => f.key === 'sales_orders')).toBe(true)
    })

    it('should return more enabled features for higher plans', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { result: freeResult } = renderHook(() => useEnabledFeatures())
      const freeEnabledCount = freeResult.current.length

      act(() => {
        useFeatureStore.getState().setPlan('enterprise')
      })

      const { result: enterpriseResult } = renderHook(() => useEnabledFeatures())
      expect(enterpriseResult.current.length).toBeGreaterThan(freeEnabledCount)
    })
  })

  describe('usePlan', () => {
    it('should return current plan info', () => {
      act(() => {
        useFeatureStore.getState().initialize('pro', 'tenant-123')
      })

      const { result } = renderHook(() => usePlan())

      expect(result.current.plan).toBe('pro')
      expect(result.current.planName).toBe('Professional')
    })

    it('should update when plan changes', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { result, rerender } = renderHook(() => usePlan())

      expect(result.current.plan).toBe('free')

      act(() => {
        useFeatureStore.getState().setPlan('basic')
      })

      rerender()
      expect(result.current.plan).toBe('basic')
      expect(result.current.planName).toBe('Basic')
    })
  })

  describe('useFeatureReady', () => {
    it('should return false when not initialized', () => {
      const { result } = renderHook(() => useFeatureReady())
      expect(result.current).toBe(false)
    })

    it('should return true after initialization', () => {
      act(() => {
        useFeatureStore.getState().initialize('free', 'tenant-123')
      })

      const { result } = renderHook(() => useFeatureReady())
      expect(result.current).toBe(true)
    })
  })
})
