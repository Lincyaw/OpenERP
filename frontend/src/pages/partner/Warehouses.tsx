import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Tag, Toast, Select, Space, Modal, Spin } from '@douyinfe/semi-ui-19'
import { IconPlus, IconRefresh, IconStar } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  DataTable,
  TableToolbar,
  BulkActionBar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { useFormatters } from '@/hooks/useFormatters'
import {
  listWarehouses,
  enableWarehouse,
  disableWarehouse,
  setDefaultWarehouse,
  deleteWarehouse,
} from '@/api/warehouses/warehouses'
import type {
  HandlerWarehouseListResponse,
  HandlerWarehouseListResponseStatus,
  HandlerWarehouseListResponseType,
  ListWarehousesParams,
  ListWarehousesStatus,
  ListWarehousesType,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './Warehouses.css'

const { Title } = Typography

// Warehouse type with index signature for DataTable compatibility
type Warehouse = HandlerWarehouseListResponse & Record<string, unknown>

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerWarehouseListResponseStatus, 'green' | 'grey'> = {
  enabled: 'green',
  disabled: 'grey',
}

// Type tag color mapping
const TYPE_TAG_COLORS: Record<HandlerWarehouseListResponseType, 'blue' | 'purple' | 'cyan'> = {
  normal: 'blue',
  virtual: 'purple',
  transit: 'cyan',
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
  const { t } = useTranslation(['partner', 'common'])
  const { formatDate } = useFormatters()

  // Memoized options with translations
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('warehouses.allStatus'), value: '' },
      { label: t('warehouses.status.enabled'), value: 'enabled' },
      { label: t('warehouses.status.disabled'), value: 'disabled' },
    ],
    [t]
  )

  const TYPE_OPTIONS = useMemo(
    () => [
      { label: t('warehouses.allTypes'), value: '' },
      { label: t('warehouses.type.normal'), value: 'normal' },
      { label: t('warehouses.type.virtual'), value: 'virtual' },
      { label: t('warehouses.type.transit'), value: 'transit' },
    ],
    [t]
  )

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
      const params: ListWarehousesParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as ListWarehousesStatus | undefined,
        type: (typeFilter || undefined) as ListWarehousesType | undefined,
        order_by: state.sort.field || 'sort_order',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      const response = await listWarehouses(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setWarehouseList(response.data.data as Warehouse[])
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
      Toast.error(t('warehouses.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    t,
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
        await enableWarehouse(warehouse.id, {})
        Toast.success(t('warehouses.messages.enableSuccess', { name: warehouse.name }))
        fetchWarehouses()
      } catch {
        Toast.error(t('warehouses.messages.enableError'))
      }
    },
    [fetchWarehouses, t]
  )

  // Handle disable warehouse
  const handleDisable = useCallback(
    async (warehouse: Warehouse) => {
      if (!warehouse.id) return
      if (warehouse.is_default) {
        Toast.warning(t('warehouses.messages.disableDefaultWarning'))
        return
      }
      try {
        await disableWarehouse(warehouse.id, {})
        Toast.success(t('warehouses.messages.disableSuccess', { name: warehouse.name }))
        fetchWarehouses()
      } catch {
        Toast.error(t('warehouses.messages.disableError'))
      }
    },
    [fetchWarehouses, t]
  )

  // Handle set default warehouse
  const handleSetDefault = useCallback(
    async (warehouse: Warehouse) => {
      if (!warehouse.id) return
      if (warehouse.is_default) {
        Toast.info(t('warehouses.messages.alreadyDefault'))
        return
      }
      Modal.confirm({
        title: t('warehouses.confirm.setDefaultTitle'),
        content: t('warehouses.confirm.setDefaultContent', { name: warehouse.name }),
        okText: t('warehouses.confirm.setDefaultOk'),
        cancelText: t('common:actions.cancel'),
        onOk: async () => {
          try {
            await setDefaultWarehouse(warehouse.id!, {})
            Toast.success(t('warehouses.messages.setDefaultSuccess', { name: warehouse.name }))
            fetchWarehouses()
          } catch {
            Toast.error(t('warehouses.messages.setDefaultError'))
          }
        },
      })
    },
    [fetchWarehouses, t]
  )

  // Handle delete warehouse
  const handleDelete = useCallback(
    async (warehouse: Warehouse) => {
      if (!warehouse.id) return
      if (warehouse.is_default) {
        Toast.warning(t('warehouses.messages.deleteDefaultWarning'))
        return
      }
      Modal.confirm({
        title: t('warehouses.confirm.deleteTitle'),
        content: t('warehouses.confirm.deleteContent', { name: warehouse.name }),
        okText: t('warehouses.confirm.deleteOk'),
        cancelText: t('common:actions.cancel'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await deleteWarehouse(warehouse.id!)
            Toast.success(t('warehouses.messages.deleteSuccess', { name: warehouse.name }))
            fetchWarehouses()
          } catch {
            Toast.error(t('warehouses.messages.deleteError'))
          }
        },
      })
    },
    [fetchWarehouses, t]
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
      await Promise.all(selectedRowKeys.map((id) => enableWarehouse(id, {})))
      Toast.success(t('warehouses.messages.batchEnableSuccess', { count: selectedRowKeys.length }))
      setSelectedRowKeys([])
      fetchWarehouses()
    } catch {
      Toast.error(t('warehouses.messages.batchEnableError'))
    }
  }, [selectedRowKeys, fetchWarehouses, t])

  // Handle bulk disable
  const handleBulkDisable = useCallback(async () => {
    // Check if any selected warehouse is default
    const hasDefault = warehouseList.some(
      (w) => selectedRowKeys.includes(w.id || '') && w.is_default
    )
    if (hasDefault) {
      Toast.warning(t('warehouses.messages.batchDisableDefaultWarning'))
      return
    }
    try {
      await Promise.all(selectedRowKeys.map((id) => disableWarehouse(id, {})))
      Toast.success(t('warehouses.messages.batchDisableSuccess', { count: selectedRowKeys.length }))
      setSelectedRowKeys([])
      fetchWarehouses()
    } catch {
      Toast.error(t('warehouses.messages.batchDisableError'))
    }
  }, [selectedRowKeys, fetchWarehouses, warehouseList, t])

  // Table columns
  const tableColumns: DataTableColumn<Warehouse>[] = useMemo(
    () => [
      {
        title: t('warehouses.columns.code'),
        dataIndex: 'code',
        width: 120,
        sortable: true,
        render: (code: unknown) => (
          <span className="warehouse-code">{(code as string) || '-'}</span>
        ),
      },
      {
        title: t('warehouses.columns.name'),
        dataIndex: 'name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown, record: Warehouse) => (
          <div className="warehouse-name-cell">
            <span className="warehouse-name">
              <span
                className="table-cell-link"
                onClick={() => {
                  if (record.id) navigate(`/partner/warehouses/${record.id}`)
                }}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault()
                    if (record.id) navigate(`/partner/warehouses/${record.id}`)
                  }
                }}
                role="link"
                tabIndex={0}
              >
                {(name as string) || '-'}
              </span>
              {record.is_default && (
                <Tag className="default-tag" color="light-blue" size="small">
                  <IconStar size="small" /> {t('warehouses.defaultTag')}
                </Tag>
              )}
            </span>
            {record.short_name && <span className="warehouse-short-name">{record.short_name}</span>}
          </div>
        ),
      },
      {
        title: t('warehouses.columns.type'),
        dataIndex: 'type',
        width: 100,
        align: 'center',
        render: (type: unknown) => {
          const typeValue = type as HandlerWarehouseListResponseType | undefined
          if (!typeValue) return '-'
          return (
            <Tag className="type-tag" color={TYPE_TAG_COLORS[typeValue]}>
              {t(`warehouses.type.${typeValue}`)}
            </Tag>
          )
        },
      },
      {
        title: t('warehouses.columns.region'),
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
        title: t('warehouses.columns.sortOrder'),
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
        title: t('warehouses.columns.status'),
        dataIndex: 'status',
        width: 90,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as HandlerWarehouseListResponseStatus | undefined
          if (!statusValue) return '-'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>
              {t(`warehouses.status.${statusValue}`)}
            </Tag>
          )
        },
      },
      {
        title: t('warehouses.columns.createdAt'),
        dataIndex: 'created_at',
        width: 120,
        sortable: true,
        render: (date: unknown) => {
          const dateStr = date as string | undefined
          return dateStr ? formatDate(dateStr) : '-'
        },
      },
    ],
    [t, formatDate, navigate]
  )

  // Table row actions
  const tableActions: TableAction<Warehouse>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('warehouses.actions.view'),
        onClick: handleView,
      },
      {
        key: 'edit',
        label: t('warehouses.actions.edit'),
        onClick: handleEdit,
      },
      {
        key: 'setDefault',
        label: t('warehouses.actions.setDefault'),
        type: 'primary',
        onClick: handleSetDefault,
        hidden: (record) => !!record.is_default,
      },
      {
        key: 'enable',
        label: t('warehouses.actions.enable'),
        type: 'primary',
        onClick: handleEnable,
        hidden: (record) => record.status === 'enabled',
      },
      {
        key: 'disable',
        label: t('warehouses.actions.disable'),
        type: 'warning',
        onClick: handleDisable,
        hidden: (record) => record.status !== 'enabled' || !!record.is_default,
      },
      {
        key: 'delete',
        label: t('warehouses.actions.delete'),
        type: 'danger',
        onClick: handleDelete,
        hidden: (record) => !!record.is_default,
      },
    ],
    [t, handleView, handleEdit, handleSetDefault, handleEnable, handleDisable, handleDelete]
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
            {t('warehouses.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('warehouses.searchPlaceholder')}
          primaryAction={{
            label: t('warehouses.addWarehouse'),
            icon: <IconPlus />,
            onClick: () => navigate('/partner/warehouses/new'),
          }}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('common:actions.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="warehouses-filter-container">
              <Select
                placeholder={t('warehouses.statusFilter')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('warehouses.typeFilter')}
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
              {t('warehouses.actions.batchEnable')}
            </Tag>
            <Tag color="orange" onClick={handleBulkDisable} style={{ cursor: 'pointer' }}>
              {t('warehouses.actions.batchDisable')}
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
