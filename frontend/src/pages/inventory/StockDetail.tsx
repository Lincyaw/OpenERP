import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Tag, Toast, Descriptions, Tabs, TabPane, Empty } from '@douyinfe/semi-ui-19'
import { IconRefresh, IconEdit } from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '@/components/common/layout'
import {
  DataTable,
  useTableState,
  type DataTableColumn,
  DetailPageHeader,
  type DetailPageHeaderAction,
  type DetailPageHeaderStatus,
  type DetailPageHeaderMetric,
} from '@/components/common'
import { useFormatters } from '@/hooks/useFormatters'
import { getInventoryById, listInventoryTransactionsByItem } from '@/api/inventory/inventory'
import { listWarehouses } from '@/api/warehouses/warehouses'
import { listProducts } from '@/api/products/products'
import type {
  HandlerInventoryItemResponse,
  HandlerTransactionResponse,
  HandlerWarehouseListResponse,
  HandlerProductListResponse,
  ListInventoryTransactionsParams,
  ListInventoryTransactionsOrderDir,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './StockDetail.css'

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

// Transaction type colors
const TRANSACTION_TYPE_COLORS: Record<string, TagColor> = {
  INBOUND: 'green',
  OUTBOUND: 'red',
  LOCK: 'orange',
  UNLOCK: 'blue',
  ADJUSTMENT: 'purple',
}

/**
 * Inventory Stock Detail Page
 *
 * Features:
 * - Display inventory item details using DetailPageHeader
 * - Show transaction history with filtering
 * - Display stock locks
 */
export default function StockDetailPage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation(['inventory', 'common'])
  const { formatCurrency: formatCurrencyBase, formatDate: formatDateBase } = useFormatters()

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
      const response = await getInventoryById(id)
      if (response.status === 200 && response.data.success && response.data.data) {
        setInventoryItem(response.data.data as HandlerInventoryItemResponse)
      }
    } catch {
      Toast.error(t('detail.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [id, t])

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
        setWarehouseName(warehouse?.name || warehouse?.code || inventoryItem.warehouse_id || '-')
      }
    } catch {
      // Silently fail - use ID as fallback
      setWarehouseName(inventoryItem.warehouse_id || '-')
    }
  }, [inventoryItem?.warehouse_id])

  // Fetch product name
  const fetchProductName = useCallback(async () => {
    if (!inventoryItem?.product_id) return

    try {
      const response = await listProducts({
        page_size: 100,
      })
      if (response.status === 200 && response.data.success && response.data.data) {
        const products = response.data.data as HandlerProductListResponse[]
        const product = products.find((p) => p.id === inventoryItem.product_id)
        setProductName(product?.name || product?.code || inventoryItem.product_id || '-')
      }
    } catch {
      // Silently fail - use ID as fallback
      setProductName(inventoryItem.product_id || '-')
    }
  }, [inventoryItem?.product_id])

  // Fetch transactions
  const fetchTransactions = useCallback(async () => {
    if (!id) return

    setTransactionsLoading(true)
    try {
      const params: ListInventoryTransactionsParams = {
        page: transactionsState.pagination.page,
        page_size: transactionsState.pagination.pageSize,
        order_by: transactionsState.sort.field || 'transaction_date',
        order_dir: (transactionsState.sort.order === 'asc'
          ? 'asc'
          : 'desc') as ListInventoryTransactionsOrderDir,
      }

      const response = await listInventoryTransactionsByItem(id, params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setTransactions(response.data.data as Transaction[])
        if (response.data.meta) {
          setTransactionsPagination({
            page: response.data.meta.page || 1,
            page_size: response.data.meta.page_size || 20,
            total: response.data.meta.total || 0,
            total_pages: response.data.meta.total_pages || 1,
          })
        }
      }
    } catch {
      Toast.error(t('detail.messages.fetchTransactionsError'))
    } finally {
      setTransactionsLoading(false)
    }
  }, [id, transactionsState.pagination, transactionsState.sort, t])

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

  // Get stock status
  const getStockStatus = useCallback((): {
    label: string
    variant: 'default' | 'success' | 'warning' | 'danger'
  } => {
    if (!inventoryItem) return { label: '-', variant: 'default' }

    if (inventoryItem.is_below_minimum) {
      return { label: t('stock.status.lowStock'), variant: 'warning' }
    }
    if (inventoryItem.is_above_maximum) {
      return { label: t('stock.status.overMax'), variant: 'warning' }
    }
    const totalQty = inventoryItem.total_quantity
    if (totalQty === undefined || totalQty === null || totalQty <= 0) {
      return { label: t('stock.status.noStock'), variant: 'danger' }
    }
    return { label: t('stock.status.normal'), variant: 'success' }
  }, [inventoryItem, t])

  // Build status for DetailPageHeader
  const headerStatus = useMemo((): DetailPageHeaderStatus | undefined => {
    const status = getStockStatus()
    return {
      label: status.label,
      variant: status.variant,
    }
  }, [getStockStatus])

  // Build metrics for DetailPageHeader
  const headerMetrics = useMemo((): DetailPageHeaderMetric[] => {
    if (!inventoryItem) return []
    return [
      {
        label: t('detail.quantity.available'),
        value: formatQuantity(inventoryItem.available_quantity),
        variant: inventoryItem.is_below_minimum ? 'warning' : 'success',
      },
      {
        label: t('detail.quantity.locked'),
        value: formatQuantity(inventoryItem.locked_quantity),
        variant:
          inventoryItem.locked_quantity && inventoryItem.locked_quantity > 0
            ? 'warning'
            : 'default',
      },
      {
        label: t('detail.quantity.total'),
        value: formatQuantity(inventoryItem.total_quantity),
      },
      {
        label: t('detail.cost.totalValue'),
        value: formatCurrency(inventoryItem.total_value),
        variant: 'primary',
      },
    ]
  }, [inventoryItem, t, formatCurrency])

  // Build primary action for DetailPageHeader
  const primaryAction = useMemo((): DetailPageHeaderAction => {
    return {
      key: 'adjust',
      label: t('detail.adjustStock'),
      icon: <IconEdit />,
      type: 'primary',
      onClick: handleAdjustStock,
    }
  }, [t, handleAdjustStock])

  // Build secondary actions for DetailPageHeader
  const secondaryActions = useMemo((): DetailPageHeaderAction[] => {
    return [
      {
        key: 'refresh',
        label: t('detail.refresh'),
        icon: <IconRefresh />,
        onClick: handleRefresh,
      },
    ]
  }, [t, handleRefresh])

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
          const label = String(
            t(`detail.transactions.type.${typeStr}`, { defaultValue: typeStr || '-' })
          )
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
          return String(
            t(`detail.transactions.sourceType.${typeStr}`, { defaultValue: typeStr || '-' })
          )
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
    [t, formatCurrency, formatDate]
  )

  if (loading) {
    return (
      <Container size="full" className="stock-detail-page">
        <DetailPageHeader
          title={t('detail.title')}
          loading={true}
          showBack={true}
          onBack={handleBack}
          backLabel={t('detail.back')}
        />
      </Container>
    )
  }

  if (!inventoryItem) {
    return (
      <Container size="full" className="stock-detail-page">
        <Card>
          <Empty title={t('detail.notFound')} description={t('detail.notFoundDesc')}>
            <button onClick={handleBack}>{t('detail.backToList')}</button>
          </Empty>
        </Card>
      </Container>
    )
  }

  return (
    <Container size="full" className="stock-detail-page">
      {/* Unified Header */}
      <DetailPageHeader
        title={t('detail.title')}
        documentNumber={productName}
        status={headerStatus}
        metrics={headerMetrics}
        primaryAction={primaryAction}
        secondaryActions={secondaryActions}
        onBack={handleBack}
        backLabel={t('detail.back')}
      />

      {/* Basic Info Card */}
      <Card className="detail-card">
        <Descriptions
          data={[
            { key: t('detail.basicInfo.warehouse'), value: warehouseName },
            { key: t('detail.basicInfo.product'), value: productName },
            {
              key: t('detail.basicInfo.updatedAt'),
              value: formatDate(inventoryItem.updated_at, 'dateTime'),
            },
          ]}
        />
      </Card>

      {/* Cost Info Card */}
      <Card className="detail-card" title={t('detail.cost.title')}>
        <Descriptions
          data={[
            { key: t('detail.cost.unitCost'), value: formatCurrency(inventoryItem.unit_cost) },
            { key: t('detail.cost.totalValue'), value: formatCurrency(inventoryItem.total_value) },
          ]}
        />
      </Card>

      {/* Threshold Info Card */}
      <Card className="detail-card" title={t('detail.threshold.title')}>
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
          </TabPane>
        </Tabs>
      </Card>
    </Container>
  )
}
