/**
 * Audit Logs Page
 *
 * Displays admin action audit logs for super admin with:
 * - Paginated log list (50 per page)
 * - Filtering by action type, target type, time range
 * - Search by operator or target ID
 * - Expandable row for old/new value comparison
 * - CSV export functionality
 * - Auto-refresh option (30 seconds)
 */
import { useState, useMemo, useCallback, useEffect, useRef } from 'react'
import {
  Typography,
  Card,
  Table,
  Space,
  Tag,
  Input,
  DatePicker,
  Select,
  Button,
  Empty,
  Spin,
  Tooltip,
  Checkbox,
  Descriptions,
  Toast,
} from '@douyinfe/semi-ui-19'
import type { TagColor } from '@douyinfe/semi-ui-19/lib/es/tag'
import type { OnExpand } from '@douyinfe/semi-ui-19/lib/es/table/interface'
import {
  IconSearch,
  IconRefresh,
  IconDownload,
  IconChevronDown,
  IconChevronRight,
  IconClock,
  IconFilter,
} from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'

import { Container } from '@/components/common/layout'
import { useAdminAuditLogs, adminQueryKeys } from '@/api/admin'
import type { AdminAuditLogEntry, AdminAuditLogListParams } from '@/api/admin'
import { useFormatters } from '@/hooks/useFormatters'
import { useQueryClient } from '@tanstack/react-query'

const { Title, Text } = Typography

// ============================================================================
// Types
// ============================================================================

interface FilterState {
  search: string
  action: string | undefined
  targetType: string | undefined
  startTime: string | undefined
  endTime: string | undefined
}

// ============================================================================
// Constants
// ============================================================================

const PAGE_SIZE = 50
const AUTO_REFRESH_INTERVAL_MS = 30000

// Audit action types based on backend enum
const AUDIT_ACTIONS = [
  'TENANT_CREATE',
  'TENANT_UPDATE',
  'TENANT_DELETE',
  'TENANT_SUSPEND',
  'TENANT_ACTIVATE',
  'SUBSCRIPTION_CHANGE',
  'QUOTA_UPDATE',
  'USER_CREATE',
  'USER_UPDATE',
  'USER_DELETE',
  'USER_SUSPEND',
  'USER_ACTIVATE',
  'ROLE_CREATE',
  'ROLE_UPDATE',
  'ROLE_DELETE',
  'PERMISSION_GRANT',
  'PERMISSION_REVOKE',
  'SYSTEM_CONFIG_CHANGE',
] as const

// Target types based on backend enum
const TARGET_TYPES = [
  'tenant',
  'user',
  'role',
  'permission',
  'subscription',
  'quota',
  'system_config',
] as const

// Action type color mapping
const ACTION_COLOR_MAP: Record<string, TagColor> = {
  TENANT_CREATE: 'green',
  TENANT_UPDATE: 'blue',
  TENANT_DELETE: 'red',
  TENANT_SUSPEND: 'orange',
  TENANT_ACTIVATE: 'green',
  SUBSCRIPTION_CHANGE: 'purple',
  QUOTA_UPDATE: 'cyan',
  USER_CREATE: 'green',
  USER_UPDATE: 'blue',
  USER_DELETE: 'red',
  USER_SUSPEND: 'orange',
  USER_ACTIVATE: 'green',
  ROLE_CREATE: 'green',
  ROLE_UPDATE: 'blue',
  ROLE_DELETE: 'red',
  PERMISSION_GRANT: 'lime',
  PERMISSION_REVOKE: 'pink',
  SYSTEM_CONFIG_CHANGE: 'amber',
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Format action name for display
 */
function formatActionName(action: string): string {
  return action
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(' ')
}

/**
 * Format target type for display
 */
function formatTargetType(targetType: string): string {
  return targetType
    .split('_')
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1).toLowerCase())
    .join(' ')
}

/**
 * Generate CSV content from audit logs
 */
function generateCSV(logs: AdminAuditLogEntry[], formatDateTime: (date: string) => string): string {
  const headers = [
    'ID',
    'Timestamp',
    'Admin User ID',
    'Action',
    'Target Type',
    'Target ID',
    'IP Address',
    'User Agent',
    'Old Value',
    'New Value',
  ]

  const rows = logs.map((log) => [
    log.id,
    formatDateTime(log.created_at),
    log.admin_user_id,
    log.action,
    log.target_type,
    log.target_id ?? '',
    log.ip_address ?? '',
    log.user_agent ?? '',
    log.old_value ? JSON.stringify(log.old_value) : '',
    log.new_value ? JSON.stringify(log.new_value) : '',
  ])

  const csvContent = [
    headers.join(','),
    ...rows.map((row) => row.map((cell) => `"${String(cell).replace(/"/g, '""')}"`).join(',')),
  ].join('\n')

  return csvContent
}

