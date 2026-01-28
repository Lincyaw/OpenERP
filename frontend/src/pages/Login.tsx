import { useState, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { Card, Form, Button, Typography, Toast, Banner } from '@douyinfe/semi-ui-19'
import { IconUser, IconLock } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '@/store'
import { useLoginAuth } from '@/api/auth/auth'
import { resetRedirectFlag } from '@/services/token-refresh'
import type { User } from '@/store/types'

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
  const { t } = useTranslation('auth')
  const [error, setError] = useState<string | null>(null)
  const login = useAuthStore((state) => state.login)
  const loginMutation = useLoginAuth()

  // Get intended destination from location state
  const from = (location.state as { from?: Location })?.from?.pathname || '/'

  // Check for redirect message (e.g., session expired)
  useEffect(() => {
    const message = sessionStorage.getItem('auth_redirect_message')
    if (message) {
      setError(message)
      sessionStorage.removeItem('auth_redirect_message')
    }
  }, [])

  const handleSubmit = async (values: LoginFormValues) => {
    setError(null)

    try {
      const response = await loginMutation.mutateAsync({
        data: {
          username: values.username,
          password: values.password,
        },
      })

      // Check response status and data
      if (response.status !== 200 || !response.data.success || !response.data.data) {
        const apiError = response.data as unknown as ApiError
        throw new Error(apiError.error?.message || t('login.failed'))
      }

      const { token, user: apiUser } = response.data.data

      if (!apiUser || !token) {
        throw new Error(t('login.failed'))
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

      // Reset redirect flag so future session expirations can redirect again
      resetRedirectFlag()

      Toast.success({ content: t('login.success') })

      // Redirect to intended destination
      navigate(from, { replace: true })
    } catch (err) {
      let errorMessage = t('login.failed')

      // Handle error response from API
      if (err instanceof Error) {
        errorMessage = err.message
      }

      // Try to extract error code from mutation error
      const mutationError = loginMutation.error as { data?: ApiError } | null
      if (mutationError?.data?.error) {
        const apiError = mutationError.data.error
        // Handle specific error codes
        switch (apiError.code) {
          case 'INVALID_CREDENTIALS':
            errorMessage = t('login.invalidCredentials')
            break
          case 'ACCOUNT_LOCKED':
            errorMessage = t('login.accountLocked')
            break
          case 'ACCOUNT_DISABLED':
          case 'ACCOUNT_DEACTIVATED':
            errorMessage = t('login.accountDisabled')
            break
          case 'USER_NOT_FOUND':
            errorMessage = t('login.userNotFound')
            break
          default:
            errorMessage = apiError.message || errorMessage
        }
      }

      setError(errorMessage)
      Toast.error({ content: errorMessage })
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
          <Title heading={3}>{t('login.title')}</Title>
          <Text type="secondary">{t('login.subtitle')}</Text>
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
            label={t('login.username')}
            prefix={<IconUser />}
            placeholder={t('login.usernamePlaceholder')}
            rules={[
              { required: true, message: t('validation.usernameRequired') },
              { min: 3, message: t('validation.usernameMinLength') },
              { max: 100, message: t('validation.usernameMaxLength') },
            ]}
            disabled={loginMutation.isPending}
          />

          <Form.Input
            field="password"
            label={t('login.password')}
            mode="password"
            prefix={<IconLock />}
            placeholder={t('login.passwordPlaceholder')}
            rules={[
              { required: true, message: t('validation.passwordRequired') },
              { min: 8, message: t('validation.passwordMinLength') },
              { max: 128, message: t('validation.passwordMaxLength') },
            ]}
            disabled={loginMutation.isPending}
          />

          <Button
            type="primary"
            htmlType="submit"
            theme="solid"
            block
            loading={loginMutation.isPending}
            style={{ marginTop: 'var(--spacing-4)' }}
          >
            {t('login.submit')}
          </Button>
        </Form>
      </Card>
    </div>
  )
}
