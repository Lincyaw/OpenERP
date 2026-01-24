import { useState, useMemo } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Card, Form, Button, Typography, Toast, Banner } from '@douyinfe/semi-ui'
import { IconUser, IconLock } from '@douyinfe/semi-icons'
import { useAuthStore } from '@/store'
import { getAuth } from '@/api/auth'
import type { User } from '@/store/types'
import type { AxiosError } from 'axios'

const { Title, Text } = Typography

interface LoginFormValues {
  username: string
  password: string
}

interface ApiError {
  success: boolean
  error?: {
    code: string
    message: string
    details?: string
  }
}

/**
 * Login page
 * Handles user authentication using the Auth API
 */
export default function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const login = useAuthStore((state) => state.login)
  const authApi = useMemo(() => getAuth(), [])

  // Get intended destination from location state
  const from = (location.state as { from?: Location })?.from?.pathname || '/'

  const handleSubmit = async (values: LoginFormValues) => {
    setLoading(true)
    setError(null)

    try {
      const response = await authApi.postAuthLogin({
        username: values.username,
        password: values.password,
      })

      if (!response.success || !response.data) {
        throw new Error(response.error?.message || 'Login failed')
      }

      const { token, user: apiUser } = response.data

      if (!apiUser || !token) {
        throw new Error('Invalid login response: missing user or token')
      }

      // Convert API user response to store User type
      const user: User = {
        id: apiUser.id ?? '',
        username: apiUser.username ?? '',
        displayName: apiUser.display_name,
        email: apiUser.email,
        avatar: apiUser.avatar,
        tenantId: apiUser.tenant_id,
        permissions: apiUser.permissions,
        roles: apiUser.role_ids,
      }

      // Use auth store to login with tokens
      login(user, token.access_token ?? '', token.refresh_token ?? '')

      Toast.success({ content: 'Login successful!' })

      // Redirect to intended destination
      navigate(from, { replace: true })
    } catch (err) {
      const axiosError = err as AxiosError<ApiError>
      let errorMessage = 'Login failed. Please check your credentials.'

      if (axiosError.response?.data?.error) {
        const apiError = axiosError.response.data.error
        // Handle specific error codes
        switch (apiError.code) {
          case 'INVALID_CREDENTIALS':
            errorMessage = 'Invalid username or password'
            break
          case 'ACCOUNT_LOCKED':
            errorMessage = 'Account is locked. Please try again later.'
            break
          case 'ACCOUNT_DISABLED':
          case 'ACCOUNT_DEACTIVATED':
            errorMessage = 'Account is disabled. Please contact support.'
            break
          case 'USER_NOT_FOUND':
            errorMessage = 'User not found'
            break
          default:
            errorMessage = apiError.message || errorMessage
        }
      } else if (axiosError.message) {
        errorMessage = axiosError.message
      }

      setError(errorMessage)
      Toast.error({ content: errorMessage })
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
        padding: 'var(--spacing-4)',
      }}
    >
      <Card
        style={{
          width: '100%',
          maxWidth: 400,
          padding: 'var(--spacing-6)',
        }}
      >
        <div style={{ textAlign: 'center', marginBottom: 'var(--spacing-6)' }}>
          <Title heading={3}>ERP System</Title>
          <Text type="secondary">Sign in to your account</Text>
        </div>

        {error && (
          <Banner
            type="danger"
            description={error}
            style={{ marginBottom: 'var(--spacing-4)' }}
            closeIcon={null}
          />
        )}

        <Form onSubmit={handleSubmit} labelPosition="top">
          <Form.Input
            field="username"
            label="Username"
            prefix={<IconUser />}
            placeholder="Enter username"
            rules={[
              { required: true, message: 'Username is required' },
              { min: 3, message: 'Username must be at least 3 characters' },
              { max: 100, message: 'Username must be at most 100 characters' },
            ]}
            disabled={loading}
          />

          <Form.Input
            field="password"
            label="Password"
            mode="password"
            prefix={<IconLock />}
            placeholder="Enter password"
            rules={[
              { required: true, message: 'Password is required' },
              { min: 8, message: 'Password must be at least 8 characters' },
              { max: 128, message: 'Password must be at most 128 characters' },
            ]}
            disabled={loading}
          />

          <Button
            type="primary"
            htmlType="submit"
            theme="solid"
            block
            loading={loading}
            style={{ marginTop: 'var(--spacing-4)' }}
          >
            Sign In
          </Button>
        </Form>
      </Card>
    </div>
  )
}
