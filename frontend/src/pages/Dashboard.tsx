import { Card, Typography, Space, Button, Row, Col } from '@douyinfe/semi-ui'
import {
  IconHome,
  IconGridView,
  IconUserGroup,
  IconList,
  IconSend,
  IconPriceTag,
} from '@douyinfe/semi-icons'
import { useUser } from '@/store'

const { Title, Paragraph, Text } = Typography

/**
 * Dashboard/Home page placeholder
 * Will be replaced with actual dashboard implementation in P5-FE-006
 */
export default function DashboardPage() {
  const user = useUser()

  // Placeholder statistics
  const stats = [
    {
      key: 'products',
      label: 'Products',
      value: '1,234',
      icon: <IconGridView />,
      color: 'var(--semi-color-primary)',
    },
    {
      key: 'customers',
      label: 'Customers',
      value: '567',
      icon: <IconUserGroup />,
      color: 'var(--semi-color-success)',
    },
    {
      key: 'inventory',
      label: 'Stock Items',
      value: '8,901',
      icon: <IconList />,
      color: 'var(--semi-color-warning)',
    },
    {
      key: 'orders',
      label: 'Orders Today',
      value: '42',
      icon: <IconSend />,
      color: 'var(--semi-color-info)',
    },
    {
      key: 'revenue',
      label: 'Revenue Today',
      value: '\xa512,345',
      icon: <IconPriceTag />,
      color: 'var(--semi-color-danger)',
    },
  ]

  return (
    <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
      {/* Welcome card */}
      <Card style={{ width: '100%' }}>
        <Space vertical align="start" spacing="tight">
          <Space>
            <IconHome size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={2} style={{ margin: 0 }}>
              Welcome back, {user?.displayName || user?.username || 'User'}!
            </Title>
          </Space>
          <Paragraph type="secondary">
            Here&apos;s an overview of your business today. This dashboard will display key metrics
            and quick actions.
          </Paragraph>
        </Space>
      </Card>

      {/* Statistics cards */}
      <Row gutter={[16, 16]} style={{ width: '100%' }}>
        {stats.map((stat) => (
          <Col key={stat.key} xs={24} sm={12} md={8} lg={6} xl={4}>
            <Card>
              <Space>
                <div
                  style={{
                    width: 48,
                    height: 48,
                    borderRadius: 8,
                    backgroundColor: stat.color + '20',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    color: stat.color,
                    fontSize: 24,
                  }}
                >
                  {stat.icon}
                </div>
                <div>
                  <Text type="tertiary" size="small">
                    {stat.label}
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

      {/* Quick actions */}
      <Card title="Quick Actions" style={{ width: '100%' }}>
        <Space>
          <Button type="primary" icon={<IconSend />}>
            New Sale
          </Button>
          <Button type="secondary" icon={<IconUserGroup />}>
            Add Customer
          </Button>
          <Button type="tertiary" icon={<IconList />}>
            View Inventory
          </Button>
        </Space>
      </Card>
    </Space>
  )
}
