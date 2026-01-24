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
import { IconPlus, IconRefresh, IconTickCircle } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { getSalesReturns } from '@/api/sales-returns/sales-returns'
import { getCustomers } from '@/api/customers/customers'
import type {
  HandlerSalesReturnListResponse,
  GetTradeSalesReturnsParams,
  GetTradeSalesReturnsStatus,
  HandlerCustomerListResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './SalesReturns.css'

const { Title } = Typography

// Sales return type with index signature for DataTable compatibility
type SalesReturn = HandlerSalesReturnListResponse & Record<string, unknown>

// Customer option type
interface CustomerOption {
  label: string
  value: string
}

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '草稿', value: 'DRAFT' },
  { label: '待审批', value: 'PENDING' },
  { label: '已审批', value: 'APPROVED' },
  { label: '已拒绝', value: 'REJECTED' },
  { label: '已完成', value: 'COMPLETED' },
  { label: '已取消', value: 'CANCELLED' },
]

// Status tag color mapping
const STATUS_TAG_COLORS: Record<
  string,
  'blue' | 'cyan' | 'green' | 'grey' | 'red' | 'orange' | 'amber'
> = {
  DRAFT: 'blue',
  PENDING: 'orange',
  APPROVED: 'cyan',
  REJECTED: 'red',
  COMPLETED: 'green',
  CANCELLED: 'grey',
}

