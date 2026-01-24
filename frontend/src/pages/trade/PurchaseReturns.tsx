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
import './PurchaseReturns.css'

const { Title } = Typography

// Purchase return type with index signature for DataTable compatibility
type PurchaseReturn = HandlerPurchaseReturnListResponse & Record<string, unknown>

// Supplier option type
interface SupplierOption {
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
  { label: '已发货', value: 'SHIPPED' },
  { label: '已完成', value: 'COMPLETED' },
  { label: '已取消', value: 'CANCELLED' },
]

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

// Status labels
const STATUS_LABELS: Record<string, string> = {
  DRAFT: '草稿',
  PENDING: '待审批',
  APPROVED: '已审批',
  REJECTED: '已拒绝',
  SHIPPED: '已发货',
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
      Toast.error('获取采购退货列表失败')
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
        title: '提交审批',
        content: `确定要提交退货单 "${returnItem.return_number}" 进行审批吗？`,
        okText: '确认',
        cancelText: '取消',
        onOk: async () => {
          try {
            await purchaseReturnApi.postTradePurchaseReturnsIdSubmit(returnItem.id!)
            Toast.success(`退货单 "${returnItem.return_number}" 已提交审批`)
            fetchReturns()
          } catch {
            Toast.error('提交审批失败')
          }
        },
      })
    },
    [purchaseReturnApi, fetchReturns]
  )

  // Handle approve return
  const handleApprove = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: '审批通过',
        content: `确定要通过退货单 "${returnItem.return_number}" 的审批吗？`,
        okText: '通过',
        cancelText: '取消',
        onOk: async () => {
          try {
            await purchaseReturnApi.postTradePurchaseReturnsIdApprove(returnItem.id!, { note: '' })
            Toast.success(`退货单 "${returnItem.return_number}" 已审批通过`)
            fetchReturns()
          } catch {
            Toast.error('审批失败')
          }
        },
      })
    },
    [purchaseReturnApi, fetchReturns]
  )

  // Handle reject return
  const handleReject = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: '拒绝退货',
        content: `确定要拒绝退货单 "${returnItem.return_number}" 吗？`,
        okText: '拒绝',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await purchaseReturnApi.postTradePurchaseReturnsIdReject(returnItem.id!, {
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
    [purchaseReturnApi, fetchReturns]
  )

  // Handle ship return (send goods back to supplier)
  const handleShip = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: '确认发货',
        content: `确定要将退货单 "${returnItem.return_number}" 的商品发货给供应商吗？发货后库存将被扣减。`,
        okText: '确认发货',
        cancelText: '取消',
        onOk: async () => {
          try {
            await purchaseReturnApi.postTradePurchaseReturnsIdShip(returnItem.id!, {
              note: `从仓库发货`,
            })
            Toast.success(`退货单 "${returnItem.return_number}" 已发货`)
            fetchReturns()
          } catch {
            Toast.error('发货失败')
          }
        },
      })
    },
    [purchaseReturnApi, fetchReturns]
  )

  // Handle complete return
  const handleComplete = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      try {
        await purchaseReturnApi.postTradePurchaseReturnsIdComplete(returnItem.id!)
        Toast.success(`退货单 "${returnItem.return_number}" 已完成`)
        fetchReturns()
      } catch {
        Toast.error('完成退货失败')
      }
    },
    [purchaseReturnApi, fetchReturns]
  )

  // Handle cancel return
  const handleCancel = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: '取消退货',
        content: `确定要取消退货单 "${returnItem.return_number}" 吗？`,
        okText: '确认取消',
        cancelText: '返回',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await purchaseReturnApi.postTradePurchaseReturnsIdCancel(returnItem.id!, {
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
    [purchaseReturnApi, fetchReturns]
  )

  // Handle delete return
  const handleDelete = useCallback(
    async (returnItem: PurchaseReturn) => {
      if (!returnItem.id) return
      Modal.confirm({
        title: '删除退货单',
        content: `确定要删除退货单 "${returnItem.return_number}" 吗？删除后无法恢复。`,
        okText: '确认删除',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await purchaseReturnApi.deleteTradePurchaseReturnsId(returnItem.id!)
            Toast.success(`退货单 "${returnItem.return_number}" 已删除`)
            fetchReturns()
          } catch {
            Toast.error('删除退货单失败')
          }
        },
      })
    },
    [purchaseReturnApi, fetchReturns]
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
        dataIndex: 'purchase_order_number',
        width: 150,
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
        title: '发货时间',
        dataIndex: 'shipped_at',
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
  const tableActions: TableAction<PurchaseReturn>[] = useMemo(
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
        key: 'ship',
        label: '发货',
        type: 'primary',
        onClick: handleShip,
        hidden: (record) => record.status !== 'APPROVED',
      },
      {
        key: 'complete',
        label: '完成',
        type: 'primary',
        onClick: handleComplete,
        hidden: (record) => record.status !== 'SHIPPED',
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
      handleShip,
      handleComplete,
      handleCancel,
      handleDelete,
    ]
  )

  return (
    <Container size="full" className="purchase-returns-page">
      <Card className="purchase-returns-card">
        <div className="purchase-returns-header">
          <Title heading={4} style={{ margin: 0 }}>
            采购退货
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索退货单号..."
          primaryAction={{
            label: '新建退货',
            icon: <IconPlus />,
            onClick: () => navigate('/trade/purchase-returns/new'),
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
