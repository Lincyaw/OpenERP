import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import { Typography, Card, Table, Button, Space, Tag, Input } from '@douyinfe/semi-ui-19'
import { IconSearch, IconPlus, IconRefresh } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'

const { Title, Text } = Typography

/**
 * Tenant list page for super admin
 *
 * Displays all tenants with filtering and management options
 */
export default function TenantListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  // Placeholder data - will be replaced with API call
  const tenants = [
    {
      id: '1',
      name: 'Acme Corp',
      plan: 'professional',
      status: 'active',
      userCount: 15,
      createdAt: '2024-01-15',
    },
    {
      id: '2',
      name: 'Tech Solutions',
      plan: 'enterprise',
      status: 'active',
      userCount: 45,
      createdAt: '2024-02-20',
    },
    {
      id: '3',
      name: 'Small Business',
      plan: 'starter',
      status: 'suspended',
      userCount: 3,
      createdAt: '2024-03-10',
    },
  ]

  const columns = [
    {
      title: t('admin.tenants.name', 'Name'),
      dataIndex: 'name',
      render: (text: string, record: (typeof tenants)[0]) => (
        <Text
          link
          onClick={() => navigate(`/super-admin/tenants/${record.id}`)}
          style={{ cursor: 'pointer' }}
        >
          {text}
        </Text>
      ),
    },
    {
      title: t('admin.tenants.plan', 'Plan'),
      dataIndex: 'plan',
      render: (plan: string) => {
        const colors: Record<string, TagColor> = {
          starter: 'grey',
          professional: 'blue',
          enterprise: 'purple',
        }
        return <Tag color={colors[plan] || 'grey'}>{plan}</Tag>
      },
    },
    {
      title: t('admin.tenants.status', 'Status'),
      dataIndex: 'status',
      render: (status: string) => <Tag color={status === 'active' ? 'green' : 'red'}>{status}</Tag>,
    },
    {
      title: t('admin.tenants.users', 'Users'),
      dataIndex: 'userCount',
    },
    {
      title: t('admin.tenants.createdAt', 'Created'),
      dataIndex: 'createdAt',
    },
    {
      title: t('admin.tenants.actions', 'Actions'),
      render: (_: unknown, record: (typeof tenants)[0]) => (
        <Space>
          <Button size="small" onClick={() => navigate(`/super-admin/tenants/${record.id}`)}>
            {t('common.view', 'View')}
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div className="tenant-list-page">
      <Card>
        <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Title heading={4} style={{ margin: 0 }}>
              {t('admin.tenants.title', 'Tenant Management')}
            </Title>
            <Space>
              <Button icon={<IconRefresh />}>{t('common.refresh', 'Refresh')}</Button>
              <Button type="primary" icon={<IconPlus />}>
                {t('admin.tenants.create', 'Create Tenant')}
              </Button>
            </Space>
          </Space>

          <Input
            prefix={<IconSearch />}
            placeholder={t('admin.tenants.search', 'Search tenants...')}
            style={{ width: 300 }}
            showClear
          />

          <Table
            columns={columns}
            dataSource={tenants}
            rowKey="id"
            pagination={{ pageSize: 10 }}
            style={{ width: '100%' }}
          />
        </Space>
      </Card>
    </div>
  )
}
