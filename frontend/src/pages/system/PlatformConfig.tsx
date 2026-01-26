import { useState, useCallback, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Toast,
  Tabs,
  TabPane,
  Form,
  Switch,
  Button,
  Banner,
  Spin,
  Space,
  Tag,
  Descriptions,
  InputNumber,
  Tooltip,
} from '@douyinfe/semi-ui-19'
import {
  IconTick,
  IconClose,
  IconRefresh,
  IconLink,
  IconEyeOpened,
  IconEyeClosed,
  IconInfoCircle,
} from '@douyinfe/semi-icons'
import { Container } from '@/components/common/layout'
import './PlatformConfig.css'

const { Title, Text } = Typography

/**
 * E-commerce platform codes matching backend domain
 */
type PlatformCode = 'TAOBAO' | 'DOUYIN' | 'JD' | 'PDD' | 'WECHAT' | 'KUAISHOU'

/**
 * Platform configuration interface
 */
interface PlatformConfig {
  enabled: boolean
  appKey: string
  appSecret: string
  accessToken: string
  refreshToken: string
  shopId: string
  shopName: string
  syncEnabled: boolean
  syncIntervalMinutes: number
  inventorySyncEnabled: boolean
  orderAutoImport: boolean
  notifyUrl: string
  isSandbox: boolean
  lastSyncAt?: string
}

/**
 * Platform status for display
 */
interface PlatformStatus {
  code: PlatformCode
  enabled: boolean
  configured: boolean
  connected: boolean
  lastTestedAt?: string
  testResult?: 'success' | 'failed'
  errorMessage?: string
}

/**
 * Platform metadata for display
 */
interface PlatformMeta {
  code: PlatformCode
  name: string
  icon: string
  color: string
  description: string
}

/**
 * Available platforms with metadata
 */
const PLATFORMS: PlatformMeta[] = [
  {
    code: 'TAOBAO',
    name: '淘宝/天猫',
    icon: 'TB',
    color: '#FF5000',
    description: '淘宝开放平台 & 天猫商家',
  },
  {
    code: 'DOUYIN',
    name: '抖音',
    icon: 'DY',
    color: '#000000',
    description: '抖音电商开放平台',
  },
  {
    code: 'JD',
    name: '京东',
    icon: 'JD',
    color: '#E2231A',
    description: '京东开放平台',
  },
  {
    code: 'PDD',
    name: '拼多多',
    icon: 'PDD',
    color: '#E02E24',
    description: '拼多多开放平台',
  },
  {
    code: 'WECHAT',
    name: '微信小商店',
    icon: 'WX',
    color: '#07C160',
    description: '微信小程序/视频号小店',
  },
  {
    code: 'KUAISHOU',
    name: '快手',
    icon: 'KS',
    color: '#FF4906',
    description: '快手电商开放平台',
  },
]

/**
 * Default platform configuration
 */
const defaultConfig: PlatformConfig = {
  enabled: false,
  appKey: '',
  appSecret: '',
  accessToken: '',
  refreshToken: '',
  shopId: '',
  shopName: '',
  syncEnabled: false,
  syncIntervalMinutes: 15,
  inventorySyncEnabled: false,
  orderAutoImport: true,
  notifyUrl: '',
  isSandbox: false,
}

/**
 * E-commerce Platform Configuration Page
 *
 * Features:
 * - Configure platform authorization (OAuth credentials)
 * - Configure sync parameters (intervals, toggles)
 * - Test platform connection
 * - Enable/disable platforms
 */
