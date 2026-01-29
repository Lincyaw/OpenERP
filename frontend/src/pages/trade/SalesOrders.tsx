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
import {
  listSalesOrders,
  confirmSalesOrder,
  shipSalesOrder,
  completeSalesOrder,
  cancelSalesOrder,
  deleteSalesOrder,
} from '@/api/sales-orders/sales-orders'
import { listCustomers } from '@/api/customers/customers'
import type {
  HandlerSalesOrderListResponse,
  ListSalesOrdersParams,
  ListSalesOrdersStatus,
  HandlerCustomerListResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import { ShipOrderModal } from './components'
import { PrintPreviewModal } from '@/components/printing'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import './SalesOrders.css'

const { Title } = Typography

// Sales order type with index signature for DataTable compatibility
type SalesOrder = HandlerSalesOrderListResponse & Record<string, unknown>

// Customer option type
interface CustomerOption {
  label: string
  value: string
}

// Status tag color mapping
const STATUS_TAG_COLORS: Record<string, 'blue' | 'cyan' | 'green' | 'grey' | 'red'> = {
  draft: 'blue',
  confirmed: 'cyan',
  shipped: 'green',
  completed: 'grey',
  cancelled: 'red',
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
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDate } = useFormatters()

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

  // Ship modal state
  const [shipModalVisible, setShipModalVisible] = useState(false)
  const [selectedOrderForShip, setSelectedOrderForShip] = useState<SalesOrder | null>(null)

  // Print modal state
  const [printModalVisible, setPrintModalVisible] = useState(false)
  const [selectedOrderForPrint, setSelectedOrderForPrint] = useState<SalesOrder | null>(null)

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Status options for filter (memoized with translations)
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('salesOrder.allStatus'), value: '' },
      { label: t('salesOrder.status.draft'), value: 'draft' },
      { label: t('salesOrder.status.confirmed'), value: 'confirmed' },
      { label: t('salesOrder.status.shipped'), value: 'shipped' },
      { label: t('salesOrder.status.completed'), value: 'completed' },
      { label: t('salesOrder.status.cancelled'), value: 'cancelled' },
    ],
    [t]
  )

  // Fetch customers for filter dropdown
  const fetchCustomers = useCallback(
    async (signal?: AbortSignal) => {
      setCustomersLoading(true)
      try {
        const response = await listCustomers({ page_size: 100 }, { signal })
        if (response.status === 200 && response.data.success && response.data.data) {
          const options: CustomerOption[] = response.data.data.map(
            (customer: HandlerCustomerListResponse) => ({
              label: customer.name || customer.code || '',
              value: customer.id || '',
            })
          )
          setCustomerOptions([{ label: t('salesOrder.allCustomers'), value: '' }, ...options])
        }
      } catch (error) {
        if (error instanceof Error && error.name === 'CanceledError') return
        // Silently fail - customer filter just won't be available
      } finally {
        setCustomersLoading(false)
      }
    },
    [t]
  )

  // Fetch customers on mount
  useEffect(() => {
    const abortController = new AbortController()
    fetchCustomers(abortController.signal)
    return () => abortController.abort()
  }, [fetchCustomers])

  // Fetch sales orders
  const fetchOrders = useCallback(
    async (signal?: AbortSignal) => {
      setLoading(true)
      try {
        const params: ListSalesOrdersParams = {
          page: state.pagination.page,
          page_size: state.pagination.pageSize,
          search: searchKeyword || undefined,
          status: (statusFilter || undefined) as ListSalesOrdersStatus | undefined,
          customer_id: customerFilter || undefined,
          order_by: state.sort.field || 'created_at',
          order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
        }

        // Add date range filter
        if (dateRange && dateRange[0] && dateRange[1]) {
          params.start_date = dateRange[0].toISOString()
          params.end_date = dateRange[1].toISOString()
        }

        const response = await listSalesOrders(params, { signal })

        if (response.status === 200 && response.data.success && response.data.data) {
          setOrderList(response.data.data as SalesOrder[])
          if (response.data.meta) {
            setPaginationMeta({
              page: response.data.meta.page || 1,
              page_size: response.data.meta.page_size || 20,
              total: response.data.meta.total || 0,
              total_pages: response.data.meta.total_pages || 1,
            })
          }
        }
      } catch (error) {
        if (error instanceof Error && error.name === 'CanceledError') return
        Toast.error(t('salesOrder.messages.fetchError'))
      } finally {
        setLoading(false)
      }
    },
    [
      state.pagination.page,
      state.pagination.pageSize,
      state.sort,
      searchKeyword,
      statusFilter,
      customerFilter,
      dateRange,
      t,
    ]
  )

  // Fetch on mount and when state changes
  useEffect(() => {
    const abortController = new AbortController()
    fetchOrders(abortController.signal)
    return () => abortController.abort()
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
        title: t('salesOrder.modal.confirmTitle'),
        content: t('salesOrder.modal.confirmContent', { orderNumber: order.order_number }),
        okText: t('salesOrder.modal.confirmOk'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        onOk: async () => {
          try {
            await confirmSalesOrder(order.id!, {})
            Toast.success(
              t('salesOrder.messages.confirmSuccess', { orderNumber: order.order_number })
            )
            fetchOrders()
          } catch (error) {
            Toast.error(t('salesOrder.messages.confirmError'))
            throw error // Re-throw to keep modal loading state and prevent double-click
          }
        },
      })
    },
    [fetchOrders, t]
  )

  // Handle ship order - open modal
  const handleShip = useCallback((order: SalesOrder) => {
    if (!order.id) return
    setSelectedOrderForShip(order)
    setShipModalVisible(true)
  }, [])

  // Handle ship confirm from modal
  const handleShipConfirm = useCallback(
    async (warehouseId: string) => {
      if (!selectedOrderForShip?.id) return

      try {
        await shipSalesOrder(selectedOrderForShip.id, {
          warehouse_id: warehouseId,
        })
        Toast.success(
          t('salesOrder.messages.shipSuccess', { orderNumber: selectedOrderForShip.order_number })
        )
        setShipModalVisible(false)
        setSelectedOrderForShip(null)
        fetchOrders()
      } catch {
        Toast.error(t('salesOrder.messages.shipError'))
        throw new Error(t('salesOrder.messages.shipError')) // Re-throw to keep modal open
      }
    },
    [selectedOrderForShip, fetchOrders, t]
  )

  // Handle complete order
  const handleComplete = useCallback(
    async (order: SalesOrder) => {
      if (!order.id) return
      try {
        await completeSalesOrder(order.id, {})
        Toast.success(t('salesOrder.messages.completeSuccess', { orderNumber: order.order_number }))
        fetchOrders()
      } catch {
        Toast.error(t('salesOrder.messages.completeError'))
      }
    },
    [fetchOrders, t]
  )

  // Handle cancel order
  const handleCancel = useCallback(
    async (order: SalesOrder) => {
      if (!order.id) return
      Modal.confirm({
        title: t('salesOrder.modal.cancelTitle'),
        content: t('salesOrder.modal.cancelContent', { orderNumber: order.order_number }),
        okText: t('salesOrder.modal.cancelOk'),
        cancelText: t('salesOrder.modal.backBtn'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await cancelSalesOrder(order.id!, {
              reason: t('common.userCancel'),
            })
            Toast.success(
              t('salesOrder.messages.cancelSuccess', { orderNumber: order.order_number })
            )
            fetchOrders()
          } catch (error) {
            Toast.error(t('salesOrder.messages.cancelError'))
            throw error // Re-throw to keep modal loading state and prevent double-click
          }
        },
      })
    },
    [fetchOrders, t]
  )

  // Handle delete order
  const handleDelete = useCallback(
    async (order: SalesOrder) => {
      if (!order.id) return
      Modal.confirm({
        title: t('salesOrder.modal.deleteTitle'),
        content: t('salesOrder.modal.deleteContent', { orderNumber: order.order_number }),
        okText: t('salesOrder.modal.deleteOk'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await deleteSalesOrder(order.id!)
            Toast.success(
              t('salesOrder.messages.deleteSuccess', { orderNumber: order.order_number })
            )
            fetchOrders()
          } catch (error) {
            Toast.error(t('salesOrder.messages.deleteError'))
            throw error // Re-throw to keep modal loading state and prevent double-click
          }
        },
      })
    },
    [fetchOrders, t]
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

  // Handle print order
  const handlePrint = useCallback((order: SalesOrder) => {
    setSelectedOrderForPrint(order)
    setPrintModalVisible(true)
  }, [])

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchOrders()
  }, [fetchOrders])

  // Table columns - Simplified to 6 essential columns (UX-007)
  // Removed: item_count, total_amount (keeping payable_amount as "amount"), confirmed_at, shipped_at
  // These details can be viewed in the detail page
  const tableColumns: DataTableColumn<SalesOrder>[] = useMemo(
    () => [
      {
        title: t('salesOrder.columns.orderNumber'),
        dataIndex: 'order_number',
        width: 160,
        sortable: true,
        render: (orderNumber: unknown, record: SalesOrder) => (
          <span
            className="order-number table-cell-link"
            onClick={() => {
              if (record.id) navigate(`/trade/sales/${record.id}`)
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                if (record.id) navigate(`/trade/sales/${record.id}`)
              }
            }}
            role="link"
            tabIndex={0}
          >
            {(orderNumber as string) || '-'}
          </span>
        ),
      },
      {
        title: t('salesOrder.columns.customer'),
        dataIndex: 'customer_name',
        width: 180,
        ellipsis: true,
        render: (name: unknown) => (name as string) || '-',
      },
      {
        title: t('salesOrder.columns.amount'),
        dataIndex: 'payable_amount',
        width: 140,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => {
          const value = amount as number | undefined
          return (
            <span className="payable-amount">
              {value !== undefined && value !== null ? formatCurrency(value) : '-'}
            </span>
          )
        },
      },
      {
        title: t('salesOrder.columns.status'),
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as string | undefined
          if (!statusValue) return '-'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>
              {t(`salesOrder.status.${statusValue}`)}
            </Tag>
          )
        },
      },
      {
        title: t('salesOrder.columns.createdAt'),
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
  const tableActions: TableAction<SalesOrder>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('salesOrder.actions.view'),
        onClick: handleView,
      },
      {
        key: 'print',
        label: t('salesOrder.actions.print'),
        onClick: handlePrint,
      },
      {
        key: 'edit',
        label: t('salesOrder.actions.edit'),
        onClick: handleEdit,
        hidden: (record) => record.status !== 'draft',
      },
      {
        key: 'confirm',
        label: t('salesOrder.actions.confirm'),
        type: 'primary',
        onClick: handleConfirm,
        hidden: (record) => record.status !== 'draft',
      },
      {
        key: 'ship',
        label: t('salesOrder.actions.ship'),
        type: 'primary',
        onClick: handleShip,
        hidden: (record) => record.status !== 'confirmed',
      },
      {
        key: 'complete',
        label: t('salesOrder.actions.complete'),
        type: 'primary',
        onClick: handleComplete,
        hidden: (record) => record.status !== 'shipped',
      },
      {
        key: 'cancel',
        label: t('salesOrder.actions.cancel'),
        type: 'warning',
        onClick: handleCancel,
        hidden: (record) => record.status !== 'draft' && record.status !== 'confirmed',
      },
      {
        key: 'delete',
        label: t('salesOrder.actions.delete'),
        type: 'danger',
        onClick: handleDelete,
        hidden: (record) => record.status !== 'draft',
      },
    ],
    [
      handleView,
      handlePrint,
      handleEdit,
      handleConfirm,
      handleShip,
      handleComplete,
      handleCancel,
      handleDelete,
      t,
    ]
  )

  return (
    <Container size="full" className="sales-orders-page">
      <Card className="sales-orders-card">
        <div className="sales-orders-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('salesOrder.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('salesOrder.searchPlaceholder')}
          primaryAction={{
            label: t('salesOrder.newOrder'),
            icon: <IconPlus />,
            onClick: () => navigate('/trade/sales/new'),
          }}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('salesOrder.refresh'),
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
          <DataTable<SalesOrder>
            data={orderList}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 900 }}
          />
        </Spin>
      </Card>

      {/* Ship Order Modal */}
      <ShipOrderModal
        visible={shipModalVisible}
        order={
          selectedOrderForShip
            ? {
                id: selectedOrderForShip.id!,
                order_number: selectedOrderForShip.order_number!,
                customer_name: selectedOrderForShip.customer_name,
                warehouse_id: selectedOrderForShip.warehouse_id,
                item_count: selectedOrderForShip.item_count,
                payable_amount: selectedOrderForShip.payable_amount,
              }
            : null
        }
        onConfirm={handleShipConfirm}
        onCancel={() => {
          setShipModalVisible(false)
          setSelectedOrderForShip(null)
        }}
      />

      {/* Print Preview Modal */}
      {selectedOrderForPrint && (
        <PrintPreviewModal
          visible={printModalVisible}
          onClose={() => {
            setPrintModalVisible(false)
            setSelectedOrderForPrint(null)
          }}
          documentType="SALES_ORDER"
          documentId={selectedOrderForPrint.id || ''}
          documentNumber={selectedOrderForPrint.order_number || ''}
        />
      )}
    </Container>
  )
}
