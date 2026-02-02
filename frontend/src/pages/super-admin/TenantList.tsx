import { useState, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Select,
  Space,
  Modal,
  Button,
  Dropdown,
} from '@douyinfe/semi-ui-19'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import {
  IconPlus,
  IconRefresh,
  IconMore,
  IconEyeOpened,
  IconEdit,
  IconStop,
  IconPlay,
} from '@douyinfe/semi-icons'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type BulkAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import {
  useListTenants,
  useSuspendTenant,
  useActivateTenant,
  getListTenantsQueryKey,
} from '@/api/tenants/tenants'
import type {
  HandlerTenantResponse,
  ListTenantsParams,
  ListTenantsStatus,
  ListTenantsPlan,
  ListTenantsSortBy,
  ListTenantsSortDir,
} from '@/api/models'
import { useFormatters } from '@/hooks/useFormatters'
import { useQueryClient } from '@tanstack/react-query'

const { Title, Text } = Typography

// Type alias for tenant row with index signature for DataTable compatibility
type TenantRow = HandlerTenantResponse & Record<string, unknown>

/**
 * Get color for tenant status badge
 */
function getStatusColor(status: string | undefined): TagColor {
  switch (status) {
    case 'active':
      return 'green'
    case 'suspended':
      return 'red'
    case 'inactive':
      return 'grey'
    case 'trial':
      return 'orange'
    default:
      return 'grey'
  }
}

/**
 * Get color for tenant plan badge
 */
function getPlanColor(plan: string | undefined): TagColor {
  switch (plan) {
    case 'free':
      return 'grey'
    case 'basic':
      return 'blue'
    case 'pro':
      return 'purple'
    case 'enterprise':
      return 'amber'
    default:
      return 'grey'
  }
}

/**
 * Tenant List Page for Super Admin
 *
 * Displays all tenants with filtering, sorting, and management options.
 *
 * Features:
 * - Paginated list of tenants (20 per page)
 * - Search by name or email
 * - Filter by status (active/suspended/inactive/trial)
 * - Filter by plan (free/basic/pro/enterprise)
 * - Sort by created_at, name, status
 * - Quick actions: view, edit, suspend/activate
 * - Batch selection and operations
 * - Create new tenant button
 */
