import { useState, useCallback, useMemo } from 'react'
import {
  Typography,
  Card,
  Descriptions,
  Button,
  Space,
  Tag,
  Tabs,
  TabPane,
  Progress,
  Toast,
  Modal,
  Spin,
  Empty,
  Timeline,
} from '@douyinfe/semi-ui-19'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import {
  IconArrowLeft,
  IconEdit,
  IconStop,
  IconPlay,
  IconDelete,
  IconRefresh,
  IconSetting,
  IconPriceTag,
} from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { Container } from '@/components/common/layout'
import {
  useGetTenantById,
  useActivateTenant,
  getGetTenantByIdQueryKey,
} from '@/api/tenants/tenants'
import type { HandlerTenantResponse } from '@/api/models'
import { useFormatters } from '@/hooks/useFormatters'
import {
  ChangePlanModal,
  UpdateQuotaModal,
  SuspendTenantModal,
  DeleteTenantModal,
} from '@/components/admin'

const { Title, Text } = Typography

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

// Mock status history data (will be replaced with API)
interface StatusHistoryItem {
  id: string
  action: 'suspended' | 'activated' | 'created' | 'deactivated'
  reason?: string
  timestamp: string
  actor: string
}

// Mock subscription history data (will be replaced with API)
interface SubscriptionHistoryItem {
  id: string
  fromPlan: string
  toPlan: string
  timestamp: string
  actor: string
}

/**
 * Tenant detail page for super admin
 *
 * Displays detailed information about a specific tenant including:
 * - Basic information (name, email, contact, created date)
 * - Subscription info (plan, quota usage, expiration)
 * - Usage statistics (users, orders, products, storage)
 * - Status history (suspend/activate records)
 * - Subscription history (plan changes)
 * - Action buttons (edit, suspend/activate, change plan, delete)
 * - Tab navigation (Overview, Users, Activity Log)
 */
