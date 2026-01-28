import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Spin,
  Button,
  Select,
  DatePicker,
  Space,
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconRefresh } from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { DataTable, TableToolbar, useTableState, type DataTableColumn } from '@/components/common'
import { getInventory } from '@/api/inventory/inventory'
import { listWarehouses } from '@/api/warehouses/warehouses'
import { listProducts } from '@/api/products/products'
import type {
  HandlerInventoryItemResponse,
  HandlerTransactionResponse,
  HandlerWarehouseListResponse,
  HandlerProductListResponse,
  ListInventoryTransactionsParams,
  ListInventoryTransactionsOrderDir,
  ListInventoryTransactionsTransactionType,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './StockTransactions.css'

const { Title, Text } = Typography

// Transaction type with index signature for DataTable compatibility
type Transaction = HandlerTransactionResponse & Record<string, unknown>

// Transaction type options for filter
const TRANSACTION_TYPE_OPTIONS = [
  { label: '全部类型', value: '' },
  { label: '入库', value: 'INBOUND' },
  { label: '出库', value: 'OUTBOUND' },
  { label: '锁定', value: 'LOCK' },
  { label: '解锁', value: 'UNLOCK' },
  { label: '调整', value: 'ADJUSTMENT' },
]

// Semi Design Tag color type
type TagColor =
  | 'amber'
  | 'blue'
  | 'cyan'
  | 'green'
  | 'grey'
  | 'indigo'
  | 'light-blue'
  | 'light-green'
  | 'lime'
  | 'orange'
  | 'pink'
  | 'purple'
  | 'red'
  | 'teal'
  | 'violet'
  | 'yellow'
  | 'white'

// Transaction type label map
const TRANSACTION_TYPE_LABELS: Record<string, { label: string; color: TagColor }> = {
  INBOUND: { label: '入库', color: 'green' },
  OUTBOUND: { label: '出库', color: 'red' },
  LOCK: { label: '锁定', color: 'orange' },
  UNLOCK: { label: '解锁', color: 'blue' },
  ADJUSTMENT: { label: '调整', color: 'purple' },
}

// Source type label map
const SOURCE_TYPE_LABELS: Record<string, string> = {
  PURCHASE_ORDER: '采购订单',
  SALES_ORDER: '销售订单',
  SALES_RETURN: '销售退货',
  PURCHASE_RETURN: '采购退货',
  STOCK_TAKE: '盘点',
  MANUAL: '手动调整',
  INITIAL: '期初',
}

/**
 * Format quantity for display with 2 decimal places
 * Safely handles both number and string inputs from API
 */
function formatQuantity(quantity?: number | string): string {
  if (quantity === undefined || quantity === null) return '-'
  const num = typeof quantity === 'string' ? parseFloat(quantity) : quantity
  if (typeof num !== 'number' || isNaN(num)) return '-'
  return num.toFixed(2)
}

/**
 * Format signed quantity (positive/negative indicator)
 * Safely handles both number and string inputs from API
 */
function formatSignedQuantity(quantity?: number | string): string {
  if (quantity === undefined || quantity === null) return '-'
  const num = typeof quantity === 'string' ? parseFloat(quantity) : quantity
  if (typeof num !== 'number' || isNaN(num)) return '-'
  const sign = num > 0 ? '+' : ''
  return `${sign}${num.toFixed(2)}`
}

/**
 * Format currency value
 * Safely handles both number and string inputs from API
 */
function formatCurrency(value?: number | string): string {
  if (value === undefined || value === null) return '-'
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (typeof num !== 'number' || isNaN(num)) return '-'
  return `¥${num.toFixed(2)}`
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
 * Format date to ISO string for API
 */
function formatDateToISO(date: Date | null): string | undefined {
  if (!date) return undefined
  return date.toISOString().split('T')[0]
}

/**
 * Inventory Stock Transactions Page
 *
 * Features:
 * - Display transaction history for a specific inventory item
 * - Filter by transaction type
 * - Filter by date range
 * - Sortable columns
 * - Pagination
 */
export default function StockTransactionsPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const inventoryApi = useMemo(() => getInventory(), [])

  // State for inventory item info (for header display)
  const [inventoryItem, setInventoryItem] = useState<HandlerInventoryItemResponse | null>(null)
  const [warehouseName, setWarehouseName] = useState<string>('')
  const [productName, setProductName] = useState<string>('')

  // State for transactions
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)

  // Filter state
  const [transactionTypeFilter, setTransactionTypeFilter] = useState<string>('')
  const [dateRange, setDateRange] = useState<[Date | null, Date | null]>([null, null])

  // Table state
  const { state, handleStateChange } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'transaction_date',
    defaultSortOrder: 'desc',
  })

  // Fetch inventory item details
  const fetchInventoryItem = useCallback(async () => {
    if (!id) return

    try {
      const response = await inventoryApi.getInventoryById(id)
      if (response.success && response.data) {
        setInventoryItem(response.data as HandlerInventoryItemResponse)
      }
    } catch {
      // Silently fail - header info is supplementary
    }
  }, [id, inventoryApi])

  // Fetch warehouse name
  const fetchWarehouseName = useCallback(async () => {
    if (!inventoryItem?.warehouse_id) return

    try {
      const response = await listWarehouses({
        page_size: 100,
        status: 'enabled',
      })
      if (response.status === 200 && response.data.success && response.data.data) {
        const warehouses = response.data.data as HandlerWarehouseListResponse[]
        const warehouse = warehouses.find(
          (w: HandlerWarehouseListResponse) => w.id === inventoryItem.warehouse_id
        )
        setWarehouseName(warehouse?.name || warehouse?.code || '-')
      }
    } catch {
      setWarehouseName('-')
    }
  }, [inventoryItem?.warehouse_id])

  // Fetch product name
  const fetchProductName = useCallback(async () => {
    if (!inventoryItem?.product_id) return

    try {
      const response = await listProducts({
        page_size: 500,
      })
      if (response.status === 200 && response.data.success && response.data.data) {
        const products = response.data.data as HandlerProductListResponse[]
        const product = products.find((p) => p.id === inventoryItem.product_id)
        setProductName(product?.name || product?.code || '-')
      }
    } catch {
      setProductName('-')
    }
  }, [inventoryItem?.product_id])

  // Fetch transactions
  const fetchTransactions = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const params: ListInventoryTransactionsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        order_by: state.sort.field || 'transaction_date',
        order_dir: (state.sort.order === 'asc'
          ? 'asc'
          : 'desc') as ListInventoryTransactionsOrderDir,
      }

      // Apply transaction type filter
      if (transactionTypeFilter) {
        params.transaction_type = transactionTypeFilter as ListInventoryTransactionsTransactionType
      }

      // Apply date range filter
      if (dateRange[0]) {
        params.start_date = formatDateToISO(dateRange[0])
      }
      if (dateRange[1]) {
        params.end_date = formatDateToISO(dateRange[1])
      }

      const response = await inventoryApi.listInventoryTransactionsByItem(id, params)

      if (response.success && response.data) {
        setTransactions(response.data as Transaction[])
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
      Toast.error('获取流水记录失败')
    } finally {
      setLoading(false)
    }
  }, [id, inventoryApi, state.pagination, state.sort, transactionTypeFilter, dateRange])

  // Fetch inventory item on mount
  useEffect(() => {
    fetchInventoryItem()
  }, [fetchInventoryItem])

  // Fetch warehouse and product names when inventory item is loaded
  useEffect(() => {
    if (inventoryItem) {
      fetchWarehouseName()
      fetchProductName()
    }
  }, [inventoryItem, fetchWarehouseName, fetchProductName])

  // Fetch transactions when filters change
  useEffect(() => {
    fetchTransactions()
  }, [fetchTransactions])

  // Handle back navigation
  const handleBack = useCallback(() => {
    if (id) {
      navigate(`/inventory/stock/${id}`)
    } else {
      navigate('/inventory/stock')
    }
  }, [id, navigate])

  // Handle refresh
  const handleRefresh = useCallback(() => {
    fetchTransactions()
  }, [fetchTransactions])

  // Handle transaction type filter change
  const handleTransactionTypeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const typeValue = typeof value === 'string' ? value : ''
      setTransactionTypeFilter(typeValue)
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle date range change

  const handleDateRangeChange = useCallback(
    (dates: any) => {
      if (Array.isArray(dates) && dates.length === 2) {
        const [start, end] = dates
        setDateRange([start instanceof Date ? start : null, end instanceof Date ? end : null])
      } else {
        setDateRange([null, null])
      }
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Table columns
  const columns: DataTableColumn<Transaction>[] = useMemo(
    () => [
      {
        title: '交易时间',
        dataIndex: 'transaction_date',
        width: 160,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
      {
        title: '类型',
        dataIndex: 'transaction_type',
        width: 80,
        align: 'center',
        render: (type: unknown) => {
          const typeStr = type as string | undefined
          const typeInfo = TRANSACTION_TYPE_LABELS[typeStr || ''] || {
            label: typeStr || '-',
            color: 'grey' as TagColor,
          }
          return <Tag color={typeInfo.color}>{typeInfo.label}</Tag>
        },
      },
      {
        title: '变动数量',
        dataIndex: 'signed_quantity',
        width: 100,
        align: 'right',
        sortable: true,
        render: (qty: unknown) => {
          const signedQty = qty as number | undefined
          const isPositive = signedQty !== undefined && signedQty > 0
          return (
            <span className={isPositive ? 'quantity-positive' : 'quantity-negative'}>
              {formatSignedQuantity(signedQty)}
            </span>
          )
        },
      },
      {
        title: '变动前余额',
        dataIndex: 'balance_before',
        width: 100,
        align: 'right',
        render: (qty: unknown) => formatQuantity(qty as number | undefined),
      },
      {
        title: '变动后余额',
        dataIndex: 'balance_after',
        width: 100,
        align: 'right',
        render: (qty: unknown) => formatQuantity(qty as number | undefined),
      },
      {
        title: '单位成本',
        dataIndex: 'unit_cost',
        width: 100,
        align: 'right',
        render: (cost: unknown) => formatCurrency(cost as number | undefined),
      },
      {
        title: '总成本',
        dataIndex: 'total_cost',
        width: 110,
        align: 'right',
        render: (cost: unknown) => formatCurrency(cost as number | undefined),
      },
      {
        title: '来源类型',
        dataIndex: 'source_type',
        width: 100,
        render: (type: unknown) => {
          const typeStr = type as string | undefined
          return SOURCE_TYPE_LABELS[typeStr || ''] || typeStr || '-'
        },
      },
      {
        title: '来源单号',
        dataIndex: 'source_id',
        width: 140,
        ellipsis: true,
        render: (sourceId: unknown) => (sourceId as string) || '-',
      },
      {
        title: '参考号',
        dataIndex: 'reference',
        width: 120,
        ellipsis: true,
        render: (ref: unknown) => (ref as string) || '-',
      },
      {
        title: '备注',
        dataIndex: 'reason',
        width: 150,
        ellipsis: true,
        render: (reason: unknown) => (reason as string) || '-',
      },
    ],
    []
  )

  return (
    <Container size="full" className="stock-transactions-page">
      {/* Header */}
      <div className="stock-transactions-header">
        <div className="header-left">
          <Button icon={<IconArrowLeft />} theme="borderless" onClick={handleBack}>
            返回
          </Button>
          <Title heading={4} style={{ margin: 0 }}>
            库存流水记录
          </Title>
        </div>
        <div className="header-right">
          <Button icon={<IconRefresh />} onClick={handleRefresh}>
            刷新
          </Button>
        </div>
      </div>

      {/* Item info summary */}
      {inventoryItem && (
        <Card className="info-summary-card">
          <div className="info-summary">
            <div className="info-item">
              <Text type="tertiary">仓库</Text>
              <Text strong>{warehouseName}</Text>
            </div>
            <div className="info-item">
              <Text type="tertiary">商品</Text>
              <Text strong>{productName}</Text>
            </div>
            <div className="info-item">
              <Text type="tertiary">当前数量</Text>
              <Text strong>{formatQuantity(inventoryItem.total_quantity)}</Text>
            </div>
            <div className="info-item">
              <Text type="tertiary">可用数量</Text>
              <Text strong>{formatQuantity(inventoryItem.available_quantity)}</Text>
            </div>
          </div>
        </Card>
      )}

      {/* Main content */}
      <Card className="transactions-card">
        <TableToolbar
          searchPlaceholder="搜索..."
          secondaryActions={[]}
          filters={
            <Space>
              <Select
                placeholder="交易类型"
                value={transactionTypeFilter}
                onChange={handleTransactionTypeChange}
                optionList={TRANSACTION_TYPE_OPTIONS}
                style={{ width: 120 }}
              />
              {}
              <DatePicker
                type="dateRange"
                placeholder={['开始日期', '结束日期']}
                value={(dateRange[0] && dateRange[1] ? dateRange : undefined) as any}
                onChange={handleDateRangeChange}
                style={{ width: 240 }}
              />
            </Space>
          }
        />

        <Spin spinning={loading}>
          <DataTable<Transaction>
            data={transactions}
            columns={columns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 1400 }}
          />
        </Spin>
      </Card>
    </Container>
  )
}
