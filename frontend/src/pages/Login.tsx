import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Card, Form, Button, Typography, Toast } from '@douyinfe/semi-ui'
import { IconUser, IconLock } from '@douyinfe/semi-icons'

const { Title, Text } = Typography

interface LoginFormValues {
  username: string
  password: string
}

/**
 * Login page
 * Handles user authentication
 */
export default function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [loading, setLoading] = useState(false)

  // Get intended destination from location state
  const from = (location.state as { from?: Location })?.from?.pathname || '/'

  const handleSubmit = async (values: LoginFormValues) => {
    setLoading(true)
    try {
      // TODO: Replace with actual login API call
      // Simulate API call
      await new Promise((resolve) => setTimeout(resolve, 1000))

      // TODO: Store actual token from API response
      localStorage.setItem('access_token', 'mock-token')
      localStorage.setItem('user', JSON.stringify({ username: values.username }))

      Toast.success({ content: 'Login successful!' })

      // Redirect to intended destination
      navigate(from, { replace: true })
    } catch {
      Toast.error({ content: 'Login failed. Please check your credentials.' })
    } finally {
      setLoading(false)
    }
  }

  return (
    <div
      style={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        minHeight: '100vh',
        background: 'var(--semi-color-bg-0)',
      }}
    >
      <Card
        style={{
          width: 400,
          padding: '24px',
        }}
      >
        <div style={{ textAlign: 'center', marginBottom: '24px' }}>
          <Title heading={3}>ERP System</Title>
          <Text type="secondary">Sign in to your account</Text>
        </div>

        <Form onSubmit={handleSubmit} labelPosition="top">
          <Form.Input
            field="username"
            label="Username"
            prefix={<IconUser />}
            placeholder="Enter username"
            rules={[{ required: true, message: 'Username is required' }]}
          />

          <Form.Input
            field="password"
            label="Password"
            mode="password"
            prefix={<IconLock />}
            placeholder="Enter password"
            rules={[{ required: true, message: 'Password is required' }]}
          />

          <Button
            type="primary"
            htmlType="submit"
            theme="solid"
            block
            loading={loading}
            style={{ marginTop: '16px' }}
          >
            Sign In
          </Button>
        </Form>

        <div style={{ textAlign: 'center', marginTop: '16px' }}>
          <Text type="tertiary">Demo: any username/password will work</Text>
        </div>
      </Card>
    </div>
  )
}
