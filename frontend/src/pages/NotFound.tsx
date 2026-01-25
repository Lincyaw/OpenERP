import { useNavigate } from 'react-router-dom'
import { Button, Typography, Space } from '@douyinfe/semi-ui-19'
import { IconAlertTriangle } from '@douyinfe/semi-icons'

const { Title, Paragraph } = Typography

/**
 * 404 Not Found page
 */
export default function NotFoundPage() {
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
        <IconAlertTriangle
          size="extra-large"
          style={{
            fontSize: '72px',
            color: 'var(--semi-color-warning)',
          }}
        />
        <Title heading={1} style={{ margin: 0, color: 'var(--semi-color-text-0)' }}>
          404
        </Title>
        <Title heading={3} style={{ margin: 0 }}>
          Page Not Found
        </Title>
        <Paragraph type="secondary" style={{ maxWidth: '400px' }}>
          The page you are looking for does not exist or has been moved.
        </Paragraph>
        <Button type="primary" onClick={() => navigate('/')} style={{ marginTop: '8px' }}>
          Back to Home
        </Button>
      </Space>
    </div>
  )
}
