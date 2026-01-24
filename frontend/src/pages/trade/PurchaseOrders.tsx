import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Select,
  Space,
  Modal,
  Spin,
  DatePicker,
  Progress,
} from '@douyinfe/semi-ui'
import { IconPlus, IconRefresh } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { getPurchaseOrders } from '@/api/purchase-orders/purchase-orders'
import { getSuppliers } from '@/api/suppliers/suppliers'
import type {
  HandlerPurchaseOrderListResponse,
  GetTradePurchaseOrdersParams,
  GetTradePurchaseOrdersStatus,
  HandlerSupplierListResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './PurchaseOrders.css'

const { Title } = Typography

// Purchase order type with index signature for DataTable compatibility
type PurchaseOrder = HandlerPurchaseOrderListResponse & Record<string, unknown>

// Supplier option type
interface SupplierOption {
  label: string
  value: string
}

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '草稿', value: 'draft' },
  { label: '已确认', value: 'confirmed' },
  { label: '部分收货', value: 'partial_received' },
  { label: '已完成', value: 'completed' },
  { label: '已取消', value: 'cancelled' },
]

// Status tag color mapping
const STATUS_TAG_COLORS: Record<string, 'blue' | 'cyan' | 'green' | 'orange' | 'grey' | 'red'> = {
  draft: 'blue',
  confirmed: 'cyan',
  partial_received: 'orange',
  completed: 'green',
  cancelled: 'grey',
}

// Status labels
const STATUS_LABELS: Record<string, string> = {
  draft: '草稿',
  confirmed: '已确认',
  partial_received: '部分收货',
  completed: '已完成',
  cancelled: '已取消',
}

/**
 * Format price for display
 */
function formatPrice(price?: number): string {
  if (price === undefined || price === null) return '-'
  return `¥${price.toFixed(2)}`
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
  })
}

/**
 * Format datetime for display
 */
