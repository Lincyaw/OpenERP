import { Card, Typography, Space, Tag } from '@douyinfe/semi-ui'
import { IconCreditCard } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * Payables page placeholder
 * Will be implemented in P4-FE-002
 */
export default function PayablesPage() {
  return (
    <div style={{ padding: '24px' }}>
      <Card style={{ width: '100%' }}>
        <Space vertical align="start" spacing="tight">
          <Space>
            <IconCreditCard size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
            <Title heading={2} style={{ margin: 0 }}>
              Accounts Payable
            </Title>
            <Tag color="red">Finance</Tag>
          </Space>
          <Paragraph type="secondary">
            Accounts payable management page. Features will include payable tracking, payment, and
            reconciliation.
          </Paragraph>
        </Space>
      </Card>
    </div>
  )
}
