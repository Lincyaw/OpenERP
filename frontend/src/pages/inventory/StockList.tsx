import { Card, Typography, Space, Tag } from '@douyinfe/semi-ui'
import { IconList } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * Stock list page placeholder
 * Will be implemented in P2-FE-001
 */
export default function StockListPage() {
  return (
    <div style={{ padding: '24px' }}>
      <Card style={{ width: '100%' }}>
        <Space vertical align="start" spacing="tight">
          <Space>
            <IconList size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={2} style={{ margin: 0 }}>
              Stock List
            </Title>
            <Tag color="orange">Inventory</Tag>
          </Space>
          <Paragraph type="secondary">
            Inventory listing page. Features will include stock levels, batch information, and
            filtering.
          </Paragraph>
        </Space>
      </Card>
    </div>
  )
}
