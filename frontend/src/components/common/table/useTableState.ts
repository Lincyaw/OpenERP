import { useState, useCallback, useMemo } from 'react'
import type {
  TableState,
  TableStateChange,
  SortOrder,
  FilterState,
  FilterValue,
  UseTableStateOptions,
  UseTableStateReturn,
} from './types'
import type { PaginationParams } from '@/types/api'

const DEFAULT_PAGE_SIZE = 10

/**
 * Hook for managing table state (pagination, sorting, filtering)
 *
 * @example
 * ```tsx
 * const { state, handleStateChange, toApiParams } = useTableState({
 *   defaultPageSize: 20,
 *   defaultSortField: 'created_at',
 *   defaultSortOrder: 'desc',
 * })
 *
 * // Use with API call
 * const { data } = useQuery(['products', toApiParams()], () =>
 *   fetchProducts(toApiParams())
 * )
 *
 * // Pass to DataTable
 * <DataTable
 *   data={data}
 *   onStateChange={handleStateChange}
 *   sortState={state.sort}
 *   filterState={state.filters}
 * />
 * ```
 */
export function useTableState(options: UseTableStateOptions = {}): UseTableStateReturn {
  const {
    defaultPageSize = DEFAULT_PAGE_SIZE,
    defaultSortField,
    defaultSortOrder,
    defaultFilters = {},
  } = options

  const initialState: TableState = useMemo(
    () => ({
      pagination: {
        page: 1,
        pageSize: defaultPageSize,
      },
      sort: {
        field: defaultSortField,
        order: defaultSortOrder,
      },
      filters: defaultFilters,
    }),
    [defaultPageSize, defaultSortField, defaultSortOrder, defaultFilters]
  )

  const [state, setState] = useState<TableState>(initialState)

  const setPagination = useCallback((page: number, pageSize?: number) => {
    setState((prev) => ({
      ...prev,
      pagination: {
        page,
        pageSize: pageSize ?? prev.pagination.pageSize,
      },
    }))
  }, [])

  const setSort = useCallback((field: string | undefined, order?: SortOrder) => {
    setState((prev) => ({
      ...prev,
      sort: { field, order },
      // Reset to page 1 when sorting changes
      pagination: { ...prev.pagination, page: 1 },
    }))
  }, [])

  const setFilters = useCallback((filters: FilterState) => {
    setState((prev) => ({
      ...prev,
      filters,
      // Reset to page 1 when filters change
      pagination: { ...prev.pagination, page: 1 },
    }))
  }, [])

  const setFilter = useCallback((key: string, value: FilterValue) => {
    setState((prev) => ({
      ...prev,
      filters: { ...prev.filters, [key]: value },
      // Reset to page 1 when filter changes
      pagination: { ...prev.pagination, page: 1 },
    }))
  }, [])

  const clearFilters = useCallback(() => {
    setState((prev) => ({
      ...prev,
      filters: {},
      pagination: { ...prev.pagination, page: 1 },
    }))
  }, [])

  const reset = useCallback(() => {
    setState(initialState)
  }, [initialState])

  const handleStateChange = useCallback((change: TableStateChange) => {
    setState((prev) => {
      const newState = { ...prev }

      if (change.pagination) {
        newState.pagination = change.pagination
      }

      if (change.sort !== undefined) {
        newState.sort = change.sort
        // Reset to page 1 on sort change if pagination wasn't also changed
        if (!change.pagination) {
          newState.pagination = { ...prev.pagination, page: 1 }
        }
      }

      if (change.filters !== undefined) {
        newState.filters = change.filters
        // Reset to page 1 on filter change if pagination wasn't also changed
        if (!change.pagination) {
          newState.pagination = { ...prev.pagination, page: 1 }
        }
      }

      return newState
    })
  }, [])

  const toApiParams = useCallback((): PaginationParams & FilterState => {
    const { pagination, sort, filters } = state

    const params: PaginationParams & FilterState = {
      page: pagination.page,
      page_size: pagination.pageSize,
      ...filters,
    }

    if (sort.field && sort.order) {
      params.sort_by = sort.field
      params.sort_order = sort.order
    }

    return params
  }, [state])

  return {
    state,
    setPagination,
    setSort,
    setFilters,
    setFilter,
    clearFilters,
    reset,
    handleStateChange,
    toApiParams,
  }
}
