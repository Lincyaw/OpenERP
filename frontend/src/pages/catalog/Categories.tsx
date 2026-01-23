import { Card, Typography, Space, Tag } from '@douyinfe/semi-ui'
import { IconTreeTriangleDown } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * Categories page placeholder
 * Will be implemented in P1-FE-003
 */
export default function CategoriesPage() {
  return (
    <div style={{ padding: '24px' }}>
      <Card style={{ width: '100%' }}>
        <Space vertical align="start" spacing="tight">
          <Space>
            <IconTreeTriangleDown
              size="extra-large"
              style={{ color: 'var(--semi-color-primary)' }}
            />
            <Title heading={2} style={{ margin: 0 }}>
              Categories
            </Title>
            <Tag color="blue">Catalog</Tag>
          </Space>
          <Paragraph type="secondary">
            Category management page. Features will include tree structure display and category CRUD
            operations.
          </Paragraph>
        </Space>
      </Card>
    </div>
  )
}
