import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Tag, Toast, Select, Space, Modal, Spin, Rating } from '@douyinfe/semi-ui'
import { IconPlus, IconRefresh } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import {
  DataTable,
  TableToolbar,
  BulkActionBar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { getSuppliers } from '@/api/suppliers/suppliers'
import type {
  HandlerSupplierListResponse,
  HandlerSupplierListResponseStatus,
  GetPartnerSuppliersParams,
  GetPartnerSuppliersStatus,
  GetPartnerSuppliersType,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './Suppliers.css'

const { Title } = Typography

// Supplier type with index signature for DataTable compatibility
type Supplier = HandlerSupplierListResponse & Record<string, unknown>

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '启用', value: 'active' },
  { label: '停用', value: 'inactive' },
  { label: '拉黑', value: 'blocked' },
]

// Type options for filter
const TYPE_OPTIONS = [
  { label: '全部类型', value: '' },
  { label: '生产商', value: 'manufacturer' },
  { label: '经销商', value: 'distributor' },
  { label: '零售商', value: 'retailer' },
  { label: '服务商', value: 'service' },
]

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerSupplierListResponseStatus, 'green' | 'grey' | 'red'> = {
  active: 'green',
  inactive: 'grey',
  blocked: 'red',
}

// Status labels
const STATUS_LABELS: Record<HandlerSupplierListResponseStatus, string> = {
  active: '启用',
  inactive: '停用',
  blocked: '拉黑',
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
 * Suppliers list page
 *
 * Features:
 * - Supplier listing with pagination
 * - Search by name, code, phone, email
 * - Filter by status and type
 * - Activate/deactivate/block supplier actions
 * - Navigate to supplier form for create/edit
 */
export default function SuppliersPage() {
  const navigate = useNavigate()
  const api = useMemo(() => getSuppliers(), [])

  // State for data
  const [supplierList, setSupplierList] = useState<Supplier[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [typeFilter, setTypeFilter] = useState<string>('')

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Fetch suppliers
  const fetchSuppliers = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetPartnerSuppliersParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as GetPartnerSuppliersStatus | undefined,
        type: (typeFilter || undefined) as GetPartnerSuppliersType | undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      const response = await api.getPartnerSuppliers(params)

      if (response.success && response.data) {
        setSupplierList(response.data as Supplier[])
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
      Toast.error('获取供应商列表失败')
    } finally {
      setLoading(false)
    }
  }, [
    api,
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    searchKeyword,
    statusFilter,
    typeFilter,
  ])

  // Fetch on mount and when state changes
  useEffect(() => {
    fetchSuppliers()
  }, [fetchSuppliers])

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

  // Handle type filter change
  const handleTypeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const typeValue = typeof value === 'string' ? value : ''
      setTypeFilter(typeValue)
      setFilter('type', typeValue || null)
    },
    [setFilter]
  )

  // Handle activate supplier
  const handleActivate = useCallback(
    async (supplier: Supplier) => {
      if (!supplier.id) return
      try {
        await api.postPartnerSuppliersIdActivate(supplier.id)
        Toast.success(`供应商 "${supplier.name}" 已启用`)
        fetchSuppliers()
      } catch {
        Toast.error('启用供应商失败')
      }
    },
    [api, fetchSuppliers]
  )

  // Handle deactivate supplier
  const handleDeactivate = useCallback(
    async (supplier: Supplier) => {
      if (!supplier.id) return
      try {
        await api.postPartnerSuppliersIdDeactivate(supplier.id)
        Toast.success(`供应商 "${supplier.name}" 已停用`)
        fetchSuppliers()
      } catch {
        Toast.error('停用供应商失败')
      }
    },
    [api, fetchSuppliers]
  )

  // Handle block supplier
  const handleBlock = useCallback(
    async (supplier: Supplier) => {
      if (!supplier.id) return
      Modal.confirm({
        title: '确认拉黑',
        content: `确定要拉黑供应商 "${supplier.name}" 吗？拉黑后该供应商将无法进行业务往来。`,
        okText: '确认拉黑',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await api.postPartnerSuppliersIdBlock(supplier.id!)
            Toast.success(`供应商 "${supplier.name}" 已拉黑`)
            fetchSuppliers()
          } catch {
            Toast.error('拉黑供应商失败')
          }
        },
      })
    },
    [api, fetchSuppliers]
  )

  // Handle delete supplier
  const handleDelete = useCallback(
    async (supplier: Supplier) => {
      if (!supplier.id) return
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除供应商 "${supplier.name}" 吗？删除后无法恢复。`,
        okText: '确认删除',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await api.deletePartnerSuppliersId(supplier.id!)
            Toast.success(`供应商 "${supplier.name}" 已删除`)
            fetchSuppliers()
          } catch {
            Toast.error('删除供应商失败，该供应商可能有关联订单')
          }
        },
      })
    },
    [api, fetchSuppliers]
  )

  // Handle edit supplier
  const handleEdit = useCallback(
    (supplier: Supplier) => {
      if (supplier.id) {
        navigate(`/partner/suppliers/${supplier.id}/edit`)
      }
    },
    [navigate]
  )

  // Handle view supplier
  const handleView = useCallback(
    (supplier: Supplier) => {
      if (supplier.id) {
        navigate(`/partner/suppliers/${supplier.id}`)
      }
    },
    [navigate]
  )

  // Handle bulk activate
  const handleBulkActivate = useCallback(async () => {
    try {
      await Promise.all(selectedRowKeys.map((id) => api.postPartnerSuppliersIdActivate(id)))
      Toast.success(`已启用 ${selectedRowKeys.length} 个供应商`)
      setSelectedRowKeys([])
      fetchSuppliers()
    } catch {
      Toast.error('批量启用失败')
    }
  }, [api, selectedRowKeys, fetchSuppliers])

  // Handle bulk deactivate
  const handleBulkDeactivate = useCallback(async () => {
    try {
      await Promise.all(selectedRowKeys.map((id) => api.postPartnerSuppliersIdDeactivate(id)))
      Toast.success(`已停用 ${selectedRowKeys.length} 个供应商`)
      setSelectedRowKeys([])
      fetchSuppliers()
    } catch {
      Toast.error('批量停用失败')
    }
  }, [api, selectedRowKeys, fetchSuppliers])

  // Table columns
  const tableColumns: DataTableColumn<Supplier>[] = useMemo(
    () => [
      {
        title: '供应商编码',
        dataIndex: 'code',
        width: 120,
        sortable: true,
        render: (code: unknown) => <span className="supplier-code">{(code as string) || '-'}</span>,
      },
      {
        title: '供应商名称',
        dataIndex: 'name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown, record: Supplier) => (
          <div className="supplier-name-cell">
            <span className="supplier-name">{(name as string) || '-'}</span>
            {record.short_name && <span className="supplier-short-name">{record.short_name}</span>}
          </div>
        ),
      },
      {
        title: '联系方式',
        dataIndex: 'phone',
        width: 160,
        render: (_phone: unknown, record: Supplier) => (
          <div className="supplier-contact-cell">
            {record.phone && <span className="supplier-phone">{record.phone}</span>}
            {record.email && <span className="supplier-email">{record.email}</span>}
            {!record.phone && !record.email && '-'}
          </div>
        ),
      },
      {
        title: '地区',
        dataIndex: 'city',
        width: 120,
        render: (_city: unknown, record: Supplier) => (
          <span className="supplier-location-cell">
            {record.province || record.city
              ? `${record.province || ''}${record.city ? ` ${record.city}` : ''}`
              : '-'}
          </span>
        ),
      },
      {
        title: '评级',
        dataIndex: 'rating',
        width: 130,
        align: 'center',
        sortable: true,
        render: (rating: unknown) => {
          const ratingValue = rating as number | undefined
          if (ratingValue === undefined || ratingValue === null) return '-'
          return <Rating value={ratingValue} disabled size="small" count={5} allowHalf />
        },
      },
      {
        title: '账期(天)',
        dataIndex: 'payment_term_days',
        width: 90,
        align: 'right',
        sortable: true,
        render: (days: unknown) => {
          const daysValue = days as number | undefined
          if (daysValue === undefined || daysValue === null) return '-'
          return <span className="supplier-payment-days">{daysValue}</span>
        },
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as HandlerSupplierListResponseStatus | undefined
          if (!statusValue) return '-'
          return <Tag color={STATUS_TAG_COLORS[statusValue]}>{STATUS_LABELS[statusValue]}</Tag>
        },
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        width: 120,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
    ],
    []
  )

  // Table row actions
  const tableActions: TableAction<Supplier>[] = useMemo(
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
      },
      {
        key: 'activate',
        label: '启用',
        type: 'primary',
        onClick: handleActivate,
        hidden: (record) => record.status === 'active',
      },
      {
        key: 'deactivate',
        label: '停用',
        type: 'warning',
        onClick: handleDeactivate,
        hidden: (record) => record.status !== 'active',
      },
      {
        key: 'block',
        label: '拉黑',
        type: 'danger',
        onClick: handleBlock,
        hidden: (record) => record.status === 'blocked',
      },
      {
        key: 'delete',
        label: '删除',
        type: 'danger',
        onClick: handleDelete,
      },
    ],
    [handleView, handleEdit, handleActivate, handleDeactivate, handleBlock, handleDelete]
  )

  // Row selection handler
  const onSelectionChange = useCallback((keys: string[], _rows: Supplier[]) => {
    setSelectedRowKeys(keys)
  }, [])

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchSuppliers()
  }, [fetchSuppliers])

  return (
    <Container size="full" className="suppliers-page">
      <Card className="suppliers-card">
        <div className="suppliers-header">
          <Title heading={4} style={{ margin: 0 }}>
            供应商管理
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索供应商名称、编码、电话、邮箱..."
          primaryAction={{
            label: '新增供应商',
            icon: <IconPlus />,
            onClick: () => navigate('/partner/suppliers/new'),
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
            <Space className="suppliers-filter-container">
              <Select
                placeholder="状态筛选"
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="类型筛选"
                value={typeFilter}
                onChange={handleTypeChange}
                optionList={TYPE_OPTIONS}
                style={{ width: 120 }}
              />
            </Space>
          }
        />

        {selectedRowKeys.length > 0 && (
          <BulkActionBar
            selectedCount={selectedRowKeys.length}
            onCancel={() => setSelectedRowKeys([])}
          >
            <Tag color="blue" onClick={handleBulkActivate} style={{ cursor: 'pointer' }}>
              批量启用
            </Tag>
            <Tag color="orange" onClick={handleBulkDeactivate} style={{ cursor: 'pointer' }}>
              批量停用
            </Tag>
          </BulkActionBar>
        )}

        <Spin spinning={loading}>
          <DataTable<Supplier>
            data={supplierList}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            rowSelection={{
              selectedRowKeys,
              onChange: onSelectionChange,
            }}
            scroll={{ x: 1100 }}
          />
        </Spin>
      </Card>
    </Container>
  )
}
