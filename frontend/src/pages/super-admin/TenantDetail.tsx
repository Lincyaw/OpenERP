import {
  Typography,
  Card,
  Descriptions,
  Button,
  Space,
  Tag,
  Tabs,
  TabPane,
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconEdit } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import { useNavigate, useParams } from 'react-router-dom'

const { Title, Text } = Typography

/**
 * Tenant detail page for super admin
 *
 * Displays detailed information about a specific tenant
 */
export default function TenantDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()

  // Placeholder data - will be replaced with API call
  const tenant = {
    id,
    name: 'Acme Corp',
    plan: 'professional',
    status: 'active',
    userCount: 15,
    storageUsed: '2.5 GB',
    apiCalls: 15000,
    createdAt: '2024-01-15',
    contactEmail: 'admin@acme.com',
    contactPhone: '+1 234 567 8900',
  }

  const descData = [
    { key: t('admin.tenants.name', 'Name'), value: tenant.name },
    {
      key: t('admin.tenants.plan', 'Plan'),
      value: <Tag color="blue">{tenant.plan}</Tag>,
    },
    {
      key: t('admin.tenants.status', 'Status'),
      value: <Tag color={tenant.status === 'active' ? 'green' : 'red'}>{tenant.status}</Tag>,
    },
    { key: t('admin.tenants.users', 'Users'), value: tenant.userCount },
    { key: t('admin.tenants.storage', 'Storage Used'), value: tenant.storageUsed },
    {
      key: t('admin.tenants.apiCalls', 'API Calls (30d)'),
      value: tenant.apiCalls.toLocaleString(),
    },
    { key: t('admin.tenants.createdAt', 'Created'), value: tenant.createdAt },
    { key: t('admin.tenants.contactEmail', 'Contact Email'), value: tenant.contactEmail },
    { key: t('admin.tenants.contactPhone', 'Contact Phone'), value: tenant.contactPhone },
  ]

  return (
    <div className="tenant-detail-page">
      <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
        <Space style={{ width: '100%', justifyContent: 'space-between' }}>
          <Space>
            <Button icon={<IconArrowLeft />} onClick={() => navigate('/super-admin/tenants')}>
              {t('common.back', 'Back')}
            </Button>
            <Title heading={4} style={{ margin: 0 }}>
              {tenant.name}
            </Title>
          </Space>
          <Button type="primary" icon={<IconEdit />}>
            {t('common.edit', 'Edit')}
          </Button>
        </Space>

        <Tabs type="line" style={{ width: '100%' }}>
          <TabPane tab={t('admin.tenants.overview', 'Overview')} itemKey="overview">
            <Card>
              <Descriptions data={descData} />
            </Card>
          </TabPane>

          <TabPane tab={t('admin.tenants.users', 'Users')} itemKey="users">
            <Card>
              <Text type="secondary">
                {t('admin.tenants.usersPlaceholder', 'User management coming soon...')}
              </Text>
            </Card>
          </TabPane>

          <TabPane tab={t('admin.tenants.subscription', 'Subscription')} itemKey="subscription">
            <Card>
              <Text type="secondary">
                {t('admin.tenants.subscriptionPlaceholder', 'Subscription history coming soon...')}
              </Text>
            </Card>
          </TabPane>

          <TabPane tab={t('admin.tenants.activity', 'Activity')} itemKey="activity">
            <Card>
              <Text type="secondary">
                {t('admin.tenants.activityPlaceholder', 'Activity log coming soon...')}
              </Text>
            </Card>
          </TabPane>
        </Tabs>
      </Space>
    </div>
  )
}