function formatDateTime(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/**
 * Purchase orders list page
 *
 * Features:
 * - Order listing with pagination
 * - Search by order number
 * - Filter by status, supplier, date range
 * - Order status actions (confirm, receive, cancel)
 * - Navigate to order detail/edit pages
 */
export default function PurchaseOrdersPage() {
  const navigate = useNavigate()
  const purchaseOrderApi = useMemo(() => getPurchaseOrders(), [])
  const supplierApi = useMemo(() => getSuppliers(), [])

  // State for data
  const [orderList, setOrderList] = useState<PurchaseOrder[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)

  // Supplier options for filter
  const [supplierOptions, setSupplierOptions] = useState<SupplierOption[]>([])
  const [suppliersLoading, setSuppliersLoading] = useState(false)

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [supplierFilter, setSupplierFilter] = useState<string>('')
  const [dateRange, setDateRange] = useState<[Date, Date] | null>(null)

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Fetch suppliers for filter dropdown
  const fetchSuppliers = useCallback(async () => {
    setSuppliersLoading(true)
    try {
      const response = await supplierApi.getPartnerSuppliers({ page_size: 100 })
      if (response.success && response.data) {
        const options: SupplierOption[] = response.data.map(
          (supplier: HandlerSupplierListResponse) => ({
            label: supplier.name || supplier.code || '',
            value: supplier.id || '',
          })
        )
        setSupplierOptions([{ label: '全部供应商', value: '' }, ...options])
      }
    } catch {
      // Silently fail - supplier filter just won't be available
    } finally {
      setSuppliersLoading(false)
    }
  }, [supplierApi])

  // Fetch suppliers on mount
  useEffect(() => {
    fetchSuppliers()
  }, [fetchSuppliers])

  // Fetch purchase orders
  const fetchOrders = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetTradePurchaseOrdersParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as GetTradePurchaseOrdersStatus | undefined,
        supplier_id: supplierFilter || undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      // Add date range filter
      if (dateRange && dateRange[0] && dateRange[1]) {
        params.start_date = dateRange[0].toISOString()
        params.end_date = dateRange[1].toISOString()
      }

      const response = await purchaseOrderApi.getTradePurchaseOrders(params)

      if (response.success && response.data) {
        setOrderList(response.data as PurchaseOrder[])
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
      Toast.error('获取采购订单列表失败')
    } finally {
      setLoading(false)
    }
  }, [
    purchaseOrderApi,
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    searchKeyword,
    statusFilter,
    supplierFilter,
    dateRange,
  ])

  // Fetch on mount and when state changes
  useEffect(() => {
    fetchOrders()
  }, [fetchOrders])

  // Handle search
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
      // Reset to page 1 when searching
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle status filter change
  const handleStatusChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const statusValue = typeof value === 'string' ? value : ''
      setStatusFilter(statusValue)
      setFilter('status', statusValue || null)
    },
    [setFilter]
  )

  // Handle supplier filter change
  const handleSupplierChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const supplierValue = typeof value === 'string' ? value : ''
      setSupplierFilter(supplierValue)
      setFilter('supplier_id', supplierValue || null)
    },
    [setFilter]
  )

  // Handle date range change
  const handleDateRangeChange = useCallback(
    (dates: Date | Date[] | string | string[] | undefined) => {
      if (Array.isArray(dates) && dates.length === 2) {
        const dateValues = dates.map((d) => (typeof d === 'string' ? new Date(d) : d)) as [
          Date,
          Date,
        ]
        setDateRange(dateValues)
      } else {
        setDateRange(null)
      }
      // Reset to page 1 when filter changes
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle confirm order
  const handleConfirm = useCallback(
    async (order: PurchaseOrder) => {
      if (!order.id) return
      Modal.confirm({
        title: '确认订单',
        content: `确定要确认采购订单 "${order.order_number}" 吗？`,
        okText: '确认',
        cancelText: '取消',
        onOk: async () => {
          try {
            await purchaseOrderApi.postTradePurchaseOrdersIdConfirm(order.id!, {})
            Toast.success(`采购订单 "${order.order_number}" 已确认`)
            fetchOrders()
          } catch {
            Toast.error('确认订单失败')
          }
        },
      })
    },
    [purchaseOrderApi, fetchOrders]
  )

  // Handle receive order - navigate to receive page
  const handleReceive = useCallback(
    (order: PurchaseOrder) => {
      if (order.id) {
        navigate(`/trade/purchase/${order.id}/receive`)
      }
    },
    [navigate]
  )

  // Handle cancel order
  const handleCancel = useCallback(
    async (order: PurchaseOrder) => {
      if (!order.id) return
      Modal.confirm({
        title: '取消订单',
        content: `确定要取消采购订单 "${order.order_number}" 吗？`,
        okText: '确认取消',
        cancelText: '返回',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await purchaseOrderApi.postTradePurchaseOrdersIdCancel(order.id!, {
              reason: '用户取消',
            })
            Toast.success(`采购订单 "${order.order_number}" 已取消`)
            fetchOrders()
          } catch {
            Toast.error('取消订单失败')
          }
        },
      })
    },
    [purchaseOrderApi, fetchOrders]
  )

  // Handle delete order
  const handleDelete = useCallback(
    async (order: PurchaseOrder) => {
      if (!order.id) return
      Modal.confirm({
        title: '删除订单',
        content: `确定要删除采购订单 "${order.order_number}" 吗？删除后无法恢复。`,
        okText: '确认删除',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await purchaseOrderApi.deleteTradePurchaseOrdersId(order.id!)
            Toast.success(`采购订单 "${order.order_number}" 已删除`)
            fetchOrders()
          } catch {
            Toast.error('删除订单失败')
          }
        },
      })
    },
    [purchaseOrderApi, fetchOrders]
  )

  // Handle view order
  const handleView = useCallback(
    (order: PurchaseOrder) => {
      if (order.id) {
        navigate(`/trade/purchase/${order.id}`)
      }
    },
    [navigate]
  )

  // Handle edit order
  const handleEdit = useCallback(
    (order: PurchaseOrder) => {
      if (order.id) {
        navigate(`/trade/purchase/${order.id}/edit`)
      }
    },
    [navigate]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchOrders()
  }, [fetchOrders])

  // Table columns
  const tableColumns: DataTableColumn<PurchaseOrder>[] = useMemo(
    () => [
      {
        title: '订单编号',
        dataIndex: 'order_number',
        width: 160,
        sortable: true,
        render: (orderNumber: unknown) => (
          <span className="order-number">{(orderNumber as string) || '-'}</span>
        ),
      },
      {
        title: '供应商',
        dataIndex: 'supplier_name',
        width: 150,
        ellipsis: true,
        render: (name: unknown) => (name as string) || '-',
      },
      {
        title: '商品数量',
        dataIndex: 'item_count',
        width: 100,
        align: 'center',
        render: (count: unknown) => `${(count as number) || 0} 件`,
      },
      {
        title: '订单金额',
        dataIndex: 'total_amount',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => formatPrice(amount as number | undefined),
      },
      {
        title: '应付金额',
        dataIndex: 'payable_amount',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => (
          <span className="payable-amount">{formatPrice(amount as number | undefined)}</span>
        ),
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as string | undefined
          if (!statusValue) return '-'
          return <Tag color={STATUS_TAG_COLORS[statusValue]}>{STATUS_LABELS[statusValue]}</Tag>
        },
      },
      {
        title: '收货进度',
        dataIndex: 'receive_progress',
        width: 120,
        align: 'center',
        render: (progress: unknown, record: PurchaseOrder) => {
          const progressValue = progress as number | undefined
          if (progressValue === undefined || progressValue === null) return '-'
          // Only show progress for confirmed or partial_received orders
          if (
            record.status !== 'confirmed' &&
            record.status !== 'partial_received' &&
            record.status !== 'completed'
          ) {
            return '-'
          }
          return (
            <Progress
              percent={Math.round(progressValue * 100)}
              size="small"
              showInfo
              style={{ width: '80px' }}
            />
          )
        },
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        width: 150,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
      {
        title: '确认时间',
        dataIndex: 'confirmed_at',
        width: 150,
        render: (date: unknown) => formatDateTime(date as string | undefined),
      },
    ],
    []
  )

  // Table row actions
  const tableActions: TableAction<PurchaseOrder>[] = useMemo(
    () => [
      {
        key: 'view',
        label: '查看',
        onClick: handleView,
      },
      {
        key: 'edit',
        label: '编辑',
        onClick: handleEdit,
        hidden: (record) => record.status !== 'draft',
      },
      {
        key: 'confirm',
        label: '确认',
        type: 'primary',
        onClick: handleConfirm,
        hidden: (record) => record.status !== 'draft',
      },
      {
        key: 'receive',
        label: '收货',
        type: 'primary',
        onClick: handleReceive,
        hidden: (record) => record.status !== 'confirmed' && record.status !== 'partial_received',
      },
      {
        key: 'cancel',
        label: '取消',
        type: 'warning',
        onClick: handleCancel,
        hidden: (record) => record.status !== 'draft' && record.status !== 'confirmed',
      },
      {
        key: 'delete',
        label: '删除',
        type: 'danger',
        onClick: handleDelete,
        hidden: (record) => record.status !== 'draft',
      },
    ],
    [handleView, handleEdit, handleConfirm, handleReceive, handleCancel, handleDelete]
  )

  return (
    <Container size="full" className="purchase-orders-page">
      <Card className="purchase-orders-card">
        <div className="purchase-orders-header">
          <Title heading={4} style={{ margin: 0 }}>
            采购订单
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索订单编号..."
          primaryAction={{
            label: '新建订单',
            icon: <IconPlus />,
            onClick: () => navigate('/trade/purchase/new'),
          }}
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
                placeholder="状态筛选"
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="供应商筛选"
                value={supplierFilter}
                onChange={handleSupplierChange}
                optionList={supplierOptions}
                loading={suppliersLoading}
                filter
                style={{ width: 150 }}
              />
              <DatePicker
                type="dateRange"
                placeholder={['开始日期', '结束日期']}
                value={dateRange || undefined}
                onChange={handleDateRangeChange}
                style={{ width: 260 }}
              />
            </Space>
          }
        />

        <Spin spinning={loading}>
          <DataTable<PurchaseOrder>
            data={orderList}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 1400 }}
          />
        </Spin>
      </Card>
    </Container>
  )
}
