import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Tag, Toast, Select, Space, Modal, Spin } from '@douyinfe/semi-ui-19'
import { IconPlus, IconRefresh } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  DataTable,
  TableToolbar,
  BulkActionBar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { useFormatters } from '@/hooks/useFormatters'
import {
  listCustomers,
  activateCustomer,
  deactivateCustomer,
  deleteCustomer,
} from '@/api/customers/customers'
import type {
  HandlerCustomerListResponse,
  HandlerCustomerListResponseStatus,
  HandlerCustomerListResponseLevel,
  HandlerCustomerListResponseType,
  ListCustomersParams,
  ListCustomersStatus,
  ListCustomersLevel,
  ListCustomersType,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './Customers.css'

const { Title } = Typography

// Customer type with index signature for DataTable compatibility
type Customer = HandlerCustomerListResponse & Record<string, unknown>

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerCustomerListResponseStatus, 'green' | 'grey' | 'orange'> = {
  active: 'green',
  inactive: 'grey',
  suspended: 'orange',
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
  const { t } = useTranslation(['partner', 'common'])
  const { formatDate } = useFormatters()

  // Memoized options with translations
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('customers.allStatus'), value: '' },
      { label: t('customers.status.active'), value: 'active' },
      { label: t('customers.status.inactive'), value: 'inactive' },
      { label: t('customers.status.suspended'), value: 'suspended' },
    ],
    [t]
  )

  const TYPE_OPTIONS = useMemo(
    () => [
      { label: t('customers.allTypes'), value: '' },
      { label: t('customers.type.individual'), value: 'individual' },
      { label: t('customers.type.organization'), value: 'organization' },
    ],
    [t]
  )

  const LEVEL_OPTIONS = useMemo(
    () => [
      { label: t('customers.allLevels'), value: '' },
      { label: t('customers.level.normal'), value: 'normal' },
      { label: t('customers.level.silver'), value: 'silver' },
      { label: t('customers.level.gold'), value: 'gold' },
      { label: t('customers.level.platinum'), value: 'platinum' },
      { label: t('customers.level.vip'), value: 'vip' },
    ],
    [t]
  )

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
      const params: ListCustomersParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as ListCustomersStatus | undefined,
        type: (typeFilter || undefined) as ListCustomersType | undefined,
        level: (levelFilter || undefined) as ListCustomersLevel | undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      const response = await listCustomers(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setCustomerList(response.data.data as Customer[])
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
      Toast.error(t('customers.messages.fetchError'))
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
        await activateCustomer(customer.id, {})
        Toast.success(t('customers.messages.activateSuccess', { name: customer.name }))
        fetchCustomers()
      } catch {
        Toast.error(t('customers.messages.activateError'))
      }
    },
    [fetchCustomers, t]
  )

  // Handle deactivate customer
  const handleDeactivate = useCallback(
    async (customer: Customer) => {
      if (!customer.id) return
      try {
        await deactivateCustomer(customer.id, {})
        Toast.success(t('customers.messages.deactivateSuccess', { name: customer.name }))
        fetchCustomers()
      } catch {
        Toast.error(t('customers.messages.deactivateError'))
      }
    },
    [fetchCustomers, t]
  )

  // Handle suspend customer
  const handleSuspend = useCallback(
    async (customer: Customer) => {
      if (!customer.id) return
      Modal.confirm({
        title: t('customers.confirm.suspendTitle'),
        content: t('customers.confirm.suspendContent', { name: customer.name }),
        okText: t('customers.confirm.suspendOk'),
        cancelText: t('common:actions.cancel'),
        okButtonProps: { type: 'warning' },
        onOk: async () => {
          try {
            await deactivateCustomer(customer.id!, {})
            Toast.success(t('customers.messages.suspendSuccess', { name: customer.name }))
            fetchCustomers()
          } catch {
            Toast.error(t('customers.messages.suspendError'))
          }
        },
      })
    },
    [fetchCustomers, t]
  )

  // Handle delete customer
  const handleDelete = useCallback(
    async (customer: Customer) => {
      if (!customer.id) return
      Modal.confirm({
        title: t('customers.confirm.deleteTitle'),
        content: t('customers.confirm.deleteContent', { name: customer.name }),
        okText: t('customers.confirm.deleteOk'),
        cancelText: t('common:actions.cancel'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await deleteCustomer(customer.id!)
            Toast.success(t('customers.messages.deleteSuccess', { name: customer.name }))
            fetchCustomers()
          } catch {
            Toast.error(t('customers.messages.deleteError'))
          }
        },
      })
    },
    [fetchCustomers, t]
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

  // Handle bulk activate using Promise.allSettled for partial success handling
  const handleBulkActivate = useCallback(async () => {
    const results = await Promise.allSettled(selectedRowKeys.map((id) => activateCustomer(id, {})))

    const successCount = results.filter((r) => r.status === 'fulfilled').length
    const failureCount = results.filter((r) => r.status === 'rejected').length

    if (failureCount === 0) {
      // All succeeded
      Toast.success(t('customers.messages.batchActivateSuccess', { count: successCount }))
    } else if (successCount === 0) {
      // All failed
      Toast.error(t('customers.messages.batchActivateError'))
    } else {
      // Partial success
      Toast.warning(
        t('customers.messages.batchActivatePartial', {
          successCount,
          failureCount,
        })
      )
    }

    setSelectedRowKeys([])
    fetchCustomers()
  }, [selectedRowKeys, fetchCustomers, t])

  // Handle bulk deactivate using Promise.allSettled for partial success handling
  const handleBulkDeactivate = useCallback(async () => {
    const results = await Promise.allSettled(
      selectedRowKeys.map((id) => deactivateCustomer(id, {}))
    )

    const successCount = results.filter((r) => r.status === 'fulfilled').length
    const failureCount = results.filter((r) => r.status === 'rejected').length

    if (failureCount === 0) {
      // All succeeded
      Toast.success(t('customers.messages.batchDeactivateSuccess', { count: successCount }))
    } else if (successCount === 0) {
      // All failed
      Toast.error(t('customers.messages.batchDeactivateError'))
    } else {
      // Partial success
      Toast.warning(
        t('customers.messages.batchDeactivatePartial', {
          successCount,
          failureCount,
        })
      )
    }

    setSelectedRowKeys([])
    fetchCustomers()
  }, [selectedRowKeys, fetchCustomers, t])

  // Table columns
  const tableColumns: DataTableColumn<Customer>[] = useMemo(
    () => [
      {
        title: t('customers.columns.code'),
        dataIndex: 'code',
        width: 120,
        sortable: true,
        render: (code: unknown) => <span className="customer-code">{(code as string) || '-'}</span>,
      },
      {
        title: t('customers.columns.name'),
        dataIndex: 'name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown, record: Customer) => (
          <div className="customer-name-cell">
            <span
              className="customer-name table-cell-link"
              onClick={() => {
                if (record.id) navigate(`/partner/customers/${record.id}`)
              }}
              onKeyDown={(e) => {
                if ((e.key === 'Enter' || e.key === ' ') && record.id) {
                  e.preventDefault()
                  navigate(`/partner/customers/${record.id}`)
                }
              }}
              role="link"
              tabIndex={0}
            >
              {(name as string) || '-'}
            </span>
            {record.short_name && <span className="customer-short-name">{record.short_name}</span>}
          </div>
        ),
      },
      {
        title: t('customers.columns.type'),
        dataIndex: 'type',
        width: 80,
        align: 'center',
        render: (type: unknown) => {
          const typeValue = type as HandlerCustomerListResponseType | undefined
          if (!typeValue) return '-'
          return (
            <Tag className="type-tag" color={typeValue === 'organization' ? 'blue' : 'light-blue'}>
              {t(`customers.type.${typeValue}`)}
            </Tag>
          )
        },
      },
      {
        title: t('customers.columns.contact'),
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
        title: t('customers.columns.region'),
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
        title: t('customers.columns.level'),
        dataIndex: 'level',
        width: 80,
        align: 'center',
        sortable: true,
        render: (level: unknown) => {
          const levelValue = level as HandlerCustomerListResponseLevel | undefined
          if (!levelValue) return '-'
          return (
            <Tag className="level-tag" color={LEVEL_TAG_COLORS[levelValue]}>
              {t(`customers.level.${levelValue}`)}
            </Tag>
          )
        },
      },
      {
        title: t('customers.columns.status'),
        dataIndex: 'status',
        width: 90,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as HandlerCustomerListResponseStatus | undefined
          if (!statusValue) return '-'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>{t(`customers.status.${statusValue}`)}</Tag>
          )
        },
      },
      {
        title: t('customers.columns.createdAt'),
        dataIndex: 'created_at',
        width: 120,
        sortable: true,
        render: (date: unknown) => {
          const dateStr = date as string | undefined
          return dateStr ? formatDate(dateStr) : '-'
        },
      },
    ],
    [t, formatDate, navigate]
  )

  // Handle view balance
  const handleViewBalance = useCallback(
    (customer: Customer) => {
      if (customer.id) {
        navigate(`/partner/customers/${customer.id}/balance`)
      }
    },
    [navigate]
  )

  // Table row actions
  const tableActions: TableAction<Customer>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('customers.actions.view'),
        onClick: handleView,
      },
      {
        key: 'edit',
        label: t('customers.actions.edit'),
        onClick: handleEdit,
      },
      {
        key: 'balance',
        label: t('customers.actions.balance'),
        type: 'primary',
        onClick: handleViewBalance,
      },
      {
        key: 'activate',
        label: t('customers.actions.activate'),
        type: 'primary',
        onClick: handleActivate,
        hidden: (record) => record.status === 'active',
      },
      {
        key: 'deactivate',
        label: t('customers.actions.deactivate'),
        type: 'warning',
        onClick: handleDeactivate,
        hidden: (record) => record.status !== 'active',
      },
      {
        key: 'suspend',
        label: t('customers.actions.suspend'),
        type: 'warning',
        onClick: handleSuspend,
        hidden: (record) => record.status !== 'active',
      },
      {
        key: 'delete',
        label: t('customers.actions.delete'),
        type: 'danger',
        onClick: handleDelete,
      },
    ],
    [
      t,
      handleView,
      handleEdit,
      handleViewBalance,
      handleActivate,
      handleDeactivate,
      handleSuspend,
      handleDelete,
    ]
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
            {t('customers.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('customers.searchPlaceholder')}
          primaryAction={{
            label: t('customers.addCustomer'),
            icon: <IconPlus />,
            onClick: () => navigate('/partner/customers/new'),
          }}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('common:actions.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="customers-filter-container">
              <Select
                placeholder={t('customers.statusFilter')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('customers.typeFilter')}
                value={typeFilter}
                onChange={handleTypeChange}
                optionList={TYPE_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('customers.levelFilter')}
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
              {t('customers.actions.batchActivate')}
            </Tag>
            <Tag color="orange" onClick={handleBulkDeactivate} style={{ cursor: 'pointer' }}>
              {t('customers.actions.batchDeactivate')}
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
