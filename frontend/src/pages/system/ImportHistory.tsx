import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Select,
  Space,
  Spin,
  Banner,
  Empty,
  Dropdown,
  Button,
} from '@douyinfe/semi-ui-19'
import { IconRefresh, IconDownload, IconChevronDown } from '@douyinfe/semi-icons'
import { useTranslation } from 'react-i18next'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { useFormatters } from '@/hooks/useFormatters'
import { axiosInstance } from '@/services/axios-instance'
import type { PaginationMeta } from '@/types/api'
import './ImportHistory.css'

const { Title } = Typography

// Import history record type
interface ImportHistoryRecord {
  id: string
  entity_type: string
  filename: string
  status: 'pending' | 'processing' | 'completed' | 'partial' | 'failed'
  total_rows: number
  success_rows: number
  error_rows: number
  created_by: string
  created_by_name?: string
  created_at: string
  duration_ms?: number
  error_report_url?: string
}

// Status tag color mapping
const STATUS_TAG_COLORS: Record<string, 'blue' | 'orange' | 'green' | 'yellow' | 'red'> = {
  pending: 'blue',
  processing: 'orange',
  completed: 'green',
  partial: 'yellow',
  failed: 'red',
}

// Entity type tag color mapping
const ENTITY_TYPE_COLORS: Record<string, string> = {
  products: 'cyan',
  customers: 'purple',
  suppliers: 'indigo',
  inventory: 'teal',
  categories: 'pink',
}

// Template URLs for each entity type
const TEMPLATE_URLS: Record<string, string> = {
  products: '/templates/products_import_template.csv',
  customers: '/templates/customers_import_template.csv',
  suppliers: '/templates/suppliers_import_template.csv',
  inventory: '/templates/inventory_import_template.csv',
}

/**
 * Import History Page
 *
 * Features:
 * - List all import operations with pagination
 * - Filter by entity type and status
 * - View import details
 * - Download error reports
 */
