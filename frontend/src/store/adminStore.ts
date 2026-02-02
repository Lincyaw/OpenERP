/**
 * Admin Store
 *
 * Zustand store for managing admin UI state and providing
 * enhanced hooks with optimistic updates for tenant operations.
 *
 * This store complements React Query by managing:
 * - Selected tenant state
 * - Loading states for operations
 * - Error states
 * - Optimistic update rollback context
 *
 * Server state (tenants, stats, audit logs) is managed by React Query.
 */
import { create } from 'zustand'
import { devtools } from 'zustand/middleware'
import { useCallback } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import {
  useListTenants,
  useGetTenantById,
  useCreateTenant,
  useUpdateTenant,
  useDeleteTenant,
  useSuspendTenant,
  useActivateTenant,
  useSetPlanTenant,
  useUpdateTenantConfig,
  useGetTenantStats,
  getListTenantsQueryKey,
  getGetTenantByIdQueryKey,
  getGetTenantStatsQueryKey,
} from '@/api/tenants/tenants'
import type { HandlerTenantResponse, ListTenantsParams } from '@/api/models'
import {
  usePlatformStats,
  useTenantGrowthTrend,
  useTenantUsageStats,
  useAdminAuditLogs,
  adminQueryKeys,
} from '@/api/admin'

// ============================================================================
// Store State Types
// ============================================================================

export interface AdminState {
  // Selected tenant
  selectedTenantId: string | null
  selectedTenantName: string | null

  // Loading states for specific operations
  operationLoading: {
    create: boolean
    update: boolean
    delete: boolean
    suspend: boolean
    activate: boolean
    changePlan: boolean
    updateQuota: boolean
  }

  // Error state
  lastError: string | null
}

export interface AdminActions {
  // Selection
  selectTenant: (id: string | null, name?: string) => void
  clearSelection: () => void

  // Loading state management
  setOperationLoading: (operation: keyof AdminState['operationLoading'], loading: boolean) => void

  // Error management
  setError: (error: string | null) => void
  clearError: () => void

  // Reset store
  reset: () => void
}

// ============================================================================
// Initial State
// ============================================================================

const initialState: AdminState = {
  selectedTenantId: null,
  selectedTenantName: null,
  operationLoading: {
    create: false,
    update: false,
    delete: false,
    suspend: false,
    activate: false,
    changePlan: false,
    updateQuota: false,
  },
  lastError: null,
}

// ============================================================================
// Store
// ============================================================================

export const useAdminStore = create<AdminState & AdminActions>()(
  devtools(
    (set) => ({
      ...initialState,

      selectTenant: (id, name) =>
        set(
          { selectedTenantId: id, selectedTenantName: name || null },
          false,
          'admin/selectTenant'
        ),

      clearSelection: () =>
        set({ selectedTenantId: null, selectedTenantName: null }, false, 'admin/clearSelection'),

      setOperationLoading: (operation, loading) =>
        set(
          (state) => ({
            operationLoading: { ...state.operationLoading, [operation]: loading },
          }),
          false,
          `admin/setOperationLoading/${operation}`
        ),

      setError: (error) => set({ lastError: error }, false, 'admin/setError'),

      clearError: () => set({ lastError: null }, false, 'admin/clearError'),

      reset: () => set(initialState, false, 'admin/reset'),
    }),
    { name: 'AdminStore' }
  )
)

// ============================================================================
// Selector Hooks
// ============================================================================

export const useSelectedTenantId = () => useAdminStore((state) => state.selectedTenantId)
export const useSelectedTenantName = () => useAdminStore((state) => state.selectedTenantName)
export const useOperationLoading = () => useAdminStore((state) => state.operationLoading)
export const useAdminError = () => useAdminStore((state) => state.lastError)

// ============================================================================
// Enhanced Hooks with Optimistic Updates
// ============================================================================

/**
 * Hook for listing tenants with pagination and filtering
 */
export function useAdminTenants(params?: ListTenantsParams) {
  return useListTenants(params)
}

/**
 * Hook for fetching a single tenant by ID
 */
export function useAdminTenant(tenantId: string) {
  return useGetTenantById(tenantId, {
    query: {
      enabled: !!tenantId,
    },
  })
}

/**
 * Hook for tenant statistics
 */
export function useAdminTenantStats() {
  return useGetTenantStats()
}

/**
 * Hook for platform-wide statistics
 */
export { usePlatformStats as useAdminPlatformStats }

/**
 * Hook for tenant growth trend
 */
export { useTenantGrowthTrend as useAdminGrowthTrend }

/**
 * Hook for tenant usage statistics
 */
