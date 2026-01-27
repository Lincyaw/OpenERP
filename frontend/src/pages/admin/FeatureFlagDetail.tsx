import { useState, useEffect, useCallback, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Spin,
  Tabs,
  TabPane,
  Descriptions,
  Button,
  Switch,
  Space,
  Empty,
} from '@douyinfe/semi-ui-19'
import { IconEdit, IconArrowLeft, IconRefresh, IconChevronRight } from '@douyinfe/semi-icons'
import { Container } from '@/components/common/layout'
import { getFeatureFlags } from '@/api/feature-flags'
import type {
  FeatureFlag,
  FlagType,
  FlagStatus,
  TargetingRule,
  FlagValue,
} from '@/api/feature-flags'
import { OverridesTab } from './components/OverridesTab'
import { AuditLogTimeline } from './components/AuditLogTimeline'
import './FeatureFlagDetail.css'

const { Title, Text } = Typography

/**
 * Get color for flag type badge
 */
function getTypeColor(type: FlagType): string {
  switch (type) {
    case 'boolean':
      return 'blue'
    case 'percentage':
      return 'orange'
    case 'variant':
      return 'purple'
    case 'user_segment':
      return 'cyan'
    default:
      return 'grey'
  }
}

/**
 * Get color for flag status
 */
function getStatusColor(status: FlagStatus): string {
  switch (status) {
    case 'enabled':
      return 'green'
    case 'disabled':
      return 'grey'
    case 'archived':
      return 'red'
    default:
      return 'grey'
  }
}

/**
 * Format date for display
 */