export default function PlatformConfigPage() {
  const { t } = useTranslation('system')

  // Platform configurations (one per platform)
  const [configs, setConfigs] = useState<Record<PlatformCode, PlatformConfig>>(() =>
    PLATFORMS.reduce(
      (acc, p) => ({ ...acc, [p.code]: { ...defaultConfig } }),
      {} as Record<PlatformCode, PlatformConfig>
    )
  )

  // Platform statuses
  const [statuses, setStatuses] = useState<PlatformStatus[]>(() =>
    PLATFORMS.map((p) => ({
      code: p.code,
      enabled: false,
      configured: false,
      connected: false,
    }))
  )

  // Loading states
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState<PlatformCode | null>(null)
  const [testing, setTesting] = useState<PlatformCode | null>(null)

  // Visibility states for sensitive fields
  const [showSecrets, setShowSecrets] = useState<Record<PlatformCode, boolean>>(() =>
    PLATFORMS.reduce((acc, p) => ({ ...acc, [p.code]: false }), {} as Record<PlatformCode, boolean>)
  )

  // Active tab
  const [activeTab, setActiveTab] = useState<string>(PLATFORMS[0].code)

  // Load configurations on mount
  useEffect(() => {
    loadConfigurations()
  }, [])

  /**
   * Load platform configurations from backend
   */
  const loadConfigurations = useCallback(async () => {
    setLoading(true)
    try {
      // TODO: Implement API call to load configurations
      // const response = await api.getPlatformConfigs()
      await new Promise((resolve) => setTimeout(resolve, 500))

      // For now, use default configs
      // In production, update configs state from API response
    } catch {
      Toast.error(t('platformConfig.messages.loadError'))
    } finally {
      setLoading(false)
    }
  }, [t])

  /**
   * Get status for a platform
   */
  const getStatus = useCallback(
    (code: PlatformCode): PlatformStatus | undefined => {
      return statuses.find((s) => s.code === code)
    },
    [statuses]
  )

  /**
   * Update platform status
   */
  const updateStatus = useCallback((code: PlatformCode, update: Partial<PlatformStatus>) => {
    setStatuses((prev) => prev.map((s) => (s.code === code ? { ...s, ...update } : s)))
  }, [])

  /**
   * Handle config field change
   */
  const handleConfigChange = useCallback(
    <K extends keyof PlatformConfig>(code: PlatformCode, field: K, value: PlatformConfig[K]) => {
      setConfigs((prev) => ({
        ...prev,
        [code]: { ...prev[code], [field]: value },
      }))
    },
    []
  )

  /**
   * Validate platform configuration
   */
  const validateConfig = useCallback(
    (code: PlatformCode): string | null => {
      const config = configs[code]
      if (!config.appKey) return t('platformConfig.errors.appKeyRequired')
      if (!config.appSecret) return t('platformConfig.errors.appSecretRequired')
      if (config.syncEnabled && config.syncIntervalMinutes < 5) {
        return t('platformConfig.errors.syncIntervalMin')
      }
      if (config.syncEnabled && config.syncIntervalMinutes > 60) {
        return t('platformConfig.errors.syncIntervalMax')
      }
      return null
    },
    [configs, t]
  )

  /**
   * Save platform configuration
   */
  const handleSave = useCallback(
    async (code: PlatformCode) => {
      const error = validateConfig(code)
      if (error) {
        Toast.error(error)
        return
      }

      setSaving(code)
      try {
        // TODO: Implement API call to save configuration
        // await api.savePlatformConfig(code, configs[code])
        await new Promise((resolve) => setTimeout(resolve, 1000))

        const config = configs[code]
        updateStatus(code, {
          configured: true,
          enabled: config.enabled,
        })

        Toast.success(t('platformConfig.messages.saveSuccess'))
      } catch {
        Toast.error(t('platformConfig.messages.saveError'))
      } finally {
        setSaving(null)
      }
    },
    [configs, validateConfig, updateStatus, t]
  )

  /**
   * Test platform connection
   */
  const handleTestConnection = useCallback(
    async (code: PlatformCode) => {
      const error = validateConfig(code)
      if (error) {
        Toast.error(error)
        return
      }

      setTesting(code)
      try {
        // TODO: Implement API call to test connection
        // const result = await api.testPlatformConnection(code)
        await new Promise((resolve) => setTimeout(resolve, 2000))

        // Simulate random success/failure for demo
        const success = Math.random() > 0.3

        updateStatus(code, {
          lastTestedAt: new Date().toISOString(),
          testResult: success ? 'success' : 'failed',
          connected: success,
          errorMessage: success ? undefined : 'Token invalid or expired',
        })

        if (success) {
          Toast.success(t('platformConfig.messages.testSuccess'))
        } else {
          Toast.error(t('platformConfig.messages.testFailed'))
        }
      } catch {
        updateStatus(code, {
          lastTestedAt: new Date().toISOString(),
          testResult: 'failed',
          connected: false,
          errorMessage: 'Network error',
        })
        Toast.error(t('platformConfig.messages.testError'))
      } finally {
        setTesting(null)
      }
    },
    [validateConfig, updateStatus, t]
  )

  /**
   * Toggle secret visibility
   */
  const toggleSecretVisibility = useCallback((code: PlatformCode) => {
    setShowSecrets((prev) => ({ ...prev, [code]: !prev[code] }))
  }, [])

  /**
   * Render status tag
   */
  const renderStatusTag = useCallback(
    (status: PlatformStatus | undefined) => {
      if (!status) return null

      if (!status.configured) {
        return <Tag color="grey">{t('platformConfig.status.notConfigured')}</Tag>
      }

      if (!status.enabled) {
        return <Tag color="orange">{t('platformConfig.status.disabled')}</Tag>
      }

      if (status.testResult === 'success') {
        return (
          <Tag color="green" prefixIcon={<IconTick />}>
            {t('platformConfig.status.connected')}
          </Tag>
        )
      }

      if (status.testResult === 'failed') {
        return (
          <Tag color="red" prefixIcon={<IconClose />}>
            {t('platformConfig.status.failed')}
          </Tag>
        )
      }

      return <Tag color="blue">{t('platformConfig.status.configured')}</Tag>
    },
    [t]
  )

  /**
   * Render platform form
   */
  const renderPlatformForm = useCallback(
    (platform: PlatformMeta) => {
      const config = configs[platform.code]
      const status = getStatus(platform.code)
      const showSecret = showSecrets[platform.code]

      return (
        <div className="platform-config-form">
          {/* Authorization Section */}
          <div className="form-section">
            <div className="form-section-header">
              <Title heading={5}>{t('platformConfig.sections.authorization')}</Title>
              <Space>
                {renderStatusTag(status)}
                <Switch
                  checked={config.enabled}
                  onChange={(checked) => handleConfigChange(platform.code, 'enabled', checked)}
                  checkedText={t('platformConfig.enabled')}
                  uncheckedText={t('platformConfig.disabled')}
                />
              </Space>
            </div>

            <Form labelPosition="left" labelWidth={140}>
              <Form.Input
                field="appKey"
                label={t('platformConfig.fields.appKey')}
                placeholder={t('platformConfig.placeholders.appKey')}
                value={config.appKey}
                onChange={(value) => handleConfigChange(platform.code, 'appKey', value)}
                rules={[{ required: true }]}
              />
              <Form.Input
                field="appSecret"
                label={t('platformConfig.fields.appSecret')}
                placeholder={t('platformConfig.placeholders.appSecret')}
                value={config.appSecret}
                onChange={(value) => handleConfigChange(platform.code, 'appSecret', value)}
                mode={showSecret ? undefined : 'password'}
                suffix={
                  <Button
                    icon={showSecret ? <IconEyeClosed /> : <IconEyeOpened />}
                    theme="borderless"
                    onClick={() => toggleSecretVisibility(platform.code)}
                  />
                }
                rules={[{ required: true }]}
              />
              <Form.Input
                field="shopId"
                label={t('platformConfig.fields.shopId')}
                placeholder={t('platformConfig.placeholders.shopId')}
                value={config.shopId}
                onChange={(value) => handleConfigChange(platform.code, 'shopId', value)}
              />
              <Form.Input
                field="shopName"
                label={t('platformConfig.fields.shopName')}
                placeholder={t('platformConfig.placeholders.shopName')}
                value={config.shopName}
                onChange={(value) => handleConfigChange(platform.code, 'shopName', value)}
              />
            </Form>
          </div>

          {/* Token Section */}
          <div className="form-section">
            <Title heading={5}>{t('platformConfig.sections.tokens')}</Title>
            <Banner
              type="info"
              description={t('platformConfig.tokenHint')}
              className="form-banner"
            />

            <Form labelPosition="left" labelWidth={140}>
              <Form.TextArea
                field="accessToken"
                label={t('platformConfig.fields.accessToken')}
                placeholder={t('platformConfig.placeholders.accessToken')}
                value={config.accessToken}
                onChange={(value) => handleConfigChange(platform.code, 'accessToken', value)}
                rows={3}
              />
              <Form.TextArea
                field="refreshToken"
                label={t('platformConfig.fields.refreshToken')}
                placeholder={t('platformConfig.placeholders.refreshToken')}
                value={config.refreshToken}
                onChange={(value) => handleConfigChange(platform.code, 'refreshToken', value)}
                rows={3}
              />
            </Form>
          </div>

          {/* Sync Settings Section */}
          <div className="form-section">
            <Title heading={5}>{t('platformConfig.sections.syncSettings')}</Title>

            <Form labelPosition="left" labelWidth={140}>
              <Form.Switch
                field="syncEnabled"
                label={t('platformConfig.fields.syncEnabled')}
                checked={config.syncEnabled}
                onChange={(checked) => handleConfigChange(platform.code, 'syncEnabled', checked)}
                extraText={t('platformConfig.hints.syncEnabled')}
              />
              {config.syncEnabled && (
                <>
                  <div className="form-inline-field">
                    <Form.Label>{t('platformConfig.fields.syncInterval')}</Form.Label>
                    <Space>
                      <InputNumber
                        value={config.syncIntervalMinutes}
                        onChange={(value) =>
                          handleConfigChange(platform.code, 'syncIntervalMinutes', value as number)
                        }
                        min={5}
                        max={60}
                        step={5}
                        style={{ width: 120 }}
                      />
                      <Text type="tertiary">{t('platformConfig.units.minutes')}</Text>
                      <Tooltip content={t('platformConfig.hints.syncInterval')}>
                        <IconInfoCircle style={{ color: 'var(--semi-color-text-2)' }} />
                      </Tooltip>
                    </Space>
                  </div>
                  <Form.Switch
                    field="inventorySyncEnabled"
                    label={t('platformConfig.fields.inventorySync')}
                    checked={config.inventorySyncEnabled}
                    onChange={(checked) =>
                      handleConfigChange(platform.code, 'inventorySyncEnabled', checked)
                    }
                    extraText={t('platformConfig.hints.inventorySync')}
                  />
                  <Form.Switch
                    field="orderAutoImport"
                    label={t('platformConfig.fields.orderAutoImport')}
                    checked={config.orderAutoImport}
                    onChange={(checked) =>
                      handleConfigChange(platform.code, 'orderAutoImport', checked)
                    }
                    extraText={t('platformConfig.hints.orderAutoImport')}
                  />
                </>
              )}
            </Form>
          </div>

          {/* Callback/Webhook Section */}
          <div className="form-section">
            <Title heading={5}>{t('platformConfig.sections.callbacks')}</Title>

            <Form labelPosition="left" labelWidth={140}>
              <Form.Input
                field="notifyUrl"
                label={t('platformConfig.fields.notifyUrl')}
                placeholder={t('platformConfig.placeholders.notifyUrl')}
                value={config.notifyUrl}
                onChange={(value) => handleConfigChange(platform.code, 'notifyUrl', value)}
                extraText={t('platformConfig.hints.notifyUrl')}
              />
            </Form>
          </div>

          {/* Environment Section */}
          <div className="form-section">
            <Title heading={5}>{t('platformConfig.sections.environment')}</Title>

            <Form labelPosition="left" labelWidth={140}>
              <Form.Switch
                field="isSandbox"
                label={t('platformConfig.fields.sandbox')}
                checked={config.isSandbox}
                onChange={(checked) => handleConfigChange(platform.code, 'isSandbox', checked)}
                extraText={t('platformConfig.hints.sandbox')}
              />
            </Form>
          </div>

          {/* Status Summary */}
          {status?.lastTestedAt && (
            <div className="form-section">
              <Title heading={5}>{t('platformConfig.sections.status')}</Title>
              <Descriptions
                data={[
                  {
                    key: t('platformConfig.status.lastTested'),
                    value: new Date(status.lastTestedAt).toLocaleString(),
                  },
                  {
                    key: t('platformConfig.status.result'),
                    value:
                      status.testResult === 'success'
                        ? t('platformConfig.status.connected')
                        : status.errorMessage || t('platformConfig.status.failed'),
                  },
                  ...(config.lastSyncAt
                    ? [
                        {
                          key: t('platformConfig.status.lastSync'),
                          value: new Date(config.lastSyncAt).toLocaleString(),
                        },
                      ]
                    : []),
                ]}
              />
            </div>
          )}

          {/* Actions */}
          <div className="form-actions">
            <Space>
              <Button
                type="primary"
                onClick={() => handleSave(platform.code)}
                loading={saving === platform.code}
                disabled={testing !== null}
              >
                {t('common.save')}
              </Button>
              <Button
                icon={<IconLink />}
                onClick={() => handleTestConnection(platform.code)}
                loading={testing === platform.code}
                disabled={saving !== null}
              >
                {t('platformConfig.testConnection')}
              </Button>
            </Space>
          </div>
        </div>
      )
    },
    [
      configs,
      showSecrets,
      saving,
      testing,
      handleConfigChange,
      handleSave,
      handleTestConnection,
      toggleSecretVisibility,
      getStatus,
      renderStatusTag,
      t,
    ]
  )

  return (
    <Container size="lg" className="platform-config-page">
      <Card className="platform-config-card">
        <div className="platform-config-header">
          <div>
            <Title heading={4} style={{ margin: 0 }}>
              {t('platformConfig.title')}
            </Title>
            <Text type="tertiary">{t('platformConfig.subtitle')}</Text>
          </div>
          <Button icon={<IconRefresh />} onClick={loadConfigurations} loading={loading}>
            {t('common.refresh')}
          </Button>
        </div>

        <Banner
          type="warning"
          description={t('platformConfig.securityWarning')}
          className="platform-config-banner"
        />

        <Spin spinning={loading}>
          <Tabs
            type="line"
            activeKey={activeTab}
            onChange={setActiveTab}
            className="platform-config-tabs"
          >
            {PLATFORMS.map((platform) => (
              <TabPane
                key={platform.code}
                tab={
                  <span className="tab-title">
                    <span className="tab-icon" style={{ backgroundColor: platform.color }}>
                      {platform.icon}
                    </span>
                    {platform.name}
                  </span>
                }
                itemKey={platform.code}
              >
                {renderPlatformForm(platform)}
              </TabPane>
            ))}
          </Tabs>
        </Spin>
      </Card>
    </Container>
  )
}
