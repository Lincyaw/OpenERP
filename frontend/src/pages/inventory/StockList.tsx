import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Tag, Toast, Select, Space, Spin, Tooltip } from '@douyinfe/semi-ui'
import { IconRefresh, IconAlertTriangle } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { getInventory } from '@/api/inventory/inventory'
import { getWarehouses } from '@/api/warehouses/warehouses'
import { getProducts } from '@/api/products/products'
import type {
  HandlerInventoryItemResponse,
  GetInventoryItemsParams,
  GetInventoryItemsOrderDir,
  HandlerWarehouseListResponse,
  HandlerProductListResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './StockList.css'

const { Title } = Typography

// Inventory item type with index signature for DataTable compatibility
type InventoryItem = HandlerInventoryItemResponse & Record<string, unknown>

// Warehouse option type
type WarehouseOption = {
  label: string
  value: string
}

// Stock status filter options
const STOCK_STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '有库存', value: 'has_stock' },
  { label: '低库存预警', value: 'below_minimum' },
  { label: '无库存', value: 'no_stock' },
]

/**
 * Format quantity for display with 2 decimal places
 */
function formatQuantity(quantity?: number): string {
  if (quantity === undefined || quantity === null) return '-'
  return quantity.toFixed(2)
}

/**
 * Format currency value
 */
function formatCurrency(value?: number): string {
  if (value === undefined || value === null) return '-'
  return `¥${value.toFixed(2)}`
}

/**
 * Format date for display
 */
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
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
  const inventoryApi = useMemo(() => getInventory(), [])
  const warehousesApi = useMemo(() => getWarehouses(), [])
  const productsApi = useMemo(() => getProducts(), [])

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
  const fetchWarehouses = useCallback(async () => {
    try {
      const response = await warehousesApi.getPartnerWarehouses({
        page_size: 100,
        status: 'active',
      })
      if (response.success && response.data) {
        const warehouses = response.data as HandlerWarehouseListResponse[]
        const options: WarehouseOption[] = [
          { label: '全部仓库', value: '' },
          ...warehouses.map((w) => ({
            label: w.name || w.code || '',
            value: w.id || '',
          })),
        ]
        setWarehouseOptions(options)

        // Build warehouse map for display
        const map = new Map<string, string>()
        warehouses.forEach((w) => {
          if (w.id) {
            map.set(w.id, w.name || w.code || w.id)
          }
        })
        setWarehouseMap(map)
      }
    } catch {
      console.error('Failed to fetch warehouses')
    }
  }, [warehousesApi])

  // Fetch products for display names
  const fetchProducts = useCallback(async () => {
    try {
      const response = await productsApi.getCatalogProducts({ page_size: 500 })
      if (response.success && response.data) {
        const products = response.data as HandlerProductListResponse[]
        const map = new Map<string, string>()
        products.forEach((p) => {
          if (p.id) {
            map.set(p.id, p.name || p.code || p.id)
          }
        })
        setProductMap(map)
      }
    } catch {
      console.error('Failed to fetch products')
    }
  }, [productsApi])

  // Fetch inventory items
  const fetchInventory = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetInventoryItemsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        warehouse_id: warehouseFilter || undefined,
        order_by: state.sort.field || 'updated_at',
        order_dir: (state.sort.order === 'asc' ? 'asc' : 'desc') as GetInventoryItemsOrderDir,
      }

      // Apply stock status filter
      if (stockStatusFilter === 'has_stock') {
        params.has_stock = true
      } else if (stockStatusFilter === 'below_minimum') {
        params.below_minimum = true
      } else if (stockStatusFilter === 'no_stock') {
        params.has_stock = false
      }

      const response = await inventoryApi.getInventoryItems(params)

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
    } catch {
      Toast.error('获取库存列表失败')
    } finally {
      setLoading(false)
    }
  }, [
    inventoryApi,
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    searchKeyword,
    warehouseFilter,
    stockStatusFilter,
  ])

  // Fetch warehouses and products on mount
  useEffect(() => {
    fetchWarehouses()
    fetchProducts()
  }, [fetchWarehouses, fetchProducts])

  // Fetch inventory on mount and when state changes
  useEffect(() => {
    fetchInventory()
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

  // Table columns
  const tableColumns: DataTableColumn<InventoryItem>[] = useMemo(
    () => [
      {
        title: '仓库',
        dataIndex: 'warehouse_id',
        width: 120,
        render: (warehouseId: unknown) => getWarehouseName(warehouseId as string | undefined),
      },
      {
        title: '商品',
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
        title: '可用数量',
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
                <Tooltip content="低于安全库存">
                  <IconAlertTriangle className="warning-icon" />
                </Tooltip>
              )}
            </div>
          )
        },
      },
      {
        title: '锁定数量',
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
        title: '总数量',
        dataIndex: 'total_quantity',
        width: 100,
        align: 'right',
        sortable: true,
        render: (qty: unknown) => formatQuantity(qty as number | undefined),
      },
      {
        title: '单位成本',
        dataIndex: 'unit_cost',
        width: 100,
        align: 'right',
        sortable: true,
        render: (cost: unknown) => formatCurrency(cost as number | undefined),
      },
      {
        title: '库存总值',
        dataIndex: 'total_value',
        width: 110,
        align: 'right',
        sortable: true,
        render: (value: unknown) => formatCurrency(value as number | undefined),
      },
      {
        title: '状态',
        dataIndex: 'is_below_minimum',
        width: 90,
        align: 'center',
        render: (_: unknown, record: InventoryItem) => {
          if (record.is_below_minimum) {
            return <Tag color="orange">低库存</Tag>
          }
          if (record.is_above_maximum) {
            return <Tag color="blue">超上限</Tag>
          }
          const totalQty = record.total_quantity
          if (totalQty === undefined || totalQty === null || totalQty <= 0) {
            return <Tag color="red">无库存</Tag>
          }
          return <Tag color="green">正常</Tag>
        },
      },
      {
        title: '更新时间',
        dataIndex: 'updated_at',
        width: 150,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
    ],
    [getWarehouseName, getProductName]
  )

  // Table row actions
  const tableActions: TableAction<InventoryItem>[] = useMemo(
    () => [
      {
        key: 'view',
        label: '查看明细',
        onClick: handleViewDetail,
      },
      {
        key: 'transactions',
        label: '流水记录',
        onClick: handleViewTransactions,
      },
      {
        key: 'adjust',
        label: '库存调整',
        onClick: handleAdjustStock,
      },
    ],
    [handleViewDetail, handleViewTransactions, handleAdjustStock]
  )

  return (
    <Container size="full" className="stock-list-page">
      <Card className="stock-list-card">
        <div className="stock-list-header">
          <Title heading={4} style={{ margin: 0 }}>
            库存查询
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索商品..."
          secondaryActions={[
            {
              key: 'refresh',
              label: '刷新',
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space>
              <Select
                placeholder="选择仓库"
                value={warehouseFilter}
                onChange={handleWarehouseChange}
                optionList={warehouseOptions}
                style={{ width: 150 }}
              />
              <Select
                placeholder="库存状态"
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
