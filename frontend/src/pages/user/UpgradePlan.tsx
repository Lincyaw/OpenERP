import { useState, useCallback, useMemo, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Button,
  Toast,
  Tag,
  Spin,
  Banner,
  Skeleton,
  Progress,
} from '@douyinfe/semi-ui-19'
import { IconTick, IconCrown, IconStar, IconVerify } from '@douyinfe/semi-icons'

import { Container } from '@/components/common/layout'
import {
  useGetCurrentSubscription,
  useGetPlans,
  type SubscriptionQuota,
  type SubscriptionPlan,
  type CurrentSubscriptionResponse,
} from '@/api/billing'

import './UpgradePlan.css'

const { Title, Text } = Typography

/**
 * Plan tier definition for display
 */
interface PlanTier {
  id: string
  name: string
  price: number
  priceUnit: string
  description: string
  features: string[]
  highlighted?: boolean
  current?: boolean
  icon: React.ReactNode
}

/**
 * Get icon for plan
 */
const getPlanIcon = (planId: string): React.ReactNode => {
  switch (planId) {
    case 'free':
      return <IconStar size="large" />
    case 'basic':
      return <IconVerify size="large" />
    case 'pro':
    case 'enterprise':
      return <IconCrown size="large" />
    default:
      return <IconStar size="large" />
  }
}

/**
 * Format quota display
 */
const formatQuotaDisplay = (quota: SubscriptionQuota, t: (key: string) => string): string => {
  const typeLabels: Record<string, string> = {
    users: t('usage.metrics.users'),
    warehouses: t('usage.metrics.warehouses'),
    products: t('usage.metrics.products'),
  }

  const label = typeLabels[quota.type] || quota.type

  if (quota.limit === -1) {
    return `${label}: ${t('usage.unlimited')}`
  }

  return `${label}: ${quota.used}/${quota.limit}`
}

/**
 * Calculate quota percentage
 */
const getQuotaPercentage = (quota: SubscriptionQuota): number => {
  if (quota.limit === -1 || quota.limit === 0) {
    return 0
  }
  return Math.min(100, Math.round((quota.used / quota.limit) * 100))
}

/**
 * Get progress bar status based on percentage
 */
const getProgressStatus = (percentage: number): 'success' | 'warning' | 'exception' => {
  if (percentage >= 90) return 'exception'
  if (percentage >= 70) return 'warning'
  return 'success'
}

/**
 * Subscription Upgrade Plan Page
 *
 * Features:
 * - Display available subscription plans from API
 * - Show current plan status with quota usage
 * - Allow plan selection and upgrade
 * - Integrate with Stripe for payment
 */