export { useTenantUsageStats as useAdminTenantUsage }

/**
 * Hook for admin audit logs
 */
export { useAdminAuditLogs }

/**
 * Hook for creating a tenant with optimistic update
 */
export function useCreateAdminTenant() {
  const queryClient = useQueryClient()
  const { setOperationLoading, setError, clearError } = useAdminStore.getState()

  return useCreateTenant({
    mutation: {
      onMutate: () => {
        clearError()
        setOperationLoading('create', true)
      },
      onSuccess: () => {
        // Invalidate tenant list to refetch with new tenant
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        queryClient.invalidateQueries({ queryKey: getGetTenantStatsQueryKey() })
        queryClient.invalidateQueries({ queryKey: adminQueryKeys.platformStats() })
      },
      onError: () => {
        setError('Failed to create tenant')
      },
      onSettled: () => {
        setOperationLoading('create', false)
      },
    },
  })
}

/**
 * Hook for updating a tenant with optimistic update
 */
export function useUpdateAdminTenant() {
  const queryClient = useQueryClient()
  const { setOperationLoading, setError, clearError } = useAdminStore.getState()

  return useUpdateTenant({
    mutation: {
      onMutate: async ({ id, data }) => {
        clearError()
        setOperationLoading('update', true)

        // Cancel outgoing refetches
        await queryClient.cancelQueries({ queryKey: getGetTenantByIdQueryKey(id) })

        // Snapshot previous value
        const previousTenant = queryClient.getQueryData(getGetTenantByIdQueryKey(id))

        // Optimistically update
        queryClient.setQueryData(getGetTenantByIdQueryKey(id), (old: unknown) => {
          if (!old) return old
          const response = old as {
            data: { data: HandlerTenantResponse }
            status: number
          }
          return {
            ...response,
            data: {
              ...response.data,
              data: { ...response.data.data, ...data },
            },
          }
        })

        return { previousTenant }
      },
      onError: (_, { id }, context) => {
        // Rollback on error
        if (context?.previousTenant) {
          queryClient.setQueryData(getGetTenantByIdQueryKey(id), context.previousTenant)
        }
        setError('Failed to update tenant')
      },
      onSuccess: (_, { id }) => {
        // Invalidate related queries
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        queryClient.invalidateQueries({ queryKey: getGetTenantByIdQueryKey(id) })
      },
      onSettled: () => {
        setOperationLoading('update', false)
      },
    },
  })
}

/**
 * Hook for deleting a tenant with optimistic update
 */
export function useDeleteAdminTenant() {
  const queryClient = useQueryClient()
  const { setOperationLoading, setError, clearError, clearSelection } = useAdminStore.getState()

  return useDeleteTenant({
    mutation: {
      onMutate: async ({ id }) => {
        clearError()
        setOperationLoading('delete', true)

        // Cancel outgoing refetches
        await queryClient.cancelQueries({ queryKey: getListTenantsQueryKey() })

        // Snapshot previous value
        const previousTenants = queryClient.getQueryData(getListTenantsQueryKey())

        // Optimistically remove from list
        queryClient.setQueryData(getListTenantsQueryKey(), (old: unknown) => {
          if (!old) return old
          const response = old as {
            data: { data: { tenants: HandlerTenantResponse[]; total: number } }
            status: number
          }
          return {
            ...response,
            data: {
              ...response.data,
              data: {
                ...response.data.data,
                tenants: response.data.data.tenants.filter((t) => t.id !== id),
                total: response.data.data.total - 1,
              },
            },
          }
        })

        return { previousTenants }
      },
      onError: (_, __, context) => {
        // Rollback on error
        if (context?.previousTenants) {
          queryClient.setQueryData(getListTenantsQueryKey(), context.previousTenants)
        }
        setError('Failed to delete tenant')
      },
      onSuccess: () => {
        // Clear selection if the deleted tenant was selected
        clearSelection()
        // Invalidate related queries
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        queryClient.invalidateQueries({ queryKey: getGetTenantStatsQueryKey() })
        queryClient.invalidateQueries({ queryKey: adminQueryKeys.platformStats() })
      },
      onSettled: () => {
        setOperationLoading('delete', false)
      },
    },
  })
}

/**
 * Hook for suspending a tenant with optimistic update
 */
