import { useNavigate } from 'react-router-dom'
import { Card, Typography, Space, Button, Avatar, Descriptions } from '@douyinfe/semi-ui'
import { IconHome, IconUser, IconExit } from '@douyinfe/semi-icons'
import { useAuthStore, useUser } from '@/store'

const { Title, Paragraph } = Typography

/**
 * Dashboard/Home page placeholder
 * Will be replaced with actual dashboard implementation in P5-FE-006
 *
 * Demonstrates Zustand store usage:
 * - useUser() selector hook for getting current user
 * - useAuthStore() for accessing logout action
 */
export default function DashboardPage() {
  const navigate = useNavigate()
  const user = useUser()
  const logout = useAuthStore((state) => state.logout)

  const handleLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  return (
    <div style={{ padding: '24px' }}>
      <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
        <Card style={{ width: '100%' }}>
          <Space vertical align="start" spacing="tight">
            <Space>
              <IconHome size="extra-large" style={{ color: 'var(--semi-color-primary)' }} />
              <Title heading={2} style={{ margin: 0 }}>
                Dashboard
              </Title>
            </Space>
            <Paragraph type="secondary">
              Welcome to the ERP System. This dashboard will display key metrics and quick actions.
            </Paragraph>
          </Space>
        </Card>

        {/* User Info Card - Demonstrates Zustand auth store usage */}
        <Card title="Current User" style={{ width: '100%' }}>
          <Space align="start" spacing="medium">
            <Avatar size="large" color="blue">
              {user?.displayName?.charAt(0).toUpperCase() || <IconUser />}
            </Avatar>
            <div style={{ flex: 1 }}>
              <Descriptions
                data={[
                  { key: 'Username', value: user?.username || '-' },
                  { key: 'Display Name', value: user?.displayName || '-' },
                  { key: 'Roles', value: user?.roles?.join(', ') || '-' },
                ]}
              />
            </div>
            <Button icon={<IconExit />} type="danger" onClick={handleLogout}>
              Logout
            </Button>
          </Space>
        </Card>

        <Card title="Quick Actions" style={{ width: '100%' }}>
          <Space>
            <Button type="primary">New Sale</Button>
            <Button type="secondary">New Purchase</Button>
            <Button type="tertiary">View Inventory</Button>
          </Space>
        </Card>
      </Space>
    </div>
  )
}
