import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Tag, Toast, Select, Space, Spin, Button } from '@douyinfe/semi-ui-19'
import { IconRefresh, IconPlus } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
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
import { getStockTaking } from '@/api/stock-taking/stock-taking'
import { getWarehouses } from '@/api/warehouses/warehouses'
import type {
  HandlerStockTakingListResponse,
  ListStockTakingsParams,
  ListStockTakingsOrderDir,
  ListStockTakingsOrderBy,
  HandlerWarehouseListResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './StockTakingList.css'
import { createScopedLogger } from '@/utils'

const log = createScopedLogger('StockTakingList')

const { Title } = Typography

// Stock taking type with index signature for DataTable compatibility
type StockTakingItem = HandlerStockTakingListResponse & Record<string, unknown>

// Warehouse option type
type WarehouseOption = {
  label: string
  value: string
}

import type { TagProps } from '@douyinfe/semi-ui-19/lib/es/tag'

type TagColor = TagProps['color']

// Status colors
const STATUS_COLORS: Record<string, TagColor> = {
  DRAFT: 'grey',
  COUNTING: 'blue',
  PENDING_APPROVAL: 'orange',
  APPROVED: 'green',
  REJECTED: 'red',
  CANCELLED: 'grey',
}

/**
 * Stock Taking List Page
 *
 * Features:
 * - List all stock takings with pagination
 * - Filter by warehouse and status
 * - Navigate to create, detail, and execution pages
 * - Display progress and difference information
 */
export default function StockTakingListPage() {
  const navigate = useNavigate()
  const { t } = useTranslation(['inventory', 'common'])
  const { formatCurrency: formatCurrencyBase, formatDate: formatDateBase } = useFormatters()
  const stockTakingApi = useMemo(() => getStockTaking(), [])
  const warehousesApi = useMemo(() => getWarehouses(), [])

  // Wrapper functions to handle undefined values
  const formatCurrency = useCallback(
    (value?: number): string => (value !== undefined ? formatCurrencyBase(value) : '-'),
    [formatCurrencyBase]
  )
  const formatDate = useCallback(
    (date?: string, style?: 'date' | 'dateTime'): string =>
      date ? formatDateBase(date, style === 'dateTime' ? 'medium' : 'short') : '-',
    [formatDateBase]
  )

  // Status filter options
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('stockTaking.list.status.all'), value: '' },
      { label: t('stockTaking.list.status.DRAFT'), value: 'DRAFT' },
      { label: t('stockTaking.list.status.COUNTING'), value: 'COUNTING' },
      { label: t('stockTaking.list.status.PENDING_APPROVAL'), value: 'PENDING_APPROVAL' },
      { label: t('stockTaking.list.status.APPROVED'), value: 'APPROVED' },
      { label: t('stockTaking.list.status.REJECTED'), value: 'REJECTED' },
      { label: t('stockTaking.list.status.CANCELLED'), value: 'CANCELLED' },
    ],
    [t]
  )

  // State for data
  const [stockTakingList, setStockTakingList] = useState<StockTakingItem[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [warehouseFilter, setWarehouseFilter] = useState<string>('')
  const [statusFilter, setStatusFilter] = useState<string>('')

  // Warehouse options
  const [warehouseOptions, setWarehouseOptions] = useState<WarehouseOption[]>([])

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Fetch warehouses for filter dropdown
  const fetchWarehouses = useCallback(async () => {
    try {
      const response = await warehousesApi.listWarehouses({
        page_size: 100,
        status: 'enabled',
      })
      if (response.success && response.data) {
        const warehouses = response.data as HandlerWarehouseListResponse[]
        const options: WarehouseOption[] = [
          { label: t('stockTaking.list.allWarehouses'), value: '' },
          ...warehouses.map((w) => ({
            label: w.name || w.code || '',
            value: w.id || '',
          })),
        ]
        setWarehouseOptions(options)
      }
    } catch {
      log.error('Failed to fetch warehouses')
    }
  }, [warehousesApi, t])

  // Fetch stock takings
  const fetchStockTakings = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListStockTakingsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        warehouse_id: warehouseFilter || undefined,
        status: (statusFilter as ListStockTakingsParams['status']) || undefined,
        order_by: (state.sort.field || 'created_at') as ListStockTakingsOrderBy,
        order_dir: (state.sort.order === 'asc'
          ? 'asc'
          : 'desc') as ListStockTakingsOrderDir,
      }

      const response = await stockTakingApi.listStockTakings(params)

      if (response.success && response.data) {
        setStockTakingList(response.data as StockTakingItem[])
        if (response.meta) {
          setPaginationMeta({
            page: response.meta.page || 1,
            page_size: response.meta.page_size || 20,
            total: response.meta.total || 0,
            total_pages: response.meta.total_pages || 1,
          })
        }
      }
    } catch {
      Toast.error(t('stockTaking.list.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    stockTakingApi,
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    searchKeyword,
    warehouseFilter,
    statusFilter,
  ])

  // Fetch warehouses on mount
  useEffect(() => {
    fetchWarehouses()
  }, [fetchWarehouses])

  // Fetch stock takings on mount and when state changes
  useEffect(() => {
    fetchStockTakings()
  }, [fetchStockTakings])

  // Handle search
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle warehouse filter change
  const handleWarehouseChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const warehouseValue = typeof value === 'string' ? value : ''
      setWarehouseFilter(warehouseValue)
      setFilter('warehouse_id', warehouseValue || null)
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

  // Handle create new stock taking
  const handleCreate = useCallback(() => {
    navigate('/inventory/stock-taking/new')
  }, [navigate])

  // Handle view detail
  const handleViewDetail = useCallback(
    (item: StockTakingItem) => {
      if (item.id) {
        navigate(`/inventory/stock-taking/${item.id}`)
      }
    },
    [navigate]
  )

  // Handle execute stock taking
  const handleExecute = useCallback(
    (item: StockTakingItem) => {
      if (item.id) {
        navigate(`/inventory/stock-taking/${item.id}/execute`)
      }
    },
    [navigate]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchStockTakings()
  }, [fetchStockTakings])

  // Table columns
  const tableColumns: DataTableColumn<StockTakingItem>[] = useMemo(
    () => [
      {
        title: t('stockTaking.list.columns.takingNumber'),
        dataIndex: 'taking_number',
        width: 160,
        render: (number: unknown) => <span className="taking-number">{number as string}</span>,
      },
      {
        title: t('stockTaking.list.columns.warehouse'),
        dataIndex: 'warehouse_name',
        width: 120,
      },
      {
        title: t('stockTaking.list.columns.status'),
        dataIndex: 'status',
        width: 100,
        render: (status: unknown) => {
          const statusStr = status as string
          return (
            <Tag color={STATUS_COLORS[statusStr] || 'grey'}>
              {String(t(`stockTaking.list.status.${statusStr}`, { defaultValue: statusStr }))}
            </Tag>
          )
        },
      },
      {
        title: t('stockTaking.list.columns.takingDate'),
        dataIndex: 'taking_date',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined, 'date'),
      },
      {
        title: t('stockTaking.list.columns.progress'),
        dataIndex: 'progress',
        width: 100,
        align: 'right',
        render: (_: unknown, record: StockTakingItem) => {
          const counted = record.counted_items || 0
          const total = record.total_items || 0
          const percent = total > 0 ? ((counted / total) * 100).toFixed(0) : 0
          return `${counted}/${total} (${percent}%)`
        },
      },
      {
        title: t('stockTaking.list.columns.totalDifference'),
        dataIndex: 'total_difference',
        width: 110,
        align: 'right',
        render: (value: unknown) => {
          // Handle string values from API (decimal.Decimal serializes as string)
          const numValue = typeof value === 'string' ? parseFloat(value) : (value as number)
          if (numValue === undefined || numValue === null || isNaN(numValue) || numValue === 0) {
            return formatCurrency(0)
          }
          return (
            <span className={numValue > 0 ? 'diff-positive' : 'diff-negative'}>
              {numValue > 0 ? '+' : ''}
              {formatCurrency(numValue)}
            </span>
          )
        },
      },
      {
        title: t('stockTaking.list.columns.createdBy'),
        dataIndex: 'created_by_name',
        width: 100,
      },
      {
        title: t('stockTaking.list.columns.createdAt'),
        dataIndex: 'created_at',
        width: 150,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined, 'dateTime'),
      },
    ],
    [t, formatDate, formatCurrency]
  )

  // Table row actions
  const tableActions: TableAction<StockTakingItem>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('stockTaking.list.actions.view'),
        onClick: handleViewDetail,
      },
      {
        key: 'execute',
        label: t('stockTaking.list.actions.execute'),
        onClick: handleExecute,
        condition: (record: StockTakingItem) =>
          record.status === 'DRAFT' || record.status === 'COUNTING',
      },
    ],
    [handleViewDetail, handleExecute, t]
  )

  return (
    <Container size="full" className="stock-taking-list-page">
      <Card className="stock-taking-list-card">
        <div className="stock-taking-list-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('stockTaking.list.title')}
          </Title>
          <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
            {t('stockTaking.list.newStockTaking')}
          </Button>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('stockTaking.list.searchPlaceholder')}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('stockTaking.list.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space>
              <Select
                placeholder={t('stockTaking.list.selectWarehouse')}
                value={warehouseFilter}
                onChange={handleWarehouseChange}
                optionList={warehouseOptions}
                style={{ width: 150 }}
              />
              <Select
                placeholder={t('stockTaking.list.stockTakingStatus')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 130 }}
              />
            </Space>
          }
        />

        <Spin spinning={loading}>
          <DataTable<StockTakingItem>
            data={stockTakingList}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 1100 }}
          />
        </Spin>
      </Card>
    </Container>
  )
}