export default function TenantDetailPage() {
  const { t } = useTranslation('admin')
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const { formatDateTime } = useFormatters()
  const queryClient = useQueryClient()

  // State for modals
  const [changePlanModalVisible, setChangePlanModalVisible] = useState(false)
  const [updateQuotaModalVisible, setUpdateQuotaModalVisible] = useState(false)
  const [suspendModalVisible, setSuspendModalVisible] = useState(false)
  const [deleteModalVisible, setDeleteModalVisible] = useState(false)
  // Key counters to force modal remount on open (resets form state)
  const [changePlanModalKey, setChangePlanModalKey] = useState(0)
  const [updateQuotaModalKey, setUpdateQuotaModalKey] = useState(0)
  const [suspendModalKey, setSuspendModalKey] = useState(0)

  // Fetch tenant data
  const {
    data: tenantResponse,
    isLoading,
    isError,
    refetch,
  } = useGetTenantById(id || '', {
    query: {
      enabled: !!id,
    },
  })

  // Extract tenant data
  const tenant: HandlerTenantResponse | undefined = useMemo(() => {
    if (tenantResponse?.status === 200 && tenantResponse.data.data) {
      return tenantResponse.data.data
    }
    return undefined
  }, [tenantResponse])

  // Mutations
  const activateMutation = useActivateTenant({
    mutation: {
      onSuccess: () => {
        Toast.success(t('tenants.messages.activateSuccess', 'Tenant activated successfully'))
        queryClient.invalidateQueries({ queryKey: getGetTenantByIdQueryKey(id || '') })
      },
      onError: () => {
        Toast.error(t('tenants.messages.activateError', 'Failed to activate tenant'))
      },
    },
  })

  // Note: Delete mutation is handled by DeleteTenantModal component

  // Mock data for status history (will be replaced with API)
  const statusHistory: StatusHistoryItem[] = useMemo(
    () => [
      {
        id: '1',
        action: 'created',
        timestamp: tenant?.created_at || '',
        actor: 'System',
      },
      {
        id: '2',
        action: 'activated',
        timestamp: tenant?.created_at || '',
        actor: 'System',
      },
    ],
    [tenant?.created_at]
  )

  // Mock data for subscription history (will be replaced with API)
  const subscriptionHistory: SubscriptionHistoryItem[] = useMemo(
    () => [
      {
        id: '1',
        fromPlan: 'free',
        toPlan: tenant?.plan || 'free',
        timestamp: tenant?.created_at || '',
        actor: 'System',
      },
    ],
    [tenant?.plan, tenant?.created_at]
  )

  // Handlers
  const handleBack = useCallback(() => {
    navigate('/super-admin/tenants')
  }, [navigate])

  const handleEdit = useCallback(() => {
    navigate(`/super-admin/tenants/${id}/edit`)
  }, [navigate, id])

  const handleRefresh = useCallback(() => {
    refetch()
  }, [refetch])

  const handleSuspend = useCallback(() => {
    setSuspendModalKey((k) => k + 1)
    setSuspendModalVisible(true)
  }, [])

  const handleActivate = useCallback(() => {
    if (!id) return

    Modal.confirm({
      title: t('tenants.confirm.activateTitle', 'Confirm Activate'),
      content: t(
        'tenants.confirm.activateContent',
        'Are you sure you want to activate tenant "{{name}}"?',
        {
          name: tenant?.name,
        }
      ),
      okText: t('tenants.confirm.activateOk', 'Activate'),
      cancelText: t('common.cancel', 'Cancel'),
      onOk: () => {
        activateMutation.mutate({ id, data: { reason: 'Admin action' } })
      },
    })
  }, [id, tenant?.name, t, activateMutation])

  const handleDelete = useCallback(() => {
    setDeleteModalVisible(true)
  }, [])

  const handleOpenChangePlanModal = useCallback(() => {
    setChangePlanModalKey((k) => k + 1)
    setChangePlanModalVisible(true)
  }, [])

  const handleOpenUpdateQuotaModal = useCallback(() => {
    setUpdateQuotaModalKey((k) => k + 1)
    setUpdateQuotaModalVisible(true)
  }, [])

  // Calculate quota usage percentages (mock data - will be replaced with actual usage API)
  const quotaUsage = useMemo(() => {
    const config = tenant?.config
    // Mock current usage values - in production these would come from a usage API
    const mockUsage = {
      users: 5,
      products: 150,
      warehouses: 2,
    }

    // Helper to safely calculate percentage (avoid division by zero)
    const calcPercent = (current: number, max: number | undefined): number => {
      if (!max || max <= 0) return 0
      return Math.min(100, Math.round((current / max) * 100))
    }

    return {
      users: {
        current: mockUsage.users,
        max: config?.max_users || 10,
        percent: calcPercent(mockUsage.users, config?.max_users || 10),
      },
      products: {
        current: mockUsage.products,
        max: config?.max_products || 1000,
        percent: calcPercent(mockUsage.products, config?.max_products || 1000),
      },
      warehouses: {
        current: mockUsage.warehouses,
        max: config?.max_warehouses || 5,
        percent: calcPercent(mockUsage.warehouses, config?.max_warehouses || 5),
      },
    }
  }, [tenant?.config])

  // Basic info descriptions data
  const basicInfoData = useMemo(
    () => [
      { key: t('tenants.detail.name', 'Name'), value: tenant?.name || '-' },
      { key: t('tenants.detail.code', 'Code'), value: tenant?.code || '-' },
      { key: t('tenants.detail.shortName', 'Short Name'), value: tenant?.short_name || '-' },
      {
        key: t('tenants.detail.contactEmail', 'Contact Email'),
        value: tenant?.contact_email || '-',
      },
      { key: t('tenants.detail.contactName', 'Contact Name'), value: tenant?.contact_name || '-' },
      {
        key: t('tenants.detail.contactPhone', 'Contact Phone'),
        value: tenant?.contact_phone || '-',
      },
      { key: t('tenants.detail.address', 'Address'), value: tenant?.address || '-' },
      { key: t('tenants.detail.domain', 'Domain'), value: tenant?.domain || '-' },
      {
        key: t('tenants.detail.createdAt', 'Created At'),
        value: tenant?.created_at ? formatDateTime(tenant.created_at) : '-',
      },
      {
        key: t('tenants.detail.updatedAt', 'Updated At'),
        value: tenant?.updated_at ? formatDateTime(tenant.updated_at) : '-',
      },
    ],
    [tenant, t, formatDateTime]
  )

  // Subscription info descriptions data
  const subscriptionInfoData = useMemo(
    () => [
      {
        key: t('tenants.detail.plan', 'Plan'),
        value: (
          <Tag color={getPlanColor(tenant?.plan)} size="small">
            {tenant?.plan || '-'}
          </Tag>
        ),
      },
      {
        key: t('tenants.detail.status', 'Status'),
        value: (
          <Tag color={getStatusColor(tenant?.status)} size="small">
            {tenant?.status || '-'}
          </Tag>
        ),
      },
      {
        key: t('tenants.detail.expiresAt', 'Expires At'),
        value: tenant?.expires_at
          ? formatDateTime(tenant.expires_at)
          : t('tenants.detail.noExpiration', 'No expiration'),
      },
      {
        key: t('tenants.detail.trialEndsAt', 'Trial Ends At'),
        value: tenant?.trial_ends_at ? formatDateTime(tenant.trial_ends_at) : '-',
      },
    ],
    [tenant, t, formatDateTime]
  )

  // Config info descriptions data
  const configInfoData = useMemo(
    () => [
      { key: t('tenants.detail.currency', 'Currency'), value: tenant?.config?.currency || '-' },
      { key: t('tenants.detail.locale', 'Locale'), value: tenant?.config?.locale || '-' },
      { key: t('tenants.detail.timezone', 'Timezone'), value: tenant?.config?.timezone || '-' },
      {
        key: t('tenants.detail.costStrategy', 'Cost Strategy'),
        value: tenant?.config?.cost_strategy || '-',
      },
    ],
    [tenant?.config, t]
  )

  // Loading state
  if (isLoading) {
    return (
      <Container size="full" className="tenant-detail-page">
        <Card>
          <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}>
            <Spin size="large" />
          </div>
        </Card>
      </Container>
    )
  }

  // Error state
  if (isError || !tenant) {
    return (
      <Container size="full" className="tenant-detail-page">
        <Card>
          <Empty
            title={t('tenants.detail.notFound', 'Tenant not found')}
            description={t(
              'tenants.detail.notFoundDescription',
              'The requested tenant does not exist'
            )}
          >
            <Button onClick={handleBack}>{t('tenants.detail.backToList', 'Back to list')}</Button>
          </Empty>
        </Card>
      </Container>
    )
  }

  return (
    <Container size="full" className="tenant-detail-page">
      <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
        {/* Header */}
        <Card style={{ width: '100%' }}>
          <Space
            style={{ width: '100%', justifyContent: 'space-between', flexWrap: 'wrap', gap: 16 }}
          >
            <Space>
              <Button icon={<IconArrowLeft />} onClick={handleBack}>
                {t('common.back', 'Back')}
              </Button>
              <Title heading={4} style={{ margin: 0 }}>
                {tenant.name}
              </Title>
              <Tag color={getStatusColor(tenant.status)} size="large">
                {tenant.status}
              </Tag>
              <Tag color={getPlanColor(tenant.plan)} size="large">
                {tenant.plan}
              </Tag>
            </Space>
            <Space>
              <Button icon={<IconRefresh />} onClick={handleRefresh}>
                {t('common.refresh', 'Refresh')}
              </Button>
              <Button icon={<IconEdit />} onClick={handleEdit}>
                {t('common.edit', 'Edit')}
              </Button>
              <Button icon={<IconSetting />} onClick={handleOpenChangePlanModal}>
                {t('tenants.actions.changePlan', 'Change Plan')}
              </Button>
              <Button icon={<IconPriceTag />} onClick={handleOpenUpdateQuotaModal}>
                {t('tenants.actions.updateQuota', 'Update Quota')}
              </Button>
              {tenant.status === 'suspended' ? (
                <Button
                  icon={<IconPlay />}
                  onClick={handleActivate}
                  loading={activateMutation.isPending}
                >
                  {t('tenants.activate', 'Activate')}
                </Button>
              ) : (
                <Button icon={<IconStop />} type="danger" onClick={handleSuspend}>
                  {t('tenants.suspend', 'Suspend')}
                </Button>
              )}
              <Button icon={<IconDelete />} type="danger" onClick={handleDelete}>
                {t('common.delete', 'Delete')}
              </Button>
            </Space>
          </Space>
        </Card>

        {/* Tabs */}
        <Tabs type="line" style={{ width: '100%' }}>
          {/* Overview Tab */}
          <TabPane tab={t('tenants.tabs.overview', 'Overview')} itemKey="overview">
            <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
              {/* Basic Information */}
              <Card
                title={t('tenants.detail.basicInfo', 'Basic Information')}
                style={{ width: '100%' }}
              >
                <Descriptions data={basicInfoData} />
              </Card>

              {/* Subscription Information */}
              <Card
                title={t('tenants.detail.subscriptionInfo', 'Subscription Information')}
                style={{ width: '100%' }}
              >
                <Descriptions data={subscriptionInfoData} />
              </Card>

              {/* Quota Usage */}
              <Card title={t('tenants.detail.quotaUsage', 'Quota Usage')} style={{ width: '100%' }}>
                <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
                  <div style={{ width: '100%' }}>
                    <Text>{t('tenants.detail.users', 'Users')}</Text>
                    <Progress
                      percent={quotaUsage.users.percent}
                      showInfo
                      format={() => `${quotaUsage.users.current} / ${quotaUsage.users.max}`}
                      style={{ marginTop: 8 }}
                    />
                  </div>
                  <div style={{ width: '100%' }}>
                    <Text>{t('tenants.detail.products', 'Products')}</Text>
                    <Progress
                      percent={quotaUsage.products.percent}
                      showInfo
                      format={() => `${quotaUsage.products.current} / ${quotaUsage.products.max}`}
                      style={{ marginTop: 8 }}
                    />
                  </div>
                  <div style={{ width: '100%' }}>
                    <Text>{t('tenants.detail.warehouses', 'Warehouses')}</Text>
                    <Progress
                      percent={quotaUsage.warehouses.percent}
                      showInfo
                      format={() =>
                        `${quotaUsage.warehouses.current} / ${quotaUsage.warehouses.max}`
                      }
                      style={{ marginTop: 8 }}
                    />
                  </div>
                </Space>
              </Card>

              {/* Configuration */}
              <Card
                title={t('tenants.detail.configuration', 'Configuration')}
                style={{ width: '100%' }}
              >
                <Descriptions data={configInfoData} />
              </Card>

              {/* Status History */}
              <Card
                title={t('tenants.detail.statusHistory', 'Status History')}
                style={{ width: '100%' }}
              >
                {statusHistory.length > 0 ? (
                  <Timeline>
                    {statusHistory.map((item) => (
                      <Timeline.Item key={item.id}>
                        <Space vertical align="start">
                          <Text strong>{t(`tenants.history.${item.action}`, item.action)}</Text>
                          <Text type="secondary">
                            {item.timestamp ? formatDateTime(item.timestamp) : '-'} •{' '}
                            {t('tenants.history.by', 'by')} {item.actor}
                          </Text>
                          {item.reason && <Text type="tertiary">{item.reason}</Text>}
                        </Space>
                      </Timeline.Item>
                    ))}
                  </Timeline>
                ) : (
                  <Empty description={t('tenants.detail.noStatusHistory', 'No status history')} />
                )}
              </Card>

              {/* Subscription History */}
              <Card
                title={t('tenants.detail.subscriptionHistory', 'Subscription History')}
                style={{ width: '100%' }}
              >
                {subscriptionHistory.length > 0 ? (
                  <Timeline>
                    {subscriptionHistory.map((item) => (
                      <Timeline.Item key={item.id}>
                        <Space vertical align="start">
                          <Space>
                            <Tag color={getPlanColor(item.fromPlan)} size="small">
                              {item.fromPlan}
                            </Tag>
                            <Text>→</Text>
                            <Tag color={getPlanColor(item.toPlan)} size="small">
                              {item.toPlan}
                            </Tag>
                          </Space>
                          <Text type="secondary">
                            {item.timestamp ? formatDateTime(item.timestamp) : '-'} •{' '}
                            {t('tenants.history.by', 'by')} {item.actor}
                          </Text>
                        </Space>
                      </Timeline.Item>
                    ))}
                  </Timeline>
                ) : (
                  <Empty
                    description={t(
                      'tenants.detail.noSubscriptionHistory',
                      'No subscription history'
                    )}
                  />
                )}
              </Card>
            </Space>
          </TabPane>

          {/* Users Tab */}
          <TabPane tab={t('tenants.tabs.users', 'Users')} itemKey="users">
            <Card>
              <Empty
                description={t('tenants.usersPlaceholder', 'User management coming soon...')}
              />
            </Card>
          </TabPane>

          {/* Activity Log Tab */}
          <TabPane tab={t('tenants.tabs.activityLog', 'Activity Log')} itemKey="activity">
            <Card>
              <Empty
                description={t('tenants.activityPlaceholder', 'Activity log coming soon...')}
              />
            </Card>
          </TabPane>
        </Tabs>
      </Space>

      {/* Change Plan Modal */}
      <ChangePlanModal
        key={`change-plan-${changePlanModalKey}`}
        visible={changePlanModalVisible}
        tenantId={id || null}
        tenantName={tenant.name}
        currentPlan={tenant.plan}
        onClose={() => setChangePlanModalVisible(false)}
      />

      {/* Update Quota Modal */}
      <UpdateQuotaModal
        key={`update-quota-${updateQuotaModalKey}`}
        visible={updateQuotaModalVisible}
        tenantId={id || null}
        tenantName={tenant.name}
        onClose={() => setUpdateQuotaModalVisible(false)}
      />

      {/* Suspend Tenant Modal */}
      <SuspendTenantModal
        key={`suspend-${suspendModalKey}`}
        visible={suspendModalVisible}
        tenantId={id || null}
        tenantName={tenant.name}
        onClose={() => setSuspendModalVisible(false)}
      />

      {/* Delete Tenant Modal */}
      <DeleteTenantModal
        visible={deleteModalVisible}
        tenantId={id || null}
        tenantName={tenant.name}
        onClose={() => setDeleteModalVisible(false)}
        onSuccess={() => navigate('/super-admin/tenants')}
      />
    </Container>
  )
}
