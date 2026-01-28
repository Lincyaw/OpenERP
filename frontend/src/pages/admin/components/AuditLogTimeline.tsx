import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Timeline,
  Typography,
  Tag,
  Toast,
  Button,
  Spin,
  Empty,
  Collapsible,
} from '@douyinfe/semi-ui-19'
import { IconChevronDown, IconChevronUp } from '@douyinfe/semi-icons'
import { getFeatureFlagAuditLogs } from '@/api/feature-flags/feature-flags'
import type { DtoAuditLogResponse } from '@/api/models'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'

const { Text } = Typography

// Type alias for cleaner code
type AuditLog = DtoAuditLogResponse

/**
 * Format date for display
 */
function formatDate(dateStr: string, locale: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString(locale, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

/**
 * Get color for action type
 */
function getActionColor(action: string): TagColor {
  switch (action.toLowerCase()) {
    case 'created':
      return 'green'
    case 'updated':
      return 'blue'
    case 'enabled':
      return 'green'
    case 'disabled':
      return 'orange'
    case 'archived':
      return 'red'
    case 'override_created':
      return 'cyan'
    case 'override_deleted':
      return 'pink'
    default:
      return 'grey'
  }
}

/**
 * Get action icon color for timeline
 */
function getTimelineColor(action: string): 'green' | 'blue' | 'grey' | 'red' | 'pink' {
  switch (action.toLowerCase()) {
    case 'created':
      return 'green'
    case 'enabled':
      return 'green'
    case 'updated':
      return 'blue'
    case 'disabled':
      return 'grey'
    case 'archived':
      return 'red'
    case 'override_deleted':
      return 'pink'
    default:
      return 'blue'
  }
}

interface AuditLogTimelineProps {
  flagKey: string
}

/**
 * Audit Log Timeline Component
 *
 * Features:
 * - Display audit logs in timeline format
 * - Show change details
 * - Show user who made the change
 * - Support pagination (load more)
 */
export function AuditLogTimeline({ flagKey }: AuditLogTimelineProps) {
  const { t, i18n } = useTranslation('admin')

  // State
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [loading, setLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [total, setTotal] = useState(0)
  const [currentPage, setCurrentPage] = useState(1)
  const [pageSize] = useState(20)

  // Whether there are more logs to load
  const hasMore = useMemo(() => logs.length < total, [logs.length, total])

  // Fetch audit logs
  const fetchLogs = useCallback(
    async (page: number, append: boolean = false) => {
      if (append) {
        setLoadingMore(true)
      } else {
        setLoading(true)
      }

      try {
        const response = await getFeatureFlagAuditLogs(flagKey, {
          page,
          page_size: pageSize,
        })
        if (response.status === 200 && response.data.success && response.data.data) {
          if (append) {
            setLogs((prev) => [...prev, ...(response.data.data?.logs || [])])
          } else {
            setLogs(response.data.data.logs || [])
          }
          setTotal(response.data.data.total || 0)
          setCurrentPage(page)
        }
      } catch {
        Toast.error(t('featureFlags.auditLog.fetchError', 'Failed to load audit logs'))
      } finally {
        setLoading(false)
        setLoadingMore(false)
      }
    },
    [flagKey, pageSize, t]
  )

  // Load on mount
  useEffect(() => {
    fetchLogs(1)
  }, [fetchLogs])

  // Handle load more
  const handleLoadMore = useCallback(() => {
    fetchLogs(currentPage + 1, true)
  }, [currentPage, fetchLogs])

  // Render loading state
  if (loading) {
    return (
      <div className="audit-log-timeline-loading">
        <Spin size="large" />
      </div>
    )
  }

  // Render empty state
  if (logs.length === 0) {
    return (
      <Empty
        title={t('featureFlags.auditLog.empty', 'No audit logs')}
        description={t(
          'featureFlags.auditLog.emptyDescription',
          'Changes to this feature flag will appear here.'
        )}
      />
    )
  }

  return (
    <div className="audit-log-timeline">
      <div className="audit-log-timeline-header">
        <Text type="secondary">
          {t(
            'featureFlags.auditLog.description',
            'History of all changes made to this feature flag.'
          )}
        </Text>
        <Text type="tertiary" size="small">
          {t('featureFlags.auditLog.totalLogs', '{{count}} entries', { count: total })}
        </Text>
      </div>

      <Timeline mode="left" className="audit-log-timeline-content">
        {logs.map((log) => (
          <Timeline.Item
            key={log.id}
            time={formatDate(log.created_at || '', i18n.language)}
            color={getTimelineColor(log.action || '')}
          >
            <AuditLogItem log={log} />
          </Timeline.Item>
        ))}
      </Timeline>

      {/* Load More Button */}
      {hasMore && (
        <div className="audit-log-timeline-footer">
          <Button onClick={handleLoadMore} loading={loadingMore} theme="borderless">
            {t('featureFlags.auditLog.loadMore', 'Load more')}
          </Button>
        </div>
      )}
    </div>
  )
}

/**
 * Single Audit Log Item Component
 */
interface AuditLogItemProps {
  log: AuditLog
}

function AuditLogItem({ log }: AuditLogItemProps) {
  const { t } = useTranslation('admin')
  const [expanded, setExpanded] = useState(false)

  // Use new_value/old_value for changes display
  const hasChanges = log.new_value && Object.keys(log.new_value).length > 0

  return (
    <div className="audit-log-item">
      <div className="audit-log-item-header">
        <Tag color={getActionColor(log.action || '')}>
          {String(
            t(`featureFlags.auditLog.actions.${(log.action || '').toLowerCase()}`, log.action || '')
          )}
        </Tag>
        <Text type="secondary" size="small">
          {t('featureFlags.auditLog.by', 'by')}{' '}
          {log.user_id || t('featureFlags.auditLog.unknown', 'Unknown')}
        </Text>
      </div>

      {/* Changes Details */}
      {hasChanges && (
        <div className="audit-log-item-changes">
          <Button
            icon={expanded ? <IconChevronUp /> : <IconChevronDown />}
            theme="borderless"
            size="small"
            onClick={() => setExpanded(!expanded)}
          >
            {expanded
              ? t('featureFlags.auditLog.hideDetails', 'Hide details')
              : t('featureFlags.auditLog.showDetails', 'Show details')}
          </Button>

          <Collapsible isOpen={expanded}>
            <div className="audit-log-changes-content">
              <ChangesDisplay changes={log.new_value!} />
            </div>
          </Collapsible>
        </div>
      )}
    </div>
  )
}

/**
 * Changes Display Component
 */
interface ChangesDisplayProps {
  changes: Record<string, unknown>
}

function ChangesDisplay({ changes }: ChangesDisplayProps) {
  const { t } = useTranslation('admin')

  // Format a single change value
  const formatValue = (value: unknown): string => {
    if (value === null || value === undefined) {
      return '-'
    }
    if (typeof value === 'boolean') {
      return value ? 'true' : 'false'
    }
    if (typeof value === 'object') {
      return JSON.stringify(value, null, 2)
    }
    return String(value)
  }

  // Check if the change is an old/new pair
  const isOldNewPair = (value: unknown): value is { old: unknown; new: unknown } => {
    return typeof value === 'object' && value !== null && 'old' in value && 'new' in value
  }

  return (
    <div className="changes-display">
      {Object.entries(changes).map(([key, value]) => (
        <div key={key} className="change-item">
          <Text strong className="change-field">
            {String(t(`featureFlags.auditLog.fields.${key}`, key))}:
          </Text>
          {isOldNewPair(value) ? (
            <div className="change-values">
              <div className="change-old">
                <Tag size="small" color="red">
                  {t('featureFlags.auditLog.old', 'Old')}
                </Tag>
                <Text className="change-value">{formatValue(value.old)}</Text>
              </div>
              <div className="change-new">
                <Tag size="small" color="green">
                  {t('featureFlags.auditLog.new', 'New')}
                </Tag>
                <Text className="change-value">{formatValue(value.new)}</Text>
              </div>
            </div>
          ) : (
            <Text className="change-value">{formatValue(value)}</Text>
          )}
        </div>
      ))}
    </div>
  )
}

export default AuditLogTimeline
