import { useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  Typography,
  Button,
  Tag,
  Descriptions,
  Table,
  Skeleton,
  Banner,
} from '@douyinfe/semi-ui-19'
import { IconTick, IconClose, IconCrown, IconStar, IconVerify } from '@douyinfe/semi-icons'

import { Container } from '@/components/common/layout'
import { UsageGauge, UsageChart, QuotaAlertList } from '@/components/usage'
import { useUser } from '@/store'
import { useGetTenantById } from '@/api/tenants/tenants'
import { useGetCurrentUsage } from '@/api/usage'
import {
  useTenantPlan,
  useFeatureStore,
  getPlanDisplayName,
  type TenantPlan,
} from '@/store/featureStore'

import './SubscriptionPage.css'

const { Title, Text } = Typography

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

  // Fetch usage data from API
  const { data: usageResponse, isLoading: isUsageLoading } = useGetCurrentUsage({
    query: {
      enabled: !!user?.tenantId,
    },
  })

  const tenant = tenantResponse?.status === 200 ? tenantResponse.data.data : null
  const usageData = usageResponse?.status === 200 ? usageResponse.data.data : null

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

      {/* Quota Alerts - Show warnings for resources approaching limits */}
      {usageData?.metrics && (
        <QuotaAlertList
          metrics={usageData.metrics}
          warningThreshold={70}
          criticalThreshold={90}
          showUpgradeButton={currentPlan !== 'enterprise'}
          className="quota-alerts-section"
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

      {/* Resource Usage Section - Using real API data */}
      <Card className="resource-usage-card">
        <Title heading={5} className="section-title">
          {t('subscriptionPage.resourceUsage')}
        </Title>

        {isUsageLoading ? (
          <Skeleton.Paragraph rows={3} />
        ) : usageData?.metrics ? (
          <div className="resource-usage-grid">
            {usageData.metrics.map((metric) => (
              <UsageGauge
                key={metric.name}
                metric={metric}
                showPercentage
                showUsageText
                size="default"
                warningThreshold={70}
                criticalThreshold={90}
              />
            ))}
          </div>
        ) : (
          <Text type="tertiary">{t('usage.noData')}</Text>
        )}
      </Card>

      {/* Usage Trends Chart */}
      <UsageChart
        title={t('usage.chart.title')}
        defaultPeriod="daily"
        height={300}
        showPeriodSelector
        className="usage-chart-section"
      />

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
