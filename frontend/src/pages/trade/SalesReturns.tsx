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
} from '@douyinfe/semi-ui-19'
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
import {
  listSalesReturns,
  deleteSalesReturn,
  submitSalesReturn,
  approveSalesReturn,
  rejectSalesReturn,
  completeSalesReturn,
  receiveSalesReturn,
  cancelSalesReturn,
} from '@/api/sales-returns/sales-returns'
import { listCustomers } from '@/api/customers/customers'
import type {
  HandlerSalesReturnListResponse,
  ListSalesReturnsParams,
  ListSalesReturnsStatus,
  HandlerCustomerListResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import './SalesReturns.css'

const { Title } = Typography

// Sales return type with index signature for DataTable compatibility
type SalesReturn = HandlerSalesReturnListResponse & Record<string, unknown>

// Customer option type
interface CustomerOption {
  label: string
  value: string
}

// Status tag color mapping
const STATUS_TAG_COLORS: Record<
  string,
  'blue' | 'cyan' | 'green' | 'grey' | 'red' | 'orange' | 'amber' | 'light-blue'
> = {
  DRAFT: 'blue',
  PENDING: 'orange',
  APPROVED: 'cyan',
  RECEIVING: 'light-blue',
  REJECTED: 'red',
  COMPLETED: 'green',
  CANCELLED: 'grey',
}

// Status key mapping for i18n
const STATUS_KEYS: Record<string, string> = {
  DRAFT: 'draft',
  PENDING: 'pendingApproval',
  APPROVED: 'approved',
  RECEIVING: 'receiving',
  REJECTED: 'rejected',
  COMPLETED: 'completed',
  CANCELLED: 'cancelled',
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
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDate } = useFormatters()

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

  // Status options for filter (memoized with translations)
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('salesReturn.status.all'), value: '' },
      { label: t('salesReturn.status.draft'), value: 'DRAFT' },
      { label: t('salesReturn.status.pendingApproval'), value: 'PENDING' },
      { label: t('salesReturn.status.approved'), value: 'APPROVED' },
      { label: t('salesReturn.status.receiving'), value: 'RECEIVING' },
      { label: t('salesReturn.status.rejected'), value: 'REJECTED' },
      { label: t('salesReturn.status.completed'), value: 'COMPLETED' },
      { label: t('salesReturn.status.cancelled'), value: 'CANCELLED' },
    ],
    [t]
  )

  // Fetch customers for filter dropdown
  const fetchCustomers = useCallback(async () => {
    setCustomersLoading(true)
    try {
      const response = await listCustomers({ page_size: 100 })
      if (response.status === 200 && response.data.success && response.data.data) {
        const options: CustomerOption[] = response.data.data.map(
          (customer: HandlerCustomerListResponse) => ({
            label: customer.name || customer.code || '',
            value: customer.id || '',
          })
        )
        setCustomerOptions([{ label: t('salesOrder.allCustomers'), value: '' }, ...options])
      }
    } catch {
      // Silently fail - customer filter just won't be available
    } finally {
      setCustomersLoading(false)
    }
  }, [t])

  // Fetch customers on mount
  useEffect(() => {
    fetchCustomers()
  }, [fetchCustomers])

  // Fetch sales returns
  const fetchReturns = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListSalesReturnsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as ListSalesReturnsStatus | undefined,
        customer_id: customerFilter || undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      // Add date range filter
      if (dateRange && dateRange[0] && dateRange[1]) {
        params.start_date = dateRange[0].toISOString()
        params.end_date = dateRange[1].toISOString()
      }

      const response = await listSalesReturns(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setReturnList(response.data.data as SalesReturn[])
        if (response.data.meta) {
          setPaginationMeta({
            page: response.data.meta.page || 1,
            page_size: response.data.meta.page_size || 20,
            total: response.data.meta.total || 0,
            total_pages: response.data.meta.total_pages || 1,
          })
        }
      }
    } catch {
      Toast.error(t('salesReturn.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    searchKeyword,
    statusFilter,
    customerFilter,
    dateRange,
    t,
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
        title: t('salesReturn.modal.submitTitle'),
        content: t('salesReturn.modal.submitContent', { returnNumber: returnItem.return_number }),
        okText: t('salesOrder.modal.confirmOk'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        onOk: async () => {
          try {
            await submitSalesReturn(returnItem.id!, {})
            Toast.success(t('salesReturn.messages.submitSuccess'))
            fetchReturns()
          } catch {
            Toast.error(t('salesReturn.messages.submitError'))
          }
        },
      })
    },
    [fetchReturns, t]
  )

  // Handle approve return
  const handleApprove = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: t('salesReturn.modal.approveTitle'),
        content: t('salesReturn.modal.approveContent', { returnNumber: returnItem.return_number }),
        okText: t('salesReturn.actions.approve'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        onOk: async () => {
          try {
            await approveSalesReturn(returnItem.id!, { note: '' })
            Toast.success(t('salesReturn.messages.approveSuccess'))
            fetchReturns()
          } catch {
            Toast.error(t('salesReturn.messages.approveError'))
          }
        },
      })
    },
    [fetchReturns, t]
  )

  // Handle reject return
  const handleReject = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: t('salesReturn.modal.rejectTitle'),
        content: t('salesReturn.modal.rejectContent', { returnNumber: returnItem.return_number }),
        okText: t('salesReturn.actions.reject'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await rejectSalesReturn(returnItem.id!, {
              reason: t('salesReturn.actions.reject'),
            })
            Toast.success(t('salesReturn.messages.rejectSuccess'))
            fetchReturns()
          } catch {
            Toast.error(t('salesReturn.messages.rejectError'))
          }
        },
      })
    },
    [fetchReturns, t]
  )

  // Handle complete return
  const handleComplete = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      try {
        await completeSalesReturn(returnItem.id!, {})
        Toast.success(t('salesReturn.messages.completeSuccess'))
        fetchReturns()
      } catch {
        Toast.error(t('salesReturn.messages.completeError'))
      }
    },
    [fetchReturns, t]
  )

  // Handle receive returned goods (start receiving)
  const handleReceive = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      try {
        await receiveSalesReturn(returnItem.id!, {})
        Toast.success(t('salesReturn.messages.receiveSuccess'))
        fetchReturns()
      } catch {
        Toast.error(t('salesReturn.messages.receiveError'))
      }
    },
    [fetchReturns, t]
  )

  // Handle cancel return
  const handleCancel = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: t('salesReturn.modal.cancelTitle'),
        content: t('salesReturn.modal.cancelContent', { returnNumber: returnItem.return_number }),
        okText: t('salesOrder.modal.cancelOk'),
        cancelText: t('salesOrder.modal.backBtn'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await cancelSalesReturn(returnItem.id!, {
              reason: t('common.userCancel'),
            })
            Toast.success(t('salesReturn.messages.cancelSuccess'))
            fetchReturns()
          } catch {
            Toast.error(t('salesReturn.messages.cancelError'))
          }
        },
      })
    },
    [fetchReturns, t]
  )

  // Handle delete return
  const handleDelete = useCallback(
    async (returnItem: SalesReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: t('salesReturn.modal.deleteTitle'),
        content: t('salesReturn.modal.deleteContent', { returnNumber: returnItem.return_number }),
        okText: t('salesOrder.modal.deleteOk'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await deleteSalesReturn(returnItem.id!)
            Toast.success(t('salesReturn.messages.deleteSuccess'))
            fetchReturns()
          } catch {
            Toast.error(t('salesReturn.messages.deleteError'))
          }
        },
      })
    },
    [fetchReturns, t]
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
        title: t('salesReturn.columns.returnNumber'),
        dataIndex: 'return_number',
        width: 150,
        sortable: true,
        render: (returnNumber: unknown, record: SalesReturn) => (
          <span
            className="return-number table-cell-link"
            onClick={() => {
              if (record.id) navigate(`/trade/sales-returns/${record.id}`)
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                if (record.id) navigate(`/trade/sales-returns/${record.id}`)
              }
            }}
            role="link"
            tabIndex={0}
          >
            {(returnNumber as string) || '-'}
          </span>
        ),
      },
      {
        title: t('salesReturn.columns.orderNumber'),
        dataIndex: 'sales_order_number',
        width: 150,
        render: (orderNumber: unknown) => (
          <span className="order-number">{(orderNumber as string) || '-'}</span>
        ),
      },
      {
        title: t('salesReturn.columns.customer'),
        dataIndex: 'customer_name',
        width: 150,
        ellipsis: true,
        render: (name: unknown) => (name as string) || '-',
      },
      {
        title: t('salesReturn.columns.itemCount'),
        dataIndex: 'item_count',
        width: 100,
        align: 'center',
        render: (count: unknown) => `${(count as number) || 0} ${t('salesOrder.unit')}`,
      },
      {
        title: t('salesReturn.columns.totalAmount'),
        dataIndex: 'total_refund',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => {
          const value = amount as number | undefined
          return (
            <span className="refund-amount">
              {value !== undefined && value !== null ? formatCurrency(value) : '-'}
            </span>
          )
        },
      },
      {
        title: t('salesReturn.columns.status'),
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as string | undefined
          if (!statusValue) return '-'
          const statusKey = STATUS_KEYS[statusValue] || 'draft'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>{t(`salesReturn.status.${statusKey}`)}</Tag>
          )
        },
      },
      {
        title: t('salesReturn.columns.createdAt'),
        dataIndex: 'created_at',
        width: 150,
        sortable: true,
        render: (date: unknown) => {
          const value = date as string | undefined
          return value ? formatDate(value) : '-'
        },
      },
    ],
    [t, formatCurrency, formatDate, navigate]
  )

  // Table row actions
  const tableActions: TableAction<SalesReturn>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('salesReturn.actions.view'),
        onClick: handleView,
      },
      {
        key: 'submit',
        label: t('salesReturn.actions.submit'),
        type: 'primary',
        onClick: handleSubmit,
        hidden: (record) => record.status !== 'DRAFT',
      },
      {
        key: 'approve',
        label: t('salesReturn.actions.approve'),
        type: 'primary',
        onClick: handleApprove,
        hidden: (record) => record.status !== 'PENDING',
      },
      {
        key: 'reject',
        label: t('salesReturn.actions.reject'),
        type: 'warning',
        onClick: handleReject,
        hidden: (record) => record.status !== 'PENDING',
      },
      {
        key: 'receive',
        label: t('salesReturn.actions.receive'),
        type: 'primary',
        onClick: handleReceive,
        hidden: (record) => record.status !== 'APPROVED',
      },
      {
        key: 'complete',
        label: t('salesReturn.actions.complete'),
        type: 'primary',
        onClick: handleComplete,
        hidden: (record) => record.status !== 'RECEIVING',
      },
      {
        key: 'cancel',
        label: t('salesReturn.actions.cancel'),
        type: 'warning',
        onClick: handleCancel,
        hidden: (record) =>
          record.status !== 'DRAFT' &&
          record.status !== 'PENDING' &&
          record.status !== 'APPROVED' &&
          record.status !== 'RECEIVING',
      },
      {
        key: 'delete',
        label: t('salesReturn.actions.delete'),
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
      handleReceive,
      handleComplete,
      handleCancel,
      handleDelete,
      t,
    ]
  )

  return (
    <Container size="full" className="sales-returns-page">
      <Card className="sales-returns-card">
        <div className="sales-returns-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('salesReturn.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('salesReturn.searchPlaceholder')}
          primaryAction={{
            label: t('salesReturn.newReturn'),
            icon: <IconPlus />,
            onClick: () => navigate('/trade/sales-returns/new'),
          }}
          secondaryActions={[
            {
              key: 'approval',
              label: t('salesReturn.actions.approve'),
              icon: <IconTickCircle />,
              onClick: () => navigate('/trade/sales-returns/approval'),
            },
            {
              key: 'refresh',
              label: t('salesReturn.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space>
              <Select
                placeholder={t('salesOrder.statusFilter')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('salesOrder.customerFilter')}
                value={customerFilter}
                onChange={handleCustomerChange}
                optionList={customerOptions}
                loading={customersLoading}
                filter
                style={{ width: 150 }}
              />
              <DatePicker
                type="dateRange"
                placeholder={[t('salesOrder.startDate'), t('salesOrder.endDate')]}
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