export function useSuspendAdminTenant() {
  const queryClient = useQueryClient()
  const { setOperationLoading, setError, clearError } = useAdminStore.getState()

  return useSuspendTenant({
    mutation: {
      onMutate: async ({ id }) => {
        clearError()
        setOperationLoading('suspend', true)

        // Cancel outgoing refetches
        await queryClient.cancelQueries({ queryKey: getGetTenantByIdQueryKey(id) })

        // Snapshot previous value
        const previousTenant = queryClient.getQueryData(getGetTenantByIdQueryKey(id))

        // Optimistically update status
        queryClient.setQueryData(getGetTenantByIdQueryKey(id), (old: unknown) => {
          if (!old) return old
          const response = old as {
            data: { data: HandlerTenantResponse }
            status: number
          }
          return {
            ...response,
            data: {
              ...response.data,
              data: { ...response.data.data, status: 'suspended' },
            },
          }
        })

        return { previousTenant }
      },
      onError: (_, { id }, context) => {
        // Rollback on error
        if (context?.previousTenant) {
          queryClient.setQueryData(getGetTenantByIdQueryKey(id), context.previousTenant)
        }
        setError('Failed to suspend tenant')
      },
      onSuccess: (_, { id }) => {
        // Invalidate related queries
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        queryClient.invalidateQueries({ queryKey: getGetTenantByIdQueryKey(id) })
        queryClient.invalidateQueries({ queryKey: getGetTenantStatsQueryKey() })
        queryClient.invalidateQueries({ queryKey: adminQueryKeys.platformStats() })
      },
      onSettled: () => {
        setOperationLoading('suspend', false)
      },
    },
  })
}

/**
 * Hook for activating a tenant with optimistic update
 */
export function useActivateAdminTenant() {
  const queryClient = useQueryClient()
  const { setOperationLoading, setError, clearError } = useAdminStore.getState()

  return useActivateTenant({
    mutation: {
      onMutate: async ({ id }) => {
        clearError()
        setOperationLoading('activate', true)

        // Cancel outgoing refetches
        await queryClient.cancelQueries({ queryKey: getGetTenantByIdQueryKey(id) })

        // Snapshot previous value
        const previousTenant = queryClient.getQueryData(getGetTenantByIdQueryKey(id))

        // Optimistically update status
        queryClient.setQueryData(getGetTenantByIdQueryKey(id), (old: unknown) => {
          if (!old) return old
          const response = old as {
            data: { data: HandlerTenantResponse }
            status: number
          }
          return {
            ...response,
            data: {
              ...response.data,
              data: { ...response.data.data, status: 'active' },
            },
          }
        })

        return { previousTenant }
      },
      onError: (_, { id }, context) => {
        // Rollback on error
        if (context?.previousTenant) {
          queryClient.setQueryData(getGetTenantByIdQueryKey(id), context.previousTenant)
        }
        setError('Failed to activate tenant')
      },
      onSuccess: (_, { id }) => {
        // Invalidate related queries
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        queryClient.invalidateQueries({ queryKey: getGetTenantByIdQueryKey(id) })
        queryClient.invalidateQueries({ queryKey: getGetTenantStatsQueryKey() })
        queryClient.invalidateQueries({ queryKey: adminQueryKeys.platformStats() })
      },
      onSettled: () => {
        setOperationLoading('activate', false)
      },
    },
  })
}

/**
 * Hook for changing tenant plan with optimistic update
 */
export function useChangePlanAdminTenant() {
  const queryClient = useQueryClient()
  const { setOperationLoading, setError, clearError } = useAdminStore.getState()

  return useSetPlanTenant({
    mutation: {
      onMutate: async ({ id, data }) => {
        clearError()
        setOperationLoading('changePlan', true)

        // Cancel outgoing refetches
        await queryClient.cancelQueries({ queryKey: getGetTenantByIdQueryKey(id) })

        // Snapshot previous value
        const previousTenant = queryClient.getQueryData(getGetTenantByIdQueryKey(id))

        // Optimistically update plan
        queryClient.setQueryData(getGetTenantByIdQueryKey(id), (old: unknown) => {
          if (!old) return old
          const response = old as {
            data: { data: HandlerTenantResponse }
            status: number
          }
          return {
            ...response,
            data: {
              ...response.data,
              data: { ...response.data.data, plan: data.plan },
            },
          }
        })

        return { previousTenant }
      },
      onError: (_, { id }, context) => {
        // Rollback on error
        if (context?.previousTenant) {
          queryClient.setQueryData(getGetTenantByIdQueryKey(id), context.previousTenant)
        }
        setError('Failed to change tenant plan')
      },
      onSuccess: (_, { id }) => {
        // Invalidate related queries
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        queryClient.invalidateQueries({ queryKey: getGetTenantByIdQueryKey(id) })
        queryClient.invalidateQueries({ queryKey: getGetTenantStatsQueryKey() })
        queryClient.invalidateQueries({ queryKey: adminQueryKeys.platformStats() })
      },
      onSettled: () => {
        setOperationLoading('changePlan', false)
      },
    },
  })
}

