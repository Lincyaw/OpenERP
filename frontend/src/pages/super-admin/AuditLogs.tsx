import {
  Typography,
  Card,
  Table,
  Space,
  Tag,
  Input,
  DatePicker,
  Select,
  Button,
} from '@douyinfe/semi-ui-19'
import { IconSearch, IconRefresh } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'

const { Title } = Typography

/**
 * Audit logs page for super admin
 *
 * Displays admin action audit logs
 */
export default function AuditLogsPage() {
  const { t } = useTranslation()

  // Placeholder data - will be replaced with API call
  const logs = [
    {
      id: '1',
      action: 'tenant.create',
      actor: 'admin@system.com',
      target: 'Acme Corp',
      timestamp: '2024-02-01 10:30:00',
      status: 'success',
    },
    {
      id: '2',
      action: 'tenant.suspend',
      actor: 'admin@system.com',
      target: 'Small Business',
      timestamp: '2024-02-01 09:15:00',
      status: 'success',
    },
    {
      id: '3',
      action: 'tenant.plan_change',
      actor: 'admin@system.com',
      target: 'Tech Solutions',
      timestamp: '2024-01-31 16:45:00',
      status: 'success',
    },
    {
      id: '4',
      action: 'tenant.quota_update',
      actor: 'admin@system.com',
      target: 'Acme Corp',
      timestamp: '2024-01-31 14:20:00',
      status: 'failed',
    },
  ]

  const actionLabels: Record<string, string> = {
    'tenant.create': t('admin.audit.actions.tenantCreate', 'Create Tenant'),
    'tenant.suspend': t('admin.audit.actions.tenantSuspend', 'Suspend Tenant'),
    'tenant.activate': t('admin.audit.actions.tenantActivate', 'Activate Tenant'),
    'tenant.plan_change': t('admin.audit.actions.planChange', 'Change Plan'),
    'tenant.quota_update': t('admin.audit.actions.quotaUpdate', 'Update Quota'),
    'tenant.delete': t('admin.audit.actions.tenantDelete', 'Delete Tenant'),
  }

  const columns = [
    {
      title: t('admin.audit.timestamp', 'Timestamp'),
      dataIndex: 'timestamp',
      width: 180,
    },
    {
      title: t('admin.audit.action', 'Action'),
      dataIndex: 'action',
      render: (action: string) => <Tag>{actionLabels[action] || action}</Tag>,
    },
    {
      title: t('admin.audit.actor', 'Actor'),
      dataIndex: 'actor',
    },
    {
      title: t('admin.audit.target', 'Target'),
      dataIndex: 'target',
    },
    {
      title: t('admin.audit.status', 'Status'),
      dataIndex: 'status',
      render: (status: string) => (
        <Tag color={status === 'success' ? 'green' : 'red'}>{status}</Tag>
      ),
    },
  ]

  return (
    <div className="audit-logs-page">
      <Card>
        <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
          <Space style={{ width: '100%', justifyContent: 'space-between' }}>
            <Title heading={4} style={{ margin: 0 }}>
              {t('admin.audit.title', 'Audit Logs')}
            </Title>
            <Button icon={<IconRefresh />}>{t('common.refresh', 'Refresh')}</Button>
          </Space>

          <Space wrap>
            <Input
              prefix={<IconSearch />}
              placeholder={t('admin.audit.searchPlaceholder', 'Search by actor or target...')}
              style={{ width: 250 }}
              showClear
            />
            <Select
              placeholder={t('admin.audit.filterAction', 'Filter by action')}
              style={{ width: 180 }}
              optionList={Object.entries(actionLabels).map(([value, label]) => ({
                value,
                label,
              }))}
            />
            <DatePicker
              type="dateRange"
              placeholder={[
                t('admin.audit.startDate', 'Start date'),
                t('admin.audit.endDate', 'End date'),
              ]}
            />
          </Space>

          <Table
            columns={columns}
            dataSource={logs}
            rowKey="id"
            pagination={{ pageSize: 20 }}
            style={{ width: '100%' }}
          />
        </Space>
      </Card>
    </div>
  )
}
