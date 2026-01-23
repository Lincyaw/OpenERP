import { Card, Typography, Space, Button } from '@douyinfe/semi-ui'
import { IconHome } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * Dashboard/Home page placeholder
 * Will be replaced with actual dashboard implementation in P5-FE-006
 */
export default function DashboardPage() {
  return (
    <div style={{ padding: '24px' }}>
      <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
        <Card style={{ width: '100%' }}>
          <Space vertical align="start" spacing="tight">
            <Space>
              <IconHome size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
              <Title heading={2} style={{ margin: 0 }}>
                Dashboard
              </Title>
            </Space>
            <Paragraph type="secondary">
              Welcome to the ERP System. This dashboard will display key metrics and quick actions.
            </Paragraph>
          </Space>
        </Card>

        <Card title="Quick Actions" style={{ width: '100%' }}>
          <Space>
            <Button type="primary">New Sale</Button>
            <Button type="secondary">New Purchase</Button>
            <Button type="tertiary">View Inventory</Button>
          </Space>
        </Card>
      </Space>
    </div>
  )
}
