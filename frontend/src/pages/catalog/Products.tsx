import { Card, Typography, Space, Tag } from '@douyinfe/semi-ui'
import { IconGridView } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * Products list page placeholder
 * Will be implemented in P1-FE-001
 */
export default function ProductsPage() {
  return (
    <div style={{ padding: '24px' }}>
      <Card style={{ width: '100%' }}>
        <Space vertical align="start" spacing="tight">
          <Space>
            <IconGridView size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={2} style={{ margin: 0 }}>
              Products
            </Title>
            <Tag color="blue">Catalog</Tag>
          </Space>
          <Paragraph type="secondary">
            Product management page. Features will include product listing, filtering, and CRUD
            operations.
          </Paragraph>
        </Space>
      </Card>
    </div>
  )
}
