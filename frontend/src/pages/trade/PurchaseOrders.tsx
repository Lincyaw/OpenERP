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
  listPurchaseOrders,
  confirmPurchaseOrder,
  cancelPurchaseOrder,
  deletePurchaseOrder,
} from '@/api/purchase-orders/purchase-orders'
import { listSuppliers } from '@/api/suppliers/suppliers'
import type {
  HandlerPurchaseOrderListResponse,
  ListPurchaseOrdersParams,
  ListPurchaseOrdersStatus,
  HandlerSupplierListResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import { PrintPreviewModal } from '@/components/printing'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import './PurchaseOrders.css'

const { Title } = Typography

// Purchase order type with index signature for DataTable compatibility
type PurchaseOrder = HandlerPurchaseOrderListResponse & Record<string, unknown>

// Supplier option type
interface SupplierOption {
  label: string
  value: string
}

// Status tag color mapping
const STATUS_TAG_COLORS: Record<string, 'blue' | 'cyan' | 'green' | 'orange' | 'grey' | 'red'> = {
  draft: 'blue',
  confirmed: 'cyan',
  partial_received: 'orange',
  completed: 'green',
  cancelled: 'grey',
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
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDate, formatDateTime } = useFormatters()

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

  // Print modal state
  const [printModalVisible, setPrintModalVisible] = useState(false)
  const [selectedOrderForPrint, setSelectedOrderForPrint] = useState<PurchaseOrder | null>(null)

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
      { label: t('purchaseOrder.status.draft'), value: 'draft' },
      { label: t('purchaseOrder.status.confirmed'), value: 'confirmed' },
      { label: t('purchaseOrder.status.partialReceived'), value: 'partial_received' },
      { label: t('purchaseOrder.status.completed'), value: 'completed' },
      { label: t('purchaseOrder.status.cancelled'), value: 'cancelled' },
    ],
    [t]
  )

  // Fetch suppliers for filter dropdown
  const fetchSuppliers = useCallback(async () => {
    setSuppliersLoading(true)
    try {
      const response = await listSuppliers({ page_size: 100 })
      if (response.status === 200 && response.data.success && response.data.data) {
        const options: SupplierOption[] = response.data.data.map(
          (supplier: HandlerSupplierListResponse) => ({
            label: supplier.name || supplier.code || '',
            value: supplier.id || '',
          })
        )
        setSupplierOptions([{ label: t('salesOrder.allSuppliers'), value: '' }, ...options])
      }
    } catch {
      // Silently fail - supplier filter just won't be available
    } finally {
      setSuppliersLoading(false)
    }
  }, [t])

  // Fetch suppliers on mount
  useEffect(() => {
    fetchSuppliers()
  }, [fetchSuppliers])

  // Fetch purchase orders
  const fetchOrders = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListPurchaseOrdersParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as ListPurchaseOrdersStatus | undefined,
        supplier_id: supplierFilter || undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      // Add date range filter
      if (dateRange && dateRange[0] && dateRange[1]) {
        params.start_date = dateRange[0].toISOString()
        params.end_date = dateRange[1].toISOString()
      }

      const response = await listPurchaseOrders(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setOrderList(response.data.data as PurchaseOrder[])
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
      Toast.error(t('purchaseOrder.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    t,
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
        title: t('purchaseOrder.modal.confirmTitle'),
        content: t('purchaseOrder.modal.confirmContent', { orderNumber: order.order_number }),
        okText: t('salesOrder.modal.confirmOk'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        onOk: async () => {
          try {
            await confirmPurchaseOrder(order.id!, {})
            Toast.success(
              t('purchaseOrder.messages.confirmSuccess', { orderNumber: order.order_number })
            )
            fetchOrders()
          } catch {
            Toast.error(t('purchaseOrder.messages.confirmError'))
          }
        },
      })
    },
    [fetchOrders, t]
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
        title: t('purchaseOrder.modal.cancelTitle'),
        content: t('purchaseOrder.modal.cancelContent', { orderNumber: order.order_number }),
        okText: t('salesOrder.modal.cancelOk'),
        cancelText: t('salesOrder.modal.backBtn'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await cancelPurchaseOrder(order.id!, {
              reason: t('common.userCancel'),
            })
            Toast.success(
              t('purchaseOrder.messages.cancelSuccess', { orderNumber: order.order_number })
            )
            fetchOrders()
          } catch {
            Toast.error(t('purchaseOrder.messages.cancelError'))
          }
        },
      })
    },
    [fetchOrders, t]
  )

  // Handle delete order
  const handleDelete = useCallback(
    async (order: PurchaseOrder) => {
      if (!order.id) return
      Modal.confirm({
        title: t('purchaseOrder.modal.deleteTitle'),
        content: t('purchaseOrder.modal.deleteContent', { orderNumber: order.order_number }),
        okText: t('salesOrder.modal.deleteOk'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await deletePurchaseOrder(order.id!)
            Toast.success(
              t('purchaseOrder.messages.deleteSuccess', { orderNumber: order.order_number })
            )
            fetchOrders()
          } catch {
            Toast.error(t('purchaseOrder.messages.deleteError'))
          }
        },
      })
    },
    [fetchOrders, t]
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

  // Handle print order
  const handlePrint = useCallback((order: PurchaseOrder) => {
    setSelectedOrderForPrint(order)
    setPrintModalVisible(true)
  }, [])

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchOrders()
  }, [fetchOrders])

  // Table columns
  const tableColumns: DataTableColumn<PurchaseOrder>[] = useMemo(
    () => [
      {
        title: t('purchaseOrder.columns.orderNumber'),
        dataIndex: 'order_number',
        width: 160,
        sortable: true,
        render: (orderNumber: unknown, record: PurchaseOrder) => (
          <span
            className="order-number table-cell-link"
            onClick={() => {
              if (record.id) navigate(`/trade/purchase/${record.id}`)
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault()
                if (record.id) navigate(`/trade/purchase/${record.id}`)
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
        title: t('purchaseOrder.columns.supplier'),
        dataIndex: 'supplier_name',
        width: 150,
        ellipsis: true,
        render: (name: unknown) => (name as string) || '-',
      },
      {
        title: t('purchaseOrder.columns.itemCount'),
        dataIndex: 'item_count',
        width: 100,
        align: 'center',
        render: (count: unknown) => `${(count as number) || 0} ${t('salesOrder.unit')}`,
      },
      {
        title: t('purchaseOrder.columns.totalAmount'),
        dataIndex: 'total_amount',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => {
          const value = amount as number | undefined
          return value !== undefined && value !== null ? formatCurrency(value) : '-'
        },
      },
      {
        title: t('purchaseOrder.columns.payableAmount'),
        dataIndex: 'payable_amount',
        width: 120,
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
        title: t('purchaseOrder.columns.status'),
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as string | undefined
          if (!statusValue) return '-'
          const statusKey = statusValue === 'partial_received' ? 'partialReceived' : statusValue
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>
              {t(`purchaseOrder.status.${statusKey}`)}
            </Tag>
          )
        },
      },
      {
        title: t('purchaseOrder.columns.receiveProgress'),
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
        title: t('purchaseOrder.columns.createdAt'),
        dataIndex: 'created_at',
        width: 150,
        sortable: true,
        render: (date: unknown) => {
          const value = date as string | undefined
          return value ? formatDate(value) : '-'
        },
      },
      {
        title: t('purchaseOrder.columns.confirmedAt'),
        dataIndex: 'confirmed_at',
        width: 150,
        render: (date: unknown) => {
          const value = date as string | undefined
          return value ? formatDateTime(value) : '-'
        },
      },
    ],
    [t, formatCurrency, formatDate, formatDateTime, navigate]
  )

  // Table row actions
  const tableActions: TableAction<PurchaseOrder>[] = useMemo(
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
        key: 'receive',
        label: t('salesOrder.actions.receive'),
        type: 'primary',
        onClick: handleReceive,
        hidden: (record) => record.status !== 'confirmed' && record.status !== 'partial_received',
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
      handleReceive,
      handleCancel,
      handleDelete,
      t,
    ]
  )

  return (
    <Container size="full" className="purchase-orders-page">
      <Card className="purchase-orders-card">
        <div className="purchase-orders-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('purchaseOrder.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('purchaseOrder.searchPlaceholder')}
          primaryAction={{
            label: t('purchaseOrder.newOrder'),
            icon: <IconPlus />,
            onClick: () => navigate('/trade/purchase/new'),
          }}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('purchaseOrder.refresh'),
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
                placeholder={t('salesOrder.supplierFilter')}
                value={supplierFilter}
                onChange={handleSupplierChange}
                optionList={supplierOptions}
                loading={suppliersLoading}
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

      {/* Print Preview Modal */}
      {selectedOrderForPrint && (
        <PrintPreviewModal
          visible={printModalVisible}
          onClose={() => {
            setPrintModalVisible(false)
            setSelectedOrderForPrint(null)
          }}
          documentType="PURCHASE_ORDER"
          documentId={selectedOrderForPrint.id || ''}
          documentNumber={selectedOrderForPrint.order_number || ''}
        />
      )}
    </Container>
  )
}
