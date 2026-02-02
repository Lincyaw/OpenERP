/**
 * Print Service Settings Page
 *
 * Allows users to configure and manage the local print service connection.
 *
 * Features:
 * - Display print service status (connected/disconnected)
 * - Show print service version information
 * - List available printers from the local print service
 * - Provide download links for print service installers
 * - Configuration wizard to generate config.yaml
 * - Test print functionality
 * - View recent print logs (last 50 records)
 */

import { useState, useMemo, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Toast,
  Table,
  Button,
  Modal,
  Form,
  Select,
  Banner,
  Space,
  Tag,
  Tooltip,
  Empty,
  Spin,
  Descriptions,
  Tabs,
  TabPane,
  Collapsible,
  TextArea,
} from '@douyinfe/semi-ui-19'
import type { ColumnProps } from '@douyinfe/semi-ui-19/lib/es/table'
import {
  IconLink,
  IconUnlink,
  IconRefresh,
  IconPrint,
  IconDownload,
  IconSetting,
  IconCopy,
  IconAlertCircle,
  IconHistory,
  IconInfoCircle,
  IconServer,
  IconCode,
} from '@douyinfe/semi-icons'
import { Container } from '@/components/common/layout'
import { usePrintServiceStatus, type PrinterInfo } from '@/hooks'
import { useListPrintJobJobs } from '@/api/print-jobs/print-jobs'
import type { HandlerPrintJobResponse } from '@/api/models'
import { useAuthStore } from '@/store'

import './PrintServiceSettings.css'

const { Title, Text, Paragraph } = Typography

/** Download links for print service installers */
const DOWNLOAD_LINKS = {
  windows:
    'https://github.com/your-org/print-service/releases/latest/download/print-service-windows.exe',
  macos:
    'https://github.com/your-org/print-service/releases/latest/download/print-service-macos.dmg',
  linux:
    'https://github.com/your-org/print-service/releases/latest/download/print-service-linux.tar.gz',
}

/** Operating system options */
type OperatingSystem = 'windows' | 'macos' | 'linux'

/**
 * Print Service Status Card Component
 */
function PrintServiceStatusCard() {
  const { t } = useTranslation('system')
  const { status, version, error, lastChecked, refresh, printers, isLoadingPrinters } =
    usePrintServiceStatus()

  const statusConfig = useMemo(() => {
    switch (status) {
      case 'connected':
        return {
          color: 'green' as const,
          icon: <IconLink size="large" />,
          label: t('printServiceSettings.status.connected'),
          type: 'success' as const,
        }
      case 'disconnected':
        return {
          color: 'red' as const,
          icon: <IconUnlink size="large" />,
          label: t('printServiceSettings.status.disconnected'),
          type: 'danger' as const,
        }
      case 'checking':
      default:
        return {
          color: 'grey' as const,
          icon: <IconRefresh size="large" spin />,
          label: t('printServiceSettings.status.checking'),
          type: 'tertiary' as const,
        }
    }
  }, [status, t])

  const formatLastChecked = useCallback((date: Date | null) => {
    if (!date) return '-'
    return date.toLocaleTimeString()
  }, [])

  return (
    <Card className="print-service-status-card" title={t('printServiceSettings.serviceStatus')}>
      <div className="status-content">
        <div className="status-indicator">
          <div className={`status-icon ${status}`}>{statusConfig.icon}</div>
          <div className="status-info">
            <Text type={statusConfig.type} strong style={{ fontSize: '1rem' }}>
              {statusConfig.label}
            </Text>
            {version && (
              <Text type="secondary" size="small">
                {t('printServiceSettings.version')}: {version}
              </Text>
            )}
          </div>
        </div>

        {error && (
          <Banner
            type="warning"
            description={error}
            className="status-error-banner"
            icon={<IconAlertCircle />}
          />
        )}

        <Descriptions
          className="status-details"
          row
          size="small"
          data={[
            {
              key: t('printServiceSettings.lastChecked'),
              value: formatLastChecked(lastChecked),
            },
            {
              key: t('printServiceSettings.printersCount'),
              value: isLoadingPrinters ? (
                <Spin size="small" />
              ) : status === 'connected' ? (
                printers.length.toString()
              ) : (
                '-'
              ),
            },
            {
              key: t('printServiceSettings.serviceUrl'),
              value: 'http://localhost:9999',
            },
          ]}
        />

        <Button
          icon={<IconRefresh />}
          onClick={() => refresh()}
          loading={status === 'checking'}
          className="refresh-button"
        >
          {t('printServiceSettings.checkConnection')}
        </Button>
      </div>
    </Card>
  )
}