// Status labels
const STATUS_LABELS: Record<string, string> = {
  DRAFT: '草稿',
  PENDING: '待审批',
  APPROVED: '已审批',
  REJECTED: '已拒绝',
  COMPLETED: '已完成',
  CANCELLED: '已取消',
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
 * Sales returns list page
 *
 * Features:
 * - Return listing with pagination
 * - Search by return number
 * - Filter by status, customer, date range
 * - Return status actions (approve, reject, complete, cancel)
 * - Navigate to return detail/create pages
 */
export default function SalesReturnsPage() {
  const navigate = useNavigate()
  const salesReturnApi = useMemo(() => getSalesReturns(), [])
  const customerApi = useMemo(() => getCustomers(), [])

  // State for data
  const [returnList, setReturnList] = useState<SalesReturn[]>([])
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

  // Fetch sales returns
  const fetchReturns = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetTradeSalesReturnsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as GetTradeSalesReturnsStatus | undefined,
        customer_id: customerFilter || undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      // Add date range filter
      if (dateRange && dateRange[0] && dateRange[1]) {
        params.start_date = dateRange[0].toISOString()
        params.end_date = dateRange[1].toISOString()
      }

      const response = await salesReturnApi.getTradeSalesReturns(params)

      if (response.success && response.data) {
        setReturnList(response.data as SalesReturn[])
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
      Toast.error('获取销售退货列表失败')
    } finally {
      setLoading(false)
    }
  }, [
    salesReturnApi,
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
    fetchReturns()
  }, [fetchReturns])

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

  // Handle submit return for approval
  const handleSubmit = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: '提交审批',
        content: `确定要提交退货单 "${returnItem.return_number}" 进行审批吗？`,
        okText: '确认',
        cancelText: '取消',
        onOk: async () => {
          try {
            await salesReturnApi.postTradeSalesReturnsIdSubmit(returnItem.id!)
            Toast.success(`退货单 "${returnItem.return_number}" 已提交审批`)
            fetchReturns()
          } catch {
            Toast.error('提交审批失败')
          }
        },
      })
    },
    [salesReturnApi, fetchReturns]
  )

  // Handle approve return
  const handleApprove = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: '审批通过',
        content: `确定要通过退货单 "${returnItem.return_number}" 的审批吗？`,
        okText: '通过',
        cancelText: '取消',
        onOk: async () => {
          try {
            await salesReturnApi.postTradeSalesReturnsIdApprove(returnItem.id!, { note: '' })
            Toast.success(`退货单 "${returnItem.return_number}" 已审批通过`)
            fetchReturns()
          } catch {
            Toast.error('审批失败')
          }
        },
      })
    },
    [salesReturnApi, fetchReturns]
  )

  // Handle reject return
  const handleReject = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: '拒绝退货',
        content: `确定要拒绝退货单 "${returnItem.return_number}" 吗？`,
        okText: '拒绝',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await salesReturnApi.postTradeSalesReturnsIdReject(returnItem.id!, {
              reason: '审批拒绝',
            })
            Toast.success(`退货单 "${returnItem.return_number}" 已拒绝`)
            fetchReturns()
          } catch {
            Toast.error('拒绝失败')
          }
        },
      })
    },
    [salesReturnApi, fetchReturns]
  )

  // Handle complete return
  const handleComplete = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      try {
        await salesReturnApi.postTradeSalesReturnsIdComplete(returnItem.id!, {})
        Toast.success(`退货单 "${returnItem.return_number}" 已完成`)
        fetchReturns()
      } catch {
        Toast.error('完成退货失败')
      }
    },
    [salesReturnApi, fetchReturns]
  )

  // Handle cancel return
  const handleCancel = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: '取消退货',
        content: `确定要取消退货单 "${returnItem.return_number}" 吗？`,
        okText: '确认取消',
        cancelText: '返回',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await salesReturnApi.postTradeSalesReturnsIdCancel(returnItem.id!, {
              reason: '用户取消',
            })
            Toast.success(`退货单 "${returnItem.return_number}" 已取消`)
            fetchReturns()
          } catch {
            Toast.error('取消退货失败')
          }
        },
      })
    },
    [salesReturnApi, fetchReturns]
  )

  // Handle delete return
  const handleDelete = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: '删除退货单',
        content: `确定要删除退货单 "${returnItem.return_number}" 吗？删除后无法恢复。`,
        okText: '确认删除',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await salesReturnApi.deleteTradeSalesReturnsId(returnItem.id!)
            Toast.success(`退货单 "${returnItem.return_number}" 已删除`)
            fetchReturns()
          } catch {
            Toast.error('删除退货单失败')
          }
        },
      })
    },
    [salesReturnApi, fetchReturns]
  )

  // Handle view return
  const handleView = useCallback(
    (returnItem: SalesReturn) => {
      if (returnItem.id) {
        navigate(`/trade/sales-returns/${returnItem.id}`)
      }
    },
    [navigate]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchReturns()
  }, [fetchReturns])

  // Table columns
  const tableColumns: DataTableColumn<SalesReturn>[] = useMemo(
    () => [
      {
        title: '退货单号',
        dataIndex: 'return_number',
        width: 150,
        sortable: true,
        render: (returnNumber: unknown) => (
          <span className="return-number">{(returnNumber as string) || '-'}</span>
        ),
      },
      {
        title: '原订单号',
        dataIndex: 'sales_order_number',
        width: 150,
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
        title: '退款金额',
        dataIndex: 'total_refund',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => (
          <span className="refund-amount">{formatPrice(amount as number | undefined)}</span>
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
        title: '提交时间',
        dataIndex: 'submitted_at',
        width: 150,
        render: (date: unknown) => formatDateTime(date as string | undefined),
      },
      {
        title: '完成时间',
        dataIndex: 'completed_at',
        width: 150,
        render: (date: unknown) => formatDateTime(date as string | undefined),
      },
    ],
    []
  )

  // Table row actions
  const tableActions: TableAction<SalesReturn>[] = useMemo(
    () => [
      {
        key: 'view',
        label: '查看',
        onClick: handleView,
      },
      {
        key: 'submit',
        label: '提交审批',
        type: 'primary',
        onClick: handleSubmit,
        hidden: (record) => record.status !== 'DRAFT',
      },
      {
        key: 'approve',
        label: '通过',
        type: 'primary',
        onClick: handleApprove,
        hidden: (record) => record.status !== 'PENDING',
      },
      {
        key: 'reject',
        label: '拒绝',
        type: 'warning',
        onClick: handleReject,
        hidden: (record) => record.status !== 'PENDING',
      },
      {
        key: 'complete',
        label: '完成',
        type: 'primary',
        onClick: handleComplete,
        hidden: (record) => record.status !== 'APPROVED',
      },
      {
        key: 'cancel',
        label: '取消',
        type: 'warning',
        onClick: handleCancel,
        hidden: (record) =>
          record.status !== 'DRAFT' && record.status !== 'PENDING' && record.status !== 'APPROVED',
      },
      {
        key: 'delete',
        label: '删除',
        type: 'danger',
        onClick: handleDelete,
        hidden: (record) => record.status !== 'DRAFT',
      },
    ],
    [
      handleView,
      handleSubmit,
      handleApprove,
      handleReject,
      handleComplete,
      handleCancel,
      handleDelete,
    ]
  )

  return (
    <Container size="full" className="sales-returns-page">
      <Card className="sales-returns-card">
        <div className="sales-returns-header">
          <Title heading={4} style={{ margin: 0 }}>
            销售退货
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索退货单号..."
          primaryAction={{
            label: '新建退货',
            icon: <IconPlus />,
            onClick: () => navigate('/trade/sales-returns/new'),
          }}
          secondaryActions={[
            {
              key: 'approval',
              label: '审批',
              icon: <IconTickCircle />,
              onClick: () => navigate('/trade/sales-returns/approval'),
            },
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
          <DataTable<SalesReturn>
            data={returnList}
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