export default function ImportHistoryPage() {
  const { t } = useTranslation('common')
  const { formatDate } = useFormatters()

  // State for data
  const [historyList, setHistoryList] = useState<ImportHistoryRecord[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)

  // Filter state
  const [entityTypeFilter, setEntityTypeFilter] = useState<string>('')
  const [statusFilter, setStatusFilter] = useState<string>('')

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Filter options
  const ENTITY_TYPE_OPTIONS = useMemo(
    () => [
      { label: t('importHistory.filters.allEntities'), value: '' },
      { label: t('import.entityTypes.products'), value: 'products' },
      { label: t('import.entityTypes.customers'), value: 'customers' },
      { label: t('import.entityTypes.suppliers'), value: 'suppliers' },
      { label: t('import.entityTypes.inventory'), value: 'inventory' },
      { label: t('import.entityTypes.categories'), value: 'categories' },
    ],
    [t]
  )

  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('importHistory.filters.allStatus'), value: '' },
      { label: t('importHistory.status.pending'), value: 'pending' },
      { label: t('importHistory.status.processing'), value: 'processing' },
      { label: t('importHistory.status.completed'), value: 'completed' },
      { label: t('importHistory.status.partial'), value: 'partial' },
      { label: t('importHistory.status.failed'), value: 'failed' },
    ],
    [t]
  )

  // Fetch import history
  const fetchHistory = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, unknown> = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      if (entityTypeFilter) {
        params.entity_type = entityTypeFilter
      }
      if (statusFilter) {
        params.status = statusFilter
      }

      const response = await axiosInstance.get('/import/history', { params })

      if (response.data.success && response.data.data) {
        setHistoryList(response.data.data as ImportHistoryRecord[])
        if (response.data.meta) {
          setPaginationMeta({
            page: response.data.meta.page || 1,
            page_size: response.data.meta.page_size || 20,
            total: response.data.meta.total || 0,
            total_pages: response.data.meta.total_pages || 1,
          })
        }
      }
    } catch {
      Toast.error(t('importHistory.messages.fetchError'))
      // Set empty state on error
      setHistoryList([])
      setPaginationMeta(undefined)
    } finally {
      setLoading(false)
    }
  }, [
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    entityTypeFilter,
    statusFilter,
    t,
  ])

  // Fetch on mount and when state changes
  useEffect(() => {
    fetchHistory()
  }, [fetchHistory])

  // Handle entity type filter change
  const handleEntityTypeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const entityValue = typeof value === 'string' ? value : ''
      setEntityTypeFilter(entityValue)
      setFilter('entity_type', entityValue || null)
    },
    [setFilter]
  )

  // Handle status filter change
  const handleStatusChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const statusValue = typeof value === 'string' ? value : ''
      setStatusFilter(statusValue)
      setFilter('status', statusValue || null)
    },
    [setFilter]
  )

  // Handle download error report
  const handleDownloadErrors = useCallback(
    async (record: ImportHistoryRecord) => {
      if (!record.error_report_url) {
        Toast.warning(t('importHistory.messages.noHistory'))
        return
      }
      // Open error report URL in new tab
      window.open(record.error_report_url, '_blank')
    },
    [t]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchHistory()
  }, [fetchHistory])

  // Handle template download
  const handleDownloadTemplate = useCallback((entityType: string) => {
    const url = TEMPLATE_URLS[entityType]
    if (url) {
      window.open(url, '_blank')
    }
  }, [])

  // Format duration
  const formatDuration = useCallback((durationMs?: number): string => {
    if (!durationMs) return '-'
    if (durationMs < 1000) return `${durationMs}ms`
    if (durationMs < 60000) return `${(durationMs / 1000).toFixed(1)}s`
    return `${(durationMs / 60000).toFixed(1)}m`
  }, [])

  // Table columns
  const tableColumns: DataTableColumn<ImportHistoryRecord>[] = useMemo(
    () => [
      {
        title: t('importHistory.columns.entityType'),
        dataIndex: 'entity_type',
        width: 120,
        render: (entityType: unknown) => {
          const type = entityType as string | undefined
          if (!type) return '-'
          return (
            <Tag color={ENTITY_TYPE_COLORS[type] || 'grey'}>{t(`import.entityTypes.${type}`)}</Tag>
          )
        },
      },
      {
        title: t('importHistory.columns.filename'),
        dataIndex: 'filename',
        width: 180,
        ellipsis: true,
        render: (filename: unknown) => (
          <span className="import-filename">{(filename as string) || '-'}</span>
        ),
      },
      {
        title: t('importHistory.columns.status'),
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as string | undefined
          if (!statusValue) return '-'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>
              {t(`importHistory.status.${statusValue}`)}
            </Tag>
          )
        },
      },
      {
        title: t('importHistory.columns.totalRows'),
        dataIndex: 'total_rows',
        width: 90,
        align: 'right',
        render: (value: unknown) => (value as number)?.toLocaleString() || '0',
      },
      {
        title: t('importHistory.columns.successRows'),
        dataIndex: 'success_rows',
        width: 90,
        align: 'right',
        render: (value: unknown) => (
          <span className="success-count">{(value as number)?.toLocaleString() || '0'}</span>
        ),
      },
      {
        title: t('importHistory.columns.errorRows'),
        dataIndex: 'error_rows',
        width: 90,
        align: 'right',
        render: (value: unknown) => {
          const errors = value as number
          if (!errors || errors === 0) return '0'
          return <span className="error-count">{errors.toLocaleString()}</span>
        },
      },
      {
        title: t('importHistory.columns.createdBy'),
        dataIndex: 'created_by_name',
        width: 120,
        ellipsis: true,
        render: (name: unknown, record: ImportHistoryRecord) => (
          <span>{(name as string) || record.created_by || '-'}</span>
        ),
      },
      {
        title: t('importHistory.columns.createdAt'),
        dataIndex: 'created_at',
        width: 160,
        sortable: true,
        render: (date: unknown) => {
          const dateStr = date as string | undefined
          return dateStr ? formatDate(new Date(dateStr), 'medium') : '-'
        },
      },
      {
        title: t('importHistory.columns.duration'),
        dataIndex: 'duration_ms',
        width: 80,
        align: 'right',
        render: (duration: unknown) => formatDuration(duration as number | undefined),
      },
    ],
    [t, formatDate, formatDuration]
  )

  // Table row actions
  const tableActions: TableAction<ImportHistoryRecord>[] = useMemo(
    () => [
      {
        key: 'download-errors',
        label: t('importHistory.actions.downloadErrors'),
        icon: <IconDownload />,
        onClick: handleDownloadErrors,
        hidden: (record) => !record.error_rows || record.error_rows === 0,
      },
    ],
    [t, handleDownloadErrors]
  )

  // Empty state component
  const renderEmpty = () => (
    <Empty
      image={<Empty.IllustrationNoContent />}
      title={t('importHistory.empty.title')}
      description={t('importHistory.empty.description')}
    />
  )

  return (
    <Container size="full" className="import-history-page">
      <Banner
        type="info"
        description={t('importHistory.tip')}
        style={{ marginBottom: 'var(--spacing-4)' }}
      />
      <Card className="import-history-card">
        <div className="import-history-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('importHistory.title')}
          </Title>
        </div>

        <TableToolbar
          secondaryActions={[
            {
              key: 'refresh',
              label: t('actions.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space>
              <Select
                placeholder={t('importHistory.filters.allEntities')}
                value={entityTypeFilter}
                onChange={handleEntityTypeChange}
                optionList={ENTITY_TYPE_OPTIONS}
                style={{ width: 140 }}
              />
              <Select
                placeholder={t('importHistory.filters.allStatus')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 130 }}
              />
              <Dropdown
                trigger="click"
                position="bottomRight"
                render={
                  <Dropdown.Menu>
                    <Dropdown.Item onClick={() => handleDownloadTemplate('products')}>
                      <IconDownload style={{ marginRight: 8 }} />
                      {t('import.entityTypes.products')}
                    </Dropdown.Item>
                    <Dropdown.Item onClick={() => handleDownloadTemplate('customers')}>
                      <IconDownload style={{ marginRight: 8 }} />
                      {t('import.entityTypes.customers')}
                    </Dropdown.Item>
                    <Dropdown.Item onClick={() => handleDownloadTemplate('suppliers')}>
                      <IconDownload style={{ marginRight: 8 }} />
                      {t('import.entityTypes.suppliers')}
                    </Dropdown.Item>
                    <Dropdown.Item onClick={() => handleDownloadTemplate('inventory')}>
                      <IconDownload style={{ marginRight: 8 }} />
                      {t('import.entityTypes.inventory')}
                    </Dropdown.Item>
                  </Dropdown.Menu>
                }
              >
                <Button icon={<IconDownload />} iconPosition="left">
                  {t('importHistory.downloadTemplates')}
                  <IconChevronDown style={{ marginLeft: 4 }} />
                </Button>
              </Dropdown>
            </Space>
          }
        />

        <Spin spinning={loading}>
          {historyList.length === 0 && !loading ? (
            renderEmpty()
          ) : (
            <DataTable<ImportHistoryRecord>
              data={historyList}
              columns={tableColumns}
              rowKey="id"
              loading={loading}
              pagination={paginationMeta}
              actions={tableActions}
              onStateChange={handleStateChange}
              sortState={state.sort}
              scroll={{ x: 1100 }}
              mobileCardPrimaryColumns={['entity_type', 'filename']}
            />
          )}
        </Spin>
      </Card>
    </Container>
  )
}