/**
 * Download CSV file
 */
function downloadCSV(content: string, filename: string): void {
  const blob = new Blob(['\ufeff' + content], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = filename
  link.click()
  URL.revokeObjectURL(link.href)
}

// ============================================================================
// Value Diff Component
// ============================================================================

interface ValueDiffProps {
  oldValue?: Record<string, unknown>
  newValue?: Record<string, unknown>
}

function ValueDiff({ oldValue, newValue }: ValueDiffProps) {
  const { t } = useTranslation()

  // Collect all keys from both objects
  const allKeys = useMemo(() => {
    const keys = new Set<string>()
    if (oldValue) Object.keys(oldValue).forEach((k) => keys.add(k))
    if (newValue) Object.keys(newValue).forEach((k) => keys.add(k))
    return Array.from(keys).sort()
  }, [oldValue, newValue])

  if (allKeys.length === 0) {
    return (
      <Empty
        image={<IconFilter size="extra-large" />}
        description={t('admin.audit.noChanges', 'No detailed changes recorded')}
      />
    )
  }

  return (
    <div style={{ display: 'flex', gap: 24 }}>
      {/* Old Value */}
      <Card
        title={
          <Text type="danger" strong>
            {t('admin.audit.oldValue', 'Old Value')}
          </Text>
        }
        style={{ flex: 1 }}
        bodyStyle={{ padding: 12 }}
      >
        {oldValue ? (
          <Descriptions
            size="small"
            row
            data={allKeys.map((key) => ({
              key,
              label: key,
              value: (
                <Text
                  style={{
                    color:
                      JSON.stringify(oldValue[key]) !== JSON.stringify(newValue?.[key])
                        ? 'var(--semi-color-danger)'
                        : undefined,
                  }}
                >
                  {formatValue(oldValue[key])}
                </Text>
              ),
            }))}
          />
        ) : (
          <Text type="tertiary">{t('admin.audit.noOldValue', '(No previous value)')}</Text>
        )}
      </Card>

      {/* New Value */}
      <Card
        title={
          <Text type="success" strong>
            {t('admin.audit.newValue', 'New Value')}
          </Text>
        }
        style={{ flex: 1 }}
        bodyStyle={{ padding: 12 }}
      >
        {newValue ? (
          <Descriptions
            size="small"
            row
            data={allKeys.map((key) => ({
              key,
              label: key,
              value: (
                <Text
                  style={{
                    color:
                      JSON.stringify(newValue[key]) !== JSON.stringify(oldValue?.[key])
                        ? 'var(--semi-color-success)'
                        : undefined,
                  }}
                >
                  {formatValue(newValue[key])}
                </Text>
              ),
            }))}
          />
        ) : (
          <Text type="tertiary">{t('admin.audit.noNewValue', '(No new value)')}</Text>
        )}
      </Card>
    </div>
  )
}

/**
 * Format a value for display
 */
function formatValue(value: unknown): string {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'boolean') return value ? 'true' : 'false'
  if (typeof value === 'object') return JSON.stringify(value, null, 2)
  return String(value)
}

// ============================================================================
// Expanded Row Component
// ============================================================================

interface ExpandedRowProps {
  record: AdminAuditLogEntry
}

function ExpandedRow({ record }: ExpandedRowProps) {
  const { t } = useTranslation()

  return (
    <div style={{ padding: '12px 0' }}>
      <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
        {/* Metadata */}
        <Descriptions
          size="small"
          data={[
            {
              key: t('admin.audit.ipAddress', 'IP Address'),
              value: record.ip_address || '-',
            },
            {
              key: t('admin.audit.userAgent', 'User Agent'),
              value: (
                <Tooltip content={record.user_agent}>
                  <Text ellipsis={{ showTooltip: false }} style={{ maxWidth: 400 }}>
                    {record.user_agent || '-'}
                  </Text>
                </Tooltip>
              ),
            },
          ]}
        />

        {/* Value Diff */}
        <div style={{ width: '100%' }}>
          <Text strong style={{ marginBottom: 8, display: 'block' }}>
            {t('admin.audit.changes', 'Changes')}
          </Text>
          <ValueDiff oldValue={record.old_value} newValue={record.new_value} />
        </div>
      </Space>
    </div>
  )
}