/**
 * Printers List Card Component
 */
function PrintersListCard() {
  const { t } = useTranslation('system')
  const { status, printers, isLoadingPrinters, refreshPrinters } = usePrintServiceStatus()

  const columns: ColumnProps<PrinterInfo>[] = useMemo(
    () => [
      {
        title: t('printServiceSettings.printers.name'),
        dataIndex: 'name',
        key: 'name',
        render: (value: string, record: PrinterInfo) => (
          <Space>
            <IconPrint />
            <Text>{record.displayName || value}</Text>
            {record.isDefault && (
              <Tag color="blue" size="small">
                {t('printServiceSettings.printers.default')}
              </Tag>
            )}
          </Space>
        ),
      },
      {
        title: t('printServiceSettings.printers.status'),
        dataIndex: 'status',
        key: 'status',
        width: 120,
        render: (value: string) => (
          <Tag color={value === 'online' ? 'green' : 'grey'}>
            {value || t('printServiceSettings.printers.unknown')}
          </Tag>
        ),
      },
    ],
    [t]
  )

  if (status !== 'connected') {
    return (
      <Card className="printers-list-card" title={t('printServiceSettings.availablePrinters')}>
        <Empty
          image={<IconUnlink style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
          title={t('printServiceSettings.printers.serviceNotConnected')}
          description={t('printServiceSettings.printers.connectFirst')}
        />
      </Card>
    )
  }

  return (
    <Card
      className="printers-list-card"
      title={t('printServiceSettings.availablePrinters')}
      headerExtraContent={
        <Button
          icon={<IconRefresh />}
          theme="borderless"
          size="small"
          onClick={() => refreshPrinters()}
          loading={isLoadingPrinters}
        />
      }
    >
      <Table
        columns={columns}
        dataSource={printers}
        rowKey="name"
        loading={isLoadingPrinters}
        pagination={false}
        size="small"
        empty={
          <Empty
            image={<IconPrint style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
            title={t('printServiceSettings.printers.empty.title')}
            description={t('printServiceSettings.printers.empty.description')}
          />
        }
      />
    </Card>
  )
}

/**
 * Download Links Card Component
 */
function DownloadLinksCard() {
  const { t } = useTranslation('system')

  const handleDownload = useCallback((os: OperatingSystem) => {
    window.open(DOWNLOAD_LINKS[os], '_blank')
  }, [])

  return (
    <Card className="download-links-card" title={t('printServiceSettings.downloadService')}>
      <Paragraph type="secondary" className="download-description">
        {t('printServiceSettings.downloadDescription')}
      </Paragraph>

      <div className="download-buttons">
        <Button
          icon={<IconDownload />}
          onClick={() => handleDownload('windows')}
          className="download-button"
        >
          Windows
        </Button>
        <Button
          icon={<IconDownload />}
          onClick={() => handleDownload('macos')}
          className="download-button"
        >
          macOS
        </Button>
        <Button
          icon={<IconDownload />}
          onClick={() => handleDownload('linux')}
          className="download-button"
        >
          Linux
        </Button>
      </div>

      <Collapsible className="installation-guide" collapseHeight={0} isOpen={false}>
        <div className="guide-content">
          <Title heading={6}>{t('printServiceSettings.installationGuide.title')}</Title>
          <ol>
            <li>{t('printServiceSettings.installationGuide.step1')}</li>
            <li>{t('printServiceSettings.installationGuide.step2')}</li>
            <li>{t('printServiceSettings.installationGuide.step3')}</li>
            <li>{t('printServiceSettings.installationGuide.step4')}</li>
          </ol>
        </div>
      </Collapsible>
    </Card>
  )
}

/**
 * Configuration Wizard Card Component
 */
function ConfigWizardCard() {
  const { t } = useTranslation('system')
  const user = useAuthStore((state) => state.user)
  const [isModalVisible, setIsModalVisible] = useState(false)
  const [configValues, setConfigValues] = useState({
    apiUrl: window.location.origin,
    tenantId: user?.tenantId || '',
    apiToken: '',
  })

  const generateConfig = useCallback(() => {
    return `# Print Service Configuration
# Generated for tenant: ${configValues.tenantId}

server:
  port: 9999
  host: "0.0.0.0"

api:
  url: "${configValues.apiUrl}"
  tenant_id: "${configValues.tenantId}"
  token: "${configValues.apiToken}"

printing:
  default_copies: 1
  timeout: 30

logging:
  level: "info"
  file: "print-service.log"
`
  }, [configValues])

  const handleCopyConfig = useCallback(() => {
    const config = generateConfig()
    navigator.clipboard.writeText(config).then(() => {
      Toast.success(t('printServiceSettings.configWizard.copied'))
    })
  }, [generateConfig, t])

  const handleDownloadConfig = useCallback(() => {
    const config = generateConfig()
    const blob = new Blob([config], { type: 'text/yaml' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = 'config.yaml'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    URL.revokeObjectURL(url)
    Toast.success(t('printServiceSettings.configWizard.downloaded'))
  }, [generateConfig, t])

  return (
    <>
      <Card className="config-wizard-card" title={t('printServiceSettings.configWizard.title')}>
        <Paragraph type="secondary" className="config-description">
          {t('printServiceSettings.configWizard.description')}
        </Paragraph>

        <Button
          icon={<IconSetting />}
          type="primary"
          theme="solid"
          onClick={() => setIsModalVisible(true)}
        >
          {t('printServiceSettings.configWizard.openWizard')}
        </Button>
      </Card>

      <Modal
        title={t('printServiceSettings.configWizard.modalTitle')}
        visible={isModalVisible}
        onCancel={() => setIsModalVisible(false)}
        footer={
          <Space>
            <Button onClick={() => setIsModalVisible(false)}>{t('common.cancel')}</Button>
            <Button icon={<IconCopy />} onClick={handleCopyConfig}>
              {t('printServiceSettings.configWizard.copyConfig')}
            </Button>
            <Button
              icon={<IconDownload />}
              type="primary"
              theme="solid"
              onClick={handleDownloadConfig}
            >
              {t('printServiceSettings.configWizard.downloadConfig')}
            </Button>
          </Space>
        }
        width={640}
      >
        <Form labelPosition="left" labelWidth={120} className="config-wizard-form">
          <Form.Input
            field="apiUrl"
            label={t('printServiceSettings.configWizard.apiUrl')}
            initValue={configValues.apiUrl}
            onChange={(value) => setConfigValues((prev) => ({ ...prev, apiUrl: value }))}
          />
          <Form.Input
            field="tenantId"
            label={t('printServiceSettings.configWizard.tenantId')}
            initValue={configValues.tenantId}
            onChange={(value) => setConfigValues((prev) => ({ ...prev, tenantId: value }))}
            disabled
          />
          <Form.Input
            field="apiToken"
            label={t('printServiceSettings.configWizard.apiToken')}
            initValue={configValues.apiToken}
            onChange={(value) => setConfigValues((prev) => ({ ...prev, apiToken: value }))}
            placeholder={t('printServiceSettings.configWizard.apiTokenPlaceholder')}
            mode="password"
          />
        </Form>

        <div className="config-preview">
          <div className="preview-header">
            <Text strong>
              <IconCode style={{ marginRight: 4 }} />
              config.yaml
            </Text>
          </div>
          <TextArea
            value={generateConfig()}
            readonly
            autosize={{ minRows: 10, maxRows: 20 }}
            className="config-textarea"
          />
        </div>
      </Modal>
    </>
  )
}

/**
 * Test Print Card Component
 */
function TestPrintCard() {
  const { t } = useTranslation('system')
  const { status, printers } = usePrintServiceStatus()
  const [selectedPrinter, setSelectedPrinter] = useState<string>('')
  const [isTesting, setIsTesting] = useState(false)

  const printerOptions = useMemo(
    () =>
      printers.map((p) => ({
        label: p.displayName || p.name,
        value: p.name,
      })),
    [printers]
  )

  const handleTestPrint = useCallback(async () => {
    if (!selectedPrinter) {
      Toast.warning(t('printServiceSettings.testPrint.selectPrinter'))
      return
    }

    setIsTesting(true)
    try {
      // Send test print request to local print service
      const response = await fetch('http://localhost:9999/test-print', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          printer: selectedPrinter,
          content: 'ERP Print Service Test Page',
        }),
      })

      if (response.ok) {
        Toast.success(t('printServiceSettings.testPrint.success'))
      } else {
        throw new Error('Test print failed')
      }
    } catch {
      Toast.error(t('printServiceSettings.testPrint.error'))
    } finally {
      setIsTesting(false)
    }
  }, [selectedPrinter, t])

  return (
    <Card className="test-print-card" title={t('printServiceSettings.testPrint.title')}>
      {status !== 'connected' ? (
        <Banner
          type="warning"
          description={t('printServiceSettings.testPrint.serviceNotConnected')}
          icon={<IconAlertCircle />}
        />
      ) : (
        <div className="test-print-content">
          <Paragraph type="secondary">{t('printServiceSettings.testPrint.description')}</Paragraph>

          <Space className="test-print-form">
            <Select
              placeholder={t('printServiceSettings.testPrint.selectPrinterPlaceholder')}
              optionList={printerOptions}
              value={selectedPrinter}
              onChange={(value) => setSelectedPrinter(value as string)}
              style={{ width: 280 }}
              disabled={printers.length === 0}
            />
            <Button
              icon={<IconPrint />}
              type="primary"
              onClick={handleTestPrint}
              loading={isTesting}
              disabled={!selectedPrinter || printers.length === 0}
            >
              {t('printServiceSettings.testPrint.sendTest')}
            </Button>
          </Space>
        </div>
      )}
    </Card>
  )
}

/**
 * Print Logs Card Component
 */
function PrintLogsCard() {
  const { t } = useTranslation('system')

  // Fetch last 50 print jobs
  const {
    data: jobsResponse,
    isLoading,
    refetch,
  } = useListPrintJobJobs({
    page: 1,
    page_size: 50,
    order_by: 'created_at',
    order_dir: 'desc',
  })

  const jobs = useMemo(() => {
    if (jobsResponse?.status === 200) {
      return jobsResponse.data.data || []
    }
    return []
  }, [jobsResponse])

  const columns: ColumnProps<HandlerPrintJobResponse>[] = useMemo(
    () => [
      {
        title: t('printServiceSettings.logs.documentType'),
        dataIndex: 'document_type',
        key: 'document_type',
        width: 120,
        render: (value: string) => value || '-',
      },
      {
        title: t('printServiceSettings.logs.documentNumber'),
        dataIndex: 'document_number',
        key: 'document_number',
        width: 160,
        render: (value: string) => value || '-',
      },
      {
        title: t('printServiceSettings.logs.status'),
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (value: string) => {
          const statusColors: Record<string, 'green' | 'red' | 'orange' | 'grey'> = {
            completed: 'green',
            failed: 'red',
            pending: 'orange',
            printing: 'orange',
          }
          return (
            <Tag color={statusColors[value] || 'grey'}>
              {String(t(`printServiceSettings.logs.statuses.${value}`, value))}
            </Tag>
          )
        },
      },
      {
        title: t('printServiceSettings.logs.copies'),
        dataIndex: 'copies',
        key: 'copies',
        width: 80,
        align: 'center' as const,
      },
      {
        title: t('printServiceSettings.logs.createdAt'),
        dataIndex: 'created_at',
        key: 'created_at',
        width: 180,
        render: (value: string) => (value ? new Date(value).toLocaleString() : '-'),
      },
      {
        title: t('printServiceSettings.logs.printedAt'),
        dataIndex: 'printed_at',
        key: 'printed_at',
        width: 180,
        render: (value: string) => (value ? new Date(value).toLocaleString() : '-'),
      },
      {
        title: t('printServiceSettings.logs.error'),
        dataIndex: 'error_message',
        key: 'error_message',
        ellipsis: true,
        render: (value: string) =>
          value ? (
            <Tooltip content={value}>
              <Text type="danger">{value}</Text>
            </Tooltip>
          ) : (
            '-'
          ),
      },
    ],
    [t]
  )

  return (
    <Card
      className="print-logs-card"
      title={
        <Space>
          <IconHistory />
          {t('printServiceSettings.logs.title')}
        </Space>
      }
      headerExtraContent={
        <Button
          icon={<IconRefresh />}
          theme="borderless"
          size="small"
          onClick={() => refetch()}
          loading={isLoading}
        />
      }
    >
      <Table
        columns={columns}
        dataSource={jobs}
        rowKey="id"
        loading={isLoading}
        pagination={{ pageSize: 10 }}
        size="small"
        scroll={{ x: 1000 }}
        empty={
          <Empty
            image={<IconHistory style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />}
            title={t('printServiceSettings.logs.empty.title')}
            description={t('printServiceSettings.logs.empty.description')}
          />
        }
      />
    </Card>
  )
}

/**
 * Print Service Settings Page Component
 */
export default function PrintServiceSettingsPage() {
  const { t } = useTranslation('system')

  return (
    <Container size="lg" className="print-service-settings-page">
      <div className="page-header">
        <div className="page-title">
          <Title heading={3} style={{ margin: 0 }}>
            <IconServer style={{ marginRight: 8 }} />
            {t('printServiceSettings.title')}
          </Title>
          <Text type="tertiary">{t('printServiceSettings.subtitle')}</Text>
        </div>
      </div>

      <Banner
        type="info"
        description={t('printServiceSettings.infoBanner')}
        icon={<IconInfoCircle />}
        className="info-banner"
      />

      <Tabs type="line" className="settings-tabs">
        <TabPane
          tab={
            <span>
              <IconLink style={{ marginRight: 4 }} />
              {t('printServiceSettings.tabs.connection')}
            </span>
          }
          itemKey="connection"
        >
          <div className="tab-content">
            <div className="cards-grid">
              <PrintServiceStatusCard />
              <PrintersListCard />
            </div>
            <TestPrintCard />
          </div>
        </TabPane>

        <TabPane
          tab={
            <span>
              <IconSetting style={{ marginRight: 4 }} />
              {t('printServiceSettings.tabs.configuration')}
            </span>
          }
          itemKey="configuration"
        >
          <div className="tab-content">
            <div className="cards-grid">
              <DownloadLinksCard />
              <ConfigWizardCard />
            </div>
          </div>
        </TabPane>

        <TabPane
          tab={
            <span>
              <IconHistory style={{ marginRight: 4 }} />
              {t('printServiceSettings.tabs.logs')}
            </span>
          }
          itemKey="logs"
        >
          <div className="tab-content">
            <PrintLogsCard />
          </div>
        </TabPane>
      </Tabs>
    </Container>
  )
}
