import { Card, Typography, Space, Tag } from '@douyinfe/semi-ui'
import { IconUserGroup } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * Customers page placeholder
 * Will be implemented in P1-FE-004
 */
export default function CustomersPage() {
  return (
    <div style={{ padding: '24px' }}>
      <Card style={{ width: '100%' }}>
        <Space vertical align="start" spacing="tight">
          <Space>
            <IconUserGroup size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={2} style={{ margin: 0 }}>
              Customers
            </Title>
            <Tag color="green">Partner</Tag>
          </Space>
          <Paragraph type="secondary">
            Customer management page. Features will include customer listing, balance tracking, and
            CRUD operations.
          </Paragraph>
        </Space>
      </Card>
    </div>
  )
}