function formatDate(dateStr: string | undefined, locale: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/**
 * Feature Flag Detail Page
 *
 * Features:
 * - Display flag complete information
 * - Tab switching: Configuration / Overrides / Audit Logs
 * - Toggle enable/disable status
 * - Edit flag navigation
 */
export default function FeatureFlagDetailPage() {
  const { t } = useTranslation('admin')
  const { key } = useParams<{ key: string }>()
  const navigate = useNavigate()
  const api = useMemo(() => getFeatureFlags(), [])

  // State
  const [flag, setFlag] = useState<FeatureFlag | null>(null)
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState('config')

  // Fetch flag data
  const fetchFlag = useCallback(async () => {
    if (!key) return

    setLoading(true)
    try {
      const response = await api.getFlag(key)
      if (response.success && response.data) {
        setFlag(response.data)
      } else {
        Toast.error(t('featureFlags.messages.loadError', 'Failed to load feature flag'))
        navigate('/admin/feature-flags')
      }
    } catch {
      Toast.error(t('featureFlags.messages.loadError', 'Failed to load feature flag'))
      navigate('/admin/feature-flags')
    } finally {
      setLoading(false)
    }
  }, [key, api, t, navigate])

  // Load flag on mount
  useEffect(() => {
    fetchFlag()
  }, [fetchFlag])

  // Handle toggle status
  const handleToggleStatus = useCallback(
    async (checked: boolean) => {
      if (!flag) return

      try {
        const response = checked ? await api.enableFlag(flag.key) : await api.disableFlag(flag.key)

        if (response.success) {
          Toast.success(
            checked
              ? t('featureFlags.messages.enableSuccess', 'Feature flag enabled')
              : t('featureFlags.messages.disableSuccess', 'Feature flag disabled')
          )
          fetchFlag() // Refresh data
        } else {
          Toast.error(
            response.error?.message ||
              (checked
                ? t('featureFlags.messages.enableError', 'Failed to enable feature flag')
                : t('featureFlags.messages.disableError', 'Failed to disable feature flag'))
          )
        }
      } catch {
        Toast.error(
          checked
            ? t('featureFlags.messages.enableError', 'Failed to enable feature flag')
            : t('featureFlags.messages.disableError', 'Failed to disable feature flag')
        )
      }
    },
    [flag, api, t, fetchFlag]
  )

  // Handle edit navigation
  const handleEdit = useCallback(() => {
    if (flag) {
      navigate(`/admin/feature-flags/${flag.key}/edit`)
    }
  }, [flag, navigate])

  // Handle back navigation
  const handleBack = useCallback(() => {
    navigate('/admin/feature-flags')
  }, [navigate])

  // Render loading state
  if (loading) {
    return (
      <Container size="lg" className="feature-flag-detail-page">
        <Card className="feature-flag-detail-card">
          <div className="feature-flag-detail-loading">
            <Spin size="large" />
          </div>
        </Card>
      </Container>
    )
  }

  // Render not found
  if (!flag) {
    return (
      <Container size="lg" className="feature-flag-detail-page">
        <Card className="feature-flag-detail-card">
          <Empty
            title={t('featureFlags.notFound', 'Feature flag not found')}
            description={t(
              'featureFlags.notFoundDescription',
              'The requested feature flag does not exist'
            )}
          >
            <Button theme="solid" onClick={handleBack}>
              {t('featureFlags.backToList', 'Back to list')}
            </Button>
          </Empty>
        </Card>
      </Container>
    )
  }

  return (
    <Container size="lg" className="feature-flag-detail-page">
      {/* Header */}
      <div className="feature-flag-detail-header">
        <div className="feature-flag-detail-header-left">
          <Button
            icon={<IconArrowLeft />}
            theme="borderless"
            onClick={handleBack}
            className="back-button"
          />
          <div className="feature-flag-detail-title-section">
            <div className="feature-flag-detail-title-row">
              <Title heading={4} className="feature-flag-detail-title">
                {flag.name}
              </Title>
              <Space>
                <Tag color={getTypeColor(flag.type)} size="large">
                  {t(`featureFlags.type.${flag.type}`, flag.type)}
                </Tag>
                <Tag color={getStatusColor(flag.status)} size="large">
                  {t(`featureFlags.status.${flag.status}`, flag.status)}
                </Tag>
              </Space>
            </div>
            <div className="feature-flag-detail-key">
              <Text type="tertiary" className="key-label">
                Key:
              </Text>
              <Text code copyable className="key-value">
                {flag.key}
              </Text>
            </div>
          </div>
        </div>
        <div className="feature-flag-detail-header-right">
          {flag.status !== 'archived' && (
            <Space>
              <Switch
                checked={flag.status === 'enabled'}
                onChange={handleToggleStatus}
                checkedText={t('featureFlags.status.enabled', 'Enabled')}
                uncheckedText={t('featureFlags.status.disabled', 'Disabled')}
              />
              <Button icon={<IconRefresh />} onClick={fetchFlag}>
                {t('common.refresh', 'Refresh')}
              </Button>
              <Button icon={<IconEdit />} theme="solid" onClick={handleEdit}>
                {t('featureFlags.actions.edit', 'Edit')}
              </Button>
            </Space>
          )}
        </div>
      </div>

      {/* Content */}
      <Card className="feature-flag-detail-card">
        <Tabs activeKey={activeTab} onChange={setActiveTab}>
          <TabPane tab={t('featureFlags.tabs.configuration', 'Configuration')} itemKey="config">
            <ConfigurationTab flag={flag} />
          </TabPane>
          <TabPane tab={t('featureFlags.tabs.overrides', 'Overrides')} itemKey="overrides">
            <OverridesTab flagKey={flag.key} flagType={flag.type} />
          </TabPane>
          <TabPane tab={t('featureFlags.tabs.auditLog', 'Audit Log')} itemKey="audit">
            <AuditLogTimeline flagKey={flag.key} />
          </TabPane>
        </Tabs>
      </Card>
    </Container>
  )
}

/**
 * Configuration Tab Component
 */
interface ConfigurationTabProps {
  flag: FeatureFlag
}

function ConfigurationTab({ flag }: ConfigurationTabProps) {
  const { t, i18n } = useTranslation('admin')

  return (
    <div className="configuration-tab">
      {/* Basic Information */}
      <section className="config-section">
        <Title heading={5} className="config-section-title">
          {t('featureFlags.detail.basicInfo', 'Basic Information')}
        </Title>
        <Descriptions align="left" row className="config-descriptions">
          <Descriptions.Item itemKey={t('featureFlags.columns.name', 'Name')}>
            {flag.name}
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('featureFlags.columns.key', 'Key')}>
            <Text code copyable>
              {flag.key}
            </Text>
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('featureFlags.columns.type', 'Type')}>
            <Tag color={getTypeColor(flag.type)}>
              {t(`featureFlags.type.${flag.type}`, flag.type)}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('featureFlags.columns.status', 'Status')}>
            <Tag color={getStatusColor(flag.status)}>
              {t(`featureFlags.status.${flag.status}`, flag.status)}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('featureFlags.detail.description', 'Description')}>
            {flag.description || <Text type="tertiary">-</Text>}
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('featureFlags.columns.tags', 'Tags')}>
            {flag.tags && flag.tags.length > 0 ? (
              <Space wrap>
                {flag.tags.map((tag) => (
                  <Tag key={tag} color="light-blue">
                    {tag}
                  </Tag>
                ))}
              </Space>
            ) : (
              <Text type="tertiary">-</Text>
            )}
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('featureFlags.detail.version', 'Version')}>
            {flag.version}
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('featureFlags.detail.createdAt', 'Created At')}>
            {formatDate(flag.created_at, i18n.language)}
          </Descriptions.Item>
          <Descriptions.Item itemKey={t('featureFlags.columns.updatedAt', 'Updated At')}>
            {formatDate(flag.updated_at, i18n.language)}
          </Descriptions.Item>
        </Descriptions>
      </section>

      {/* Default Value */}
      <section className="config-section">
        <Title heading={5} className="config-section-title">
          {t('featureFlags.form.defaultValue', 'Default Value')}
        </Title>
        <DefaultValueDisplay value={flag.default_value} type={flag.type} />
      </section>

      {/* Targeting Rules */}
      <section className="config-section">
        <Title heading={5} className="config-section-title">
          {t('featureFlags.form.targetingRules', 'Targeting Rules')}
        </Title>
        {flag.rules && flag.rules.length > 0 ? (
          <RulesDisplay rules={flag.rules} type={flag.type} />
        ) : (
          <div className="empty-rules">
            <Text type="tertiary">
              {t(
                'featureFlags.form.noRules',
                'No targeting rules. All users will receive the default value.'
              )}
            </Text>
          </div>
        )}
      </section>
    </div>
  )
}

