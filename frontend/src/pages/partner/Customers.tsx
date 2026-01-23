import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Tag, Toast, Select, Space, Modal, Spin } from '@douyinfe/semi-ui'
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
import { getCustomers } from '@/api/customers/customers'
import type {
  HandlerCustomerListResponse,
  HandlerCustomerListResponseStatus,
  HandlerCustomerListResponseLevel,
  HandlerCustomerListResponseType,
  GetPartnerCustomersParams,
  GetPartnerCustomersStatus,
  GetPartnerCustomersLevel,
  GetPartnerCustomersType,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './Customers.css'

const { Title } = Typography

// Customer type with index signature for DataTable compatibility
type Customer = HandlerCustomerListResponse & Record<string, unknown>

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '启用', value: 'active' },
  { label: '停用', value: 'inactive' },
  { label: '暂停', value: 'suspended' },
]

// Type options for filter
const TYPE_OPTIONS = [
  { label: '全部类型', value: '' },
  { label: '个人', value: 'individual' },
  { label: '企业/组织', value: 'organization' },
]

// Level options for filter
const LEVEL_OPTIONS = [
  { label: '全部等级', value: '' },
  { label: '普通', value: 'normal' },
  { label: '白银', value: 'silver' },
  { label: '黄金', value: 'gold' },
  { label: '铂金', value: 'platinum' },
  { label: 'VIP', value: 'vip' },
]

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerCustomerListResponseStatus, 'green' | 'grey' | 'orange'> = {
  active: 'green',
  inactive: 'grey',
  suspended: 'orange',
}

// Status labels
const STATUS_LABELS: Record<HandlerCustomerListResponseStatus, string> = {
  active: '启用',
  inactive: '停用',
  suspended: '暂停',
}

// Level tag color mapping
const LEVEL_TAG_COLORS: Record<
  HandlerCustomerListResponseLevel,
  'white' | 'grey' | 'amber' | 'cyan' | 'violet'
> = {
  normal: 'white',
  silver: 'grey',
  gold: 'amber',
  platinum: 'cyan',
  vip: 'violet',
}

// Level labels
const LEVEL_LABELS: Record<HandlerCustomerListResponseLevel, string> = {
  normal: '普通',
  silver: '白银',
  gold: '黄金',
  platinum: '铂金',
  vip: 'VIP',
}