// ============================================================================
// Main Component
// ============================================================================

export default function AuditLogsPage() {
  const { t } = useTranslation()
  const { formatDateTime } = useFormatters()
  const queryClient = useQueryClient()

  // ============================================================================
  // State
  // ============================================================================

  const [page, setPage] = useState(1)
  const [filters, setFilters] = useState<FilterState>({
    search: '',
    action: undefined,
    targetType: undefined,
    startTime: undefined,
    endTime: undefined,
  })
  const [expandedRowKeys, setExpandedRowKeys] = useState<string[]>([])
  const [autoRefresh, setAutoRefresh] = useState(false)
  const autoRefreshIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  // ============================================================================
  // API Query
  // ============================================================================

  const queryParams: AdminAuditLogListParams = useMemo(
    () => ({
      page,
      page_size: PAGE_SIZE,
      action: filters.action,
      target_type: filters.targetType,
      target_id: filters.search || undefined,
      start_time: filters.startTime,
      end_time: filters.endTime,
      sort_by: 'created_at',
      sort_order: 'desc',
    }),
    [page, filters]
  )

  const queryParamsRef = useRef(queryParams)

  // Keep ref in sync for auto-refresh closure
  useEffect(() => {
    queryParamsRef.current = queryParams
  }, [queryParams])

  const auditLogsQuery = useAdminAuditLogs(queryParams)

  // Extract data from query response
  const { logs, total, totalPages } = useMemo(() => {
    const resp = auditLogsQuery.data as unknown as {
      data?: { data?: { logs?: AdminAuditLogEntry[]; total?: number; total_pages?: number } }
    }
    return {
      logs: resp?.data?.data?.logs ?? [],
      total: resp?.data?.data?.total ?? 0,
      totalPages: resp?.data?.data?.total_pages ?? 0,
    }
  }, [auditLogsQuery.data])

  // ============================================================================
  // Auto-refresh Logic
  // ============================================================================

  useEffect(() => {
    if (autoRefresh) {
      autoRefreshIntervalRef.current = setInterval(() => {
        queryClient.invalidateQueries({
          queryKey: adminQueryKeys.auditLogs(queryParamsRef.current),
        })
      }, AUTO_REFRESH_INTERVAL_MS)
    } else if (autoRefreshIntervalRef.current) {
      clearInterval(autoRefreshIntervalRef.current)
      autoRefreshIntervalRef.current = null
    }

    return () => {
      if (autoRefreshIntervalRef.current) {
        clearInterval(autoRefreshIntervalRef.current)
        autoRefreshIntervalRef.current = null
      }
    }
  }, [autoRefresh, queryClient])

  // ============================================================================
  // Event Handlers
  // ============================================================================

  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: adminQueryKeys.auditLogs(queryParams) })
    Toast.success(t('common.refreshed', 'Refreshed'))
  }, [queryClient, queryParams, t])

  const handleSearchChange = useCallback((value: string) => {
    setFilters((prev) => ({ ...prev, search: value }))
    setPage(1)
  }, [])

  const handleActionChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      setFilters((prev) => ({
        ...prev,
        action: typeof value === 'string' ? value : undefined,
      }))
      setPage(1)
    },
    []
  )

  const handleTargetTypeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      setFilters((prev) => ({
        ...prev,
        targetType: typeof value === 'string' ? value : undefined,
      }))
      setPage(1)
    },
    []
  )

  const handleDateRangeChange = useCallback(
    (
      dates?: Date | Date[] | string | string[],
      _dateStrings?: string | string[] | Date | Date[]
    ) => {
      if (Array.isArray(dates) && dates.length === 2 && dates[0] && dates[1]) {
        const [start, end] = dates as [Date, Date]
        setFilters((prev) => ({
          ...prev,
          startTime: start.toISOString(),
          endTime: end.toISOString(),
        }))
      } else {
        setFilters((prev) => ({
          ...prev,
          startTime: undefined,
          endTime: undefined,
        }))
      }
      setPage(1)
    },
    []
  )

  const handleClearFilters = useCallback(() => {
    setFilters({
      search: '',
      action: undefined,
      targetType: undefined,
      startTime: undefined,
      endTime: undefined,
    })
    setPage(1)
  }, [])

  const handlePageChange = useCallback((currentPage: number) => {
    setPage(currentPage)
    setExpandedRowKeys([])
  }, [])

  const handleExpandRow: OnExpand<AdminAuditLogEntry> = useCallback(
    (expanded, record, _mouseEvent) => {
      if (!record || expanded === undefined || !('id' in record)) return
      setExpandedRowKeys((prev) =>
        expanded ? [...prev, record.id] : prev.filter((k) => k !== record.id)
      )
    },
    []
  )

  const handleExportCSV = useCallback(() => {
    if (logs.length === 0) {
      Toast.warning(t('admin.audit.noDataToExport', 'No data to export'))
      return
    }

    try {
      const csvContent = generateCSV(logs, formatDateTime)
      const filename = `audit-logs-${new Date().toISOString().slice(0, 10)}.csv`
      downloadCSV(csvContent, filename)
      Toast.success(t('admin.audit.exportSuccess', 'Export successful'))
    } catch {
      Toast.error(t('admin.audit.exportFailed', 'Failed to export CSV'))
    }
  }, [logs, formatDateTime, t])

  const handleAutoRefreshChange = useCallback((checked: boolean) => {
    setAutoRefresh(checked)
  }, [])

  // ============================================================================
  // Table Columns
  // ============================================================================

  const columns = useMemo(
    () => [
      {
        title: '',
        dataIndex: 'expand',
        width: 40,
        render: (_: unknown, record: AdminAuditLogEntry) => {
          const isExpanded = expandedRowKeys.includes(record.id)
          return (
            <Button
              type="tertiary"
              size="small"
              icon={isExpanded ? <IconChevronDown /> : <IconChevronRight />}
              onClick={() => handleExpandRow(!isExpanded, record)}
              aria-label={
                isExpanded ? t('common.collapse', 'Collapse') : t('common.expand', 'Expand')
              }
            />
          )
        },
      },
      {
        title: t('admin.audit.timestamp', 'Timestamp'),
        dataIndex: 'created_at',
        width: 180,
        render: (val: string) => (
          <Text size="small" style={{ whiteSpace: 'nowrap' }}>
            {formatDateTime(val)}
          </Text>
        ),
      },
      {
        title: t('admin.audit.action', 'Action'),
        dataIndex: 'action',
        width: 160,
        render: (action: string) => (
          <Tag color={ACTION_COLOR_MAP[action] ?? 'grey'} size="small">
            {formatActionName(action)}
          </Tag>
        ),
      },
      {
        title: t('admin.audit.operator', 'Operator'),
        dataIndex: 'admin_user_id',
        width: 280,
        render: (id: string) => (
          <Tooltip content={id}>
            <Text copyable ellipsis={{ showTooltip: false }} style={{ maxWidth: 250 }}>
              {id}
            </Text>
          </Tooltip>
        ),
      },
      {
        title: t('admin.audit.targetType', 'Target Type'),
        dataIndex: 'target_type',
        width: 120,
        render: (targetType: string) => (
          <Tag color="light-blue" size="small">
            {formatTargetType(targetType)}
          </Tag>
        ),
      },
      {
        title: t('admin.audit.targetId', 'Target ID'),
        dataIndex: 'target_id',
        render: (id: string | undefined) =>
          id ? (
            <Tooltip content={id}>
              <Text copyable ellipsis={{ showTooltip: false }} style={{ maxWidth: 200 }}>
                {id}
              </Text>
            </Tooltip>
          ) : (
            <Text type="tertiary">-</Text>
          ),
      },
    ],
    [t, formatDateTime, expandedRowKeys, handleExpandRow]
  )

  // ============================================================================
  // Filter Options
  // ============================================================================

  const actionOptions = useMemo(
    () =>
      AUDIT_ACTIONS.map((action) => ({
        value: action,
        label: formatActionName(action),
      })),
    []
  )

  const targetTypeOptions = useMemo(
    () =>
      TARGET_TYPES.map((type) => ({
        value: type,
        label: formatTargetType(type),
      })),
    []
  )

  const hasActiveFilters =
    filters.search || filters.action || filters.targetType || filters.startTime || filters.endTime

  // ============================================================================
  // Render
  // ============================================================================

  return (
    <Container>
      <Space vertical align="start" spacing="medium" style={{ width: '100%' }}>
        {/* Header */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            width: '100%',
            flexWrap: 'wrap',
            gap: 12,
          }}
        >
          <Title heading={4} style={{ margin: 0 }}>
            {t('admin.audit.title', 'Audit Logs')}
          </Title>
          <Space>
            <Checkbox
              checked={autoRefresh}
              onChange={(e) => handleAutoRefreshChange(e.target.checked ?? false)}
            >
              <Space>
                <IconClock />
                {t('admin.audit.autoRefresh', 'Auto-refresh (30s)')}
              </Space>
            </Checkbox>
            <Button
              icon={<IconRefresh />}
              onClick={handleRefresh}
              loading={auditLogsQuery.isFetching}
            >
              {t('common.refresh', 'Refresh')}
            </Button>
            <Button icon={<IconDownload />} onClick={handleExportCSV} disabled={logs.length === 0}>
              {t('admin.audit.export', 'Export CSV')}
            </Button>
          </Space>
        </div>

        {/* Filters */}
        <Card bodyStyle={{ padding: 16 }} style={{ width: '100%' }}>
          <Space wrap style={{ width: '100%' }}>
            <Input
              prefix={<IconSearch />}
              placeholder={t('admin.audit.searchPlaceholder', 'Search by Target ID...')}
              style={{ width: 280 }}
              showClear
              value={filters.search}
              onChange={handleSearchChange}
            />
            <Select
              placeholder={t('admin.audit.filterAction', 'Filter by action')}
              style={{ width: 200 }}
              optionList={actionOptions}
              value={filters.action}
              onChange={handleActionChange}
              showClear
            />
            <Select
              placeholder={t('admin.audit.filterTargetType', 'Filter by target type')}
              style={{ width: 180 }}
              optionList={targetTypeOptions}
              value={filters.targetType}
              onChange={handleTargetTypeChange}
              showClear
            />
            <DatePicker
              type="dateTimeRange"
              placeholder={[
                t('admin.audit.startTime', 'Start time'),
                t('admin.audit.endTime', 'End time'),
              ]}
              style={{ width: 360 }}
              onChange={handleDateRangeChange}
            />
            {hasActiveFilters && (
              <Button type="tertiary" onClick={handleClearFilters}>
                {t('admin.audit.clearFilters', 'Clear filters')}
              </Button>
            )}
          </Space>
        </Card>

        {/* Summary */}
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            width: '100%',
          }}
        >
          <Text type="secondary">
            {t('admin.audit.totalLogs', 'Total: {{count}} logs', { count: total })}
          </Text>
          {autoRefresh && (
            <Text type="tertiary" size="small">
              <IconClock style={{ marginRight: 4 }} />
              {t('admin.audit.autoRefreshEnabled', 'Auto-refreshing every 30 seconds')}
            </Text>
          )}
        </div>

        {/* Table */}
        <Card style={{ width: '100%' }} bodyStyle={{ padding: 0 }}>
          {auditLogsQuery.isLoading ? (
            <div
              style={{
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                minHeight: 400,
              }}
            >
              <Spin size="large" />
            </div>
          ) : auditLogsQuery.isError ? (
            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'center',
                alignItems: 'center',
                minHeight: 400,
                gap: 16,
              }}
            >
              <Empty description={t('admin.audit.loadError', 'Failed to load audit logs')} />
              <Button icon={<IconRefresh />} onClick={handleRefresh}>
                {t('common.retry', 'Retry')}
              </Button>
            </div>
          ) : logs.length === 0 ? (
            <div
              style={{
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
                minHeight: 400,
              }}
            >
              <Empty
                description={
                  hasActiveFilters
                    ? t('admin.audit.noResults', 'No audit logs match your filters')
                    : t('admin.audit.noLogs', 'No audit logs yet')
                }
              />
            </div>
          ) : (
            <Table
              columns={columns}
              dataSource={logs}
              rowKey="id"
              pagination={{
                currentPage: page,
                pageSize: PAGE_SIZE,
                total,
                onPageChange: handlePageChange,
                showSizeChanger: false,
                showTotal: true,
                formatPageText: (pageInfo) =>
                  t('admin.audit.pageInfo', '{{current}}/{{total}} pages', {
                    current: pageInfo?.currentStart ?? 0,
                    total: totalPages,
                  }),
              }}
              expandedRowRender={(record: AdminAuditLogEntry | undefined) =>
                record ? <ExpandedRow record={record} /> : null
              }
              expandedRowKeys={expandedRowKeys}
              onExpand={handleExpandRow}
              style={{ width: '100%' }}
              virtualized={{
                itemSize: 56,
              }}
            />
          )}
        </Card>
      </Space>
    </Container>
  )
}