/**
 * Default Value Display Component
 */
interface DefaultValueDisplayProps {
  value: FlagValue
  type: FlagType
}

function DefaultValueDisplay({ value, type }: DefaultValueDisplayProps) {
  const { t } = useTranslation('admin')

  if (type === 'boolean' || type === 'user_segment') {
    return (
      <div className="default-value-display">
        <Tag color={value.enabled ? 'green' : 'grey'} size="large">
          {value.enabled
            ? t('featureFlags.status.enabled', 'Enabled')
            : t('featureFlags.status.disabled', 'Disabled')}
        </Tag>
      </div>
    )
  }

  if (type === 'percentage') {
    const percentage = (value.metadata?.percentage as number) || 0
    return (
      <div className="default-value-display">
        <div className="percentage-display">
          <div className="percentage-bar">
            <div className="percentage-bar-fill" style={{ width: `${percentage}%` }} />
          </div>
          <div className="percentage-labels">
            <Tag color="green">
              {t('featureFlags.form.enabledUsers', 'Enabled')}: {percentage}%
            </Tag>
            <Tag color="grey">
              {t('featureFlags.form.disabledUsers', 'Disabled')}: {100 - percentage}%
            </Tag>
          </div>
        </div>
      </div>
    )
  }

  if (type === 'variant') {
    const variants = (value.metadata?.variants as Array<{ name: string; weight: number }>) || []
    return (
      <div className="default-value-display">
        <div className="variant-display">
          <div className="variant-bar">
            {variants.map((variant, index) => (
              <div
                key={variant.name}
                className="variant-bar-segment"
                style={{
                  width: `${variant.weight}%`,
                  backgroundColor: getVariantColor(index),
                }}
                title={`${variant.name}: ${variant.weight}%`}
              >
                {variant.weight >= 10 && variant.name}
              </div>
            ))}
          </div>
          <div className="variant-legend">
            {variants.map((variant, index) => (
              <div key={variant.name} className="variant-legend-item">
                <div
                  className="variant-legend-color"
                  style={{ backgroundColor: getVariantColor(index) }}
                />
                <Text>
                  {variant.name}: {variant.weight}%
                </Text>
              </div>
            ))}
          </div>
          {value.variant && (
            <div className="default-variant-info">
              <Text type="secondary">
                {t('featureFlags.form.defaultVariant', 'Default Variant')}:
              </Text>
              <Tag color="blue">{value.variant}</Tag>
            </div>
          )}
        </div>
      </div>
    )
  }

  return null
}

