import { useState, useCallback, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Toast,
  Tabs,
  TabPane,
  Table,
  Button,
  Space,
  Tag,
  Progress,
  Descriptions,
  Empty,
  Spin,
  Popconfirm,
  Modal,
  Timeline,
} from '@douyinfe/semi-ui-19'
import {
  IconTick,
  IconClose,
  IconRefresh,
  IconHistory,
  IconPlay,
  IconClock,
  IconAlertCircle,
} from '@douyinfe/semi-icons'
import type { ColumnProps } from '@douyinfe/semi-ui-19/lib/es/table'
import { Container } from '@/components/common/layout'
import './PlatformSyncStatus.css'

const { Title, Text } = Typography

/**
 * E-commerce platform codes matching backend domain
 */
type PlatformCode = 'TAOBAO' | 'DOUYIN' | 'JD' | 'PDD' | 'WECHAT' | 'KUAISHOU'

/**
 * Sync type enum
 */
type SyncType = 'ORDER' | 'INVENTORY' | 'PRODUCT'

/**
 * Sync status enum
 */
type SyncStatus = 'PENDING' | 'RUNNING' | 'SUCCESS' | 'FAILED' | 'CANCELLED'

/**
 * Sync history record
 */
interface SyncRecord {
  id: string
  platformCode: PlatformCode
  syncType: SyncType
  status: SyncStatus
  startedAt: string
  completedAt?: string
  totalItems: number
  successItems: number
  failedItems: number
  errorMessage?: string
  triggeredBy: 'AUTO' | 'MANUAL'
}

/**
 * Platform sync status summary
 */
interface PlatformSyncSummary {
  code: PlatformCode
  icon: string
  color: string
  enabled: boolean
  lastOrderSync?: string
  lastInventorySync?: string
  lastProductSync?: string
  orderSyncStatus?: SyncStatus
  inventorySyncStatus?: SyncStatus
  productSyncStatus?: SyncStatus
  nextScheduledSync?: string
  syncEnabled: boolean
  syncIntervalMinutes: number
  pendingOrders: number
  pendingInventoryUpdates: number
}

/**
 * Platform metadata for display
 */
const PLATFORMS: Array<{ code: PlatformCode; icon: string; color: string }> = [
  { code: 'TAOBAO', icon: 'TB', color: '#FF5000' },
  { code: 'DOUYIN', icon: 'DY', color: '#000000' },
  { code: 'JD', icon: 'JD', color: '#E2231A' },
  { code: 'PDD', icon: 'PDD', color: '#E02E24' },
  { code: 'WECHAT', icon: 'WX', color: '#07C160' },
  { code: 'KUAISHOU', icon: 'KS', color: '#FF4906' },
]

/**
 * Generate mock sync history data
 */
function generateMockSyncHistory(platformCode?: PlatformCode): SyncRecord[] {
  const types: SyncType[] = ['ORDER', 'INVENTORY', 'PRODUCT']
  const statuses: SyncStatus[] = ['SUCCESS', 'SUCCESS', 'SUCCESS', 'FAILED', 'SUCCESS']
  const triggers: Array<'AUTO' | 'MANUAL'> = ['AUTO', 'AUTO', 'AUTO', 'MANUAL', 'AUTO']
  const platforms = platformCode ? [platformCode] : PLATFORMS.map((p) => p.code)

  const records: SyncRecord[] = []
  const now = new Date()

  for (let i = 0; i < 20; i++) {
    const startTime = new Date(now.getTime() - i * 30 * 60 * 1000) // Every 30 minutes
    const status = statuses[Math.floor(Math.random() * statuses.length)]
    const duration = Math.floor(Math.random() * 60000) + 5000 // 5-65 seconds
    const totalItems = Math.floor(Math.random() * 100) + 10
    const failedItems = status === 'FAILED' ? Math.floor(Math.random() * 10) + 1 : 0

    records.push({
      id: `sync-${Date.now()}-${i}`,
      platformCode: platforms[Math.floor(Math.random() * platforms.length)],
      syncType: types[Math.floor(Math.random() * types.length)],
      status,
      startedAt: startTime.toISOString(),
      completedAt:
        status !== 'RUNNING' ? new Date(startTime.getTime() + duration).toISOString() : undefined,
      totalItems,
      successItems: totalItems - failedItems,
      failedItems,
      errorMessage: status === 'FAILED' ? 'API rate limit exceeded' : undefined,
      triggeredBy: triggers[Math.floor(Math.random() * triggers.length)],
    })
  }

  return records.sort((a, b) => new Date(b.startedAt).getTime() - new Date(a.startedAt).getTime())
}

