import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Tag, Toast, Select, Space, Spin, Tooltip } from '@douyinfe/semi-ui-19'
import { IconRefresh, IconAlertTriangle } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { useFormatters } from '@/hooks/useFormatters'
import { getInventory } from '@/api/inventory/inventory'
import { listWarehouses } from '@/api/warehouses/warehouses'
import { listProducts } from '@/api/products/products'
import type {
  HandlerInventoryItemResponse,
  ListInventoriesParams,
  ListInventoriesOrderDir,
  HandlerWarehouseListResponse,
  HandlerProductListResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './StockList.css'
import { createScopedLogger } from '@/utils'

const log = createScopedLogger('StockList')

const { Title } = Typography

// Inventory item type with index signature for DataTable compatibility
type InventoryItem = HandlerInventoryItemResponse & Record<string, unknown>

type WarehouseOption = {
  label: string
  value: string
}

/**
 * Format quantity for display with 2 decimal places
 * Handles both number and string values from API
 */
function formatQuantity(quantity?: number | string): string {
  if (quantity === undefined || quantity === null) return '-'
  const num = typeof quantity === 'string' ? parseFloat(quantity) : quantity
  if (isNaN(num)) return '-'
  return num.toFixed(2)
}

/**
 * Inventory Stock List Page
 *
 * Features:
 * - Inventory listing with pagination
 * - Filter by warehouse
 * - Filter by stock status (has stock, low stock, no stock)
 * - Display available/locked/total quantities
 * - Show low stock warning indicators
 * - Navigate to inventory detail/transactions
 */
export default function StockListPage() {
  const navigate = useNavigate()
  const { t } = useTranslation(['inventory', 'common'])
  const { formatCurrency: formatCurrencyBase, formatDate: formatDateBase } = useFormatters()
  const inventoryApi = useMemo(() => getInventory(), [])

  // Wrapper functions to handle undefined values
  const formatCurrency = useCallback(
    (value?: number | string | null): string =>
      value !== undefined && value !== null ? formatCurrencyBase(value) : '-',
    [formatCurrencyBase]
  )
  const formatDate = useCallback(
    (date?: string, style?: 'date' | 'dateTime'): string =>
      date ? formatDateBase(date, style === 'dateTime' ? 'medium' : 'short') : '-',
    [formatDateBase]
  )

  // State for data
  const [inventoryList, setInventoryList] = useState<InventoryItem[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [warehouseFilter, setWarehouseFilter] = useState<string>('')
  const [stockStatusFilter, setStockStatusFilter] = useState<string>('')

  // Warehouse and product options for display
  const [warehouseOptions, setWarehouseOptions] = useState<WarehouseOption[]>([])
  const [warehouseMap, setWarehouseMap] = useState<Map<string, string>>(new Map())
  const [productMap, setProductMap] = useState<Map<string, string>>(new Map())

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'updated_at',
    defaultSortOrder: 'desc',
  })

  // Fetch warehouses for filter dropdown
  const fetchWarehouses = useCallback(
    async (signal?: AbortSignal) => {
      try {
        const response = await listWarehouses(
          {
            page: 1,
            page_size: 100,
            status: 'enabled',
          },
          { signal }
        )
        if (response.status === 200 && response.data.success && response.data.data) {
          const warehouses = response.data.data as HandlerWarehouseListResponse[]
          const options: WarehouseOption[] = [
            { label: t('stock.allWarehouses'), value: '' },
            ...warehouses.map((w: HandlerWarehouseListResponse) => ({
              label: w.name || w.code || '',
              value: w.id || '',
            })),
          ]
          setWarehouseOptions(options)

          // Build warehouse map for display
          const map = new Map<string, string>()
          warehouses.forEach((w: HandlerWarehouseListResponse) => {
            if (w.id) {
              map.set(w.id, w.name || w.code || w.id)
            }
          })
          setWarehouseMap(map)
        }
      } catch (error) {
        if (error instanceof Error && error.name === 'CanceledError') return
        log.error('Failed to fetch warehouses')
      }
    },
    [t]
  )

  // Fetch products for display names
  const fetchProducts = useCallback(async (signal?: AbortSignal) => {
    try {
      const response = await listProducts({ page: 1, page_size: 100 }, { signal })
      if (response.status === 200 && response.data.success && response.data.data) {
        const products = response.data.data as HandlerProductListResponse[]
        const map = new Map<string, string>()
        products.forEach((p) => {
          if (p.id) {
            map.set(p.id, p.name || p.code || p.id)
          }
        })
        setProductMap(map)
      }
    } catch (error) {
      if (error instanceof Error && error.name === 'CanceledError') return
      log.error('Failed to fetch products')
    }
  }, [])

  // Fetch inventory items
  const fetchInventory = useCallback(
    async (signal?: AbortSignal) => {
      setLoading(true)
      try {
        const params: ListInventoriesParams = {
          page: state.pagination.page,
          page_size: state.pagination.pageSize,
          search: searchKeyword || undefined,
          warehouse_id: warehouseFilter || undefined,
          order_by: state.sort.field || 'updated_at',
          order_dir: (state.sort.order === 'asc' ? 'asc' : 'desc') as ListInventoriesOrderDir,
        }

        // Apply stock status filter
        if (stockStatusFilter === 'has_stock') {
          params.has_stock = true
        } else if (stockStatusFilter === 'below_minimum') {
          params.below_minimum = true
        } else if (stockStatusFilter === 'no_stock') {
          params.has_stock = false
        }

        const response = await inventoryApi.listInventories(params, { signal })

        if (response.success && response.data) {
          setInventoryList(response.data as InventoryItem[])
          if (response.meta) {
            setPaginationMeta({
              page: response.meta.page || 1,
              page_size: response.meta.page_size || 20,
              total: response.meta.total || 0,
              total_pages: response.meta.total_pages || 1,
            })
          }
        }
      } catch (error) {
        if (error instanceof Error && error.name === 'CanceledError') return
        Toast.error(t('stock.messages.fetchError'))
      } finally {
        setLoading(false)
      }
    },
    [
      inventoryApi,
      state.pagination.page,
      state.pagination.pageSize,
      state.sort,
      searchKeyword,
      warehouseFilter,
      stockStatusFilter,
      t,
    ]
  )

  // Fetch warehouses and products on mount
  useEffect(() => {
    const abortController = new AbortController()
    fetchWarehouses(abortController.signal)
    fetchProducts(abortController.signal)
    return () => abortController.abort()
  }, [fetchWarehouses, fetchProducts])

  // Fetch inventory on mount and when state changes
  useEffect(() => {
    const abortController = new AbortController()
    fetchInventory(abortController.signal)
    return () => abortController.abort()
  }, [fetchInventory])

  // Handle search
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle warehouse filter change
  const handleWarehouseChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const warehouseValue = typeof value === 'string' ? value : ''
      setWarehouseFilter(warehouseValue)
      setFilter('warehouse_id', warehouseValue || null)
    },
    [setFilter]
  )

  // Handle stock status filter change
  const handleStockStatusChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const statusValue = typeof value === 'string' ? value : ''
      setStockStatusFilter(statusValue)
      setFilter('stock_status', statusValue || null)
    },
    [setFilter]
  )

  // Handle view transactions
  const handleViewTransactions = useCallback(
    (item: InventoryItem) => {
      if (item.id) {
        navigate(`/inventory/stock/${item.id}/transactions`)
      }
    },
    [navigate]
  )

  // Handle view detail
  const handleViewDetail = useCallback(
    (item: InventoryItem) => {
      if (item.id) {
        navigate(`/inventory/stock/${item.id}`)
      }
    },
    [navigate]
  )

  // Handle adjust stock
  const handleAdjustStock = useCallback(
    (item: InventoryItem) => {
      if (item.id) {
        navigate(
          `/inventory/adjust?warehouse_id=${item.warehouse_id}&product_id=${item.product_id}`
        )
      }
    },
    [navigate]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchInventory()
  }, [fetchInventory])

  // Get warehouse name by ID
  const getWarehouseName = useCallback(
    (warehouseId?: string): string => {
      if (!warehouseId) return '-'
      return warehouseMap.get(warehouseId) || warehouseId.substring(0, 8) + '...'
    },
    [warehouseMap]
  )

  // Get product name by ID
  const getProductName = useCallback(
    (productId?: string): string => {
      if (!productId) return '-'
      return productMap.get(productId) || productId.substring(0, 8) + '...'
    },
    [productMap]
  )

  // Stock status filter options
  const STOCK_STATUS_OPTIONS = useMemo(
    () => [
      { label: t('stock.statusFilter.all'), value: '' },
      { label: t('stock.statusFilter.hasStock'), value: 'has_stock' },
      { label: t('stock.statusFilter.belowMinimum'), value: 'below_minimum' },
      { label: t('stock.statusFilter.noStock'), value: 'no_stock' },
    ],
    [t]
  )

  // Table columns
  const tableColumns: DataTableColumn<InventoryItem>[] = useMemo(
    () => [
      {
        title: t('stock.columns.warehouse'),
        dataIndex: 'warehouse_id',
        width: 120,
        render: (warehouseId: unknown) => getWarehouseName(warehouseId as string | undefined),
      },
      {
        title: t('stock.columns.product'),
        dataIndex: 'product_id',
        width: 180,
        ellipsis: true,
        render: (productId: unknown) => (
          <span className="product-name-cell">
            {getProductName(productId as string | undefined)}
          </span>
        ),
      },
      {
        title: t('stock.columns.availableQuantity'),
        dataIndex: 'available_quantity',
        width: 110,
        align: 'right',
        sortable: true,
        render: (qty: unknown, record: InventoryItem) => {
          const availableQty = qty as number | undefined
          const isBelowMin = record.is_below_minimum
          return (
            <div className="quantity-cell">
              <span className={isBelowMin ? 'quantity-warning' : ''}>
                {formatQuantity(availableQty)}
              </span>
              {isBelowMin && (
                <Tooltip content={t('stock.tooltip.belowMinimum')}>
                  <span style={{ display: 'inline-flex' }}>
                    <IconAlertTriangle className="warning-icon" />
                  </span>
                </Tooltip>
              )}
            </div>
          )
        },
      },
      {
        title: t('stock.columns.lockedQuantity'),
        dataIndex: 'locked_quantity',
        width: 100,
        align: 'right',
        sortable: true,
        render: (qty: unknown) => {
          const lockedQty = qty as number | undefined
          return (
            <span className={lockedQty && lockedQty > 0 ? 'quantity-locked' : ''}>
              {formatQuantity(lockedQty)}
            </span>
          )
        },
      },
      {
        title: t('stock.columns.totalQuantity'),
        dataIndex: 'total_quantity',
        width: 100,
        align: 'right',
        sortable: true,
        render: (qty: unknown) => formatQuantity(qty as number | undefined),
      },
      {
        title: t('stock.columns.unitCost'),
        dataIndex: 'unit_cost',
        width: 100,
        align: 'right',
        sortable: true,
        render: (cost: unknown) => formatCurrency(cost as number | string | undefined),
      },
      {
        title: t('stock.columns.totalValue'),
        dataIndex: 'total_value',
        width: 110,
        align: 'right',
        sortable: true,
        render: (value: unknown) => formatCurrency(value as number | string | undefined),
      },
      {
        title: t('stock.columns.status'),
        dataIndex: 'is_below_minimum',
        width: 90,
        align: 'center',
        render: (_: unknown, record: InventoryItem) => {
          if (record.is_below_minimum) {
            return <Tag color="orange">{t('stock.status.lowStock')}</Tag>
          }
          if (record.is_above_maximum) {
            return <Tag color="blue">{t('stock.status.overMax')}</Tag>
          }
          const totalQty = record.total_quantity
          if (totalQty === undefined || totalQty === null || totalQty <= 0) {
            return <Tag color="red">{t('stock.status.noStock')}</Tag>
          }
          return <Tag color="green">{t('stock.status.normal')}</Tag>
        },
      },
      {
        title: t('stock.columns.updatedAt'),
        dataIndex: 'updated_at',
        width: 150,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined, 'dateTime'),
      },
    ],
    [getWarehouseName, getProductName, t, formatCurrency, formatDate]
  )

  // Table row actions
  const tableActions: TableAction<InventoryItem>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('stock.actions.viewDetail'),
        onClick: handleViewDetail,
      },
      {
        key: 'transactions',
        label: t('stock.actions.viewTransactions'),
        onClick: handleViewTransactions,
      },
      {
        key: 'adjust',
        label: t('stock.actions.adjustStock'),
        onClick: handleAdjustStock,
      },
    ],
    [handleViewDetail, handleViewTransactions, handleAdjustStock, t]
  )

  return (
    <Container size="full" className="stock-list-page">
      <Card className="stock-list-card">
        <div className="stock-list-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('stock.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('stock.searchPlaceholder')}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('stock.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space>
              <Select
                placeholder={t('stock.selectWarehouse')}
                value={warehouseFilter}
                onChange={handleWarehouseChange}
                optionList={warehouseOptions}
                style={{ width: 150 }}
              />
              <Select
                placeholder={t('stock.stockStatus')}
                value={stockStatusFilter}
                onChange={handleStockStatusChange}
                optionList={STOCK_STATUS_OPTIONS}
                style={{ width: 130 }}
              />
            </Space>
          }
        />

        <Spin spinning={loading}>
          <DataTable<InventoryItem>
            data={inventoryList}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 1200 }}
          />
        </Spin>
      </Card>
    </Container>
  )
}
