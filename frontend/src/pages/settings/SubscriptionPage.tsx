import { useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  Typography,
  Button,
  Tag,
  Progress,
  Descriptions,
  Table,
  Skeleton,
  Banner,
} from '@douyinfe/semi-ui-19'
import {
  IconTick,
  IconClose,
  IconCrown,
  IconStar,
  IconVerify,
  IconUser,
  IconInbox,
  IconBox,
} from '@douyinfe/semi-icons'

import { Container } from '@/components/common/layout'
import { useUser } from '@/store'
import { useGetTenantById } from '@/api/tenants/tenants'
import {
  useTenantPlan,
  useFeatureStore,
  getPlanDisplayName,
  type TenantPlan,
} from '@/store/featureStore'

import './SubscriptionPage.css'

const { Title, Text } = Typography

/**
 * Plan limits for resource usage display
 */
const PLAN_LIMITS: Record<TenantPlan, { users: number; warehouses: number; products: number }> = {
  free: { users: 1, warehouses: 1, products: 100 },
  basic: { users: 5, warehouses: 3, products: 1000 },
  pro: { users: 20, warehouses: 10, products: 10000 },
  enterprise: { users: -1, warehouses: -1, products: -1 }, // -1 = unlimited
}

/**
 * Plan pricing
 */
const PLAN_PRICING: Record<TenantPlan, number> = {
  free: 0,
  basic: 29,
  pro: 79,
  enterprise: -1, // Custom pricing
}

/**
 * Feature comparison data
 */
interface FeatureComparisonRow {
  key: string
  feature: string
  free: boolean | string
  basic: boolean | string
  pro: boolean | string
  enterprise: boolean | string
}

/**
 * Subscription Page
 *
 * Displays current tenant subscription status, plan features,
 * resource usage, and plan comparison table.
 */
