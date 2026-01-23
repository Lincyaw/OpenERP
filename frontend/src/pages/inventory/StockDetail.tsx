import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Descriptions,
  Spin,
  Button,
  Tabs,
  TabPane,
  Empty,
} from '@douyinfe/semi-ui'
import { IconArrowLeft, IconRefresh, IconEdit } from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { DataTable, useTableState, type DataTableColumn } from '@/components/common'
import { getInventory } from '@/api/inventory/inventory'
import { getWarehouses } from '@/api/warehouses/warehouses'
import { getProducts } from '@/api/products/products'
import type {
  HandlerInventoryItemResponse,
  HandlerTransactionResponse,
  HandlerWarehouseListResponse,
  HandlerProductListResponse,
  GetInventoryItemsIdTransactionsParams,
  GetInventoryItemsIdTransactionsOrderDir,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './StockDetail.css'

const { Title, Text } = Typography

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

// Transaction type with index signature for DataTable compatibility
type Transaction = HandlerTransactionResponse & Record<string, unknown>

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
 */
function formatQuantity(quantity?: number): string {
  if (quantity === undefined || quantity === null) return '-'
  return quantity.toFixed(2)
}

/**
 * Format signed quantity (positive/negative indicator)
 */
function formatSignedQuantity(quantity?: number): string {
  if (quantity === undefined || quantity === null) return '-'
  const sign = quantity > 0 ? '+' : ''
  return `${sign}${quantity.toFixed(2)}`
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
 * Inventory Stock Detail Page
 *
 * Features:
 * - Display inventory item details (quantities, costs, thresholds)
 * - Show transaction history with filtering
 * - Display stock locks
 */
export default function StockDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const inventoryApi = useMemo(() => getInventory(), [])
  const warehousesApi = useMemo(() => getWarehouses(), [])
  const productsApi = useMemo(() => getProducts(), [])

  // State for inventory item
  const [inventoryItem, setInventoryItem] = useState<HandlerInventoryItemResponse | null>(null)
  const [loading, setLoading] = useState(false)

  // State for transactions
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [transactionsPagination, setTransactionsPagination] = useState<PaginationMeta | undefined>(
    undefined
  )
  const [transactionsLoading, setTransactionsLoading] = useState(false)

  // Warehouse and product display names
  const [warehouseName, setWarehouseName] = useState<string>('')
  const [productName, setProductName] = useState<string>('')

  // Table state for transactions
  const { state: transactionsState, handleStateChange: handleTransactionsStateChange } =
    useTableState({
      defaultPageSize: 20,
      defaultSortField: 'transaction_date',
      defaultSortOrder: 'desc',
    })

  // Fetch inventory item details
  const fetchInventoryItem = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await inventoryApi.getInventoryItemsId(id)
      if (response.success && response.data) {
        setInventoryItem(response.data as HandlerInventoryItemResponse)
      }
    } catch {
      Toast.error('获取库存详情失败')
    } finally {
      setLoading(false)
    }
  }, [id, inventoryApi])

  // Fetch warehouse name
  const fetchWarehouseName = useCallback(async () => {
    if (!inventoryItem?.warehouse_id) return

    try {
      const response = await warehousesApi.getPartnerWarehouses({
        page_size: 100,
        status: 'active',
      })
      if (response.success && response.data) {
        const warehouses = response.data as HandlerWarehouseListResponse[]
        const warehouse = warehouses.find((w) => w.id === inventoryItem.warehouse_id)
        setWarehouseName(warehouse?.name || warehouse?.code || inventoryItem.warehouse_id || '-')
      }
    } catch {
      // Silently fail - use ID as fallback
      setWarehouseName(inventoryItem.warehouse_id || '-')
    }
  }, [inventoryItem?.warehouse_id, warehousesApi])

  // Fetch product name
  const fetchProductName = useCallback(async () => {
    if (!inventoryItem?.product_id) return

    try {
      const response = await productsApi.getCatalogProducts({
        page_size: 500,
      })
      if (response.success && response.data) {
        const products = response.data as HandlerProductListResponse[]
        const product = products.find((p) => p.id === inventoryItem.product_id)
        setProductName(product?.name || product?.code || inventoryItem.product_id || '-')
      }
    } catch {
      // Silently fail - use ID as fallback
      setProductName(inventoryItem.product_id || '-')
    }
  }, [inventoryItem?.product_id, productsApi])

  // Fetch transactions
  const fetchTransactions = useCallback(async () => {
    if (!id) return

    setTransactionsLoading(true)
    try {
      const params: GetInventoryItemsIdTransactionsParams = {
        page: transactionsState.pagination.page,
        page_size: transactionsState.pagination.pageSize,
        order_by: transactionsState.sort.field || 'transaction_date',
        order_dir: (transactionsState.sort.order === 'asc'
          ? 'asc'
          : 'desc') as GetInventoryItemsIdTransactionsOrderDir,
      }

      const response = await inventoryApi.getInventoryItemsIdTransactions(id, params)

      if (response.success && response.data) {
        setTransactions(response.data as Transaction[])
        if (response.meta) {
          setTransactionsPagination({
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
      setTransactionsLoading(false)
    }
  }, [id, inventoryApi, transactionsState.pagination, transactionsState.sort])

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

  // Fetch transactions when inventory item is loaded or pagination changes
  useEffect(() => {
    if (id) {
      fetchTransactions()
    }
  }, [id, fetchTransactions])

  // Handle back navigation
  const handleBack = useCallback(() => {
    navigate('/inventory/stock')
  }, [navigate])

  // Handle refresh
  const handleRefresh = useCallback(() => {
    fetchInventoryItem()
    fetchTransactions()
  }, [fetchInventoryItem, fetchTransactions])

  // Handle adjust stock navigation
  const handleAdjustStock = useCallback(() => {
    if (inventoryItem) {
      navigate(
        `/inventory/adjust?warehouse_id=${inventoryItem.warehouse_id}&product_id=${inventoryItem.product_id}`
      )
    }
  }, [inventoryItem, navigate])

  // Get status tag for inventory item
  const getStatusTag = useCallback(() => {
    if (!inventoryItem) return null

    if (inventoryItem.is_below_minimum) {
      return <Tag color="orange">低库存</Tag>
    }
    if (inventoryItem.is_above_maximum) {
      return <Tag color="blue">超上限</Tag>
    }
    const totalQty = inventoryItem.total_quantity
    if (totalQty === undefined || totalQty === null || totalQty <= 0) {
      return <Tag color="red">无库存</Tag>
    }
    return <Tag color="green">正常</Tag>
  }, [inventoryItem])

  // Transactions table columns
  const transactionColumns: DataTableColumn<Transaction>[] = useMemo(
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
        title: '备注',
        dataIndex: 'reason',
        width: 150,
        ellipsis: true,
        render: (reason: unknown) => (reason as string) || '-',
      },
    ],
    []
  )

  if (loading) {
    return (
      <Container size="full" className="stock-detail-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
      </Container>
    )
  }

  if (!inventoryItem) {
    return (
      <Container size="full" className="stock-detail-page">
        <Card>
          <Empty title="未找到库存记录" description="请检查库存ID是否正确">
            <Button onClick={handleBack}>返回列表</Button>
          </Empty>
        </Card>
      </Container>
    )
  }

  return (
    <Container size="full" className="stock-detail-page">
      {/* Header */}
      <div className="stock-detail-header">
        <div className="header-left">
          <Button icon={<IconArrowLeft />} theme="borderless" onClick={handleBack}>
            返回
          </Button>
          <Title heading={4} style={{ margin: 0 }}>
            库存详情
          </Title>
          {getStatusTag()}
        </div>
        <div className="header-right">
          <Button icon={<IconRefresh />} onClick={handleRefresh}>
            刷新
          </Button>
          <Button type="primary" icon={<IconEdit />} onClick={handleAdjustStock}>
            库存调整
          </Button>
        </div>
      </div>

      {/* Basic Info Card */}
      <Card className="detail-card">
        <Title heading={5}>基本信息</Title>
        <Descriptions
          data={[
            { key: '仓库', value: warehouseName },
            { key: '商品', value: productName },
            { key: '更新时间', value: formatDate(inventoryItem.updated_at) },
          ]}
        />
      </Card>

      {/* Quantity Info Card */}
      <Card className="detail-card">
        <Title heading={5}>库存数量</Title>
        <div className="quantity-grid">
          <div className="quantity-item">
            <Text type="tertiary">可用数量</Text>
            <Text
              className={`quantity-value ${inventoryItem.is_below_minimum ? 'quantity-warning' : ''}`}
            >
              {formatQuantity(inventoryItem.available_quantity)}
            </Text>
          </div>
          <div className="quantity-item">
            <Text type="tertiary">锁定数量</Text>
            <Text
              className={`quantity-value ${inventoryItem.locked_quantity && inventoryItem.locked_quantity > 0 ? 'quantity-locked' : ''}`}
            >
              {formatQuantity(inventoryItem.locked_quantity)}
            </Text>
          </div>
          <div className="quantity-item">
            <Text type="tertiary">总数量</Text>
            <Text className="quantity-value">{formatQuantity(inventoryItem.total_quantity)}</Text>
          </div>
        </div>
      </Card>

      {/* Cost Info Card */}
      <Card className="detail-card">
        <Title heading={5}>成本信息</Title>
        <Descriptions
          data={[
            { key: '单位成本', value: formatCurrency(inventoryItem.unit_cost) },
            { key: '库存总值', value: formatCurrency(inventoryItem.total_value) },
          ]}
        />
      </Card>

      {/* Threshold Info Card */}
      <Card className="detail-card">
        <Title heading={5}>库存阈值</Title>
        <Descriptions
          data={[
            {
              key: '最小库存',
              value:
                inventoryItem.min_quantity !== undefined && inventoryItem.min_quantity !== null
                  ? formatQuantity(inventoryItem.min_quantity)
                  : '未设置',
            },
            {
              key: '最大库存',
              value:
                inventoryItem.max_quantity !== undefined && inventoryItem.max_quantity !== null
                  ? formatQuantity(inventoryItem.max_quantity)
                  : '未设置',
            },
          ]}
        />
      </Card>

      {/* Transactions Tab */}
      <Card className="detail-card transactions-card">
        <Tabs type="line">
          <TabPane tab="流水记录" itemKey="transactions">
            <Spin spinning={transactionsLoading}>
              <DataTable<Transaction>
                data={transactions}
                columns={transactionColumns}
                rowKey="id"
                loading={transactionsLoading}
                pagination={transactionsPagination}
                onStateChange={handleTransactionsStateChange}
                sortState={transactionsState.sort}
                scroll={{ x: 1100 }}
              />
            </Spin>
          </TabPane>
        </Tabs>
      </Card>
    </Container>
  )
}
