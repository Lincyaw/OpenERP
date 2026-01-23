import { Card, Typography, Space, Tag } from '@douyinfe/semi-ui'
import { IconDownload } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * Purchase orders page placeholder
 * Will be implemented in P3-FE-010
 */
export default function PurchaseOrdersPage() {
  return (
    <div style={{ padding: '24px' }}>
      <Card style={{ width: '100%' }}>
        <Space vertical align="start" spacing="tight">
          <Space>
            <IconDownload size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={2} style={{ margin: 0 }}>
              Purchase Orders
            </Title>
            <Tag color="purple">Trade</Tag>
          </Space>
          <Paragraph type="secondary">
            Purchase order management page. Features will include order listing, receiving, and
            order creation.
          </Paragraph>
        </Space>
      </Card>
    </div>
  )
}