export default function TenantListPage() {
  const { t } = useTranslation('admin')
  const navigate = useNavigate()
  const { formatDateTime } = useFormatters()
  const queryClient = useQueryClient()

  // Status options for filter
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('tenants.filter.allStatus', 'All Status'), value: '' },
      { label: t('tenants.filter.active', 'Active'), value: 'active' },
      { label: t('tenants.filter.suspended', 'Suspended'), value: 'suspended' },
      { label: t('tenants.filter.inactive', 'Inactive'), value: 'inactive' },
      { label: t('tenants.filter.trial', 'Trial'), value: 'trial' },
    ],
    [t]
  )

  // Plan options for filter
  const PLAN_OPTIONS = useMemo(
    () => [
      { label: t('tenants.filter.allPlans', 'All Plans'), value: '' },
      { label: t('tenants.filter.free', 'Free'), value: 'free' },
      { label: t('tenants.filter.basic', 'Basic'), value: 'basic' },
      { label: t('tenants.filter.pro', 'Pro'), value: 'pro' },
      { label: t('tenants.filter.enterprise', 'Enterprise'), value: 'enterprise' },
    ],
    [t]
  )

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [planFilter, setPlanFilter] = useState<string>('')

  // Selection state for batch operations
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Build query params
  const queryParams: ListTenantsParams = useMemo(
    () => ({
      page: state.pagination.page,
      page_size: state.pagination.pageSize,
      keyword: searchKeyword || undefined,
      status: (statusFilter as ListTenantsStatus) || undefined,
      plan: (planFilter as ListTenantsPlan) || undefined,
      sort_by: (state.sort.field as ListTenantsSortBy) || 'created_at',
      sort_dir: (state.sort.order as ListTenantsSortDir) || 'desc',
    }),
    [state.pagination, state.sort, searchKeyword, statusFilter, planFilter]
  )

  // Fetch tenants using React Query
  const { data: tenantsResponse, isLoading, refetch } = useListTenants(queryParams)

  // Extract tenant data
  const tenants = useMemo(() => {
    if (tenantsResponse?.status === 200 && tenantsResponse.data.data) {
      return (tenantsResponse.data.data.tenants || []) as TenantRow[]
    }
    return []
  }, [tenantsResponse])

  const total = useMemo(() => {
    if (tenantsResponse?.status === 200 && tenantsResponse.data.data) {
      return tenantsResponse.data.data.total || 0
    }
    return 0
  }, [tenantsResponse])

  // Mutations for suspend/activate
  const suspendMutation = useSuspendTenant({
    mutation: {
      onSuccess: () => {
        Toast.success(t('tenants.messages.suspendSuccess', 'Tenant suspended successfully'))
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey(queryParams) })
        setSelectedRowKeys([])
      },
      onError: () => {
        Toast.error(t('tenants.messages.suspendError', 'Failed to suspend tenant'))
      },
    },
  })

  const activateMutation = useActivateTenant({
    mutation: {
      onSuccess: () => {
        Toast.success(t('tenants.messages.activateSuccess', 'Tenant activated successfully'))
        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey(queryParams) })
        setSelectedRowKeys([])
      },
      onError: () => {
        Toast.error(t('tenants.messages.activateError', 'Failed to activate tenant'))
      },
    },
  })

  // Handle search
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
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

  // Handle plan filter change
  const handlePlanChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const planValue = typeof value === 'string' ? value : ''
      setPlanFilter(planValue)
      setFilter('plan', planValue || null)
    },
    [setFilter]
  )

  // Handle view tenant detail
  const handleViewDetail = useCallback(
    (tenant: TenantRow) => {
      navigate(`/super-admin/tenants/${tenant.id}`)
    },
    [navigate]
  )

  // Handle edit tenant
  const handleEdit = useCallback(
    (tenant: TenantRow) => {
      navigate(`/super-admin/tenants/${tenant.id}/edit`)
    },
    [navigate]
  )

  // Handle suspend tenant
  const handleSuspend = useCallback(
    (tenant: TenantRow) => {
      if (!tenant.id) {
        Toast.error(t('tenants.messages.invalidTenant', 'Invalid tenant ID'))
        return
      }

      Modal.confirm({
        title: t('tenants.confirm.suspendTitle', 'Confirm Suspend'),
        content: t(
          'tenants.confirm.suspendContent',
          'Are you sure you want to suspend tenant "{{name}}"? Users will not be able to access the system.',
          { name: tenant.name }
        ),
        okText: t('tenants.confirm.suspendOk', 'Suspend'),
        cancelText: t('common.cancel', 'Cancel'),
        okButtonProps: { type: 'danger' },
        onOk: () => {
          suspendMutation.mutate({
            id: tenant.id!,
            data: { reason: 'Admin action' },
          })
        },
      })
    },
    [t, suspendMutation]
  )

  // Handle activate tenant
  const handleActivate = useCallback(
    (tenant: TenantRow) => {
      if (!tenant.id) {
        Toast.error(t('tenants.messages.invalidTenant', 'Invalid tenant ID'))
        return
      }

      Modal.confirm({
        title: t('tenants.confirm.activateTitle', 'Confirm Activate'),
        content: t(
          'tenants.confirm.activateContent',
          'Are you sure you want to activate tenant "{{name}}"?',
          { name: tenant.name }
        ),
        okText: t('tenants.confirm.activateOk', 'Activate'),
        cancelText: t('common.cancel', 'Cancel'),
        onOk: () => {
          activateMutation.mutate({
            id: tenant.id!,
            data: { reason: 'Admin action' },
          })
        },
      })
    },
    [t, activateMutation]
  )

  // Handle create tenant
  const handleCreate = useCallback(() => {
    navigate('/super-admin/tenants/new')
  }, [navigate])

  // Refresh handler
  const handleRefresh = useCallback(() => {
    refetch()
  }, [refetch])

  // Handle batch suspend
  const handleBatchSuspend = useCallback(() => {
    if (selectedRowKeys.length === 0) return

    Modal.confirm({
      title: t('tenants.confirm.batchSuspendTitle', 'Confirm Batch Suspend'),
      content: t(
        'tenants.confirm.batchSuspendContent',
        'Are you sure you want to suspend {{count}} selected tenants?',
        { count: selectedRowKeys.length }
      ),
      okText: t('tenants.confirm.suspendOk', 'Suspend'),
      cancelText: t('common.cancel', 'Cancel'),
      okButtonProps: { type: 'danger' },
      onOk: async () => {
        const results = await Promise.allSettled(
          selectedRowKeys.map((id) =>
            suspendMutation.mutateAsync({ id, data: { reason: 'Batch admin action' } })
          )
        )

        const failures = results.filter((r) => r.status === 'rejected')
        if (failures.length > 0) {
          Toast.error(
            t('tenants.messages.batchPartialError', '{{failed}} of {{total}} operations failed', {
              failed: failures.length,
              total: selectedRowKeys.length,
            })
          )
        }

        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey(queryParams) })
        setSelectedRowKeys([])
      },
    })
  }, [selectedRowKeys, t, suspendMutation, queryClient, queryParams])

  // Handle batch activate
  const handleBatchActivate = useCallback(() => {
    if (selectedRowKeys.length === 0) return

    Modal.confirm({
      title: t('tenants.confirm.batchActivateTitle', 'Confirm Batch Activate'),
      content: t(
        'tenants.confirm.batchActivateContent',
        'Are you sure you want to activate {{count}} selected tenants?',
        { count: selectedRowKeys.length }
      ),
      okText: t('tenants.confirm.activateOk', 'Activate'),
      cancelText: t('common.cancel', 'Cancel'),
      onOk: async () => {
        const results = await Promise.allSettled(
          selectedRowKeys.map((id) =>
            activateMutation.mutateAsync({ id, data: { reason: 'Batch admin action' } })
          )
        )

        const failures = results.filter((r) => r.status === 'rejected')
        if (failures.length > 0) {
          Toast.error(
            t('tenants.messages.batchPartialError', '{{failed}} of {{total}} operations failed', {
              failed: failures.length,
              total: selectedRowKeys.length,
            })
          )
        }

        queryClient.invalidateQueries({ queryKey: getListTenantsQueryKey(queryParams) })
        setSelectedRowKeys([])
      },
    })
  }, [selectedRowKeys, t, activateMutation, queryClient, queryParams])

  // Table columns
  const tableColumns: DataTableColumn<TenantRow>[] = useMemo(
    () => [
      {
        title: t('tenants.name', 'Name'),
        dataIndex: 'name',
        width: 200,
        sortable: true,
        render: (name: unknown, record: TenantRow) => (
          <Text link onClick={() => handleViewDetail(record)} style={{ cursor: 'pointer' }}>
            {name as string}
          </Text>
        ),
      },
      {
        title: t('tenants.email', 'Email'),
        dataIndex: 'contact_email',
        width: 200,
        ellipsis: true,
        render: (email: unknown) => (email as string) || '-',
      },
      {
        title: t('tenants.plan', 'Plan'),
        dataIndex: 'plan',
        width: 100,
        align: 'center',
        render: (plan: unknown) => {
          const planStr = plan as string
          return (
            <Tag color={getPlanColor(planStr)} size="small">
              {planStr || '-'}
            </Tag>
          )
        },
      },
      {
        title: t('tenants.status', 'Status'),
        dataIndex: 'status',
        width: 100,
        align: 'center',
        sortable: true,
        render: (status: unknown) => {
          const statusStr = status as string
          return (
            <Tag color={getStatusColor(statusStr)} size="small">
              {statusStr || '-'}
            </Tag>
          )
        },
      },
      {
        title: t('tenants.createdAt', 'Created'),
        dataIndex: 'created_at',
        width: 160,
        sortable: true,
        render: (date: unknown) => (date ? formatDateTime(date as string) : '-'),
      },
      {
        title: t('tenants.actions', 'Actions'),
        dataIndex: 'actions',
        width: 120,
        align: 'center',
        render: (_: unknown, record: TenantRow) => (
          <Dropdown
            trigger="click"
            position="bottomRight"
            render={
              <Dropdown.Menu>
                <Dropdown.Item onClick={() => handleViewDetail(record)}>
                  <IconEyeOpened style={{ marginRight: 8 }} />
                  {t('common.view', 'View')}
                </Dropdown.Item>
                <Dropdown.Item onClick={() => handleEdit(record)}>
                  <IconEdit style={{ marginRight: 8 }} />
                  {t('common.edit', 'Edit')}
                </Dropdown.Item>
                <Dropdown.Divider />
                {record.status === 'suspended' ? (
                  <Dropdown.Item onClick={() => handleActivate(record)}>
                    <IconPlay style={{ marginRight: 8 }} />
                    {t('tenants.activate', 'Activate')}
                  </Dropdown.Item>
                ) : (
                  <Dropdown.Item type="danger" onClick={() => handleSuspend(record)}>
                    <IconStop style={{ marginRight: 8 }} />
                    {t('tenants.suspend', 'Suspend')}
                  </Dropdown.Item>
                )}
              </Dropdown.Menu>
            }
          >
            <Button icon={<IconMore />} theme="borderless" />
          </Dropdown>
        ),
      },
    ],
    [t, formatDateTime, handleViewDetail, handleEdit, handleActivate, handleSuspend]
  )

  // Bulk actions
  const bulkActions: BulkAction[] = useMemo(
    () => [
      {
        key: 'suspend',
        label: t('tenants.batchSuspend', 'Suspend Selected'),
        icon: <IconStop />,
        type: 'danger',
        onClick: handleBatchSuspend,
      },
      {
        key: 'activate',
        label: t('tenants.batchActivate', 'Activate Selected'),
        icon: <IconPlay />,
        onClick: handleBatchActivate,
      },
    ],
    [t, handleBatchSuspend, handleBatchActivate]
  )

  return (
    <Container size="full" className="tenant-list-page">
      <Card>
        <div style={{ marginBottom: 16 }}>
          <Title heading={4} style={{ margin: 0 }}>
            {t('tenants.title', 'Tenant Management')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('tenants.searchPlaceholder', 'Search tenants...')}
          primaryAction={{
            label: t('tenants.create', 'Create Tenant'),
            icon: <IconPlus />,
            onClick: handleCreate,
          }}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('common.refresh', 'Refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space>
              <Select
                placeholder={t('tenants.filter.status', 'Status')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('tenants.filter.plan', 'Plan')}
                value={planFilter}
                onChange={handlePlanChange}
                optionList={PLAN_OPTIONS}
                style={{ width: 120 }}
              />
            </Space>
          }
        />

        <DataTable<TenantRow>
          data={tenants}
          columns={tableColumns}
          rowKey="id"
          loading={isLoading}
          pagination={{
            page: state.pagination.page,
            page_size: state.pagination.pageSize,
            total,
            total_pages: Math.ceil(total / state.pagination.pageSize),
          }}
          onStateChange={handleStateChange}
          sortState={state.sort}
          scroll={{ x: 1000 }}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys as string[]),
          }}
          bulkActions={bulkActions}
        />
      </Card>
    </Container>
  )
}
