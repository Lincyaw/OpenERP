import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Select,
  Space,
  Modal,
  Spin,
  Rating,
} from '@douyinfe/semi-ui-19'
import { IconPlus, IconRefresh } from '@douyinfe/semi-icons'
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
import { getSuppliers } from '@/api/suppliers/suppliers'
import type {
  HandlerSupplierListResponse,
  HandlerSupplierListResponseStatus,
  ListSuppliersParams,
  ListSuppliersStatus,
  ListSuppliersType,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './Suppliers.css'

const { Title } = Typography

// Supplier type with index signature for DataTable compatibility
type Supplier = HandlerSupplierListResponse & Record<string, unknown>

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerSupplierListResponseStatus, 'green' | 'grey' | 'red'> = {
  active: 'green',
  inactive: 'grey',
  blocked: 'red',
}

/**
 * Suppliers list page
 *
 * Features:
 * - Supplier listing with pagination
 * - Search by name, code, phone, email
 * - Filter by status and type
 * - Activate/deactivate/block supplier actions
 * - Navigate to supplier form for create/edit
 */
export default function SuppliersPage() {
  const navigate = useNavigate()
  const { t } = useTranslation(['partner', 'common'])
  const { formatDate } = useFormatters()
  const api = useMemo(() => getSuppliers(), [])

  // Memoized options with translations
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('suppliers.allStatus'), value: '' },
      { label: t('suppliers.status.active'), value: 'active' },
      { label: t('suppliers.status.inactive'), value: 'inactive' },
      { label: t('suppliers.status.blocked'), value: 'blocked' },
    ],
    [t]
  )

  const TYPE_OPTIONS = useMemo(
    () => [
      { label: t('suppliers.allTypes'), value: '' },
      { label: t('suppliers.type.manufacturer'), value: 'manufacturer' },
      { label: t('suppliers.type.distributor'), value: 'distributor' },
      { label: t('suppliers.type.retailer'), value: 'retailer' },
      { label: t('suppliers.type.service'), value: 'service' },
    ],
    [t]
  )

  // State for data
  const [supplierList, setSupplierList] = useState<Supplier[]>([])
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
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Fetch suppliers
  const fetchSuppliers = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListSuppliersParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as ListSuppliersStatus | undefined,
        type: (typeFilter || undefined) as ListSuppliersType | undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      const response = await api.listSuppliers(params)

      if (response.success && response.data) {
        setSupplierList(response.data as Supplier[])
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
      Toast.error(t('suppliers.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    api,
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
    fetchSuppliers()
  }, [fetchSuppliers])

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

  // Handle activate supplier
  const handleActivate = useCallback(
    async (supplier: Supplier) => {
      if (!supplier.id) return
      try {
        await api.activateSupplier(supplier.id)
        Toast.success(t('suppliers.messages.activateSuccess', { name: supplier.name }))
        fetchSuppliers()
      } catch {
        Toast.error(t('suppliers.messages.activateError'))
      }
    },
    [api, fetchSuppliers, t]
  )

  // Handle deactivate supplier
  const handleDeactivate = useCallback(
    async (supplier: Supplier) => {
      if (!supplier.id) return
      try {
        await api.deactivateSupplier(supplier.id)
        Toast.success(t('suppliers.messages.deactivateSuccess', { name: supplier.name }))
        fetchSuppliers()
      } catch {
        Toast.error(t('suppliers.messages.deactivateError'))
      }
    },
    [api, fetchSuppliers, t]
  )

  // Handle block supplier
  const handleBlock = useCallback(
    async (supplier: Supplier) => {
      if (!supplier.id) return
      Modal.confirm({
        title: t('suppliers.confirm.blockTitle'),
        content: t('suppliers.confirm.blockContent', { name: supplier.name }),
        okText: t('suppliers.confirm.blockOk'),
        cancelText: t('common:actions.cancel'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await api.blockSupplier(supplier.id!)
            Toast.success(t('suppliers.messages.blockSuccess', { name: supplier.name }))
            fetchSuppliers()
          } catch {
            Toast.error(t('suppliers.messages.blockError'))
          }
        },
      })
    },
    [api, fetchSuppliers, t]
  )

  // Handle delete supplier
  const handleDelete = useCallback(
    async (supplier: Supplier) => {
      if (!supplier.id) return
      Modal.confirm({
        title: t('suppliers.confirm.deleteTitle'),
        content: t('suppliers.confirm.deleteContent', { name: supplier.name }),
        okText: t('suppliers.confirm.deleteOk'),
        cancelText: t('common:actions.cancel'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await api.deleteSupplier(supplier.id!)
            Toast.success(t('suppliers.messages.deleteSuccess', { name: supplier.name }))
            fetchSuppliers()
          } catch {
            Toast.error(t('suppliers.messages.deleteError'))
          }
        },
      })
    },
    [api, fetchSuppliers, t]
  )

  // Handle edit supplier
  const handleEdit = useCallback(
    (supplier: Supplier) => {
      if (supplier.id) {
        navigate(`/partner/suppliers/${supplier.id}/edit`)
      }
    },
    [navigate]
  )

  // Handle view supplier
  const handleView = useCallback(
    (supplier: Supplier) => {
      if (supplier.id) {
        navigate(`/partner/suppliers/${supplier.id}`)
      }
    },
    [navigate]
  )

  // Handle bulk activate using Promise.allSettled for partial success handling
  const handleBulkActivate = useCallback(async () => {
    const results = await Promise.allSettled(
      selectedRowKeys.map((id) => api.activateSupplier(id))
    )

    const successCount = results.filter((r) => r.status === 'fulfilled').length
    const failureCount = results.filter((r) => r.status === 'rejected').length

    if (failureCount === 0) {
      // All succeeded
      Toast.success(t('suppliers.messages.batchActivateSuccess', { count: successCount }))
    } else if (successCount === 0) {
      // All failed
      Toast.error(t('suppliers.messages.batchActivateError'))
    } else {
      // Partial success
      Toast.warning(
        t('suppliers.messages.batchActivatePartial', {
          successCount,
          failureCount,
        })
      )
    }

    setSelectedRowKeys([])
    fetchSuppliers()
  }, [api, selectedRowKeys, fetchSuppliers, t])

  // Handle bulk deactivate using Promise.allSettled for partial success handling
  const handleBulkDeactivate = useCallback(async () => {
    const results = await Promise.allSettled(
      selectedRowKeys.map((id) => api.deactivateSupplier(id))
    )

    const successCount = results.filter((r) => r.status === 'fulfilled').length
    const failureCount = results.filter((r) => r.status === 'rejected').length

    if (failureCount === 0) {
      // All succeeded
      Toast.success(t('suppliers.messages.batchDeactivateSuccess', { count: successCount }))
    } else if (successCount === 0) {
      // All failed
      Toast.error(t('suppliers.messages.batchDeactivateError'))
    } else {
      // Partial success
      Toast.warning(
        t('suppliers.messages.batchDeactivatePartial', {
          successCount,
          failureCount,
        })
      )
    }

    setSelectedRowKeys([])
    fetchSuppliers()
  }, [api, selectedRowKeys, fetchSuppliers, t])

  // Table columns
  const tableColumns: DataTableColumn<Supplier>[] = useMemo(
    () => [
      {
        title: t('suppliers.columns.code'),
        dataIndex: 'code',
        width: 120,
        sortable: true,
        render: (code: unknown) => <span className="supplier-code">{(code as string) || '-'}</span>,
      },
      {
        title: t('suppliers.columns.name'),
        dataIndex: 'name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown, record: Supplier) => (
          <div className="supplier-name-cell">
            <span className="supplier-name">{(name as string) || '-'}</span>
            {record.short_name && <span className="supplier-short-name">{record.short_name}</span>}
          </div>
        ),
      },
      {
        title: t('suppliers.columns.contact'),
        dataIndex: 'phone',
        width: 160,
        render: (_phone: unknown, record: Supplier) => (
          <div className="supplier-contact-cell">
            {record.phone && <span className="supplier-phone">{record.phone}</span>}
            {record.email && <span className="supplier-email">{record.email}</span>}
            {!record.phone && !record.email && '-'}
          </div>
        ),
      },
      {
        title: t('suppliers.columns.region'),
        dataIndex: 'city',
        width: 120,
        render: (_city: unknown, record: Supplier) => (
          <span className="supplier-location-cell">
            {record.province || record.city
              ? `${record.province || ''}${record.city ? ` ${record.city}` : ''}`
              : '-'}
          </span>
        ),
      },
      {
        title: t('suppliers.columns.rating'),
        dataIndex: 'rating',
        width: 130,
        align: 'center',
        sortable: true,
        render: (rating: unknown) => {
          const ratingValue = rating as number | undefined
          if (ratingValue === undefined || ratingValue === null) return '-'
          return <Rating value={ratingValue} disabled size="small" count={5} allowHalf />
        },
      },
      {
        title: t('suppliers.columns.paymentTermDays'),
        dataIndex: 'payment_term_days',
        width: 90,
        align: 'right',
        sortable: true,
        render: (days: unknown) => {
          const daysValue = days as number | undefined
          if (daysValue === undefined || daysValue === null) return '-'
          return <span className="supplier-payment-days">{daysValue}</span>
        },
      },
      {
        title: t('suppliers.columns.status'),
        dataIndex: 'status',
        width: 90,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as HandlerSupplierListResponseStatus | undefined
          if (!statusValue) return '-'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>{t(`suppliers.status.${statusValue}`)}</Tag>
          )
        },
      },
      {
        title: t('suppliers.columns.createdAt'),
        dataIndex: 'created_at',
        width: 120,
        sortable: true,
        render: (date: unknown) => {
          const dateStr = date as string | undefined
          return dateStr ? formatDate(dateStr) : '-'
        },
      },
    ],
    [t, formatDate]
  )

  // Table row actions
  const tableActions: TableAction<Supplier>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('suppliers.actions.view'),
        onClick: handleView,
      },
      {
        key: 'edit',
        label: t('suppliers.actions.edit'),
        onClick: handleEdit,
      },
      {
        key: 'activate',
        label: t('suppliers.actions.activate'),
        type: 'primary',
        onClick: handleActivate,
        hidden: (record) => record.status === 'active',
      },
      {
        key: 'deactivate',
        label: t('suppliers.actions.deactivate'),
        type: 'warning',
        onClick: handleDeactivate,
        hidden: (record) => record.status !== 'active',
      },
      {
        key: 'block',
        label: t('suppliers.actions.block'),
        type: 'danger',
        onClick: handleBlock,
        hidden: (record) => record.status === 'blocked',
      },
      {
        key: 'delete',
        label: t('suppliers.actions.delete'),
        type: 'danger',
        onClick: handleDelete,
      },
    ],
    [t, handleView, handleEdit, handleActivate, handleDeactivate, handleBlock, handleDelete]
  )

  // Row selection handler
  const onSelectionChange = useCallback((keys: string[], _rows: Supplier[]) => {
    setSelectedRowKeys(keys)
  }, [])

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchSuppliers()
  }, [fetchSuppliers])

  return (
    <Container size="full" className="suppliers-page">
      <Card className="suppliers-card">
        <div className="suppliers-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('suppliers.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('suppliers.searchPlaceholder')}
          primaryAction={{
            label: t('suppliers.addSupplier'),
            icon: <IconPlus />,
            onClick: () => navigate('/partner/suppliers/new'),
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
            <Space className="suppliers-filter-container">
              <Select
                placeholder={t('suppliers.statusFilter')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('suppliers.typeFilter')}
                value={typeFilter}
                onChange={handleTypeChange}
                optionList={TYPE_OPTIONS}
                style={{ width: 120 }}
              />
            </Space>
          }
        />

        {selectedRowKeys.length > 0 && (
          <BulkActionBar
            selectedCount={selectedRowKeys.length}
            onCancel={() => setSelectedRowKeys([])}
          >
            <Tag color="blue" onClick={handleBulkActivate} style={{ cursor: 'pointer' }}>
              {t('suppliers.actions.batchActivate')}
            </Tag>
            <Tag color="orange" onClick={handleBulkDeactivate} style={{ cursor: 'pointer' }}>
              {t('suppliers.actions.batchDeactivate')}
            </Tag>
          </BulkActionBar>
        )}

        <Spin spinning={loading}>
          <DataTable<Supplier>
            data={supplierList}
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
            scroll={{ x: 1100 }}
          />
        </Spin>
      </Card>
    </Container>
  )
}