export default function SubscriptionPage() {
  const { t } = useTranslation('system')
  const navigate = useNavigate()
  const user = useUser()
  const currentPlan = useTenantPlan()
  const getEnabledFeatures = useFeatureStore((state) => state.getEnabledFeatures)
  const getDisabledFeatures = useFeatureStore((state) => state.getDisabledFeatures)

  // Fetch tenant data
  const {
    data: tenantResponse,
    isLoading: isTenantLoading,
    isError: isTenantError,
  } = useGetTenantById(user?.tenantId || '', {
    query: {
      enabled: !!user?.tenantId,
    },
  })

  const tenant = tenantResponse?.status === 200 ? tenantResponse.data.data : null

  // Get enabled and disabled features
  const enabledFeatures = useMemo(() => getEnabledFeatures(), [getEnabledFeatures])
  const disabledFeatures = useMemo(() => getDisabledFeatures(), [getDisabledFeatures])

  // Extract tenant dates for memoization
  const tenantExpiresAt = tenant?.expires_at
  const tenantTrialEndsAt = tenant?.trial_ends_at

  // Calculate days until expiration
  const daysUntilExpiration = useMemo(() => {
    if (!tenantExpiresAt) return null
    const expiresAt = new Date(tenantExpiresAt)
    const now = new Date()
    const diffTime = expiresAt.getTime() - now.getTime()
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24))
    return diffDays
  }, [tenantExpiresAt])

  // Check if trial is active
  const isTrialActive = useMemo(() => {
    if (!tenantTrialEndsAt) return false
    const trialEndsAt = new Date(tenantTrialEndsAt)
    return trialEndsAt > new Date()
  }, [tenantTrialEndsAt])

  // Calculate trial days remaining
  const trialDaysRemaining = useMemo(() => {
    if (!tenantTrialEndsAt) return null
    const trialEndsAt = new Date(tenantTrialEndsAt)
    const now = new Date()
    const diffTime = trialEndsAt.getTime() - now.getTime()
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24))
    return diffDays > 0 ? diffDays : 0
  }, [tenantTrialEndsAt])

  // Mock resource usage (in real app, this would come from API)
  const resourceUsage = useMemo(
    () => ({
      users: { current: 3, limit: PLAN_LIMITS[currentPlan].users },
      warehouses: { current: 2, limit: PLAN_LIMITS[currentPlan].warehouses },
      products: { current: 45, limit: PLAN_LIMITS[currentPlan].products },
    }),
    [currentPlan]
  )

  // Feature comparison table data
  const featureComparisonData: FeatureComparisonRow[] = useMemo(
    () => [
      {
        key: 'users',
        feature: t('subscriptionPage.comparison.users'),
        free: '1',
        basic: '5',
        pro: '20',
        enterprise: t('subscriptionPage.comparison.unlimited'),
      },
      {
        key: 'products',
        feature: t('subscriptionPage.comparison.products'),
        free: '100',
        basic: '1,000',
        pro: '10,000',
        enterprise: t('subscriptionPage.comparison.unlimited'),
      },
      {
        key: 'warehouses',
        feature: t('subscriptionPage.comparison.warehouses'),
        free: '1',
        basic: '3',
        pro: '10',
        enterprise: t('subscriptionPage.comparison.unlimited'),
      },
      {
        key: 'orders',
        feature: t('subscriptionPage.comparison.orders'),
        free: '50/mo',
        basic: '500/mo',
        pro: t('subscriptionPage.comparison.unlimited'),
        enterprise: t('subscriptionPage.comparison.unlimited'),
      },
      {
        key: 'multi_warehouse',
        feature: t('subscriptionPage.comparison.multiWarehouse'),
        free: false,
        basic: true,
        pro: true,
        enterprise: true,
      },
      {
        key: 'batch_management',
        feature: t('subscriptionPage.comparison.batchManagement'),
        free: false,
        basic: true,
        pro: true,
        enterprise: true,
      },
      {
        key: 'serial_tracking',
        feature: t('subscriptionPage.comparison.serialTracking'),
        free: false,
        basic: false,
        pro: true,
        enterprise: true,
      },
      {
        key: 'multi_currency',
        feature: t('subscriptionPage.comparison.multiCurrency'),
        free: false,
        basic: false,
        pro: true,
        enterprise: true,
      },
      {
        key: 'advanced_reporting',
        feature: t('subscriptionPage.comparison.advancedReporting'),
        free: false,
        basic: false,
        pro: true,
        enterprise: true,
      },
      {
        key: 'api_access',
        feature: t('subscriptionPage.comparison.apiAccess'),
        free: false,
        basic: false,
        pro: true,
        enterprise: true,
      },
      {
        key: 'integrations',
        feature: t('subscriptionPage.comparison.integrations'),
        free: false,
        basic: false,
        pro: true,
        enterprise: true,
      },
      {
        key: 'priority_support',
        feature: t('subscriptionPage.comparison.prioritySupport'),
        free: false,
        basic: false,
        pro: true,
        enterprise: true,
      },
      {
        key: 'dedicated_support',
        feature: t('subscriptionPage.comparison.dedicatedSupport'),
        free: false,
        basic: false,
        pro: false,
        enterprise: true,
      },
      {
        key: 'sla',
        feature: t('subscriptionPage.comparison.sla'),
        free: false,
        basic: false,
        pro: false,
        enterprise: true,
      },
      {
        key: 'white_labeling',
        feature: t('subscriptionPage.comparison.whiteLabeling'),
        free: false,
        basic: false,
        pro: false,
        enterprise: true,
      },
    ],
    [t]
  )

  // Table columns for feature comparison
  const comparisonColumns = [
    {
      title: t('subscriptionPage.comparison.feature'),
      dataIndex: 'feature',
      key: 'feature',
      width: 200,
    },
    {
      title: (
        <div className="plan-column-header">
          <IconStar className="plan-icon-free" />
          <span>{t('subscription.plans.free.name')}</span>
          <Text type="tertiary" size="small">
            ${PLAN_PRICING.free}/mo
          </Text>
        </div>
      ),
      dataIndex: 'free',
      key: 'free',
      align: 'center' as const,
      render: renderFeatureCell,
    },
    {
      title: (
        <div className="plan-column-header">
          <IconVerify className="plan-icon-basic" />
          <span>{t('subscription.plans.basic.name')}</span>
          <Text type="tertiary" size="small">
            ${PLAN_PRICING.basic}/mo
          </Text>
        </div>
      ),
      dataIndex: 'basic',
      key: 'basic',
      align: 'center' as const,
      render: renderFeatureCell,
    },
    {
      title: (
        <div className="plan-column-header plan-column-header--highlighted">
          <IconCrown className="plan-icon-pro" />
          <span>{t('subscription.plans.pro.name')}</span>
          <Text type="tertiary" size="small">
            ${PLAN_PRICING.pro}/mo
          </Text>
          <Tag color="orange" size="small">
            {t('subscription.popular')}
          </Tag>
        </div>
      ),
      dataIndex: 'pro',
      key: 'pro',
      align: 'center' as const,
      render: renderFeatureCell,
    },
    {
      title: (
        <div className="plan-column-header">
          <IconCrown className="plan-icon-enterprise" />
          <span>{t('subscription.plans.enterprise.name')}</span>
          <Text type="tertiary" size="small">
            {t('subscription.customPricing')}
          </Text>
        </div>
      ),
      dataIndex: 'enterprise',
      key: 'enterprise',
      align: 'center' as const,
      render: renderFeatureCell,
    },
  ]

  // Navigate to upgrade page
  const handleUpgrade = useCallback(() => {
    navigate('/upgrade')
  }, [navigate])

  // Render feature cell (checkmark, X, or text)
  function renderFeatureCell(value: boolean | string) {
    if (typeof value === 'boolean') {
      return value ? (
        <IconTick className="feature-check" />
      ) : (
        <IconClose className="feature-cross" />
      )
    }
    return <Text>{value}</Text>
  }

  // Calculate usage percentage
  function getUsagePercentage(current: number, limit: number): number {
    if (limit === -1) return 0 // Unlimited
    return Math.min(Math.round((current / limit) * 100), 100)
  }

  // Get usage status color
  function getUsageStatus(percentage: number): 'success' | 'warning' | 'exception' {
    if (percentage >= 90) return 'exception'
    if (percentage >= 70) return 'warning'
    return 'success'
  }

  // Format limit display
  function formatLimit(limit: number): string {
    if (limit === -1) return t('subscriptionPage.comparison.unlimited')
    return limit.toLocaleString()
  }

  if (isTenantLoading) {
    return (
      <Container size="lg" className="subscription-page">
        <Skeleton.Title style={{ width: 200, marginBottom: 24 }} />
        <Skeleton.Paragraph rows={4} />
      </Container>
    )
  }

  return (
    <Container size="lg" className="subscription-page">
      {/* Page Header */}
      <div className="subscription-header">
        <Title heading={3}>{t('subscriptionPage.title')}</Title>
        <Text type="tertiary">{t('subscriptionPage.subtitle')}</Text>
      </div>

      {/* Error Banner */}
      {isTenantError && (
        <Banner
          type="danger"
          description={t('common.fetchError', { defaultValue: 'Failed to load data' })}
          className="error-banner"
        />
      )}

      {/* Trial Banner */}
      {isTrialActive && trialDaysRemaining !== null && (
        <Banner
          type="warning"
          description={t('subscriptionPage.trialBanner', { days: trialDaysRemaining })}
          className="trial-banner"
        />
      )}

      {/* Expiration Warning */}
      {daysUntilExpiration !== null && daysUntilExpiration <= 30 && daysUntilExpiration > 0 && (
        <Banner
          type="warning"
          description={t('subscriptionPage.expirationWarning', { days: daysUntilExpiration })}
          className="expiration-banner"
        />
      )}

      {/* Current Plan Card */}
      <Card className="current-plan-card">
        <div className="current-plan-content">
          <div className="current-plan-info">
            <div className="current-plan-badge">
              {currentPlan === 'enterprise' ? (
                <IconCrown className="plan-icon-enterprise" size="extra-large" />
              ) : currentPlan === 'pro' ? (
                <IconCrown className="plan-icon-pro" size="extra-large" />
              ) : currentPlan === 'basic' ? (
                <IconVerify className="plan-icon-basic" size="extra-large" />
              ) : (
                <IconStar className="plan-icon-free" size="extra-large" />
              )}
            </div>
            <div className="current-plan-details">
              <div className="current-plan-name-row">
                <Title heading={4}>{getPlanDisplayName(currentPlan)}</Title>
                <Tag color="green">{t('subscription.current')}</Tag>
              </div>
              <Text type="tertiary">{t(`subscription.plans.${currentPlan}.description`)}</Text>
            </div>
          </div>

          <div className="current-plan-meta">
            <Descriptions
              data={[
                {
                  key: t('subscriptionPage.tenantName'),
                  value: tenant?.name || '-',
                },
                {
                  key: t('subscriptionPage.status'),
                  value: (
                    <Tag color={tenant?.status === 'active' ? 'green' : 'grey'}>
                      {tenant?.status || '-'}
                    </Tag>
                  ),
                },
                {
                  key: t('subscriptionPage.expiresAt'),
                  value: tenant?.expires_at
                    ? new Date(tenant.expires_at).toLocaleDateString()
                    : t('subscriptionPage.noExpiration'),
                },
              ]}
            />
          </div>

          {currentPlan !== 'enterprise' && (
            <Button theme="solid" type="primary" size="large" onClick={handleUpgrade}>
              {t('subscription.upgrade')}
            </Button>
          )}
        </div>
      </Card>

      {/* Resource Usage Section */}
      <Card className="resource-usage-card">
        <Title heading={5} className="section-title">
          {t('subscriptionPage.resourceUsage')}
        </Title>

        <div className="resource-usage-grid">
          {/* Users */}
          <div className="resource-item">
            <div className="resource-header">
              <IconUser className="resource-icon" />
              <Text strong>{t('subscriptionPage.comparison.users')}</Text>
            </div>
            <Progress
              percent={getUsagePercentage(resourceUsage.users.current, resourceUsage.users.limit)}
              stroke={getUsageStatus(
                getUsagePercentage(resourceUsage.users.current, resourceUsage.users.limit)
              )}
              showInfo
            />
            <Text type="tertiary" size="small">
              {resourceUsage.users.current} / {formatLimit(resourceUsage.users.limit)}
            </Text>
          </div>

          {/* Warehouses */}
          <div className="resource-item">
            <div className="resource-header">
              <IconInbox className="resource-icon" />
              <Text strong>{t('subscriptionPage.comparison.warehouses')}</Text>
            </div>
            <Progress
              percent={getUsagePercentage(
                resourceUsage.warehouses.current,
                resourceUsage.warehouses.limit
              )}
              stroke={getUsageStatus(
                getUsagePercentage(resourceUsage.warehouses.current, resourceUsage.warehouses.limit)
              )}
              showInfo
            />
            <Text type="tertiary" size="small">
              {resourceUsage.warehouses.current} / {formatLimit(resourceUsage.warehouses.limit)}
            </Text>
          </div>

          {/* Products */}
          <div className="resource-item">
            <div className="resource-header">
              <IconBox className="resource-icon" />
              <Text strong>{t('subscriptionPage.comparison.products')}</Text>
            </div>
            <Progress
              percent={getUsagePercentage(
                resourceUsage.products.current,
                resourceUsage.products.limit
              )}
              stroke={getUsageStatus(
                getUsagePercentage(resourceUsage.products.current, resourceUsage.products.limit)
              )}
              showInfo
            />
            <Text type="tertiary" size="small">
              {resourceUsage.products.current} / {formatLimit(resourceUsage.products.limit)}
            </Text>
          </div>
        </div>
      </Card>

      {/* Current Plan Features */}
      <Card className="features-card">
        <Title heading={5} className="section-title">
          {t('subscriptionPage.includedFeatures')}
        </Title>

        <div className="features-grid">
          {enabledFeatures.slice(0, 12).map((feature) => (
            <div key={feature.key} className="feature-item feature-item--enabled">
              <IconTick className="feature-check" />
              <Text>{feature.description}</Text>
            </div>
          ))}
        </div>

        {disabledFeatures.length > 0 && (
          <>
            <Title heading={6} className="section-subtitle">
              {t('subscriptionPage.upgradeToUnlock')}
            </Title>
            <div className="features-grid">
              {disabledFeatures.slice(0, 6).map((feature) => (
                <div key={feature.key} className="feature-item feature-item--disabled">
                  <IconClose className="feature-cross" />
                  <Text type="tertiary">{feature.description}</Text>
                  <Tag size="small" color="blue">
                    {getPlanDisplayName(feature.requiredPlan)}
                  </Tag>
                </div>
              ))}
            </div>
          </>
        )}
      </Card>

      {/* Plan Comparison Table */}
      <Card className="comparison-card">
        <Title heading={5} className="section-title">
          {t('subscriptionPage.planComparison')}
        </Title>

        <Table
          columns={comparisonColumns}
          dataSource={featureComparisonData}
          pagination={false}
          rowKey="key"
          className="comparison-table"
        />

        <div className="comparison-actions">
          <Button theme="solid" type="primary" size="large" onClick={handleUpgrade}>
            {t('subscriptionPage.viewAllPlans')}
          </Button>
        </div>
      </Card>
    </Container>
  )
}
