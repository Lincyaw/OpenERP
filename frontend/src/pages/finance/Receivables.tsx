import { Card, Typography, Space, Tag } from '@douyinfe/semi-ui'
import { IconPriceTag } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * Receivables page placeholder
 * Will be implemented in P4-FE-001
 */
export default function ReceivablesPage() {
  return (
    <div style={{ padding: '24px' }}>
      <Card style={{ width: '100%' }}>
        <Space vertical align="start" spacing="tight">
          <Space>
            <IconPriceTag size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={2} style={{ margin: 0 }}>
              Accounts Receivable
            </Title>
            <Tag color="red">Finance</Tag>
          </Space>
          <Paragraph type="secondary">
            Accounts receivable management page. Features will include receivable tracking,
            collection, and reconciliation.
          </Paragraph>
        </Space>
      </Card>
    </div>
  )
}
