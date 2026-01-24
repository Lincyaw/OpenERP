import { useNavigate, useLocation } from 'react-router-dom'
import { Button, Typography, Space } from '@douyinfe/semi-ui'
import { IconLock, IconHome } from '@douyinfe/semi-icons'

const { Title, Paragraph, Text } = Typography

/**
 * 403 Forbidden page
 *
 * Displayed when a user tries to access a route they don't have permission for.
 * Shows the attempted URL and provides navigation options.
 */
export default function ForbiddenPage() {
  const navigate = useNavigate()
  const location = useLocation()

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
        backgroundColor: 'var(--semi-color-bg-0)',
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
          You do not have permission to access this page. Please contact your administrator if you
          believe this is a mistake.
        </Paragraph>
        {location.state?.from?.pathname && (
          <Text type="tertiary" size="small">
            Attempted to access: <code>{location.state.from.pathname}</code>
          </Text>
        )}
        <Space style={{ marginTop: '16px' }}>
          <Button icon={<IconHome />} type="primary" onClick={() => navigate('/')}>
            Back to Dashboard
          </Button>
          <Button type="tertiary" onClick={() => navigate(-1)}>
            Go Back
          </Button>
        </Space>
      </Space>
    </div>
  )
}
