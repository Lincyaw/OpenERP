import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Tag, Toast, Select, Space, Spin, Button } from '@douyinfe/semi-ui'
import { IconRefresh, IconPlus } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { getStockTaking } from '@/api/stock-taking/stock-taking'
import { getWarehouses } from '@/api/warehouses/warehouses'
import type {
  HandlerStockTakingListResponse,
  GetInventoryStockTakingsParams,
  GetInventoryStockTakingsOrderDir,
  GetInventoryStockTakingsOrderBy,
  HandlerWarehouseListResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './StockTakingList.css'

const { Title } = Typography

// Stock taking type with index signature for DataTable compatibility
type StockTakingItem = HandlerStockTakingListResponse & Record<string, unknown>

// Warehouse option type
type WarehouseOption = {
  label: string
  value: string
}

import type { TagProps } from '@douyinfe/semi-ui/lib/es/tag'

type TagColor = TagProps['color']

// Status filter options
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '草稿', value: 'DRAFT' },
  { label: '盘点中', value: 'COUNTING' },
  { label: '待审批', value: 'PENDING_APPROVAL' },
  { label: '已通过', value: 'APPROVED' },
  { label: '已拒绝', value: 'REJECTED' },
  { label: '已取消', value: 'CANCELLED' },
]

// Status colors
const STATUS_COLORS: Record<string, TagColor> = {
  DRAFT: 'grey',
  COUNTING: 'blue',
  PENDING_APPROVAL: 'orange',
  APPROVED: 'green',
  REJECTED: 'red',
  CANCELLED: 'grey',
}

// Status labels
const STATUS_LABELS: Record<string, string> = {
  DRAFT: '草稿',
  COUNTING: '盘点中',
  PENDING_APPROVAL: '待审批',
  APPROVED: '已通过',
  REJECTED: '已拒绝',
  CANCELLED: '已取消',
}

/**
 * Format date for display
 */
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
}

/**
 * Format datetime for display
 */
function formatDateTime(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/**
 * Format currency value
 */
function formatCurrency(value?: number): string {
  if (value === undefined || value === null) return '-'
  return `¥${value.toFixed(2)}`
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
  const stockTakingApi = useMemo(() => getStockTaking(), [])
  const warehousesApi = useMemo(() => getWarehouses(), [])

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
      const response = await warehousesApi.getPartnerWarehouses({
        page_size: 100,
        status: 'active',
      })
      if (response.success && response.data) {
        const warehouses = response.data as HandlerWarehouseListResponse[]
        const options: WarehouseOption[] = [
          { label: '全部仓库', value: '' },
          ...warehouses.map((w) => ({
            label: w.name || w.code || '',
            value: w.id || '',
          })),
        ]
        setWarehouseOptions(options)
      }
    } catch {
      console.error('Failed to fetch warehouses')
    }
  }, [warehousesApi])

  // Fetch stock takings
  const fetchStockTakings = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetInventoryStockTakingsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        warehouse_id: warehouseFilter || undefined,
        status: (statusFilter as GetInventoryStockTakingsParams['status']) || undefined,
        order_by: (state.sort.field || 'created_at') as GetInventoryStockTakingsOrderBy,
        order_dir: (state.sort.order === 'asc' ? 'asc' : 'desc') as GetInventoryStockTakingsOrderDir,
      }

      const response = await stockTakingApi.getInventoryStockTakings(params)

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
      Toast.error('获取盘点单列表失败')
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
        title: '盘点单号',
        dataIndex: 'taking_number',
        width: 160,
        render: (number: unknown) => (
          <span className="taking-number">{number as string}</span>
        ),
      },
      {
        title: '仓库',
        dataIndex: 'warehouse_name',
        width: 120,
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 100,
        render: (status: unknown) => {
          const statusStr = status as string
          return (
            <Tag color={STATUS_COLORS[statusStr] || 'grey'}>
              {STATUS_LABELS[statusStr] || statusStr}
            </Tag>
          )
        },
      },
      {
        title: '盘点日期',
        dataIndex: 'taking_date',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string),
      },
      {
        title: '进度',
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
        title: '差异金额',
        dataIndex: 'total_difference',
        width: 110,
        align: 'right',
        render: (value: unknown) => {
          const diff = value as number
          if (diff === undefined || diff === null || diff === 0) {
            return formatCurrency(0)
          }
          return (
            <span className={diff > 0 ? 'diff-positive' : 'diff-negative'}>
              {diff > 0 ? '+' : ''}{formatCurrency(diff)}
            </span>
          )
        },
      },
      {
        title: '创建人',
        dataIndex: 'created_by_name',
        width: 100,
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        width: 150,
        sortable: true,
        render: (date: unknown) => formatDateTime(date as string),
      },
    ],
    []
  )

  // Table row actions
  const tableActions: TableAction<StockTakingItem>[] = useMemo(
    () => [
      {
        key: 'view',
        label: '查看',
        onClick: handleViewDetail,
      },
      {
        key: 'execute',
        label: '执行',
        onClick: handleExecute,
        condition: (record: StockTakingItem) =>
          record.status === 'DRAFT' || record.status === 'COUNTING',
      },
    ],
    [handleViewDetail, handleExecute]
  )

  return (
    <Container size="full" className="stock-taking-list-page">
      <Card className="stock-taking-list-card">
        <div className="stock-taking-list-header">
          <Title heading={4} style={{ margin: 0 }}>
            盘点管理
          </Title>
          <Button
            type="primary"
            icon={<IconPlus />}
            onClick={handleCreate}
          >
            新建盘点
          </Button>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索盘点单号..."
          secondaryActions={[
            {
              key: 'refresh',
              label: '刷新',
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space>
              <Select
                placeholder="选择仓库"
                value={warehouseFilter}
                onChange={handleWarehouseChange}
                optionList={warehouseOptions}
                style={{ width: 150 }}
              />
              <Select
                placeholder="盘点状态"
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
