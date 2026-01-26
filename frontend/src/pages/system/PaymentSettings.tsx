import { useState, useCallback, useMemo } from 'react'
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
} from '@douyinfe/semi-ui-19'
import {
  IconTick,
  IconClose,
  IconRefresh,
  IconEyeOpened,
  IconEyeClosed,
} from '@douyinfe/semi-icons'
import { Container } from '@/components/common/layout'
import './PaymentSettings.css'

const { Title, Text } = Typography

/**
 * Payment gateway configuration types
 */
interface WechatPayConfig {
  enabled: boolean
  mchId: string
  appId: string
  apiKey: string
  serialNo: string
  privateKey: string
  wechatCert: string
  wechatCertSerialNo: string
  notifyUrl: string
  refundNotifyUrl: string
  isSandbox: boolean
}

interface AlipayConfig {
  enabled: boolean
  appId: string
  privateKey: string
  alipayPublicKey: string
  signType: 'RSA2' | 'RSA'
  notifyUrl: string
  returnUrl: string
  isSandbox: boolean
}

interface PaymentGatewayStatus {
  type: 'WECHAT' | 'ALIPAY'
  enabled: boolean
  configured: boolean
  lastTestedAt?: string
  testResult?: 'success' | 'failed'
  errorMessage?: string
}

/**
 * Default configurations
 */
const defaultWechatConfig: WechatPayConfig = {
  enabled: false,
  mchId: '',
  appId: '',
  apiKey: '',
  serialNo: '',
  privateKey: '',
  wechatCert: '',
  wechatCertSerialNo: '',
  notifyUrl: '',
  refundNotifyUrl: '',
  isSandbox: false,
}

const defaultAlipayConfig: AlipayConfig = {
  enabled: false,
  appId: '',
  privateKey: '',
  alipayPublicKey: '',
  signType: 'RSA2',
  notifyUrl: '',
  returnUrl: '',
  isSandbox: false,
}

/**
 * Payment Settings Page
 *
 * Features:
 * - Configure WeChat Pay gateway
 * - Configure Alipay gateway
 * - Test connection for each gateway
 * - Enable/disable gateways
 */
