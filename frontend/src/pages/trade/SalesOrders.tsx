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
import { getSalesOrders } from '@/api/sales-orders/sales-orders'
import { getCustomers } from '@/api/customers/customers'
import type {
  HandlerSalesOrderListResponse,
  GetTradeSalesOrdersParams,
  GetTradeSalesOrdersStatus,
  HandlerCustomerListResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './SalesOrders.css'

const { Title } = Typography

// Sales order type with index signature for DataTable compatibility
type SalesOrder = HandlerSalesOrderListResponse & Record<string, unknown>

// Customer option type
interface CustomerOption {
  label: string
  value: string
}

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '草稿', value: 'draft' },
  { label: '已确认', value: 'confirmed' },
  { label: '已发货', value: 'shipped' },
  { label: '已完成', value: 'completed' },
  { label: '已取消', value: 'cancelled' },
]

// Status tag color mapping
const STATUS_TAG_COLORS: Record<string, 'blue' | 'cyan' | 'green' | 'grey' | 'red'> = {
  draft: 'blue',
  confirmed: 'cyan',
  shipped: 'green',
  completed: 'grey',
  cancelled: 'red',
}

// Status labels
const STATUS_LABELS: Record<string, string> = {
  draft: '草稿',
  confirmed: '已确认',
  shipped: '已发货',
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
 * Sales orders list page
 *
 * Features:
 * - Order listing with pagination
 * - Search by order number
 * - Filter by status, customer, date range
 * - Order status actions (confirm, ship, complete, cancel)
 * - Navigate to order detail/edit pages
 */
export default function SalesOrdersPage() {
  const navigate = useNavigate()
  const salesOrderApi = useMemo(() => getSalesOrders(), [])
  const customerApi = useMemo(() => getCustomers(), [])

  // State for data
  const [orderList, setOrderList] = useState<SalesOrder[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)

  // Customer options for filter
  const [customerOptions, setCustomerOptions] = useState<CustomerOption[]>([])
  const [customersLoading, setCustomersLoading] = useState(false)

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [customerFilter, setCustomerFilter] = useState<string>('')
  const [dateRange, setDateRange] = useState<[Date, Date] | null>(null)

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Fetch customers for filter dropdown
  const fetchCustomers = useCallback(async () => {
    setCustomersLoading(true)
    try {
      const response = await customerApi.getPartnerCustomers({ page_size: 100 })
      if (response.success && response.data) {
        const options: CustomerOption[] = response.data.map(
          (customer: HandlerCustomerListResponse) => ({
            label: customer.name || customer.code || '',
            value: customer.id || '',
          })
        )
        setCustomerOptions([{ label: '全部客户', value: '' }, ...options])
      }
    } catch {
      // Silently fail - customer filter just won't be available
    } finally {
      setCustomersLoading(false)
    }
  }, [customerApi])

  // Fetch customers on mount
  useEffect(() => {
    fetchCustomers()
  }, [fetchCustomers])

  // Fetch sales orders
  const fetchOrders = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetTradeSalesOrdersParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as GetTradeSalesOrdersStatus | undefined,
        customer_id: customerFilter || undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      // Add date range filter
      if (dateRange && dateRange[0] && dateRange[1]) {
        params.start_date = dateRange[0].toISOString()
        params.end_date = dateRange[1].toISOString()
      }

      const response = await salesOrderApi.getTradeSalesOrders(params)

      if (response.success && response.data) {
        setOrderList(response.data as SalesOrder[])
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
      Toast.error('获取销售订单列表失败')
    } finally {
      setLoading(false)
    }
  }, [
    salesOrderApi,
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    searchKeyword,
    statusFilter,
    customerFilter,
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

  // Handle customer filter change
  const handleCustomerChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const customerValue = typeof value === 'string' ? value : ''
      setCustomerFilter(customerValue)
      setFilter('customer_id', customerValue || null)
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
    async (order: SalesOrder) => {
      if (!order.id) return
      Modal.confirm({
        title: '确认订单',
        content: `确定要确认订单 "${order.order_number}" 吗？确认后将锁定库存。`,
        okText: '确认',
        cancelText: '取消',
        onOk: async () => {
          try {
            await salesOrderApi.postTradeSalesOrdersIdConfirm(order.id!, {})
            Toast.success(`订单 "${order.order_number}" 已确认`)
            fetchOrders()
          } catch {
            Toast.error('确认订单失败')
          }
        },
      })
    },
    [salesOrderApi, fetchOrders]
  )

  // Handle ship order
  const handleShip = useCallback(
    async (order: SalesOrder) => {
      if (!order.id) return
      Modal.confirm({
        title: '发货',
        content: `确定要为订单 "${order.order_number}" 发货吗？发货后将扣减库存。`,
        okText: '确认发货',
        cancelText: '取消',
        onOk: async () => {
          try {
            await salesOrderApi.postTradeSalesOrdersIdShip(order.id!, {
              warehouse_id: order.warehouse_id,
            })
            Toast.success(`订单 "${order.order_number}" 已发货`)
            fetchOrders()
          } catch {
            Toast.error('发货失败')
          }
        },
      })
    },
    [salesOrderApi, fetchOrders]
  )

  // Handle complete order
  const handleComplete = useCallback(
    async (order: SalesOrder) => {
      if (!order.id) return
      try {
        await salesOrderApi.postTradeSalesOrdersIdComplete(order.id)
        Toast.success(`订单 "${order.order_number}" 已完成`)
        fetchOrders()
      } catch {
        Toast.error('完成订单失败')
      }
    },
    [salesOrderApi, fetchOrders]
  )

  // Handle cancel order
  const handleCancel = useCallback(
    async (order: SalesOrder) => {
      if (!order.id) return
      Modal.confirm({
        title: '取消订单',
        content: `确定要取消订单 "${order.order_number}" 吗？`,
        okText: '确认取消',
        cancelText: '返回',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await salesOrderApi.postTradeSalesOrdersIdCancel(order.id!, { reason: '用户取消' })
            Toast.success(`订单 "${order.order_number}" 已取消`)
            fetchOrders()
          } catch {
            Toast.error('取消订单失败')
          }
        },
      })
    },
    [salesOrderApi, fetchOrders]
  )

  // Handle delete order
  const handleDelete = useCallback(
    async (order: SalesOrder) => {
      if (!order.id) return
      Modal.confirm({
        title: '删除订单',
        content: `确定要删除订单 "${order.order_number}" 吗？删除后无法恢复。`,
        okText: '确认删除',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await salesOrderApi.deleteTradeSalesOrdersId(order.id!)
            Toast.success(`订单 "${order.order_number}" 已删除`)
            fetchOrders()
          } catch {
            Toast.error('删除订单失败')
          }
        },
      })
    },
    [salesOrderApi, fetchOrders]
  )

  // Handle view order
  const handleView = useCallback(
    (order: SalesOrder) => {
      if (order.id) {
        navigate(`/trade/sales/${order.id}`)
      }
    },
    [navigate]
  )

  // Handle edit order
  const handleEdit = useCallback(
    (order: SalesOrder) => {
      if (order.id) {
        navigate(`/trade/sales/${order.id}/edit`)
      }
    },
    [navigate]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchOrders()
  }, [fetchOrders])

  // Table columns
  const tableColumns: DataTableColumn<SalesOrder>[] = useMemo(
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
        title: '客户',
        dataIndex: 'customer_name',
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
      {
        title: '发货时间',
        dataIndex: 'shipped_at',
        width: 150,
        render: (date: unknown) => formatDateTime(date as string | undefined),
      },
    ],
    []
  )

  // Table row actions
  const tableActions: TableAction<SalesOrder>[] = useMemo(
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
        key: 'ship',
        label: '发货',
        type: 'primary',
        onClick: handleShip,
        hidden: (record) => record.status !== 'confirmed',
      },
      {
        key: 'complete',
        label: '完成',
        type: 'primary',
        onClick: handleComplete,
        hidden: (record) => record.status !== 'shipped',
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
    [handleView, handleEdit, handleConfirm, handleShip, handleComplete, handleCancel, handleDelete]
  )

  return (
    <Container size="full" className="sales-orders-page">
      <Card className="sales-orders-card">
        <div className="sales-orders-header">
          <Title heading={4} style={{ margin: 0 }}>
            销售订单
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索订单编号..."
          primaryAction={{
            label: '新建订单',
            icon: <IconPlus />,
            onClick: () => navigate('/trade/sales/new'),
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
                placeholder="客户筛选"
                value={customerFilter}
                onChange={handleCustomerChange}
                optionList={customerOptions}
                loading={customersLoading}
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
          <DataTable<SalesOrder>
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
