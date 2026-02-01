import { useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  Typography,
  Button,
  Toast,
  Select,
  Switch,
  Divider,
  RadioGroup,
  Radio,
  Tag,
  Skeleton,
  Progress,
} from '@douyinfe/semi-ui-19'
import {
  IconLanguage,
  IconMoon,
  IconSun,
  IconBell,
  IconDelete,
  IconCreditCard,
} from '@douyinfe/semi-icons'

import { Container } from '@/components/common/layout'
import { useAppStore } from '@/store'
import { useGetCurrentSubscription, type SubscriptionQuota } from '@/api/billing'
import { getPlanDisplayName, type TenantPlan } from '@/store/featureStore'

import './Settings.css'

const { Title, Text } = Typography

/**
 * System Settings page
 *
 * Features:
 * - Language settings
 * - Theme settings (light/dark/system)
 * - Notification preferences
 * - Data management (clear cache)
 */
export default function SettingsPage() {
  const { t, i18n } = useTranslation('system')
  const navigate = useNavigate()

  // App store
  const theme = useAppStore((state) => state.theme)
  const setTheme = useAppStore((state) => state.setTheme)
  const locale = useAppStore((state) => state.locale)
  const setLocale = useAppStore((state) => state.setLocale)

  // Local state for other settings
  const [notifications, setNotifications] = useState(true)
  const [soundEnabled, setSoundEnabled] = useState(true)
  const [autoRefresh, setAutoRefresh] = useState(true)

  // Fetch current subscription data
  const { data: subscriptionResponse, isLoading: isSubscriptionLoading } =
    useGetCurrentSubscription()
  const subscriptionData =
    subscriptionResponse?.status === 200 ? subscriptionResponse.data.data : null
  const currentPlan = (subscriptionData?.plan_id || 'free') as TenantPlan
  const subscriptionStatus = subscriptionData?.status || 'active'

  // Get key quotas for display (users, products, warehouses)
  const quotas = subscriptionData?.quotas
  const keyQuotas = useMemo(() => {
    if (!quotas) return []
    const quotaTypes = ['users', 'products', 'warehouses']
    return quotas.filter((q: SubscriptionQuota) => quotaTypes.includes(q.type))
  }, [quotas])

  // Helper to get quota display name
  const getQuotaDisplayName = useCallback(
    (type: string) => {
      const names: Record<string, string> = {
        users: t('settings.subscription.quotaUsers'),
        products: t('settings.subscription.quotaProducts'),
        warehouses: t('settings.subscription.quotaWarehouses'),
      }
      return names[type] || type
    },
    [t]
  )

  // Helper to calculate quota percentage
  const getQuotaPercentage = useCallback((quota: SubscriptionQuota) => {
    if (quota.limit === -1) return 0 // Unlimited
    if (quota.limit === 0) return 100
    return Math.min(100, Math.round((quota.used / quota.limit) * 100))
  }, [])

  // Language options
  const languageOptions = useMemo(
    () => [
      { label: '简体中文', value: 'zh-CN' },
      { label: 'English', value: 'en-US' },
    ],
    []
  )

  // Handle language change
  const handleLanguageChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const lang = typeof value === 'string' ? value : 'zh-CN'
      setLocale(lang)
      i18n.changeLanguage(lang)
      Toast.success(t('settings.messages.languageChanged'))
    },
    [setLocale, i18n, t]
  )

  // Handle theme change
  const handleThemeChange = useCallback(
    (e: { target: { value: string } }) => {
      const newTheme = e.target.value as 'light' | 'dark'
      setTheme(newTheme)
      Toast.success(t('settings.messages.themeChanged'))
    },
    [setTheme, t]
  )

  // Handle clear cache
  const handleClearCache = useCallback(() => {
    // Clear localStorage except auth data
    const authData = localStorage.getItem('erp-auth')
    const appSettings = localStorage.getItem('erp-app-settings')
    localStorage.clear()
    if (authData) localStorage.setItem('erp-auth', authData)
    if (appSettings) localStorage.setItem('erp-app-settings', appSettings)

    // Clear sessionStorage
    sessionStorage.clear()

    Toast.success(t('settings.messages.cacheCleared'))
  }, [t])

  // Navigate to subscription page
  const handleManageSubscription = useCallback(() => {
    navigate('/subscription')
  }, [navigate])

  // Navigate to billing history page
  const handleViewBilling = useCallback(() => {
    navigate('/billing')
  }, [navigate])

  return (
    <Container size="md" className="settings-page">
      <Title heading={4} className="settings-title">
        {t('settings.title')}
      </Title>

      {/* Language Settings */}
      <Card className="settings-card">
        <div className="settings-section-header">
          <IconLanguage className="settings-section-icon" />
          <div>
            <Title heading={5} style={{ margin: 0 }}>
              {t('settings.language.title')}
            </Title>
            <Text type="tertiary">{t('settings.language.description')}</Text>
          </div>
        </div>

        <div className="settings-section-content">
          <div className="settings-item">
            <div className="settings-item-label">
              <Text>{t('settings.language.display')}</Text>
            </div>
            <Select
              value={locale}
              onChange={handleLanguageChange}
              optionList={languageOptions}
              style={{ width: 200 }}
            />
          </div>
        </div>
      </Card>

      {/* Theme Settings */}
      <Card className="settings-card">
        <div className="settings-section-header">
          {theme === 'dark' ? (
            <IconMoon className="settings-section-icon" />
          ) : (
            <IconSun className="settings-section-icon" />
          )}
          <div>
            <Title heading={5} style={{ margin: 0 }}>
              {t('settings.theme.title')}
            </Title>
            <Text type="tertiary">{t('settings.theme.description')}</Text>
          </div>
        </div>

        <div className="settings-section-content">
          <RadioGroup
            value={theme}
            onChange={handleThemeChange}
            direction="horizontal"
            className="theme-radio-group"
          >
            <Radio value="light" className="theme-radio-item">
              <div className="theme-option">
                <IconSun />
                <span>{t('settings.theme.light')}</span>
              </div>
            </Radio>
            <Radio value="dark" className="theme-radio-item">
              <div className="theme-option">
                <IconMoon />
                <span>{t('settings.theme.dark')}</span>
              </div>
            </Radio>
          </RadioGroup>
        </div>
      </Card>

      {/* Notification Settings */}
      <Card className="settings-card">
        <div className="settings-section-header">
          <IconBell className="settings-section-icon" />
          <div>
            <Title heading={5} style={{ margin: 0 }}>
              {t('settings.notifications.title')}
            </Title>
            <Text type="tertiary">{t('settings.notifications.description')}</Text>
          </div>
        </div>

        <div className="settings-section-content">
          <div className="settings-item">
            <div className="settings-item-info">
              <Text>{t('settings.notifications.enable')}</Text>
              <Text type="tertiary" size="small">
                {t('settings.notifications.enableDesc')}
              </Text>
            </div>
            <Switch checked={notifications} onChange={setNotifications} />
          </div>

          <Divider margin={16} />

          <div className="settings-item">
            <div className="settings-item-info">
              <Text>{t('settings.notifications.sound')}</Text>
              <Text type="tertiary" size="small">
                {t('settings.notifications.soundDesc')}
              </Text>
            </div>
            <Switch checked={soundEnabled} onChange={setSoundEnabled} disabled={!notifications} />
          </div>

          <Divider margin={16} />

          <div className="settings-item">
            <div className="settings-item-info">
              <Text>{t('settings.notifications.autoRefresh')}</Text>
              <Text type="tertiary" size="small">
                {t('settings.notifications.autoRefreshDesc')}
              </Text>
            </div>
            <Switch checked={autoRefresh} onChange={setAutoRefresh} />
          </div>
        </div>
      </Card>

      {/* Subscription & Billing */}
      <Card className="settings-card">
        <div className="settings-section-header">
          <IconCreditCard className="settings-section-icon" />
          <div>
            <Title heading={5} style={{ margin: 0 }}>
              {t('settings.subscription.title')}
            </Title>
            <Text type="tertiary">{t('settings.subscription.description')}</Text>
          </div>
        </div>

        <div className="settings-section-content">
          {/* Current Plan and Status */}
          <div className="settings-item">
            <div className="settings-item-info">
              <Text>{t('settings.subscription.currentPlan')}</Text>
              {isSubscriptionLoading ? (
                <Skeleton.Paragraph rows={1} style={{ width: 100 }} />
              ) : (
                <div className="subscription-plan-info">
                  <Tag color="blue" size="large">
                    {getPlanDisplayName(currentPlan)}
                  </Tag>
                  <Tag
                    color={
                      subscriptionStatus === 'active'
                        ? 'green'
                        : subscriptionStatus === 'trial'
                          ? 'orange'
                          : 'red'
                    }
                    size="small"
                  >
                    {t(`settings.subscription.status.${subscriptionStatus}`)}
                  </Tag>
                </div>
              )}
            </div>
            <Button theme="solid" type="primary" onClick={handleManageSubscription}>
              {t('settings.subscription.manage')}
            </Button>
          </div>

          {/* Quota Usage */}
          {keyQuotas.length > 0 && (
            <>
              <Divider margin={16} />
              <div className="settings-quota-section">
                <Text strong style={{ marginBottom: 'var(--spacing-3)', display: 'block' }}>
                  {t('settings.subscription.quotaUsage')}
                </Text>
                <div className="settings-quota-list">
                  {keyQuotas.map((quota: SubscriptionQuota) => (
                    <div key={quota.type} className="settings-quota-item">
                      <div className="settings-quota-header">
                        <Text size="small">{getQuotaDisplayName(quota.type)}</Text>
                        <Text size="small" type="tertiary">
                          {quota.limit === -1
                            ? `${quota.used} / ${t('settings.subscription.unlimited')}`
                            : `${quota.used} / ${quota.limit}`}
                        </Text>
                      </div>
                      <Progress
                        percent={getQuotaPercentage(quota)}
                        showInfo={false}
                        size="small"
                        stroke={
                          getQuotaPercentage(quota) >= 90
                            ? 'var(--semi-color-danger)'
                            : getQuotaPercentage(quota) >= 70
                              ? 'var(--semi-color-warning)'
                              : 'var(--semi-color-primary)'
                        }
                      />
                    </div>
                  ))}
                </div>
              </div>
            </>
          )}

          <Divider margin={16} />

          <div className="settings-item">
            <div className="settings-item-info">
              <Text>{t('settings.subscription.billingHistory')}</Text>
              <Text type="tertiary" size="small">
                {t('settings.subscription.billingHistoryDesc')}
              </Text>
            </div>
            <Button onClick={handleViewBilling}>{t('settings.subscription.viewBilling')}</Button>
          </div>
        </div>
      </Card>

      {/* Data Management */}
      <Card className="settings-card">
        <div className="settings-section-header">
          <IconDelete className="settings-section-icon settings-section-icon--danger" />
          <div>
            <Title heading={5} style={{ margin: 0 }}>
              {t('settings.data.title')}
            </Title>
            <Text type="tertiary">{t('settings.data.description')}</Text>
          </div>
        </div>

        <div className="settings-section-content">
          <div className="settings-item">
            <div className="settings-item-info">
              <Text>{t('settings.data.clearCache')}</Text>
              <Text type="tertiary" size="small">
                {t('settings.data.clearCacheDesc')}
              </Text>
            </div>
            <Button type="danger" onClick={handleClearCache}>
              {t('settings.data.clearCacheBtn')}
            </Button>
          </div>
        </div>
      </Card>
    </Container>
  )
}
