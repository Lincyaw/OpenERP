import { Card, Typography, Space, Tag } from '@douyinfe/semi-ui'
import { IconSend } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * Sales orders page placeholder
 * Will be implemented in P3-FE-001
 */
export default function SalesOrdersPage() {
  return (
    <div style={{ padding: '24px' }}>
      <Card style={{ width: '100%' }}>
        <Space vertical align="start" spacing="tight">
          <Space>
            <IconSend size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={2} style={{ margin: 0 }}>
              Sales Orders
            </Title>
            <Tag color="purple">Trade</Tag>
          </Space>
          <Paragraph type="secondary">
            Sales order management page. Features will include order listing, status tracking, and
            order creation.
          </Paragraph>
        </Space>
      </Card>
    </div>
  )
}
