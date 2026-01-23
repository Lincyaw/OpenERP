import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Tag, Toast, Select, Space, Modal, Spin } from '@douyinfe/semi-ui'
import { IconPlus, IconRefresh, IconStar } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import {
  DataTable,
  TableToolbar,
  BulkActionBar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { getWarehouses } from '@/api/warehouses/warehouses'
import type {
  HandlerWarehouseListResponse,
  HandlerWarehouseListResponseStatus,
  HandlerWarehouseListResponseType,
  GetPartnerWarehousesParams,
  GetPartnerWarehousesStatus,
  GetPartnerWarehousesType,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './Warehouses.css'

const { Title } = Typography

// Warehouse type with index signature for DataTable compatibility
type Warehouse = HandlerWarehouseListResponse & Record<string, unknown>

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '启用', value: 'enabled' },
  { label: '停用', value: 'disabled' },
]

// Type options for filter
const TYPE_OPTIONS = [
  { label: '全部类型', value: '' },
  { label: '普通仓库', value: 'normal' },
  { label: '虚拟仓库', value: 'virtual' },
  { label: '中转仓库', value: 'transit' },
]

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerWarehouseListResponseStatus, 'green' | 'grey'> = {
  enabled: 'green',
  disabled: 'grey',
}

// Status labels
const STATUS_LABELS: Record<HandlerWarehouseListResponseStatus, string> = {
  enabled: '启用',
  disabled: '停用',
}

// Type tag color mapping
const TYPE_TAG_COLORS: Record<HandlerWarehouseListResponseType, 'blue' | 'purple' | 'cyan'> = {
  normal: 'blue',
  virtual: 'purple',
  transit: 'cyan',
}

