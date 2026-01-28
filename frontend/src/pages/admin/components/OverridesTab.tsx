import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Table,
  Typography,
  Tag,
  Toast,
  Button,
  Modal,
  Empty,
  Spin,
  Popconfirm,
  Space,
  Tooltip,
} from '@douyinfe/semi-ui-19'
import { IconPlus, IconDelete, IconRefresh } from '@douyinfe/semi-icons'
import type { ColumnProps } from '@douyinfe/semi-ui-19/lib/es/table'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import { getFeatureFlags } from '@/api/feature-flags'
import type { Override, FlagType, OverrideTargetType } from '@/api/feature-flags'
import { OverrideForm } from './OverrideForm'

const { Text } = Typography

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
 * Check if override is expired
 */
function isExpired(expiresAt: string | undefined): boolean {
  if (!expiresAt) return false
  return new Date(expiresAt) < new Date()
}

/**
 * Get target type color
 */
function getTargetTypeColor(type: OverrideTargetType): TagColor {
  switch (type) {
    case 'user':
      return 'blue'
    case 'tenant':
      return 'purple'
    default:
      return 'grey'
  }
}

interface OverridesTabProps {
  flagKey: string
  flagType: FlagType
}

/**
 * Overrides Tab Component
 *
 * Features:
 * - List overrides for a feature flag
 * - Add new override
 * - Delete override
 * - Display expiration status
 */
