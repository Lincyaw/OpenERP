import { Card, Typography, Space, Tag } from '@douyinfe/semi-ui'
import { IconUserCardVideo } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * Suppliers page placeholder
 * Will be implemented in P1-FE-006
 */
export default function SuppliersPage() {
  return (
    <div style={{ padding: '24px' }}>
      <Card style={{ width: '100%' }}>
        <Space vertical align="start" spacing="tight">
          <Space>
            <IconUserCardVideo size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={2} style={{ margin: 0 }}>
              Suppliers
            </Title>
            <Tag color="green">Partner</Tag>
          </Space>
          <Paragraph type="secondary">
            Supplier management page. Features will include supplier listing and CRUD operations.
          </Paragraph>
        </Space>
      </Card>
    </div>
  )
}