// Type labels
const TYPE_LABELS: Record<HandlerCustomerListResponseType, string> = {
  individual: '个人',
  organization: '企业',
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
 * Customers list page
 *
 * Features:
 * - Customer listing with pagination
 * - Search by name, code, phone, email
 * - Filter by status, type, and level
 * - Activate/deactivate/suspend customer actions
 * - Navigate to customer form for create/edit
 */
export default function CustomersPage() {
  const navigate = useNavigate()
  const api = useMemo(() => getCustomers(), [])

  // State for data
  const [customerList, setCustomerList] = useState<Customer[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [typeFilter, setTypeFilter] = useState<string>('')
  const [levelFilter, setLevelFilter] = useState<string>('')

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Fetch customers
  const fetchCustomers = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetPartnerCustomersParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as GetPartnerCustomersStatus | undefined,
        type: (typeFilter || undefined) as GetPartnerCustomersType | undefined,
        level: (levelFilter || undefined) as GetPartnerCustomersLevel | undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      const response = await api.getPartnerCustomers(params)

      if (response.success && response.data) {
        setCustomerList(response.data as Customer[])
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
      Toast.error('获取客户列表失败')
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
    levelFilter,
  ])

  // Fetch on mount and when state changes
  useEffect(() => {
    fetchCustomers()
  }, [fetchCustomers])

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

  // Handle level filter change
  const handleLevelChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const levelValue = typeof value === 'string' ? value : ''
      setLevelFilter(levelValue)
      setFilter('level', levelValue || null)
    },
    [setFilter]
  )

  // Handle activate customer
  const handleActivate = useCallback(
    async (customer: Customer) => {
      if (!customer.id) return
      try {
        await api.postPartnerCustomersIdActivate(customer.id)
        Toast.success(`客户 "${customer.name}" 已启用`)
        fetchCustomers()
      } catch {
        Toast.error('启用客户失败')
      }
    },
    [api, fetchCustomers]
  )

  // Handle deactivate customer
  const handleDeactivate = useCallback(
    async (customer: Customer) => {
      if (!customer.id) return
      try {
        await api.postPartnerCustomersIdDeactivate(customer.id)
        Toast.success(`客户 "${customer.name}" 已停用`)
        fetchCustomers()
      } catch {
        Toast.error('停用客户失败')
      }
    },
    [api, fetchCustomers]
  )

  // Handle suspend customer
  const handleSuspend = useCallback(
    async (customer: Customer) => {
      if (!customer.id) return
      Modal.confirm({
        title: '确认暂停',
        content: `确定要暂停客户 "${customer.name}" 吗？暂停后该客户将无法下单。`,
        okText: '确认暂停',
        cancelText: '取消',
        okButtonProps: { type: 'warning' },
        onOk: async () => {
          try {
            await api.postPartnerCustomersIdSuspend(customer.id!)
            Toast.success(`客户 "${customer.name}" 已暂停`)
            fetchCustomers()
          } catch {
            Toast.error('暂停客户失败')
          }
        },
      })
    },
    [api, fetchCustomers]
  )

  // Handle delete customer
  const handleDelete = useCallback(
    async (customer: Customer) => {
      if (!customer.id) return
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除客户 "${customer.name}" 吗？删除后无法恢复。`,
        okText: '确认删除',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await api.deletePartnerCustomersId(customer.id!)
            Toast.success(`客户 "${customer.name}" 已删除`)
            fetchCustomers()
          } catch {
            Toast.error('删除客户失败，该客户可能有余额或关联订单')
          }
        },
      })
    },
    [api, fetchCustomers]
  )

  // Handle edit customer
  const handleEdit = useCallback(
    (customer: Customer) => {
      if (customer.id) {
        navigate(`/partner/customers/${customer.id}/edit`)
      }
    },
    [navigate]
  )

  // Handle view customer
  const handleView = useCallback(
    (customer: Customer) => {
      if (customer.id) {
        navigate(`/partner/customers/${customer.id}`)
      }
    },
    [navigate]
  )

  // Handle bulk activate
  const handleBulkActivate = useCallback(async () => {
    try {
      await Promise.all(selectedRowKeys.map((id) => api.postPartnerCustomersIdActivate(id)))
      Toast.success(`已启用 ${selectedRowKeys.length} 个客户`)
      setSelectedRowKeys([])
      fetchCustomers()
    } catch {
      Toast.error('批量启用失败')
    }
  }, [api, selectedRowKeys, fetchCustomers])

  // Handle bulk deactivate
  const handleBulkDeactivate = useCallback(async () => {
    try {
      await Promise.all(selectedRowKeys.map((id) => api.postPartnerCustomersIdDeactivate(id)))
      Toast.success(`已停用 ${selectedRowKeys.length} 个客户`)
      setSelectedRowKeys([])
      fetchCustomers()
    } catch {
      Toast.error('批量停用失败')
    }
  }, [api, selectedRowKeys, fetchCustomers])

  // Table columns
  const tableColumns: DataTableColumn<Customer>[] = useMemo(
    () => [
      {
        title: '客户编码',
        dataIndex: 'code',
        width: 120,
        sortable: true,
        render: (code: unknown) => <span className="customer-code">{(code as string) || '-'}</span>,
      },
      {
        title: '客户名称',
        dataIndex: 'name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown, record: Customer) => (
          <div className="customer-name-cell">
            <span className="customer-name">{(name as string) || '-'}</span>
            {record.short_name && <span className="customer-short-name">{record.short_name}</span>}
          </div>
        ),
      },
      {
        title: '类型',
        dataIndex: 'type',
        width: 80,
        align: 'center',
        render: (type: unknown) => {
          const typeValue = type as HandlerCustomerListResponseType | undefined
          if (!typeValue) return '-'
          return (
            <Tag className="type-tag" color={typeValue === 'organization' ? 'blue' : 'light-blue'}>
              {TYPE_LABELS[typeValue]}
            </Tag>
          )
        },
      },
      {
        title: '联系方式',
        dataIndex: 'phone',
        width: 160,
        render: (_phone: unknown, record: Customer) => (
          <div className="customer-contact-cell">
            {record.phone && <span className="customer-phone">{record.phone}</span>}
            {record.email && <span className="customer-email">{record.email}</span>}
            {!record.phone && !record.email && '-'}
          </div>
        ),
      },
      {
        title: '地区',
        dataIndex: 'city',
        width: 120,
        render: (_city: unknown, record: Customer) => (
          <span className="customer-location-cell">
            {record.province || record.city
              ? `${record.province || ''}${record.city ? ` ${record.city}` : ''}`
              : '-'}
          </span>
        ),
      },
      {
        title: '等级',
        dataIndex: 'level',
        width: 80,
        align: 'center',
        sortable: true,
        render: (level: unknown) => {
          const levelValue = level as HandlerCustomerListResponseLevel | undefined
          if (!levelValue) return '-'
          return (
            <Tag className="level-tag" color={LEVEL_TAG_COLORS[levelValue]}>
              {LEVEL_LABELS[levelValue]}
            </Tag>
          )
        },
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as HandlerCustomerListResponseStatus | undefined
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
  const tableActions: TableAction<Customer>[] = useMemo(
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
        key: 'suspend',
        label: '暂停',
        type: 'warning',
        onClick: handleSuspend,
        hidden: (record) => record.status !== 'active',
      },
      {
        key: 'delete',
        label: '删除',
        type: 'danger',
        onClick: handleDelete,
      },
    ],
    [handleView, handleEdit, handleActivate, handleDeactivate, handleSuspend, handleDelete]
  )

  // Row selection handler
  const onSelectionChange = useCallback((keys: string[], _rows: Customer[]) => {
    setSelectedRowKeys(keys)
  }, [])

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchCustomers()
  }, [fetchCustomers])

  return (
    <Container size="full" className="customers-page">
      <Card className="customers-card">
        <div className="customers-header">
          <Title heading={4} style={{ margin: 0 }}>
            客户管理
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索客户名称、编码、电话、邮箱..."
          primaryAction={{
            label: '新增客户',
            icon: <IconPlus />,
            onClick: () => navigate('/partner/customers/new'),
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
            <Space className="customers-filter-container">
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
              <Select
                placeholder="等级筛选"
                value={levelFilter}
                onChange={handleLevelChange}
                optionList={LEVEL_OPTIONS}
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
          <DataTable<Customer>
            data={customerList}
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