/**
 * Generate mock platform summary data
 */
function generateMockPlatformSummaries(): PlatformSyncSummary[] {
  return PLATFORMS.map((platform) => {
    const enabled = Math.random() > 0.3
    const syncEnabled = enabled && Math.random() > 0.2
    const now = new Date()

    return {
      ...platform,
      enabled,
      syncEnabled,
      syncIntervalMinutes: 15,
      lastOrderSync: enabled
        ? new Date(now.getTime() - Math.random() * 3600000).toISOString()
        : undefined,
      lastInventorySync: enabled
        ? new Date(now.getTime() - Math.random() * 7200000).toISOString()
        : undefined,
      lastProductSync: enabled
        ? new Date(now.getTime() - Math.random() * 86400000).toISOString()
        : undefined,
      orderSyncStatus: enabled ? (Math.random() > 0.2 ? 'SUCCESS' : 'FAILED') : undefined,
      inventorySyncStatus: enabled ? (Math.random() > 0.2 ? 'SUCCESS' : 'FAILED') : undefined,
      productSyncStatus: enabled ? (Math.random() > 0.2 ? 'SUCCESS' : 'FAILED') : undefined,
      nextScheduledSync:
        enabled && syncEnabled
          ? new Date(now.getTime() + Math.random() * 900000).toISOString()
          : undefined,
      pendingOrders: enabled ? Math.floor(Math.random() * 20) : 0,
      pendingInventoryUpdates: enabled ? Math.floor(Math.random() * 50) : 0,
    }
  })
}

/**
 * Platform Order Sync Status Page
 *
 * Features:
 * - Display sync history for all platforms
 * - Show current sync status per platform
 * - Support manual sync triggering
 */
