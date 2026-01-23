import { useNavigate } from 'react-router-dom'
import { Button, Typography, Space } from '@douyinfe/semi-ui'
import { IconLock } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * 403 Forbidden page
 */
export default function ForbiddenPage() {
  const navigate = useNavigate()

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100vh',
        padding: '24px',
        textAlign: 'center',
      }}
    >
      <Space vertical align="center" spacing="medium">
        <IconLock
          size="extra-large"
          style={{
            fontSize: '72px',
            color: 'var(--semi-color-danger)',
          }}
        />
        <Title heading={1} style={{ margin: 0, color: 'var(--semi-color-text-0)' }}>
          403
        </Title>
        <Title heading={3} style={{ margin: 0 }}>
          Access Denied
        </Title>
        <Paragraph type="secondary" style={{ maxWidth: '400px' }}>
          You do not have permission to access this page. Please contact your administrator.
        </Paragraph>
        <Button type="primary" onClick={() => navigate('/')} style={{ marginTop: '8px' }}>
          Back to Home
        </Button>
      </Space>
    </div>
  )
}