export default function PaymentSettingsPage() {
  const { t } = useTranslation('system')

  // Gateway configurations
  const [wechatConfig, setWechatConfig] = useState<WechatPayConfig>(defaultWechatConfig)
  const [alipayConfig, setAlipayConfig] = useState<AlipayConfig>(defaultAlipayConfig)

  // Gateway statuses
  const [gatewayStatuses, setGatewayStatuses] = useState<PaymentGatewayStatus[]>([
    { type: 'WECHAT', enabled: false, configured: false },
    { type: 'ALIPAY', enabled: false, configured: false },
  ])

  // Loading states
  const [loading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState<'WECHAT' | 'ALIPAY' | null>(null)

  // Visibility states for sensitive fields
  const [showWechatApiKey, setShowWechatApiKey] = useState(false)
  // Reserved for future use when private key field visibility toggle is implemented
  const [_showWechatPrivateKey, _setShowWechatPrivateKey] = useState(false)
  const [_showAlipayPrivateKey, _setShowAlipayPrivateKey] = useState(false)
  const [_showAlipayPublicKey, _setShowAlipayPublicKey] = useState(false)

  // Active tab
  const [activeTab, setActiveTab] = useState('wechat')

  // Get status for a gateway type
  const getGatewayStatus = useCallback(
    (type: 'WECHAT' | 'ALIPAY'): PaymentGatewayStatus | undefined => {
      return gatewayStatuses.find((s) => s.type === type)
    },
    [gatewayStatuses]
  )

  // Update gateway status
  const updateGatewayStatus = useCallback(
    (type: 'WECHAT' | 'ALIPAY', update: Partial<PaymentGatewayStatus>) => {
      setGatewayStatuses((prev) => prev.map((s) => (s.type === type ? { ...s, ...update } : s)))
    },
    []
  )

  // Handle WeChat config change
  const handleWechatChange = useCallback(
    <K extends keyof WechatPayConfig>(field: K, value: WechatPayConfig[K]) => {
      setWechatConfig((prev) => ({ ...prev, [field]: value }))
    },
    []
  )

  // Handle Alipay config change
  const handleAlipayChange = useCallback(
    <K extends keyof AlipayConfig>(field: K, value: AlipayConfig[K]) => {
      setAlipayConfig((prev) => ({ ...prev, [field]: value }))
    },
    []
  )

  // Validate WeChat configuration
  const validateWechatConfig = useCallback((): string | null => {
    if (!wechatConfig.mchId) return t('paymentSettings.wechat.errors.mchIdRequired')
    if (!wechatConfig.appId) return t('paymentSettings.wechat.errors.appIdRequired')
    if (!wechatConfig.apiKey) return t('paymentSettings.wechat.errors.apiKeyRequired')
    if (wechatConfig.apiKey.length !== 32) return t('paymentSettings.wechat.errors.apiKeyInvalid')
    if (!wechatConfig.serialNo) return t('paymentSettings.wechat.errors.serialNoRequired')
    if (!wechatConfig.privateKey) return t('paymentSettings.wechat.errors.privateKeyRequired')
    if (!wechatConfig.notifyUrl) return t('paymentSettings.wechat.errors.notifyUrlRequired')
    return null
  }, [wechatConfig, t])

  // Validate Alipay configuration
  const validateAlipayConfig = useCallback((): string | null => {
    if (!alipayConfig.appId) return t('paymentSettings.alipay.errors.appIdRequired')
    if (!alipayConfig.privateKey) return t('paymentSettings.alipay.errors.privateKeyRequired')
    if (!alipayConfig.alipayPublicKey) return t('paymentSettings.alipay.errors.publicKeyRequired')
    if (!alipayConfig.notifyUrl) return t('paymentSettings.alipay.errors.notifyUrlRequired')
    return null
  }, [alipayConfig, t])

  // Save WeChat configuration
  const handleSaveWechat = useCallback(async () => {
    const error = validateWechatConfig()
    if (error) {
      Toast.error(error)
      return
    }

    setSaving(true)
    try {
      // TODO: Implement API call to save WeChat configuration
      // await api.savePaymentGatewayConfig('WECHAT', wechatConfig)
      await new Promise((resolve) => setTimeout(resolve, 1000)) // Simulate API call

      updateGatewayStatus('WECHAT', {
        configured: true,
        enabled: wechatConfig.enabled,
      })

      Toast.success(t('paymentSettings.messages.saveSuccess'))
    } catch {
      Toast.error(t('paymentSettings.messages.saveError'))
    } finally {
      setSaving(false)
    }
  }, [wechatConfig, validateWechatConfig, updateGatewayStatus, t])

  // Save Alipay configuration
  const handleSaveAlipay = useCallback(async () => {
    const error = validateAlipayConfig()
    if (error) {
      Toast.error(error)
      return
    }

    setSaving(true)
    try {
      // TODO: Implement API call to save Alipay configuration
      // await api.savePaymentGatewayConfig('ALIPAY', alipayConfig)
      await new Promise((resolve) => setTimeout(resolve, 1000)) // Simulate API call

      updateGatewayStatus('ALIPAY', {
        configured: true,
        enabled: alipayConfig.enabled,
      })

      Toast.success(t('paymentSettings.messages.saveSuccess'))
    } catch {
      Toast.error(t('paymentSettings.messages.saveError'))
    } finally {
      setSaving(false)
    }
  }, [alipayConfig, validateAlipayConfig, updateGatewayStatus, t])

  // Test WeChat connection
  const handleTestWechat = useCallback(async () => {
    const error = validateWechatConfig()
    if (error) {
      Toast.error(error)
      return
    }

    setTesting('WECHAT')
    try {
      // TODO: Implement API call to test WeChat connection
      // const result = await api.testPaymentGateway('WECHAT')
      await new Promise((resolve) => setTimeout(resolve, 2000)) // Simulate API call

      // Simulate random success/failure for demo
      const success = Math.random() > 0.3

      updateGatewayStatus('WECHAT', {
        lastTestedAt: new Date().toISOString(),
        testResult: success ? 'success' : 'failed',
        errorMessage: success ? undefined : 'Connection timeout',
      })

      if (success) {
        Toast.success(t('paymentSettings.messages.testSuccess'))
      } else {
        Toast.error(t('paymentSettings.messages.testFailed'))
      }
    } catch {
      updateGatewayStatus('WECHAT', {
        lastTestedAt: new Date().toISOString(),
        testResult: 'failed',
        errorMessage: 'Network error',
      })
      Toast.error(t('paymentSettings.messages.testError'))
    } finally {
      setTesting(null)
    }
  }, [validateWechatConfig, updateGatewayStatus, t])

  // Test Alipay connection
  const handleTestAlipay = useCallback(async () => {
    const error = validateAlipayConfig()
    if (error) {
      Toast.error(error)
      return
    }

    setTesting('ALIPAY')
    try {
      // TODO: Implement API call to test Alipay connection
      // const result = await api.testPaymentGateway('ALIPAY')
      await new Promise((resolve) => setTimeout(resolve, 2000)) // Simulate API call

      // Simulate random success/failure for demo
      const success = Math.random() > 0.3

      updateGatewayStatus('ALIPAY', {
        lastTestedAt: new Date().toISOString(),
        testResult: success ? 'success' : 'failed',
        errorMessage: success ? undefined : 'Invalid signature',
      })

      if (success) {
        Toast.success(t('paymentSettings.messages.testSuccess'))
      } else {
        Toast.error(t('paymentSettings.messages.testFailed'))
      }
    } catch {
      updateGatewayStatus('ALIPAY', {
        lastTestedAt: new Date().toISOString(),
        testResult: 'failed',
        errorMessage: 'Network error',
      })
      Toast.error(t('paymentSettings.messages.testError'))
    } finally {
      setTesting(null)
    }
  }, [validateAlipayConfig, updateGatewayStatus, t])

  // Render status tag
  const renderStatusTag = useCallback(
    (status: PaymentGatewayStatus | undefined) => {
      if (!status) return null

      if (!status.configured) {
        return <Tag color="grey">{t('paymentSettings.status.notConfigured')}</Tag>
      }

      if (!status.enabled) {
        return <Tag color="orange">{t('paymentSettings.status.disabled')}</Tag>
      }

      if (status.testResult === 'success') {
        return (
          <Tag color="green" prefixIcon={<IconTick />}>
            {t('paymentSettings.status.connected')}
          </Tag>
        )
      }

      if (status.testResult === 'failed') {
        return (
          <Tag color="red" prefixIcon={<IconClose />}>
            {t('paymentSettings.status.failed')}
          </Tag>
        )
      }

      return <Tag color="blue">{t('paymentSettings.status.configured')}</Tag>
    },
    [t]
  )

  // WeChat Pay form
  const renderWechatForm = useMemo(
    () => (
      <div className="payment-settings-form">
        <div className="form-section">
          <div className="form-section-header">
            <Title heading={5}>{t('paymentSettings.wechat.basicInfo')}</Title>
            <Space>
              {renderStatusTag(getGatewayStatus('WECHAT'))}
              <Switch
                checked={wechatConfig.enabled}
                onChange={(checked) => handleWechatChange('enabled', checked)}
                checkedText={t('paymentSettings.enabled')}
                uncheckedText={t('paymentSettings.disabled')}
              />
            </Space>
          </div>

          <Form labelPosition="left" labelWidth={140}>
            <Form.Input
              field="mchId"
              label={t('paymentSettings.wechat.mchId')}
              placeholder={t('paymentSettings.wechat.mchIdPlaceholder')}
              value={wechatConfig.mchId}
              onChange={(value) => handleWechatChange('mchId', value)}
              rules={[{ required: true }]}
            />
            <Form.Input
              field="appId"
              label={t('paymentSettings.wechat.appId')}
              placeholder={t('paymentSettings.wechat.appIdPlaceholder')}
              value={wechatConfig.appId}
              onChange={(value) => handleWechatChange('appId', value)}
              rules={[{ required: true }]}
            />
            <Form.Input
              field="apiKey"
              label={t('paymentSettings.wechat.apiKey')}
              placeholder={t('paymentSettings.wechat.apiKeyPlaceholder')}
              value={wechatConfig.apiKey}
              onChange={(value) => handleWechatChange('apiKey', value)}
              mode={showWechatApiKey ? undefined : 'password'}
              suffix={
                <Button
                  icon={showWechatApiKey ? <IconEyeClosed /> : <IconEyeOpened />}
                  theme="borderless"
                  onClick={() => setShowWechatApiKey(!showWechatApiKey)}
                />
              }
              rules={[{ required: true }]}
              extraText={t('paymentSettings.wechat.apiKeyHint')}
            />
            <Form.Input
              field="serialNo"
              label={t('paymentSettings.wechat.serialNo')}
              placeholder={t('paymentSettings.wechat.serialNoPlaceholder')}
              value={wechatConfig.serialNo}
              onChange={(value) => handleWechatChange('serialNo', value)}
              rules={[{ required: true }]}
            />
          </Form>
        </div>

        <div className="form-section">
          <Title heading={5}>{t('paymentSettings.wechat.certificates')}</Title>

          <Form labelPosition="left" labelWidth={140}>
            <Form.TextArea
              field="privateKey"
              label={t('paymentSettings.wechat.privateKey')}
              placeholder={t('paymentSettings.wechat.privateKeyPlaceholder')}
              value={wechatConfig.privateKey}
              onChange={(value) => handleWechatChange('privateKey', value)}
              rows={6}
              rules={[{ required: true }]}
              extraText={t('paymentSettings.wechat.privateKeyHint')}
            />
            <Form.TextArea
              field="wechatCert"
              label={t('paymentSettings.wechat.wechatCert')}
              placeholder={t('paymentSettings.wechat.wechatCertPlaceholder')}
              value={wechatConfig.wechatCert}
              onChange={(value) => handleWechatChange('wechatCert', value)}
              rows={6}
            />
            <Form.Input
              field="wechatCertSerialNo"
              label={t('paymentSettings.wechat.wechatCertSerialNo')}
              placeholder={t('paymentSettings.wechat.wechatCertSerialNoPlaceholder')}
              value={wechatConfig.wechatCertSerialNo}
              onChange={(value) => handleWechatChange('wechatCertSerialNo', value)}
            />
          </Form>
        </div>

        <div className="form-section">
          <Title heading={5}>{t('paymentSettings.wechat.callbacks')}</Title>

          <Form labelPosition="left" labelWidth={140}>
            <Form.Input
              field="notifyUrl"
              label={t('paymentSettings.wechat.notifyUrl')}
              placeholder={t('paymentSettings.wechat.notifyUrlPlaceholder')}
              value={wechatConfig.notifyUrl}
              onChange={(value) => handleWechatChange('notifyUrl', value)}
              rules={[{ required: true }]}
            />
            <Form.Input
              field="refundNotifyUrl"
              label={t('paymentSettings.wechat.refundNotifyUrl')}
              placeholder={t('paymentSettings.wechat.refundNotifyUrlPlaceholder')}
              value={wechatConfig.refundNotifyUrl}
              onChange={(value) => handleWechatChange('refundNotifyUrl', value)}
            />
          </Form>
        </div>

        <div className="form-section">
          <Title heading={5}>{t('paymentSettings.environment')}</Title>

          <Form labelPosition="left" labelWidth={140}>
            <Form.Switch
              field="isSandbox"
              label={t('paymentSettings.wechat.sandbox')}
              checked={wechatConfig.isSandbox}
              onChange={(checked) => handleWechatChange('isSandbox', checked)}
              extraText={t('paymentSettings.wechat.sandboxHint')}
            />
          </Form>
        </div>

        <div className="form-actions">
          <Space>
            <Button
              type="primary"
              onClick={handleSaveWechat}
              loading={saving}
              disabled={testing !== null}
            >
              {t('common.save')}
            </Button>
            <Button
              icon={<IconRefresh />}
              onClick={handleTestWechat}
              loading={testing === 'WECHAT'}
              disabled={saving}
            >
              {t('paymentSettings.testConnection')}
            </Button>
          </Space>
        </div>
      </div>
    ),
    [
      wechatConfig,
      showWechatApiKey,
      saving,
      testing,
      handleWechatChange,
      handleSaveWechat,
      handleTestWechat,
      renderStatusTag,
      getGatewayStatus,
      t,
    ]
  )

  // Alipay form
  const renderAlipayForm = useMemo(
    () => (
      <div className="payment-settings-form">
        <div className="form-section">
          <div className="form-section-header">
            <Title heading={5}>{t('paymentSettings.alipay.basicInfo')}</Title>
            <Space>
              {renderStatusTag(getGatewayStatus('ALIPAY'))}
              <Switch
                checked={alipayConfig.enabled}
                onChange={(checked) => handleAlipayChange('enabled', checked)}
                checkedText={t('paymentSettings.enabled')}
                uncheckedText={t('paymentSettings.disabled')}
              />
            </Space>
          </div>

          <Form labelPosition="left" labelWidth={140}>
            <Form.Input
              field="appId"
              label={t('paymentSettings.alipay.appId')}
              placeholder={t('paymentSettings.alipay.appIdPlaceholder')}
              value={alipayConfig.appId}
              onChange={(value) => handleAlipayChange('appId', value)}
              rules={[{ required: true }]}
            />
            <Form.Select
              field="signType"
              label={t('paymentSettings.alipay.signType')}
              value={alipayConfig.signType}
              onChange={(value) => handleAlipayChange('signType', value as 'RSA2' | 'RSA')}
              optionList={[
                { label: 'RSA2 (Recommended)', value: 'RSA2' },
                { label: 'RSA', value: 'RSA' },
              ]}
            />
          </Form>
        </div>

        <div className="form-section">
          <Title heading={5}>{t('paymentSettings.alipay.keys')}</Title>

          <Form labelPosition="left" labelWidth={140}>
            <Form.TextArea
              field="privateKey"
              label={t('paymentSettings.alipay.privateKey')}
              placeholder={t('paymentSettings.alipay.privateKeyPlaceholder')}
              value={alipayConfig.privateKey}
              onChange={(value) => handleAlipayChange('privateKey', value)}
              rows={6}
              rules={[{ required: true }]}
              extraText={t('paymentSettings.alipay.privateKeyHint')}
            />
            <Form.TextArea
              field="alipayPublicKey"
              label={t('paymentSettings.alipay.alipayPublicKey')}
              placeholder={t('paymentSettings.alipay.alipayPublicKeyPlaceholder')}
              value={alipayConfig.alipayPublicKey}
              onChange={(value) => handleAlipayChange('alipayPublicKey', value)}
              rows={6}
              rules={[{ required: true }]}
              extraText={t('paymentSettings.alipay.alipayPublicKeyHint')}
            />
          </Form>
        </div>

        <div className="form-section">
          <Title heading={5}>{t('paymentSettings.alipay.callbacks')}</Title>

          <Form labelPosition="left" labelWidth={140}>
            <Form.Input
              field="notifyUrl"
              label={t('paymentSettings.alipay.notifyUrl')}
              placeholder={t('paymentSettings.alipay.notifyUrlPlaceholder')}
              value={alipayConfig.notifyUrl}
              onChange={(value) => handleAlipayChange('notifyUrl', value)}
              rules={[{ required: true }]}
            />
            <Form.Input
              field="returnUrl"
              label={t('paymentSettings.alipay.returnUrl')}
              placeholder={t('paymentSettings.alipay.returnUrlPlaceholder')}
              value={alipayConfig.returnUrl}
              onChange={(value) => handleAlipayChange('returnUrl', value)}
            />
          </Form>
        </div>

        <div className="form-section">
          <Title heading={5}>{t('paymentSettings.environment')}</Title>

          <Form labelPosition="left" labelWidth={140}>
            <Form.Switch
              field="isSandbox"
              label={t('paymentSettings.alipay.sandbox')}
              checked={alipayConfig.isSandbox}
              onChange={(checked) => handleAlipayChange('isSandbox', checked)}
              extraText={t('paymentSettings.alipay.sandboxHint')}
            />
          </Form>
        </div>

        <div className="form-actions">
          <Space>
            <Button
              type="primary"
              onClick={handleSaveAlipay}
              loading={saving}
              disabled={testing !== null}
            >
              {t('common.save')}
            </Button>
            <Button
              icon={<IconRefresh />}
              onClick={handleTestAlipay}
              loading={testing === 'ALIPAY'}
              disabled={saving}
            >
              {t('paymentSettings.testConnection')}
            </Button>
          </Space>
        </div>
      </div>
    ),
    [
      alipayConfig,
      saving,
      testing,
      handleAlipayChange,
      handleSaveAlipay,
      handleTestAlipay,
      renderStatusTag,
      getGatewayStatus,
      t,
    ]
  )

  return (
    <Container size="lg" className="payment-settings-page">
      <Card className="payment-settings-card">
        <div className="payment-settings-header">
          <div>
            <Title heading={4} style={{ margin: 0 }}>
              {t('paymentSettings.title')}
            </Title>
            <Text type="tertiary">{t('paymentSettings.subtitle')}</Text>
          </div>
        </div>

        <Banner
          type="warning"
          description={t('paymentSettings.securityWarning')}
          className="payment-settings-banner"
        />

        <Spin spinning={loading}>
          <Tabs
            type="line"
            activeKey={activeTab}
            onChange={setActiveTab}
            className="payment-settings-tabs"
          >
            <TabPane
              tab={
                <span className="tab-title">
                  <span className="tab-icon wechat-icon">W</span>
                  {t('paymentSettings.wechat.title')}
                </span>
              }
              itemKey="wechat"
            >
              {renderWechatForm}
            </TabPane>
            <TabPane
              tab={
                <span className="tab-title">
                  <span className="tab-icon alipay-icon">A</span>
                  {t('paymentSettings.alipay.title')}
                </span>
              }
              itemKey="alipay"
            >
              {renderAlipayForm}
            </TabPane>
          </Tabs>
        </Spin>
      </Card>
    </Container>
  )
}