export function OverridesTab({ flagKey, flagType }: OverridesTabProps) {
  const { t, i18n } = useTranslation('admin')
  const api = useMemo(() => getFeatureFlags(), [])

  // State
  const [overrides, setOverrides] = useState<Override[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(10)

  // Modal state
  const [createModalVisible, setCreateModalVisible] = useState(false)
  const [editingOverride, setEditingOverride] = useState<Override | null>(null)

  // Fetch overrides
  const fetchOverrides = useCallback(async () => {
    setLoading(true)
    try {
      const response = await api.listOverrides(flagKey, {
        page: currentPage,
        page_size: pageSize,
      })
      if (response.success && response.data) {
        setOverrides(response.data.overrides)
        setTotal(response.data.total)
      }
    } catch {
      Toast.error(t('featureFlags.overrides.fetchError', 'Failed to load overrides'))
    } finally {
      setLoading(false)
    }
  }, [api, flagKey, currentPage, pageSize, t])

  // Load on mount
  useEffect(() => {
    fetchOverrides()
  }, [fetchOverrides])

  // Handle delete
  const handleDelete = useCallback(
    async (override: Override) => {
      try {
        const response = await api.deleteOverride(flagKey, override.id)
        if (response.success) {
          Toast.success(t('featureFlags.overrides.deleteSuccess', 'Override deleted successfully'))
          fetchOverrides()
        } else {
          Toast.error(
            response.error?.message ||
              t('featureFlags.overrides.deleteError', 'Failed to delete override')
          )
        }
      } catch {
        Toast.error(t('featureFlags.overrides.deleteError', 'Failed to delete override'))
      }
    },
    [api, flagKey, t, fetchOverrides]
  )

  // Handle create success
  const handleCreateSuccess = useCallback(() => {
    setCreateModalVisible(false)
    setEditingOverride(null)
    fetchOverrides()
  }, [fetchOverrides])

  // Handle page change
  const handlePageChange = useCallback((page: number) => {
    setCurrentPage(page)
  }, [])

  // Table columns
  const columns: ColumnProps<Override>[] = useMemo(
    () => [
      {
        title: t('featureFlags.overrides.targetType', 'Target Type'),
        dataIndex: 'target_type',
        key: 'target_type',
        width: 120,
        render: (type: OverrideTargetType) => (
          <Tag color={getTargetTypeColor(type)}>
            {t(`featureFlags.overrides.targetTypes.${type}`, type)}
          </Tag>
        ),
      },
      {
        title: t('featureFlags.overrides.targetId', 'Target ID'),
        dataIndex: 'target_id',
        key: 'target_id',
        width: 180,
        render: (id: string, record: Override) => (
          <div>
            <Text copyable ellipsis={{ showTooltip: true }} style={{ maxWidth: 150 }}>
              {id}
            </Text>
            {record.target_name && (
              <div>
                <Text type="tertiary" size="small">
                  {record.target_name}
                </Text>
              </div>
            )}
          </div>
        ),
      },
      {
        title: t('featureFlags.overrides.value', 'Override Value'),
        dataIndex: 'value',
        key: 'value',
        width: 150,
        render: (_, record: Override) => {
          if (flagType === 'boolean' || flagType === 'user_segment') {
            return (
              <Tag color={record.value.enabled ? 'green' : 'grey'}>
                {record.value.enabled
                  ? t('featureFlags.status.enabled', 'Enabled')
                  : t('featureFlags.status.disabled', 'Disabled')}
              </Tag>
            )
          }
          if (flagType === 'percentage') {
            const percentage = (record.value.metadata?.percentage as number) || 0
            return <Tag color="orange">{percentage}%</Tag>
          }
          if (flagType === 'variant') {
            return <Tag color="purple">{record.value.variant || '-'}</Tag>
          }
          return '-'
        },
      },
      {
        title: t('featureFlags.overrides.reason', 'Reason'),
        dataIndex: 'reason',
        key: 'reason',
        width: 200,
        render: (reason: string | undefined) => (
          <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 180 }}>
            {reason || <Text type="tertiary">-</Text>}
          </Text>
        ),
      },
      {
        title: t('featureFlags.overrides.expiresAt', 'Expires At'),
        dataIndex: 'expires_at',
        key: 'expires_at',
        width: 160,
        render: (expiresAt: string | undefined) => {
          if (!expiresAt) {
            return (
              <Text type="tertiary">
                {t('featureFlags.overrides.noExpiration', 'No expiration')}
              </Text>
            )
          }
          const expired = isExpired(expiresAt)
          return (
            <span>
              {formatDate(expiresAt, i18n.language)}
              {expired && (
                <Tag color="red" size="small" style={{ marginLeft: 4 }}>
                  {t('featureFlags.overrides.expired', 'Expired')}
                </Tag>
              )}
            </span>
          )
        },
      },
      {
        title: t('featureFlags.overrides.createdBy', 'Created By'),
        dataIndex: 'created_by_name',
        key: 'created_by_name',
        width: 120,
        render: (name: string | undefined) => <Text>{name || <Text type="tertiary">-</Text>}</Text>,
      },
      {
        title: t('featureFlags.overrides.createdAt', 'Created At'),
        dataIndex: 'created_at',
        key: 'created_at',
        width: 160,
        render: (date: string) => formatDate(date, i18n.language),
      },
      {
        title: t('featureFlags.overrides.actions', 'Actions'),
        key: 'actions',
        width: 100,
        fixed: 'right',
        render: (_: unknown, record: Override) => (
          <Space>
            <Popconfirm
              title={t('featureFlags.overrides.deleteConfirmTitle', 'Delete Override')}
              content={t(
                'featureFlags.overrides.deleteConfirmContent',
                'Are you sure you want to delete this override?'
              )}
              okType="danger"
              onConfirm={() => handleDelete(record)}
            >
              <Tooltip content={t('common.delete', 'Delete')}>
                <Button icon={<IconDelete />} type="danger" theme="borderless" size="small" />
              </Tooltip>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [t, i18n.language, flagType, handleDelete]
  )

  return (
    <div className="overrides-tab">
      {/* Header */}
      <div className="overrides-tab-header">
        <Text type="secondary">
          {t(
            'featureFlags.overrides.description',
            'Overrides allow you to set specific values for individual users or tenants, bypassing the normal targeting rules.'
          )}
        </Text>
        <Space>
          <Button icon={<IconRefresh />} onClick={fetchOverrides}>
            {t('common.refresh', 'Refresh')}
          </Button>
          <Button icon={<IconPlus />} theme="solid" onClick={() => setCreateModalVisible(true)}>
            {t('featureFlags.overrides.addOverride', 'Add Override')}
          </Button>
        </Space>
      </div>

      {/* Table */}
      <Spin spinning={loading}>
        {overrides.length === 0 && !loading ? (
          <Empty
            title={t('featureFlags.overrides.empty', 'No overrides')}
            description={t(
              'featureFlags.overrides.emptyDescription',
              'Add an override to set specific values for individual users or tenants.'
            )}
          />
        ) : (
          <Table
            columns={columns}
            dataSource={overrides}
            rowKey="id"
            pagination={{
              currentPage,
              pageSize,
              total,
              onPageChange: handlePageChange,
            }}
            scroll={{ x: 1200 }}
          />
        )}
      </Spin>

      {/* Create Modal */}
      <Modal
        title={t('featureFlags.overrides.addOverrideTitle', 'Add Override')}
        visible={createModalVisible}
        onCancel={() => {
          setCreateModalVisible(false)
          setEditingOverride(null)
        }}
        footer={null}
        width={600}
      >
        <OverrideForm
          flagKey={flagKey}
          flagType={flagType}
          override={editingOverride}
          onSuccess={handleCreateSuccess}
          onCancel={() => {
            setCreateModalVisible(false)
            setEditingOverride(null)
          }}
        />
      </Modal>
    </div>
  )
}

export default OverridesTab
