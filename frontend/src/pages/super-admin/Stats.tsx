import { Typography, Card, Row, Col, Space } from '@douyinfe/semi-ui-19'
import { IconUserGroup, IconBox, IconCart } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'

const { Title, Text } = Typography

/**
 * Platform statistics page for super admin
 *
 * Displays platform-wide statistics and metrics
 */
export default function StatsPage() {
  const { t } = useTranslation()

  // Placeholder data - will be replaced with API call
  const stats = {
    totalTenants: 156,
    activeTenants: 142,
    totalUsers: 1250,
    totalOrders: 45000,
    monthlyRevenue: 125000,
    growthRate: 12.5,
  }

  const statCards = [
    {
      title: t('admin.stats.totalTenants', 'Total Tenants'),
      value: stats.totalTenants,
      icon: <IconUserGroup size="extra-large" />,
      color: 'var(--semi-color-primary)',
    },
    {
      title: t('admin.stats.activeTenants', 'Active Tenants'),
      value: stats.activeTenants,
      icon: <IconUserGroup size="extra-large" />,
      color: 'var(--semi-color-success)',
    },
    {
      title: t('admin.stats.totalUsers', 'Total Users'),
      value: stats.totalUsers.toLocaleString(),
      icon: <IconBox size="extra-large" />,
      color: 'var(--semi-color-info)',
    },
    {
      title: t('admin.stats.totalOrders', 'Total Orders'),
      value: stats.totalOrders.toLocaleString(),
      icon: <IconCart size="extra-large" />,
      color: 'var(--semi-color-warning)',
    },
  ]

  return (
    <div className="stats-page">
      <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
        <Title heading={4} style={{ margin: 0 }}>
          {t('admin.stats.title', 'Platform Statistics')}
        </Title>

        <Row gutter={[16, 16]} style={{ width: '100%' }}>
          {statCards.map((stat, index) => (
            <Col key={index} xs={24} sm={12} lg={6}>
              <Card>
                <Space>
                  <div
                    style={{
                      width: 48,
                      height: 48,
                      borderRadius: 8,
                      background: `${stat.color}20`,
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      color: stat.color,
                    }}
                  >
                    {stat.icon}
                  </div>
                  <div>
                    <Text type="secondary" size="small">
                      {stat.title}
                    </Text>
                    <Title heading={3} style={{ margin: 0 }}>
                      {stat.value}
                    </Title>
                  </div>
                </Space>
              </Card>
            </Col>
          ))}
        </Row>

        <Row gutter={[16, 16]} style={{ width: '100%' }}>
          <Col xs={24} lg={12}>
            <Card title={t('admin.stats.tenantGrowth', 'Tenant Growth')}>
              <div
                style={{
                  height: 300,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Text type="secondary">
                  {t('admin.stats.chartPlaceholder', 'Chart coming soon...')}
                </Text>
              </div>
            </Card>
          </Col>
          <Col xs={24} lg={12}>
            <Card title={t('admin.stats.planDistribution', 'Plan Distribution')}>
              <div
                style={{
                  height: 300,
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                }}
              >
                <Text type="secondary">
                  {t('admin.stats.chartPlaceholder', 'Chart coming soon...')}
                </Text>
              </div>
            </Card>
          </Col>
        </Row>
      </Space>
    </div>
  )
}
