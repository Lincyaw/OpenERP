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
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconRefresh, IconEdit } from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '@/components/common/layout'
import { DataTable, useTableState, type DataTableColumn } from '@/components/common'
import { useFormatters } from '@/hooks/useFormatters'
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
  const { t } = useTranslation(['inventory', 'common'])
  const { formatCurrency: formatCurrencyBase, formatDate: formatDateBase } = useFormatters()
  const inventoryApi = useMemo(() => getInventory(), [])
  const warehousesApi = useMemo(() => getWarehouses(), [])
  const productsApi = useMemo(() => getProducts(), [])

  // Wrapper functions to handle undefined values
  const formatCurrency = useCallback(
    (value?: number): string => (value !== undefined ? formatCurrencyBase(value) : '-'),
    [formatCurrencyBase]
  )
  const formatDate = useCallback(
    (date?: string, style?: 'date' | 'dateTime'): string =>
      date ? formatDateBase(date, style === 'dateTime' ? 'medium' : 'short') : '-',
    [formatDateBase]
  )

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
      Toast.error(t('detail.messages.fetchError'))
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
      Toast.error(t('detail.messages.fetchTransactionsError'))
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

  // Transaction type colors
  const TRANSACTION_TYPE_COLORS: Record<string, TagColor> = {
    INBOUND: 'green',
    OUTBOUND: 'red',
    LOCK: 'orange',
    UNLOCK: 'blue',
    ADJUSTMENT: 'purple',
  }

  // Get status tag for inventory item
  const getStatusTag = useCallback(() => {
    if (!inventoryItem) return null

    if (inventoryItem.is_below_minimum) {
      return <Tag color="orange">{t('stock.status.lowStock')}</Tag>
    }
    if (inventoryItem.is_above_maximum) {
      return <Tag color="blue">{t('stock.status.overMax')}</Tag>
    }
    const totalQty = inventoryItem.total_quantity
    if (totalQty === undefined || totalQty === null || totalQty <= 0) {
      return <Tag color="red">{t('stock.status.noStock')}</Tag>
    }
    return <Tag color="green">{t('stock.status.normal')}</Tag>
  }, [inventoryItem, t])

  // Transactions table columns
  const transactionColumns: DataTableColumn<Transaction>[] = useMemo(
    () => [
      {
        title: t('detail.transactions.columns.transactionDate'),
        dataIndex: 'transaction_date',
        width: 160,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined, 'dateTime'),
      },
      {
        title: t('detail.transactions.columns.type'),
        dataIndex: 'transaction_type',
        width: 80,
        align: 'center',
        render: (type: unknown) => {
          const typeStr = type as string | undefined
          const color = TRANSACTION_TYPE_COLORS[typeStr || ''] || 'grey'
          const label = String(t(`detail.transactions.type.${typeStr}`, { defaultValue: typeStr || '-' }))
          return <Tag color={color}>{label}</Tag>
        },
      },
      {
        title: t('detail.transactions.columns.signedQuantity'),
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
        title: t('detail.transactions.columns.balanceBefore'),
        dataIndex: 'balance_before',
        width: 100,
        align: 'right',
        render: (qty: unknown) => formatQuantity(qty as number | undefined),
      },
      {
        title: t('detail.transactions.columns.balanceAfter'),
        dataIndex: 'balance_after',
        width: 100,
        align: 'right',
        render: (qty: unknown) => formatQuantity(qty as number | undefined),
      },
      {
        title: t('detail.transactions.columns.unitCost'),
        dataIndex: 'unit_cost',
        width: 100,
        align: 'right',
        render: (cost: unknown) => formatCurrency(cost as number | undefined),
      },
      {
        title: t('detail.transactions.columns.sourceType'),
        dataIndex: 'source_type',
        width: 100,
        render: (type: unknown) => {
          const typeStr = type as string | undefined
          return String(t(`detail.transactions.sourceType.${typeStr}`, { defaultValue: typeStr || '-' }))
        },
      },
      {
        title: t('detail.transactions.columns.sourceId'),
        dataIndex: 'source_id',
        width: 140,
        ellipsis: true,
        render: (sourceId: unknown) => (sourceId as string) || '-',
      },
      {
        title: t('detail.transactions.columns.remark'),
        dataIndex: 'reason',
        width: 150,
        ellipsis: true,
        render: (reason: unknown) => (reason as string) || '-',
      },
    ],
    [t, formatCurrency, formatDate, TRANSACTION_TYPE_COLORS]
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
          <Empty title={t('detail.notFound')} description={t('detail.notFoundDesc')}>
            <Button onClick={handleBack}>{t('detail.backToList')}</Button>
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
            {t('detail.back')}
          </Button>
          <Title heading={4} style={{ margin: 0 }}>
            {t('detail.title')}
          </Title>
          {getStatusTag()}
        </div>
        <div className="header-right">
          <Button icon={<IconRefresh />} onClick={handleRefresh}>
            {t('detail.refresh')}
          </Button>
          <Button type="primary" icon={<IconEdit />} onClick={handleAdjustStock}>
            {t('detail.adjustStock')}
          </Button>
        </div>
      </div>

      {/* Basic Info Card */}
      <Card className="detail-card">
        <Title heading={5}>{t('detail.basicInfo.title')}</Title>
        <Descriptions
          data={[
            { key: t('detail.basicInfo.warehouse'), value: warehouseName },
            { key: t('detail.basicInfo.product'), value: productName },
            { key: t('detail.basicInfo.updatedAt'), value: formatDate(inventoryItem.updated_at, 'dateTime') },
          ]}
        />
      </Card>

      {/* Quantity Info Card */}
      <Card className="detail-card">
        <Title heading={5}>{t('detail.quantity.title')}</Title>
        <div className="quantity-grid">
          <div className="quantity-item">
            <Text type="tertiary">{t('detail.quantity.available')}</Text>
            <Text
              className={`quantity-value ${inventoryItem.is_below_minimum ? 'quantity-warning' : ''}`}
            >
              {formatQuantity(inventoryItem.available_quantity)}
            </Text>
          </div>
          <div className="quantity-item">
            <Text type="tertiary">{t('detail.quantity.locked')}</Text>
            <Text
              className={`quantity-value ${inventoryItem.locked_quantity && inventoryItem.locked_quantity > 0 ? 'quantity-locked' : ''}`}
            >
              {formatQuantity(inventoryItem.locked_quantity)}
            </Text>
          </div>
          <div className="quantity-item">
            <Text type="tertiary">{t('detail.quantity.total')}</Text>
            <Text className="quantity-value">{formatQuantity(inventoryItem.total_quantity)}</Text>
          </div>
        </div>
      </Card>

      {/* Cost Info Card */}
      <Card className="detail-card">
        <Title heading={5}>{t('detail.cost.title')}</Title>
        <Descriptions
          data={[
            { key: t('detail.cost.unitCost'), value: formatCurrency(inventoryItem.unit_cost) },
            { key: t('detail.cost.totalValue'), value: formatCurrency(inventoryItem.total_value) },
          ]}
        />
      </Card>

      {/* Threshold Info Card */}
      <Card className="detail-card">
        <Title heading={5}>{t('detail.threshold.title')}</Title>
        <Descriptions
          data={[
            {
              key: t('detail.threshold.minQuantity'),
              value:
                inventoryItem.min_quantity !== undefined && inventoryItem.min_quantity !== null
                  ? formatQuantity(inventoryItem.min_quantity)
                  : t('detail.threshold.notSet'),
            },
            {
              key: t('detail.threshold.maxQuantity'),
              value:
                inventoryItem.max_quantity !== undefined && inventoryItem.max_quantity !== null
                  ? formatQuantity(inventoryItem.max_quantity)
                  : t('detail.threshold.notSet'),
            },
          ]}
        />
      </Card>

      {/* Transactions Tab */}
      <Card className="detail-card transactions-card">
        <Tabs type="line">
          <TabPane tab={t('detail.transactions.title')} itemKey="transactions">
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