export default function UpgradePlanPage() {
  const { t } = useTranslation('system')

  // API queries
  const {
    data: subscriptionResponse,
    isLoading: isLoadingSubscription,
    error: subscriptionError,
  } = useGetCurrentSubscription()

  const { data: plansResponse, isLoading: isLoadingPlans, error: plansError } = useGetPlans()

  // State
  const [selectedPlan, setSelectedPlan] = useState<string | null>(null)
  const [isUpgrading, setIsUpgrading] = useState(false)

  // Extract data from responses
  const currentSubscription: CurrentSubscriptionResponse | null = useMemo(() => {
    if (subscriptionResponse?.status === 200) {
      return subscriptionResponse.data.data
    }
    return null
  }, [subscriptionResponse])

  const availablePlans: SubscriptionPlan[] = useMemo(() => {
    if (plansResponse?.status === 200) {
      return plansResponse.data.data.plans || []
    }
    return []
  }, [plansResponse])

  const currentPlanId = currentSubscription?.plan_id || 'free'

  // Show error toast on API errors
  useEffect(() => {
    if (subscriptionError) {
      Toast.error(t('billing.messages.loadError'))
    }
    if (plansError) {
      Toast.error(t('billing.messages.loadError'))
    }
  }, [subscriptionError, plansError, t])

  // Convert API plans to display format, with fallback to hardcoded plans
  const plans: PlanTier[] = useMemo(() => {
    // If we have API plans, use them
    if (availablePlans.length > 0) {
      return availablePlans.map((plan) => ({
        id: plan.id,
        name: plan.name,
        price: plan.price,
        priceUnit: plan.price_unit || t('subscription.priceUnit'),
        description: plan.description,
        features: plan.features.filter((f) => f.enabled).map((f) => f.name),
        highlighted: plan.highlighted,
        current: plan.id === currentPlanId,
        icon: getPlanIcon(plan.id),
      }))
    }

    // Fallback to hardcoded plans if API not available
    return [
      {
        id: 'free',
        name: t('subscription.plans.free.name'),
        price: 0,
        priceUnit: t('subscription.priceUnit'),
        description: t('subscription.plans.free.description'),
        features: [
          t('subscription.plans.free.features.users'),
          t('subscription.plans.free.features.products'),
          t('subscription.plans.free.features.orders'),
          t('subscription.plans.free.features.support'),
        ],
        current: currentPlanId === 'free',
        icon: <IconStar size="large" />,
      },
      {
        id: 'basic',
        name: t('subscription.plans.basic.name'),
        price: 29,
        priceUnit: t('subscription.priceUnit'),
        description: t('subscription.plans.basic.description'),
        features: [
          t('subscription.plans.basic.features.users'),
          t('subscription.plans.basic.features.products'),
          t('subscription.plans.basic.features.orders'),
          t('subscription.plans.basic.features.support'),
          t('subscription.plans.basic.features.reports'),
        ],
        current: currentPlanId === 'basic',
        icon: <IconVerify size="large" />,
      },
      {
        id: 'pro',
        name: t('subscription.plans.pro.name'),
        price: 79,
        priceUnit: t('subscription.priceUnit'),
        description: t('subscription.plans.pro.description'),
        features: [
          t('subscription.plans.pro.features.users'),
          t('subscription.plans.pro.features.products'),
          t('subscription.plans.pro.features.orders'),
          t('subscription.plans.pro.features.support'),
          t('subscription.plans.pro.features.reports'),
          t('subscription.plans.pro.features.api'),
          t('subscription.plans.pro.features.integrations'),
        ],
        highlighted: true,
        current: currentPlanId === 'pro',
        icon: <IconCrown size="large" />,
      },
      {
        id: 'enterprise',
        name: t('subscription.plans.enterprise.name'),
        price: -1,
        priceUnit: t('subscription.priceUnit'),
        description: t('subscription.plans.enterprise.description'),
        features: [
          t('subscription.plans.enterprise.features.users'),
          t('subscription.plans.enterprise.features.products'),
          t('subscription.plans.enterprise.features.orders'),
          t('subscription.plans.enterprise.features.support'),
          t('subscription.plans.enterprise.features.reports'),
          t('subscription.plans.enterprise.features.api'),
          t('subscription.plans.enterprise.features.integrations'),
          t('subscription.plans.enterprise.features.sla'),
          t('subscription.plans.enterprise.features.custom'),
        ],
        current: currentPlanId === 'enterprise',
        icon: <IconCrown size="large" />,
      },
    ]
  }, [availablePlans, currentPlanId, t])

  // Handle plan selection
  const handleSelectPlan = useCallback((planId: string) => {
    setSelectedPlan(planId)
  }, [])

  // Handle upgrade/subscribe
  const handleUpgrade = useCallback(async () => {
    if (!selectedPlan || selectedPlan === currentPlanId) {
      return
    }

    if (selectedPlan === 'enterprise') {
      Toast.info(t('subscription.messages.contactSales'))
      return
    }

    setIsUpgrading(true)

    try {
      // TODO: Integrate with Stripe Checkout
      // 1. Call backend to create Stripe Checkout Session
      // 2. Redirect to Stripe Checkout page
      // 3. Handle success/cancel callbacks

      // Simulated API call
      await new Promise((resolve) => setTimeout(resolve, 1500))

      Toast.success(t('subscription.messages.upgradeSuccess'))
    } catch {
      Toast.error(t('subscription.messages.upgradeError'))
    } finally {
      setIsUpgrading(false)
    }
  }, [selectedPlan, currentPlanId, t])

  // Handle contact sales
  const handleContactSales = useCallback(() => {
    window.open('mailto:sales@example.com?subject=Enterprise Plan Inquiry', '_blank')
  }, [])

  // Get button text based on plan status
  const getButtonText = useCallback(
    (plan: PlanTier) => {
      if (plan.current) {
        return t('subscription.currentPlan')
      }
      if (plan.id === 'enterprise') {
        return t('subscription.contactSales')
      }
      if (plan.id === 'free') {
        return t('subscription.downgrade')
      }
      return t('subscription.upgrade')
    },
    [t]
  )

  // Check if plan is selectable
  const isPlanSelectable = useCallback(
    (plan: PlanTier) => {
      return !plan.current && plan.id !== currentPlanId
    },
    [currentPlanId]
  )

  // Loading state
  const isLoading = isLoadingSubscription || isLoadingPlans

  // Render loading skeleton
  if (isLoading) {
    return (
      <Container size="lg" className="upgrade-plan-page">
        <div className="upgrade-plan-header">
          <Skeleton.Title style={{ width: 200, margin: '0 auto' }} />
          <Skeleton.Paragraph rows={1} style={{ width: 300, margin: '16px auto 0' }} />
        </div>

        <Skeleton.Paragraph rows={2} style={{ marginBottom: 24 }} />

        <div className="plan-cards-container">
          {[1, 2, 3, 4].map((i) => (
            <Card key={i} className="plan-card">
              <Skeleton.Avatar size="large" style={{ marginBottom: 16 }} />
              <Skeleton.Title style={{ width: 100, marginBottom: 8 }} />
              <Skeleton.Title style={{ width: 80, marginBottom: 16 }} />
              <Skeleton.Paragraph rows={4} />
              <Skeleton.Button style={{ width: '100%', marginTop: 16 }} />
            </Card>
          ))}
        </div>
      </Container>
    )
  }

  // Error state
  if (subscriptionError && plansError) {
    return (
      <Container size="lg" className="upgrade-plan-page">
        <Banner type="danger" description={t('billing.messages.loadError')} closeIcon={null} />
      </Container>
    )
  }

  return (
    <Container size="lg" className="upgrade-plan-page">
      <div className="upgrade-plan-header">
        <Title heading={3}>{t('subscription.title')}</Title>
        <Text type="tertiary">{t('subscription.subtitle')}</Text>
      </div>

      {/* Current Plan Banner with Quota Usage */}
      <Banner
        type="info"
        description={
          <div className="current-plan-info">
            <div className="current-plan-text">
              {t('subscription.currentPlanBanner', {
                plan:
                  currentSubscription?.plan_name || plans.find((p) => p.current)?.name || 'Free',
              })}
              {currentSubscription?.status === 'trial' && currentSubscription.trial_ends_at && (
                <Tag color="orange" style={{ marginLeft: 8 }}>
                  {t('subscriptionPage.trialBanner', {
                    days: Math.ceil(
                      (new Date(currentSubscription.trial_ends_at).getTime() - Date.now()) /
                        (1000 * 60 * 60 * 24)
                    ),
                  })}
                </Tag>
              )}
            </div>

            {/* Quota Usage Progress Bars */}
            {currentSubscription?.quotas && currentSubscription.quotas.length > 0 && (
              <div className="quota-usage-section">
                <Text strong style={{ marginBottom: 8, display: 'block' }}>
                  {t('subscriptionPage.resourceUsage')}
                </Text>
                <div className="quota-progress-list">
                  {currentSubscription.quotas.map((quota) => {
                    const percentage = getQuotaPercentage(quota)
                    const isUnlimited = quota.limit === -1

                    return (
                      <div key={quota.type} className="quota-progress-item">
                        <div className="quota-progress-header">
                          <Text>{formatQuotaDisplay(quota, t)}</Text>
                          {!isUnlimited && <Text type="tertiary">{percentage}%</Text>}
                        </div>
                        {!isUnlimited && (
                          <Progress
                            percent={percentage}
                            showInfo={false}
                            size="small"
                            stroke={
                              getProgressStatus(percentage) === 'exception'
                                ? 'var(--semi-color-danger)'
                                : getProgressStatus(percentage) === 'warning'
                                  ? 'var(--semi-color-warning)'
                                  : 'var(--semi-color-success)'
                            }
                          />
                        )}
                        {isUnlimited && (
                          <Progress
                            percent={100}
                            showInfo={false}
                            size="small"
                            stroke="var(--semi-color-primary)"
                          />
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            )}
          </div>
        }
        className="current-plan-banner"
      />

      {/* Plan Cards */}
      <div className="plan-cards-container">
        {plans.map((plan) => (
          <div
            key={plan.id}
            className={`plan-card-wrapper ${isPlanSelectable(plan) ? 'plan-card-wrapper--selectable' : ''}`}
            onClick={() => isPlanSelectable(plan) && handleSelectPlan(plan.id)}
            role={isPlanSelectable(plan) ? 'button' : undefined}
            tabIndex={isPlanSelectable(plan) ? 0 : undefined}
            onKeyDown={(e) => {
              if (isPlanSelectable(plan) && (e.key === 'Enter' || e.key === ' ')) {
                e.preventDefault()
                handleSelectPlan(plan.id)
              }
            }}
          >
            <Card
              className={`plan-card ${plan.highlighted ? 'plan-card--highlighted' : ''} ${
                selectedPlan === plan.id ? 'plan-card--selected' : ''
              } ${plan.current ? 'plan-card--current' : ''}`}
            >
              {plan.highlighted && (
                <Tag color="orange" className="plan-badge">
                  {t('subscription.popular')}
                </Tag>
              )}
              {plan.current && (
                <Tag color="green" className="plan-badge">
                  {t('subscription.current')}
                </Tag>
              )}

              <div className="plan-icon">{plan.icon}</div>

              <Title heading={4} className="plan-name">
                {plan.name}
              </Title>

              <div className="plan-price">
                {plan.price === -1 ? (
                  <Text className="plan-price-custom">{t('subscription.customPricing')}</Text>
                ) : (
                  <>
                    <Text className="plan-price-currency">$</Text>
                    <Text className="plan-price-amount">{plan.price}</Text>
                    <Text className="plan-price-unit">/{plan.priceUnit}</Text>
                  </>
                )}
              </div>

              <Text type="tertiary" className="plan-description">
                {plan.description}
              </Text>

              <ul className="plan-features">
                {plan.features.map((feature, index) => (
                  <li key={index} className="plan-feature-item">
                    <IconTick className="plan-feature-icon" />
                    <Text>{feature}</Text>
                  </li>
                ))}
              </ul>

              <Button
                theme={plan.highlighted ? 'solid' : 'light'}
                type={plan.current ? 'tertiary' : 'primary'}
                block
                disabled={plan.current}
                onClick={(e) => {
                  e.stopPropagation()
                  if (plan.id === 'enterprise') {
                    handleContactSales()
                  } else if (isPlanSelectable(plan)) {
                    handleSelectPlan(plan.id)
                  }
                }}
              >
                {getButtonText(plan)}
              </Button>
            </Card>
          </div>
        ))}
      </div>

      {/* Upgrade Action */}
      {selectedPlan && selectedPlan !== currentPlanId && selectedPlan !== 'enterprise' && (
        <div className="upgrade-action-container">
          <Card className="upgrade-action-card">
            <div className="upgrade-action-content">
              <div className="upgrade-action-info">
                <Title heading={5}>
                  {t('subscription.confirmUpgrade', {
                    plan: plans.find((p) => p.id === selectedPlan)?.name,
                  })}
                </Title>
                <Text type="tertiary">{t('subscription.upgradeDescription')}</Text>
              </div>
              <Button
                theme="solid"
                type="primary"
                size="large"
                loading={isUpgrading}
                onClick={handleUpgrade}
              >
                {isUpgrading ? <Spin /> : t('subscription.proceedToPayment')}
              </Button>
            </div>
          </Card>
        </div>
      )}

      {/* FAQ Section */}
      <div className="subscription-faq">
        <Title heading={5}>{t('subscription.faq.title')}</Title>
        <div className="faq-items">
          <div className="faq-item">
            <Text strong>{t('subscription.faq.q1')}</Text>
            <Text type="tertiary">{t('subscription.faq.a1')}</Text>
          </div>
          <div className="faq-item">
            <Text strong>{t('subscription.faq.q2')}</Text>
            <Text type="tertiary">{t('subscription.faq.a2')}</Text>
          </div>
          <div className="faq-item">
            <Text strong>{t('subscription.faq.q3')}</Text>
            <Text type="tertiary">{t('subscription.faq.a3')}</Text>
          </div>
        </div>
      </div>
    </Container>
  )
}