// Type labels
const TYPE_LABELS: Record<HandlerWarehouseListResponseType, string> = {
  normal: '普通仓库',
  virtual: '虚拟仓库',
  transit: '中转仓库',
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
 * Warehouses list page
 *
 * Features:
 * - Warehouse listing with pagination
 * - Search by name, code
 * - Filter by status and type
 * - Enable/disable warehouse actions
 * - Set default warehouse
 * - Navigate to warehouse form for create/edit
 */
export default function WarehousesPage() {
  const navigate = useNavigate()
  const api = useMemo(() => getWarehouses(), [])

  // State for data
  const [warehouseList, setWarehouseList] = useState<Warehouse[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [typeFilter, setTypeFilter] = useState<string>('')

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'sort_order',
    defaultSortOrder: 'asc',
  })

  // Fetch warehouses
  const fetchWarehouses = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetPartnerWarehousesParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as GetPartnerWarehousesStatus | undefined,
        type: (typeFilter || undefined) as GetPartnerWarehousesType | undefined,
        order_by: state.sort.field || 'sort_order',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      const response = await api.getPartnerWarehouses(params)

      if (response.success && response.data) {
        setWarehouseList(response.data as Warehouse[])
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
      Toast.error('获取仓库列表失败')
    } finally {
      setLoading(false)
    }
  }, [
    api,
    state.pagination.page,
    state.pagination.pageSize,
    state.sort,
    searchKeyword,
    statusFilter,
    typeFilter,
  ])

  // Fetch on mount and when state changes
  useEffect(() => {
    fetchWarehouses()
  }, [fetchWarehouses])

  // Handle search
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
      // Reset to page 1 when searching
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
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

  // Handle type filter change
  const handleTypeChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const typeValue = typeof value === 'string' ? value : ''
      setTypeFilter(typeValue)
      setFilter('type', typeValue || null)
    },
    [setFilter]
  )

  // Handle enable warehouse
  const handleEnable = useCallback(
    async (warehouse: Warehouse) => {
      if (!warehouse.id) return
      try {
        await api.postPartnerWarehousesIdEnable(warehouse.id)
        Toast.success(`仓库 "${warehouse.name}" 已启用`)
        fetchWarehouses()
      } catch {
        Toast.error('启用仓库失败')
      }
    },
    [api, fetchWarehouses]
  )

  // Handle disable warehouse
  const handleDisable = useCallback(
    async (warehouse: Warehouse) => {
      if (!warehouse.id) return
      if (warehouse.is_default) {
        Toast.warning('无法停用默认仓库')
        return
      }
      try {
        await api.postPartnerWarehousesIdDisable(warehouse.id)
        Toast.success(`仓库 "${warehouse.name}" 已停用`)
        fetchWarehouses()
      } catch {
        Toast.error('停用仓库失败')
      }
    },
    [api, fetchWarehouses]
  )

  // Handle set default warehouse
  const handleSetDefault = useCallback(
    async (warehouse: Warehouse) => {
      if (!warehouse.id) return
      if (warehouse.is_default) {
        Toast.info('该仓库已是默认仓库')
        return
      }
      Modal.confirm({
        title: '设为默认仓库',
        content: `确定要将 "${warehouse.name}" 设为默认仓库吗？原默认仓库将被取消。`,
        okText: '确认',
        cancelText: '取消',
        onOk: async () => {
          try {
            await api.postPartnerWarehousesIdSetDefault(warehouse.id!)
            Toast.success(`已将 "${warehouse.name}" 设为默认仓库`)
            fetchWarehouses()
          } catch {
            Toast.error('设置默认仓库失败')
          }
        },
      })
    },
    [api, fetchWarehouses]
  )

  // Handle delete warehouse
  const handleDelete = useCallback(
    async (warehouse: Warehouse) => {
      if (!warehouse.id) return
      if (warehouse.is_default) {
        Toast.warning('无法删除默认仓库')
        return
      }
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除仓库 "${warehouse.name}" 吗？删除后无法恢复。`,
        okText: '确认删除',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await api.deletePartnerWarehousesId(warehouse.id!)
            Toast.success(`仓库 "${warehouse.name}" 已删除`)
            fetchWarehouses()
          } catch {
            Toast.error('删除仓库失败，该仓库可能有库存记录')
          }
        },
      })
    },
    [api, fetchWarehouses]
  )

  // Handle edit warehouse
  const handleEdit = useCallback(
    (warehouse: Warehouse) => {
      if (warehouse.id) {
        navigate(`/partner/warehouses/${warehouse.id}/edit`)
      }
    },
    [navigate]
  )

  // Handle view warehouse
  const handleView = useCallback(
    (warehouse: Warehouse) => {
      if (warehouse.id) {
        navigate(`/partner/warehouses/${warehouse.id}`)
      }
    },
    [navigate]
  )

  // Handle bulk enable
  const handleBulkEnable = useCallback(async () => {
    try {
      await Promise.all(selectedRowKeys.map((id) => api.postPartnerWarehousesIdEnable(id)))
      Toast.success(`已启用 ${selectedRowKeys.length} 个仓库`)
      setSelectedRowKeys([])
      fetchWarehouses()
    } catch {
      Toast.error('批量启用失败')
    }
  }, [api, selectedRowKeys, fetchWarehouses])

  // Handle bulk disable
  const handleBulkDisable = useCallback(async () => {
    // Check if any selected warehouse is default
    const hasDefault = warehouseList.some(
      (w) => selectedRowKeys.includes(w.id || '') && w.is_default
    )
    if (hasDefault) {
      Toast.warning('选中的仓库中包含默认仓库，无法批量停用')
      return
    }
    try {
      await Promise.all(selectedRowKeys.map((id) => api.postPartnerWarehousesIdDisable(id)))
      Toast.success(`已停用 ${selectedRowKeys.length} 个仓库`)
      setSelectedRowKeys([])
      fetchWarehouses()
    } catch {
      Toast.error('批量停用失败')
    }
  }, [api, selectedRowKeys, fetchWarehouses, warehouseList])

  // Table columns
  const tableColumns: DataTableColumn<Warehouse>[] = useMemo(
    () => [
      {
        title: '仓库编码',
        dataIndex: 'code',
        width: 120,
        sortable: true,
        render: (code: unknown) => (
          <span className="warehouse-code">{(code as string) || '-'}</span>
        ),
      },
      {
        title: '仓库名称',
        dataIndex: 'name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown, record: Warehouse) => (
          <div className="warehouse-name-cell">
            <span className="warehouse-name">
              {(name as string) || '-'}
              {record.is_default && (
                <Tag className="default-tag" color="light-blue" size="small">
                  <IconStar size="small" /> 默认
                </Tag>
              )}
            </span>
            {record.short_name && <span className="warehouse-short-name">{record.short_name}</span>}
          </div>
        ),
      },
      {
        title: '类型',
        dataIndex: 'type',
        width: 100,
        align: 'center',
        render: (type: unknown) => {
          const typeValue = type as HandlerWarehouseListResponseType | undefined
          if (!typeValue) return '-'
          return (
            <Tag className="type-tag" color={TYPE_TAG_COLORS[typeValue]}>
              {TYPE_LABELS[typeValue]}
            </Tag>
          )
        },
      },
      {
        title: '地区',
        dataIndex: 'city',
        width: 140,
        render: (_city: unknown, record: Warehouse) => (
          <span className="warehouse-location-cell">
            {record.province || record.city
              ? `${record.province || ''}${record.city ? ` ${record.city}` : ''}`
              : '-'}
          </span>
        ),
      },
      {
        title: '排序',
        dataIndex: 'sort_order',
        width: 80,
        align: 'center',
        sortable: true,
        render: (sortOrder: unknown) => {
          const order = sortOrder as number | undefined
          return <span className="warehouse-sort-order">{order ?? 0}</span>
        },
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as HandlerWarehouseListResponseStatus | undefined
          if (!statusValue) return '-'
          return <Tag color={STATUS_TAG_COLORS[statusValue]}>{STATUS_LABELS[statusValue]}</Tag>
        },
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        width: 120,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
    ],
    []
  )

  // Table row actions
  const tableActions: TableAction<Warehouse>[] = useMemo(
    () => [
      {
        key: 'view',
        label: '查看',
        onClick: handleView,
      },
      {
        key: 'edit',
        label: '编辑',
        onClick: handleEdit,
      },
      {
        key: 'setDefault',
        label: '设为默认',
        type: 'primary',
        onClick: handleSetDefault,
        hidden: (record) => !!record.is_default,
      },
      {
        key: 'enable',
        label: '启用',
        type: 'primary',
        onClick: handleEnable,
        hidden: (record) => record.status === 'enabled',
      },
      {
        key: 'disable',
        label: '停用',
        type: 'warning',
        onClick: handleDisable,
        hidden: (record) => record.status !== 'enabled' || !!record.is_default,
      },
      {
        key: 'delete',
        label: '删除',
        type: 'danger',
        onClick: handleDelete,
        hidden: (record) => !!record.is_default,
      },
    ],
    [handleView, handleEdit, handleSetDefault, handleEnable, handleDisable, handleDelete]
  )

  // Row selection handler
  const onSelectionChange = useCallback((keys: string[], _rows: Warehouse[]) => {
    setSelectedRowKeys(keys)
  }, [])

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchWarehouses()
  }, [fetchWarehouses])

  return (
    <Container size="full" className="warehouses-page">
      <Card className="warehouses-card">
        <div className="warehouses-header">
          <Title heading={4} style={{ margin: 0 }}>
            仓库管理
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索仓库名称、编码..."
          primaryAction={{
            label: '新增仓库',
            icon: <IconPlus />,
            onClick: () => navigate('/partner/warehouses/new'),
          }}
          secondaryActions={[
            {
              key: 'refresh',
              label: '刷新',
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="warehouses-filter-container">
              <Select
                placeholder="状态筛选"
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="类型筛选"
                value={typeFilter}
                onChange={handleTypeChange}
                optionList={TYPE_OPTIONS}
                style={{ width: 140 }}
              />
            </Space>
          }
        />

        {selectedRowKeys.length > 0 && (
          <BulkActionBar
            selectedCount={selectedRowKeys.length}
            onCancel={() => setSelectedRowKeys([])}
          >
            <Tag color="blue" onClick={handleBulkEnable} style={{ cursor: 'pointer' }}>
              批量启用
            </Tag>
            <Tag color="orange" onClick={handleBulkDisable} style={{ cursor: 'pointer' }}>
              批量停用
            </Tag>
          </BulkActionBar>
        )}

        <Spin spinning={loading}>
          <DataTable<Warehouse>
            data={warehouseList}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            rowSelection={{
              selectedRowKeys,
              onChange: onSelectionChange,
            }}
            scroll={{ x: 900 }}
          />
        </Spin>
      </Card>
    </Container>
  )
}
