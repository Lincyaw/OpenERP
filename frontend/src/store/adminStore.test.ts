/**
 * Admin Store Tests (P3-ADMIN-021)
 *
 * Tests for the Zustand admin store managing admin UI state.
 * Covers: selection, loading states, error management, reset, exported hooks.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import {
  useAdminStore,
  useSelectedTenantId,
  useSelectedTenantName,
  useOperationLoading,
  useAdminError,
} from './adminStore'

describe('useAdminStore', () => {
  beforeEach(() => {
    act(() => {
      useAdminStore.getState().reset()
    })
    vi.clearAllMocks()
  })

  describe('initial state', () => {
    it('should have null selected tenant', () => {
      const state = useAdminStore.getState()
      expect(state.selectedTenantId).toBeNull()
      expect(state.selectedTenantName).toBeNull()
    })

    it('should have all operation loading states as false', () => {
      const state = useAdminStore.getState()
      expect(state.operationLoading).toEqual({
        create: false,
        update: false,
        delete: false,
        suspend: false,
        activate: false,
        changePlan: false,
        updateQuota: false,
      })
    })

    it('should have null error', () => {
      expect(useAdminStore.getState().lastError).toBeNull()
    })
  })

  describe('selectTenant', () => {
    it('should set selected tenant id and name', () => {
      act(() => {
        useAdminStore.getState().selectTenant('tenant-123', 'Acme Corp')
      })

      const state = useAdminStore.getState()
      expect(state.selectedTenantId).toBe('tenant-123')
      expect(state.selectedTenantName).toBe('Acme Corp')
    })

    it('should set name to null when not provided', () => {
      act(() => {
        useAdminStore.getState().selectTenant('tenant-123')
      })

      expect(useAdminStore.getState().selectedTenantName).toBeNull()
    })

    it('should allow selecting null to deselect', () => {
      act(() => {
        useAdminStore.getState().selectTenant('tenant-123', 'Test')
      })
      act(() => {
        useAdminStore.getState().selectTenant(null)
      })

      expect(useAdminStore.getState().selectedTenantId).toBeNull()
    })
  })

  describe('clearSelection', () => {
    it('should clear selected tenant', () => {
      act(() => {
        useAdminStore.getState().selectTenant('tenant-123', 'Test')
      })
      act(() => {
        useAdminStore.getState().clearSelection()
      })

      const state = useAdminStore.getState()
      expect(state.selectedTenantId).toBeNull()
      expect(state.selectedTenantName).toBeNull()
    })
  })

  describe('setOperationLoading', () => {
    it('should set specific operation loading state', () => {
      act(() => {
        useAdminStore.getState().setOperationLoading('create', true)
      })

      const state = useAdminStore.getState()
      expect(state.operationLoading.create).toBe(true)
      expect(state.operationLoading.update).toBe(false)
    })

    it('should toggle loading state off', () => {
      act(() => {
        useAdminStore.getState().setOperationLoading('delete', true)
      })
      act(() => {
        useAdminStore.getState().setOperationLoading('delete', false)
      })

      expect(useAdminStore.getState().operationLoading.delete).toBe(false)
    })

    it('should handle all operation types', () => {
      const operations = [
        'create',
        'update',
        'delete',
        'suspend',
        'activate',
        'changePlan',
        'updateQuota',
      ] as const

      for (const op of operations) {
        act(() => {
          useAdminStore.getState().setOperationLoading(op, true)
        })
        expect(useAdminStore.getState().operationLoading[op]).toBe(true)
      }
    })
  })

  describe('error management', () => {
    it('should set error message', () => {
      act(() => {
        useAdminStore.getState().setError('Something went wrong')
      })

      expect(useAdminStore.getState().lastError).toBe('Something went wrong')
    })

    it('should clear error', () => {
      act(() => {
        useAdminStore.getState().setError('Error')
      })
      act(() => {
        useAdminStore.getState().clearError()
      })

      expect(useAdminStore.getState().lastError).toBeNull()
    })

    it('should overwrite previous error', () => {
      act(() => {
        useAdminStore.getState().setError('Error 1')
      })
      act(() => {
        useAdminStore.getState().setError('Error 2')
      })

      expect(useAdminStore.getState().lastError).toBe('Error 2')
    })
  })

  describe('reset', () => {
    it('should reset all state to initial values', () => {
      // Modify all state
      act(() => {
        const store = useAdminStore.getState()
        store.selectTenant('tenant-1', 'Test')
        store.setOperationLoading('create', true)
        store.setError('Some error')
      })

      // Reset
      act(() => {
        useAdminStore.getState().reset()
      })

      const state = useAdminStore.getState()
      expect(state.selectedTenantId).toBeNull()
      expect(state.selectedTenantName).toBeNull()
      expect(state.operationLoading.create).toBe(false)
      expect(state.lastError).toBeNull()
    })
  })

  describe('selector hooks', () => {
    it('should provide selectedTenantId selector', () => {
      act(() => {
        useAdminStore.getState().selectTenant('test-id')
      })

      const selector = (state: ReturnType<typeof useAdminStore.getState>) => state.selectedTenantId
      expect(selector(useAdminStore.getState())).toBe('test-id')
    })

    it('should provide selectedTenantName selector', () => {
      act(() => {
        useAdminStore.getState().selectTenant('id', 'Name')
      })

      const selector = (state: ReturnType<typeof useAdminStore.getState>) =>
        state.selectedTenantName
      expect(selector(useAdminStore.getState())).toBe('Name')
    })

    it('should provide operationLoading selector', () => {
      const selector = (state: ReturnType<typeof useAdminStore.getState>) => state.operationLoading
      expect(selector(useAdminStore.getState())).toBeDefined()
      expect(selector(useAdminStore.getState()).create).toBe(false)
    })

    it('should provide lastError selector', () => {
      act(() => {
        useAdminStore.getState().setError('test error')
      })

      const selector = (state: ReturnType<typeof useAdminStore.getState>) => state.lastError
      expect(selector(useAdminStore.getState())).toBe('test error')
    })
  })

  describe('exported selector hooks', () => {
    it('useSelectedTenantId should return selected tenant id', () => {
      act(() => {
        useAdminStore.getState().selectTenant('tenant-abc', 'ABC Corp')
      })

      const { result } = renderHook(() => useSelectedTenantId())
      expect(result.current).toBe('tenant-abc')
    })

    it('useSelectedTenantName should return selected tenant name', () => {
      act(() => {
        useAdminStore.getState().selectTenant('tenant-abc', 'ABC Corp')
      })

      const { result } = renderHook(() => useSelectedTenantName())
      expect(result.current).toBe('ABC Corp')
    })

    it('useOperationLoading should return operation loading states', () => {
      act(() => {
        useAdminStore.getState().setOperationLoading('suspend', true)
      })

      const { result } = renderHook(() => useOperationLoading())
      expect(result.current.suspend).toBe(true)
      expect(result.current.create).toBe(false)
    })

    it('useAdminError should return last error', () => {
      act(() => {
        useAdminStore.getState().setError('Network error')
      })

      const { result } = renderHook(() => useAdminError())
      expect(result.current).toBe('Network error')
    })

    it('selector hooks should update when state changes', () => {
      const { result: idResult, rerender: rerenderIdHook } = renderHook(() => useSelectedTenantId())

      expect(idResult.current).toBeNull()

      act(() => {
        useAdminStore.getState().selectTenant('new-id', 'New Name')
      })

      rerenderIdHook()
      expect(idResult.current).toBe('new-id')
    })
  })
})
