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
import { getPurchaseReturns } from '@/api/purchase-returns/purchase-returns'
import { getSuppliers } from '@/api/suppliers/suppliers'
import type {
  HandlerPurchaseReturnListResponse,
  GetTradePurchaseReturnsParams,
  GetTradePurchaseReturnsStatus,
  HandlerSupplierListResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import { useI18n } from '@/hooks/useI18n'
import { useFormatters } from '@/hooks/useFormatters'
import './PurchaseReturns.css'

const { Title } = Typography

// Purchase return type with index signature for DataTable compatibility
type PurchaseReturn = HandlerPurchaseReturnListResponse & Record<string, unknown>

// Supplier option type
interface SupplierOption {
  label: string
  value: string
}

// Status tag color mapping
const STATUS_TAG_COLORS: Record<
  string,
  'blue' | 'cyan' | 'green' | 'grey' | 'red' | 'orange' | 'amber' | 'violet'
> = {
  DRAFT: 'blue',
  PENDING: 'orange',
  APPROVED: 'cyan',
  REJECTED: 'red',
  SHIPPED: 'violet',
  COMPLETED: 'green',
  CANCELLED: 'grey',
}

// Status key mapping for i18n
const STATUS_KEYS: Record<string, string> = {
  DRAFT: 'draft',
  PENDING: 'pendingApproval',
  APPROVED: 'approved',
  REJECTED: 'rejected',
  SHIPPED: 'shipped',
  COMPLETED: 'completed',
  CANCELLED: 'cancelled',
}

/**
 * Purchase returns list page
 *
 * Features:
 * - Return listing with pagination
 * - Search by return number
 * - Filter by status, supplier, date range
 * - Return status actions (approve, reject, ship, complete, cancel)
 * - Navigate to return detail/create pages
 */
export default function PurchaseReturnsPage() {
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  const { formatCurrency, formatDate, formatDateTime } = useFormatters()
  const purchaseReturnApi = useMemo(() => getPurchaseReturns(), [])
  const supplierApi = useMemo(() => getSuppliers(), [])

  // State for data
  const [returnList, setReturnList] = useState<PurchaseReturn[]>([])
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

  // Status options for filter (memoized with translations)
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('purchaseReturn.status.all'), value: '' },
      { label: t('purchaseReturn.status.draft'), value: 'DRAFT' },
      { label: t('purchaseReturn.status.pendingApproval'), value: 'PENDING' },
      { label: t('purchaseReturn.status.approved'), value: 'APPROVED' },
      { label: t('purchaseReturn.status.rejected'), value: 'REJECTED' },
      { label: t('purchaseReturn.status.shipped'), value: 'SHIPPED' },
      { label: t('purchaseReturn.status.completed'), value: 'COMPLETED' },
      { label: t('purchaseReturn.status.cancelled'), value: 'CANCELLED' },
    ],
    [t]
  )

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
        setSupplierOptions([{ label: t('salesOrder.allSuppliers'), value: '' }, ...options])
      }
    } catch {
      // Silently fail - supplier filter just won't be available
    } finally {
      setSuppliersLoading(false)
    }
  }, [supplierApi, t])

  // Fetch suppliers on mount
  useEffect(() => {
    fetchSuppliers()
  }, [fetchSuppliers])

  // Fetch purchase returns
  const fetchReturns = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetTradePurchaseReturnsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as GetTradePurchaseReturnsStatus | undefined,
        supplier_id: supplierFilter || undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      // Add date range filter
      if (dateRange && dateRange[0] && dateRange[1]) {
        params.start_date = dateRange[0].toISOString()
        params.end_date = dateRange[1].toISOString()
      }

      const response = await purchaseReturnApi.getTradePurchaseReturns(params)

      if (response.success && response.data) {
        setReturnList(response.data as PurchaseReturn[])
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
      Toast.error(t('purchaseReturn.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    purchaseReturnApi,
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    searchKeyword,
    statusFilter,
    supplierFilter,
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

  // Handle submit return for approval
  const handleSubmit = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: t('purchaseReturn.modal.submitTitle'),
        content: t('purchaseReturn.modal.submitContent', {
          returnNumber: returnItem.return_number,
        }),
        okText: t('salesOrder.modal.confirmOk'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        onOk: async () => {
          try {
            await purchaseReturnApi.postTradePurchaseReturnsIdSubmit(returnItem.id!)
            Toast.success(t('purchaseReturn.messages.submitSuccess'))
            fetchReturns()
          } catch {
            Toast.error(t('purchaseReturn.messages.submitError'))
          }
        },
      })
    },
    [purchaseReturnApi, fetchReturns, t]
  )

  // Handle approve return
  const handleApprove = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: t('purchaseReturn.modal.approveTitle'),
        content: t('purchaseReturn.modal.approveContent', {
          returnNumber: returnItem.return_number,
        }),
        okText: t('purchaseReturn.actions.approve'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        onOk: async () => {
          try {
            await purchaseReturnApi.postTradePurchaseReturnsIdApprove(returnItem.id!, { note: '' })
            Toast.success(t('purchaseReturn.messages.approveSuccess'))
            fetchReturns()
          } catch {
            Toast.error(t('purchaseReturn.messages.approveError'))
          }
        },
      })
    },
    [purchaseReturnApi, fetchReturns, t]
  )

  // Handle reject return
  const handleReject = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: t('purchaseReturn.modal.rejectTitle'),
        content: t('purchaseReturn.modal.rejectContent', {
          returnNumber: returnItem.return_number,
        }),
        okText: t('purchaseReturn.actions.reject'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await purchaseReturnApi.postTradePurchaseReturnsIdReject(returnItem.id!, {
              reason: t('purchaseReturn.actions.reject'),
            })
            Toast.success(t('purchaseReturn.messages.rejectSuccess'))
            fetchReturns()
          } catch {
            Toast.error(t('purchaseReturn.messages.rejectError'))
          }
        },
      })
    },
    [purchaseReturnApi, fetchReturns, t]
  )

  // Handle ship return (send goods back to supplier)
  const handleShip = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: t('purchaseReturn.modal.shipTitle'),
        content: t('purchaseReturn.modal.shipContent', { returnNumber: returnItem.return_number }),
        okText: t('purchaseReturn.actions.ship'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        onOk: async () => {
          try {
            await purchaseReturnApi.postTradePurchaseReturnsIdShip(returnItem.id!, {
              note: '',
            })
            Toast.success(t('purchaseReturn.messages.shipSuccess'))
            fetchReturns()
          } catch {
            Toast.error(t('purchaseReturn.messages.shipError'))
          }
        },
      })
    },
    [purchaseReturnApi, fetchReturns, t]
  )

  // Handle complete return
  const handleComplete = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      try {
        await purchaseReturnApi.postTradePurchaseReturnsIdComplete(returnItem.id!)
        Toast.success(t('purchaseReturn.messages.completeSuccess'))
        fetchReturns()
      } catch {
        Toast.error(t('purchaseReturn.messages.completeError'))
      }
    },
    [purchaseReturnApi, fetchReturns, t]
  )

  // Handle cancel return
  const handleCancel = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: t('purchaseReturn.modal.cancelTitle'),
        content: t('purchaseReturn.modal.cancelContent', {
          returnNumber: returnItem.return_number,
        }),
        okText: t('salesOrder.modal.cancelOk'),
        cancelText: t('salesOrder.modal.backBtn'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await purchaseReturnApi.postTradePurchaseReturnsIdCancel(returnItem.id!, {
              reason: t('common.userCancel'),
            })
            Toast.success(t('purchaseReturn.messages.cancelSuccess'))
            fetchReturns()
          } catch {
            Toast.error(t('purchaseReturn.messages.cancelError'))
          }
        },
      })
    },
    [purchaseReturnApi, fetchReturns, t]
  )

  // Handle delete return
  const handleDelete = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: t('purchaseReturn.modal.deleteTitle'),
        content: t('purchaseReturn.modal.deleteContent', {
          returnNumber: returnItem.return_number,
        }),
        okText: t('salesOrder.modal.deleteOk'),
        cancelText: t('salesOrder.modal.cancelBtn'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await purchaseReturnApi.deleteTradePurchaseReturnsId(returnItem.id!)
            Toast.success(t('purchaseReturn.messages.deleteSuccess'))
            fetchReturns()
          } catch {
            Toast.error(t('purchaseReturn.messages.deleteError'))
          }
        },
      })
    },
    [purchaseReturnApi, fetchReturns, t]
  )

  // Handle view return
  const handleView = useCallback(
    (returnItem: PurchaseReturn) => {
      if (returnItem.id) {
        navigate(`/trade/purchase-returns/${returnItem.id}`)
      }
    },
    [navigate]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchReturns()
  }, [fetchReturns])

  // Table columns
  const tableColumns: DataTableColumn<PurchaseReturn>[] = useMemo(
    () => [
      {
        title: t('purchaseReturn.columns.returnNumber'),
        dataIndex: 'return_number',
        width: 150,
        sortable: true,
        render: (returnNumber: unknown) => (
          <span className="return-number">{(returnNumber as string) || '-'}</span>
        ),
      },
      {
        title: t('purchaseReturn.columns.orderNumber'),
        dataIndex: 'purchase_order_number',
        width: 150,
        render: (orderNumber: unknown) => (
          <span className="order-number">{(orderNumber as string) || '-'}</span>
        ),
      },
      {
        title: t('purchaseReturn.columns.supplier'),
        dataIndex: 'supplier_name',
        width: 150,
        ellipsis: true,
        render: (name: unknown) => (name as string) || '-',
      },
      {
        title: t('purchaseReturn.columns.itemCount'),
        dataIndex: 'item_count',
        width: 100,
        align: 'center',
        render: (count: unknown) => `${(count as number) || 0} ${t('common.unit')}`,
      },
      {
        title: t('purchaseReturn.columns.totalAmount'),
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
        title: t('purchaseReturn.columns.status'),
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as string | undefined
          if (!statusValue) return '-'
          const statusKey = STATUS_KEYS[statusValue] || 'draft'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>
              {t(`purchaseReturn.status.${statusKey}`)}
            </Tag>
          )
        },
      },
      {
        title: t('purchaseReturn.columns.createdAt'),
        dataIndex: 'created_at',
        width: 150,
        sortable: true,
        render: (date: unknown) => {
          const value = date as string | undefined
          return value ? formatDate(value) : '-'
        },
      },
      {
        title: t('purchaseReturn.columns.submittedAt'),
        dataIndex: 'submitted_at',
        width: 150,
        render: (date: unknown) => {
          const value = date as string | undefined
          return value ? formatDateTime(value) : '-'
        },
      },
      {
        title: t('purchaseReturn.columns.shippedAt'),
        dataIndex: 'shipped_at',
        width: 150,
        render: (date: unknown) => {
          const value = date as string | undefined
          return value ? formatDateTime(value) : '-'
        },
      },
      {
        title: t('purchaseReturn.columns.completedAt'),
        dataIndex: 'completed_at',
        width: 150,
        render: (date: unknown) => {
          const value = date as string | undefined
          return value ? formatDateTime(value) : '-'
        },
      },
    ],
    [t, formatCurrency, formatDate, formatDateTime]
  )

  // Table row actions
  const tableActions: TableAction<PurchaseReturn>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('purchaseReturn.actions.view'),
        onClick: handleView,
      },
      {
        key: 'submit',
        label: t('purchaseReturn.actions.submit'),
        type: 'primary',
        onClick: handleSubmit,
        hidden: (record) => record.status !== 'DRAFT',
      },
      {
        key: 'approve',
        label: t('purchaseReturn.actions.approve'),
        type: 'primary',
        onClick: handleApprove,
        hidden: (record) => record.status !== 'PENDING',
      },
      {
        key: 'reject',
        label: t('purchaseReturn.actions.reject'),
        type: 'warning',
        onClick: handleReject,
        hidden: (record) => record.status !== 'PENDING',
      },
      {
        key: 'ship',
        label: t('purchaseReturn.actions.ship'),
        type: 'primary',
        onClick: handleShip,
        hidden: (record) => record.status !== 'APPROVED',
      },
      {
        key: 'complete',
        label: t('purchaseReturn.actions.complete'),
        type: 'primary',
        onClick: handleComplete,
        hidden: (record) => record.status !== 'SHIPPED',
      },
      {
        key: 'cancel',
        label: t('purchaseReturn.actions.cancel'),
        type: 'warning',
        onClick: handleCancel,
        hidden: (record) =>
          record.status !== 'DRAFT' && record.status !== 'PENDING' && record.status !== 'APPROVED',
      },
      {
        key: 'delete',
        label: t('purchaseReturn.actions.delete'),
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
      handleShip,
      handleComplete,
      handleCancel,
      handleDelete,
      t,
    ]
  )

  return (
    <Container size="full" className="purchase-returns-page">
      <Card className="purchase-returns-card">
        <div className="purchase-returns-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('purchaseReturn.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('purchaseReturn.searchPlaceholder')}
          primaryAction={{
            label: t('purchaseReturn.newReturn'),
            icon: <IconPlus />,
            onClick: () => navigate('/trade/purchase-returns/new'),
          }}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('purchaseReturn.refresh'),
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
          <DataTable<PurchaseReturn>
            data={returnList}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 1500 }}
          />
        </Spin>
      </Card>
    </Container>
  )
}