/**
 * Hook for updating tenant quota with optimistic update
 */
export function useUpdateQuotaAdminTenant() {
  const queryClient = useQueryClient()
  const { setOperationLoading, setError, clearError } = useAdminStore.getState()

  return useUpdateTenantConfig({
    mutation: {
      onMutate: async ({ id, data }) => {
        clearError()
        setOperationLoading('updateQuota', true)

        // Cancel outgoing refetches
        await queryClient.cancelQueries({ queryKey: getGetTenantByIdQueryKey(id) })

        // Snapshot previous value
        const previousTenant = queryClient.getQueryData(getGetTenantByIdQueryKey(id))

        // Optimistically update config
        queryClient.setQueryData(getGetTenantByIdQueryKey(id), (old: unknown) => {
          if (!old) return old
          const response = old as {
            data: { data: HandlerTenantResponse }
            status: number
          }
          return {
            ...response,
            data: {
              ...response.data,
              data: {
                ...response.data.data,
                config: { ...response.data.data.config, ...data },
              },
            },
          }
        })

        return { previousTenant }
      },
      onError: (_, { id }, context) => {
        // Rollback on error
        if (context?.previousTenant) {
          queryClient.setQueryData(getGetTenantByIdQueryKey(id), context.previousTenant)
        }
        setError('Failed to update tenant quota')
      },
      onSuccess: (_, { id }) => {
        // Invalidate related queries
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
        queryClient.invalidateQueries({ queryKey: getGetTenantByIdQueryKey(id) })
      },
      onSettled: () => {
        setOperationLoading('updateQuota', false)
      },
    },
  })
}

// ============================================================================
// Combined Admin Operations Hook
// ============================================================================

/**
 * Combined hook providing all admin tenant operations
 *
 * @example
 * ```tsx
 * function TenantManagement() {
 *   const {
 *     tenants,
 *     selectedTenant,
 *     operations,
 *     isLoading,
 *   } = useAdminOperations({ page: 1, page_size: 20 })
 *
 *   const handleSuspend = async (id: string) => {
 *     await operations.suspend.mutateAsync({ id, data: { reason: 'Admin action' } })
 *     Toast.success('Tenant suspended')
 *   }
 * }
 * ```
 */
export function useAdminOperations(listParams?: ListTenantsParams) {
  const store = useAdminStore()
  const queryClient = useQueryClient()

  // Queries
  const tenantsQuery = useAdminTenants(listParams)
  const selectedTenantQuery = useAdminTenant(store.selectedTenantId || '')
  const statsQuery = useAdminTenantStats()

  // Mutations
  const createMutation = useCreateAdminTenant()
  const updateMutation = useUpdateAdminTenant()
  const deleteMutation = useDeleteAdminTenant()
  const suspendMutation = useSuspendAdminTenant()
  const activateMutation = useActivateAdminTenant()
  const changePlanMutation = useChangePlanAdminTenant()
  const updateQuotaMutation = useUpdateQuotaAdminTenant()

  // Refresh all tenant data
  const refreshAll = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey() })
    queryClient.invalidateQueries({ queryKey: getGetTenantStatsQueryKey() })
    queryClient.invalidateQueries({ queryKey: adminQueryKeys.platformStats() })
    if (store.selectedTenantId) {
      queryClient.invalidateQueries({
        queryKey: getGetTenantByIdQueryKey(store.selectedTenantId),
      })
    }
  }, [queryClient, store.selectedTenantId])

  return {
    // State
    selectedTenantId: store.selectedTenantId,
    selectedTenantName: store.selectedTenantName,
    operationLoading: store.operationLoading,
    lastError: store.lastError,

    // Actions
    selectTenant: store.selectTenant,
    clearSelection: store.clearSelection,
    clearError: store.clearError,
    refreshAll,

    // Queries
    tenants: tenantsQuery,
    selectedTenant: selectedTenantQuery,
    stats: statsQuery,

    // Mutations
    operations: {
      create: createMutation,
      update: updateMutation,
      delete: deleteMutation,
      suspend: suspendMutation,
      activate: activateMutation,
      changePlan: changePlanMutation,
      updateQuota: updateQuotaMutation,
    },

    // Computed states
    isLoading: {
      tenants: tenantsQuery.isLoading,
      selectedTenant: selectedTenantQuery.isLoading,
      stats: statsQuery.isLoading,
      anyOperation: Object.values(store.operationLoading).some(Boolean),
    },
  }
}
