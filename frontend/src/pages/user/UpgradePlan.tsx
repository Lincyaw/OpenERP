import { useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, Typography, Button, Toast, Tag, Spin, Banner } from '@douyinfe/semi-ui-19'
import { IconTick, IconCrown, IconStar, IconVerify } from '@douyinfe/semi-icons'

import { Container } from '@/components/common/layout'

import './UpgradePlan.css'

const { Title, Text } = Typography

/**
 * Plan tier definition
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
 * Subscription Upgrade Plan Page
 *
 * Features:
 * - Display available subscription plans
 * - Show current plan status
 * - Allow plan selection and upgrade
 * - Integrate with Stripe for payment
 */
export default function UpgradePlanPage() {
  const { t } = useTranslation('system')

  // State
  const [selectedPlan, setSelectedPlan] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [currentPlan] = useState<string>('free') // TODO: Get from API

  // Plan definitions
  const plans: PlanTier[] = useMemo(
    () => [
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
        current: currentPlan === 'free',
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
        current: currentPlan === 'basic',
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
        current: currentPlan === 'pro',
        icon: <IconCrown size="large" />,
      },
      {
        id: 'enterprise',
        name: t('subscription.plans.enterprise.name'),
        price: -1, // Custom pricing
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
        current: currentPlan === 'enterprise',
        icon: <IconCrown size="large" />,
      },
    ],
    [t, currentPlan]
  )

  // Handle plan selection
  const handleSelectPlan = useCallback((planId: string) => {
    setSelectedPlan(planId)
  }, [])

  // Handle upgrade/subscribe
  const handleUpgrade = useCallback(async () => {
    if (!selectedPlan || selectedPlan === currentPlan) {
      return
    }

    if (selectedPlan === 'enterprise') {
      // Contact sales for enterprise
      Toast.info(t('subscription.messages.contactSales'))
      return
    }

    setIsLoading(true)

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
      setIsLoading(false)
    }
  }, [selectedPlan, currentPlan, t])

  // Handle contact sales
  const handleContactSales = useCallback(() => {
    // Open contact form or email
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
      return !plan.current && plan.id !== currentPlan
    },
    [currentPlan]
  )

  return (
    <Container size="lg" className="upgrade-plan-page">
      <div className="upgrade-plan-header">
        <Title heading={3}>{t('subscription.title')}</Title>
        <Text type="tertiary">{t('subscription.subtitle')}</Text>
      </div>

      {/* Current Plan Banner */}
      <Banner
        type="info"
        description={t('subscription.currentPlanBanner', {
          plan: plans.find((p) => p.current)?.name || 'Free',
        })}
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
      {selectedPlan && selectedPlan !== currentPlan && selectedPlan !== 'enterprise' && (
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
                loading={isLoading}
                onClick={handleUpgrade}
              >
                {isLoading ? <Spin /> : t('subscription.proceedToPayment')}
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