export default function PlatformSyncStatusPage() {
  const { t } = useTranslation('system')

  // State
  const [loading, setLoading] = useState(false)
  const [syncHistory, setSyncHistory] = useState<SyncRecord[]>([])
  const [platformSummaries, setPlatformSummaries] = useState<PlatformSyncSummary[]>([])
  const [activeTab, setActiveTab] = useState<string>('overview')
  const [syncing, setSyncing] = useState<{ platform: PlatformCode; type: SyncType } | null>(null)
  const [detailRecord, setDetailRecord] = useState<SyncRecord | null>(null)

  /**
   * Load sync data from backend
   */
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      // TODO: Implement API call to load sync data
      // const [history, summaries] = await Promise.all([
      //   api.getSyncHistory(),
      //   api.getPlatformSyncSummaries(),
      // ])
      await new Promise((resolve) => setTimeout(resolve, 800))

      // Mock data for now
      setSyncHistory(generateMockSyncHistory())
      setPlatformSummaries(generateMockPlatformSummaries())
    } catch {
      Toast.error(t('syncStatus.messages.loadError'))
    } finally {
      setLoading(false)
    }
  }, [t])

  // Load data on mount
  useEffect(() => {
    loadData()
  }, [loadData])

  /**
   * Trigger manual sync for a platform
   */
  const handleManualSync = useCallback(
    async (platformCode: PlatformCode, syncType: SyncType) => {
      setSyncing({ platform: platformCode, type: syncType })
      try {
        // TODO: Implement API call to trigger sync
        // await api.triggerSync(platformCode, syncType)
        await new Promise((resolve) => setTimeout(resolve, 2000))

        Toast.success(t('syncStatus.messages.syncStarted'))
        // Reload data after sync starts
        await loadData()
      } catch {
        Toast.error(t('syncStatus.messages.syncError'))
      } finally {
        setSyncing(null)
      }
    },
    [t, loadData]
  )

  /**
   * Format duration between two dates
   */
  const formatDuration = useCallback((start: string, end?: string): string => {
    if (!end) return '-'
    const duration = new Date(end).getTime() - new Date(start).getTime()
    if (duration < 1000) return `${duration}ms`
    if (duration < 60000) return `${Math.round(duration / 1000)}s`
    return `${Math.round(duration / 60000)}m ${Math.round((duration % 60000) / 1000)}s`
  }, [])

  /**
   * Format relative time
   */
  const formatRelativeTime = useCallback(
    (dateStr?: string): string => {
      if (!dateStr) return t('syncStatus.neverSynced')
      const date = new Date(dateStr)
      const now = new Date()
      const diff = now.getTime() - date.getTime()

      if (diff < 60000) return t('syncStatus.justNow')
      if (diff < 3600000) return t('syncStatus.minutesAgo', { count: Math.floor(diff / 60000) })
      if (diff < 86400000) return t('syncStatus.hoursAgo', { count: Math.floor(diff / 3600000) })
      return date.toLocaleString()
    },
    [t]
  )

  /**
   * Get status tag for sync status
   */
  const getStatusTag = useCallback(
    (status?: SyncStatus) => {
      if (!status) return <Tag color="grey">{t('syncStatus.status.unknown')}</Tag>

      const configs: Record<
        SyncStatus,
        { color: 'blue' | 'green' | 'red' | 'grey'; icon: React.ReactNode }
      > = {
        PENDING: { color: 'blue', icon: <IconClock /> },
        RUNNING: { color: 'blue', icon: <IconRefresh spin /> },
        SUCCESS: { color: 'green', icon: <IconTick /> },
        FAILED: { color: 'red', icon: <IconClose /> },
        CANCELLED: { color: 'grey', icon: <IconAlertCircle /> },
      }

      const config = configs[status]
      return (
        <Tag color={config.color} prefixIcon={config.icon}>
          {t(`syncStatus.status.${status.toLowerCase()}`) as string}
        </Tag>
      )
    },
    [t]
  )

  /**
   * Get sync type label
   */
  const getSyncTypeLabel = useCallback(
    (type: SyncType): string => {
      return t(`syncStatus.syncTypes.${type.toLowerCase()}`) as string
    },
    [t]
  )

  /**
   * Get platform name by code
   */
  const getPlatformName = useCallback(
    (code: PlatformCode): string => {
      return t(`syncStatus.platforms.${code}`) as string
    },
    [t]
  )

  /**
   * Render platform icon
   */
  const renderPlatformIcon = useCallback((code: PlatformCode) => {
    const platform = PLATFORMS.find((p) => p.code === code)
    if (!platform) return null
    return (
      <span className="platform-icon" style={{ backgroundColor: platform.color }}>
        {platform.icon}
      </span>
    )
  }, [])

  /**
   * Table columns for sync history
   */
  const historyColumns: ColumnProps<SyncRecord>[] = [
    {
      title: t('syncStatus.columns.platform'),
      dataIndex: 'platformCode',
      width: 140,
      render: (code: PlatformCode) => (
        <Space>
          {renderPlatformIcon(code)}
          <Text>{getPlatformName(code)}</Text>
        </Space>
      ),
    },
    {
      title: t('syncStatus.columns.syncType'),
      dataIndex: 'syncType',
      width: 120,
      render: (type: SyncType) => <Tag>{getSyncTypeLabel(type)}</Tag>,
    },
    {
      title: t('syncStatus.columns.status'),
      dataIndex: 'status',
      width: 120,
      render: (status: SyncStatus) => getStatusTag(status),
    },
    {
      title: t('syncStatus.columns.progress'),
      dataIndex: 'totalItems',
      width: 180,
      render: (_: number, record: SyncRecord) => {
        const percent =
          record.totalItems > 0 ? Math.round((record.successItems / record.totalItems) * 100) : 0
        return (
          <Space vertical align="start" spacing="tight">
            <Progress
              percent={percent}
              size="small"
              style={{ width: 120 }}
              stroke={record.failedItems > 0 ? 'var(--semi-color-warning)' : undefined}
            />
            <Text size="small" type="tertiary">
              {record.successItems}/{record.totalItems}{' '}
              {record.failedItems > 0 && (
                <Text size="small" type="danger">
                  ({record.failedItems} {t('syncStatus.failed')})
                </Text>
              )}
            </Text>
          </Space>
        )
      },
    },
    {
      title: t('syncStatus.columns.triggeredBy'),
      dataIndex: 'triggeredBy',
      width: 100,
      render: (trigger: 'AUTO' | 'MANUAL') => (
        <Tag color={trigger === 'MANUAL' ? 'blue' : 'grey'}>
          {t(`syncStatus.trigger.${trigger.toLowerCase()}`) as string}
        </Tag>
      ),
    },
    {
      title: t('syncStatus.columns.startedAt'),
      dataIndex: 'startedAt',
      width: 160,
      render: (date: string) => new Date(date).toLocaleString(),
    },
    {
      title: t('syncStatus.columns.duration'),
      dataIndex: 'completedAt',
      width: 100,
      render: (_: string | undefined, record: SyncRecord) =>
        formatDuration(record.startedAt, record.completedAt),
    },
    {
      title: t('syncStatus.columns.actions'),
      dataIndex: 'actions',
      width: 100,
      render: (_: unknown, record: SyncRecord) => (
        <Button theme="borderless" icon={<IconHistory />} onClick={() => setDetailRecord(record)}>
          {t('syncStatus.viewDetails')}
        </Button>
      ),
    },
  ]

  /**
   * Render platform summary card
   */
  const renderPlatformCard = useCallback(
    (summary: PlatformSyncSummary) => {
      const isSyncing = syncing?.platform === summary.code

      return (
        <Card
          key={summary.code}
          className="platform-status-card"
          title={
            <Space>
              <span className="platform-icon" style={{ backgroundColor: summary.color }}>
                {summary.icon}
              </span>
              <Text strong>{getPlatformName(summary.code)}</Text>
              {!summary.enabled && <Tag color="grey">{t('syncStatus.platformDisabled')}</Tag>}
            </Space>
          }
          headerExtraContent={
            summary.enabled && (
              <Tag color={summary.syncEnabled ? 'green' : 'orange'}>
                {summary.syncEnabled
                  ? t('syncStatus.autoSyncEnabled')
                  : t('syncStatus.autoSyncDisabled')}
              </Tag>
            )
          }
        >
          {!summary.enabled ? (
            <Empty
              image={
                <IconAlertCircle style={{ fontSize: 48, color: 'var(--semi-color-text-2)' }} />
              }
              description={t('syncStatus.platformNotConfigured')}
            />
          ) : (
            <>
              <Descriptions
                row
                data={[
                  {
                    key: t('syncStatus.fields.lastOrderSync'),
                    value: (
                      <Space>
                        {getStatusTag(summary.orderSyncStatus)}
                        <Text type="tertiary">{formatRelativeTime(summary.lastOrderSync)}</Text>
                      </Space>
                    ),
                  },
                  {
                    key: t('syncStatus.fields.lastInventorySync'),
                    value: (
                      <Space>
                        {getStatusTag(summary.inventorySyncStatus)}
                        <Text type="tertiary">{formatRelativeTime(summary.lastInventorySync)}</Text>
                      </Space>
                    ),
                  },
                  {
                    key: t('syncStatus.fields.nextScheduledSync'),
                    value: summary.nextScheduledSync
                      ? new Date(summary.nextScheduledSync).toLocaleString()
                      : t('syncStatus.notScheduled'),
                  },
                  {
                    key: t('syncStatus.fields.pendingOrders'),
                    value: (
                      <Tag color={summary.pendingOrders > 0 ? 'blue' : 'grey'}>
                        {summary.pendingOrders}
                      </Tag>
                    ),
                  },
                ]}
              />

              <div className="sync-actions">
                <Space>
                  <Popconfirm
                    title={t('syncStatus.confirmSync.title')}
                    content={t('syncStatus.confirmSync.orderContent')}
                    onConfirm={() => handleManualSync(summary.code, 'ORDER')}
                  >
                    <Button
                      icon={<IconPlay />}
                      loading={isSyncing && syncing?.type === 'ORDER'}
                      disabled={isSyncing}
                    >
                      {t('syncStatus.syncOrders')}
                    </Button>
                  </Popconfirm>
                  <Popconfirm
                    title={t('syncStatus.confirmSync.title')}
                    content={t('syncStatus.confirmSync.inventoryContent')}
                    onConfirm={() => handleManualSync(summary.code, 'INVENTORY')}
                  >
                    <Button
                      icon={<IconRefresh />}
                      loading={isSyncing && syncing?.type === 'INVENTORY'}
                      disabled={isSyncing}
                    >
                      {t('syncStatus.syncInventory')}
                    </Button>
                  </Popconfirm>
                </Space>
              </div>
            </>
          )}
        </Card>
      )
    },
    [syncing, t, getStatusTag, formatRelativeTime, handleManualSync, getPlatformName]
  )

  /**
   * Render sync detail modal
   */
  const renderDetailModal = useCallback(() => {
    if (!detailRecord) return null

    return (
      <Modal
        title={t('syncStatus.detailModal.title')}
        visible={!!detailRecord}
        onCancel={() => setDetailRecord(null)}
        footer={<Button onClick={() => setDetailRecord(null)}>{t('common.close')}</Button>}
        width={600}
      >
        <Descriptions
          data={[
            {
              key: t('syncStatus.columns.platform'),
              value: (
                <Space>
                  {renderPlatformIcon(detailRecord.platformCode)}
                  {getPlatformName(detailRecord.platformCode)}
                </Space>
              ),
            },
            {
              key: t('syncStatus.columns.syncType'),
              value: getSyncTypeLabel(detailRecord.syncType),
            },
            {
              key: t('syncStatus.columns.status'),
              value: getStatusTag(detailRecord.status),
            },
            {
              key: t('syncStatus.columns.triggeredBy'),
              value: t(`syncStatus.trigger.${detailRecord.triggeredBy.toLowerCase()}`) as string,
            },
            {
              key: t('syncStatus.columns.startedAt'),
              value: new Date(detailRecord.startedAt).toLocaleString(),
            },
            {
              key: t('syncStatus.detailModal.completedAt'),
              value: detailRecord.completedAt
                ? new Date(detailRecord.completedAt).toLocaleString()
                : '-',
            },
            {
              key: t('syncStatus.columns.duration'),
              value: formatDuration(detailRecord.startedAt, detailRecord.completedAt),
            },
            {
              key: t('syncStatus.detailModal.totalItems'),
              value: detailRecord.totalItems,
            },
            {
              key: t('syncStatus.detailModal.successItems'),
              value: <Text type="success">{detailRecord.successItems}</Text>,
            },
            {
              key: t('syncStatus.detailModal.failedItems'),
              value: (
                <Text type={detailRecord.failedItems > 0 ? 'danger' : undefined}>
                  {detailRecord.failedItems}
                </Text>
              ),
            },
          ]}
        />

        {detailRecord.errorMessage && (
          <div className="error-section">
            <Title heading={6}>{t('syncStatus.detailModal.errorMessage')}</Title>
            <div className="error-box">
              <Text type="danger">{detailRecord.errorMessage}</Text>
            </div>
          </div>
        )}

        <div className="timeline-section">
          <Title heading={6}>{t('syncStatus.detailModal.timeline')}</Title>
          <Timeline>
            <Timeline.Item time={new Date(detailRecord.startedAt).toLocaleTimeString()}>
              {t('syncStatus.timeline.started')}
            </Timeline.Item>
            {detailRecord.status === 'RUNNING' ? (
              <Timeline.Item type="ongoing" time={new Date().toLocaleTimeString()}>
                {t('syncStatus.timeline.running')}
              </Timeline.Item>
            ) : detailRecord.completedAt ? (
              <Timeline.Item
                type={detailRecord.status === 'SUCCESS' ? 'success' : 'error'}
                time={new Date(detailRecord.completedAt).toLocaleTimeString()}
              >
                {detailRecord.status === 'SUCCESS'
                  ? t('syncStatus.timeline.completed')
                  : t('syncStatus.timeline.failed')}
              </Timeline.Item>
            ) : null}
          </Timeline>
        </div>
      </Modal>
    )
  }, [
    detailRecord,
    t,
    renderPlatformIcon,
    getPlatformName,
    getSyncTypeLabel,
    getStatusTag,
    formatDuration,
  ])

  return (
    <Container size="lg" className="sync-status-page">
      <Card className="sync-status-card">
        <div className="sync-status-header">
          <div>
            <Title heading={4} style={{ margin: 0 }}>
              {t('syncStatus.title')}
            </Title>
            <Text type="tertiary">{t('syncStatus.subtitle')}</Text>
          </div>
          <Button icon={<IconRefresh />} onClick={loadData} loading={loading}>
            {t('common.refresh')}
          </Button>
        </div>

        <Spin spinning={loading}>
          <Tabs
            type="line"
            activeKey={activeTab}
            onChange={setActiveTab}
            className="sync-status-tabs"
          >
            <TabPane
              tab={
                <span>
                  <IconHistory style={{ marginRight: 8 }} />
                  {t('syncStatus.tabs.overview')}
                </span>
              }
              itemKey="overview"
            >
              <div className="platform-cards-grid">{platformSummaries.map(renderPlatformCard)}</div>
            </TabPane>

            <TabPane
              tab={
                <span>
                  <IconClock style={{ marginRight: 8 }} />
                  {t('syncStatus.tabs.history')}
                </span>
              }
              itemKey="history"
            >
              <Table
                columns={historyColumns}
                dataSource={syncHistory}
                rowKey="id"
                pagination={{
                  pageSize: 10,
                  showTotal: true,
                  showSizeChanger: true,
                  pageSizeOpts: [10, 20, 50],
                }}
                empty={<Empty description={t('syncStatus.noHistory')} />}
              />
            </TabPane>
          </Tabs>
        </Spin>
      </Card>

      {renderDetailModal()}
    </Container>
  )
}
