import { Card, Typography, Space, Tag } from '@douyinfe/semi-ui'
import { IconInbox } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * Warehouses page placeholder
 * Will be implemented in P1-FE-008
 */
export default function WarehousesPage() {
  return (
    <div style={{ padding: '24px' }}>
      <Card style={{ width: '100%' }}>
        <Space vertical align="start" spacing="tight">
          <Space>
            <IconInbox size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={2} style={{ margin: 0 }}>
              Warehouses
            </Title>
            <Tag color="green">Partner</Tag>
          </Space>
          <Paragraph type="secondary">
            Warehouse management page. Features will include warehouse listing and CRUD operations.
          </Paragraph>
        </Space>
      </Card>
    </div>
  )
}