// Variant color palette
const VARIANT_COLORS = [
  '#0077FA', // Primary blue
  '#F5222D', // Red
  '#52C41A', // Green
  '#FA8C16', // Orange
  '#722ED1', // Purple
  '#13C2C2', // Cyan
  '#EB2F96', // Pink
  '#FAAD14', // Yellow
]

function getVariantColor(index: number): string {
  return VARIANT_COLORS[index % VARIANT_COLORS.length]
}

/**
 * Rules Display Component
 */
interface RulesDisplayProps {
  rules: TargetingRule[]
  type: FlagType
}

function RulesDisplay({ rules, type }: RulesDisplayProps) {
  const { t } = useTranslation('admin')

  return (
    <div className="rules-display">
      {rules.map((rule, index) => (
        <div key={rule.rule_id} className="rule-display-item">
          <div className="rule-display-header">
            <div className="rule-display-priority">
              <Tag color="blue">
                {t('featureFlags.form.rulePriority', 'Rule {{n}}', { n: index + 1 })}
              </Tag>
            </div>
          </div>
          <div className="rule-display-conditions">
            <Text type="secondary" size="small">
              {t('featureFlags.detail.ifConditions', 'If conditions match')}:
            </Text>
            <div className="conditions-list">
              {rule.conditions.map((condition, condIndex) => (
                <div key={condIndex} className="condition-display">
                  {condIndex > 0 && <Tag size="small">AND</Tag>}
                  <Tag color="light-blue">
                    {t(`featureFlags.form.attributes.${condition.attribute}`, condition.attribute)}
                  </Tag>
                  <Text>{getOperatorLabel(condition.operator)}</Text>
                  <Tag color="amber">{condition.values.join(', ')}</Tag>
                </div>
              ))}
            </div>
          </div>
          <div className="rule-display-value">
            <IconChevronRight style={{ color: 'var(--semi-color-text-2)' }} />
            <Text type="secondary" size="small">
              {t('featureFlags.detail.thenReturn', 'Then return')}:
            </Text>
            <RuleValueDisplay value={rule.value} type={type} percentage={rule.percentage} />
          </div>
        </div>
      ))}
    </div>
  )
}

function getOperatorLabel(operator: string): string {
  const operatorLabels: Record<string, string> = {
    eq: '=',
    neq: '≠',
    contains: 'contains',
    not_contains: 'not contains',
    starts_with: 'starts with',
    ends_with: 'ends with',
    gt: '>',
    gte: '≥',
    lt: '<',
    lte: '≤',
    in: 'in',
    not_in: 'not in',
    regex: 'regex',
  }
  return operatorLabels[operator] || operator
}

/**
 * Rule Value Display Component
 */
interface RuleValueDisplayProps {
  value: FlagValue
  type: FlagType
  percentage: number
}

function RuleValueDisplay({ value, type, percentage }: RuleValueDisplayProps) {
  const { t } = useTranslation('admin')

  if (type === 'boolean' || type === 'user_segment') {
    return (
      <Tag color={value.enabled ? 'green' : 'grey'}>
        {value.enabled
          ? t('featureFlags.status.enabled', 'Enabled')
          : t('featureFlags.status.disabled', 'Disabled')}
      </Tag>
    )
  }

  if (type === 'percentage') {
    return <Tag color="orange">{percentage}%</Tag>
  }

  if (type === 'variant') {
    return <Tag color="purple">{value.variant || '-'}</Tag>
  }

  return <Text>-</Text>
}
