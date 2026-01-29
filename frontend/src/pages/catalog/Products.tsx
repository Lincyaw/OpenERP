import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Tag, Toast, Select, Space, Modal, Spin } from '@douyinfe/semi-ui-19'
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
import {
  listProducts,
  activateProduct,
  deactivateProduct,
  discontinueProduct,
  deleteProduct,
} from '@/api/products/products'
import type {
  HandlerProductListResponse,
  HandlerProductListResponseStatus,
  GetProductByCategoryParams,
  GetProductByCategoryStatus,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './Products.css'

const { Title } = Typography

// Product type with index signature for DataTable compatibility
type Product = HandlerProductListResponse & Record<string, unknown>

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerProductListResponseStatus, 'green' | 'grey' | 'red'> = {
  active: 'green',
  inactive: 'grey',
  discontinued: 'red',
}

/**
 * Products list page
 *
 * Features:
 * - Product listing with pagination
 * - Search by name, code, barcode
 * - Filter by status
 * - Enable/disable product actions
 * - Navigate to product form for create/edit
 */
export default function ProductsPage() {
  const navigate = useNavigate()
  const { t } = useTranslation(['catalog', 'common'])
  const { formatCurrency, formatDate } = useFormatters()

  // State for data
  const [productList, setProductList] = useState<Product[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('')

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Fetch products
  const fetchProducts = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetProductByCategoryParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as GetProductByCategoryStatus | undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      const response = await listProducts(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setProductList(response.data.data as Product[])
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
      Toast.error(t('products.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [state.pagination.page, state.pagination.pageSize, state.sort, searchKeyword, statusFilter])

  // Fetch on mount and when state changes
  useEffect(() => {
    fetchProducts()
  }, [fetchProducts])

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

  // Handle activate product
  const handleActivate = useCallback(
    async (product: Product) => {
      if (!product.id) return
      try {
        await activateProduct(product.id, {})
        Toast.success(t('products.messages.activateSuccess', { name: product.name }))
        fetchProducts()
      } catch {
        Toast.error(t('products.messages.activateError'))
      }
    },
    [fetchProducts, t]
  )

  // Handle deactivate product
  const handleDeactivate = useCallback(
    async (product: Product) => {
      if (!product.id) return
      try {
        await deactivateProduct(product.id, {})
        Toast.success(t('products.messages.deactivateSuccess', { name: product.name }))
        fetchProducts()
      } catch {
        Toast.error(t('products.messages.deactivateError'))
      }
    },
    [fetchProducts, t]
  )

  // Handle discontinue product
  const handleDiscontinue = useCallback(
    async (product: Product) => {
      if (!product.id) return
      Modal.confirm({
        title: t('products.confirm.discontinueTitle'),
        content: t('products.confirm.discontinueContent', { name: product.name }),
        okText: t('products.confirm.discontinueOk'),
        cancelText: t('common:actions.cancel'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await discontinueProduct(product.id!, {})
            Toast.success(t('products.messages.discontinueSuccess', { name: product.name }))
            fetchProducts()
          } catch {
            Toast.error(t('products.messages.discontinueError'))
          }
        },
      })
    },
    [fetchProducts, t]
  )

  // Handle delete product
  const handleDelete = useCallback(
    async (product: Product) => {
      if (!product.id) return
      Modal.confirm({
        title: t('products.confirm.deleteTitle'),
        content: t('products.confirm.deleteContent', { name: product.name }),
        okText: t('products.confirm.deleteOk'),
        cancelText: t('common:actions.cancel'),
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await deleteProduct(product.id!)
            Toast.success(t('products.messages.deleteSuccess', { name: product.name }))
            fetchProducts()
          } catch {
            Toast.error(t('products.messages.deleteError'))
          }
        },
      })
    },
    [fetchProducts, t]
  )

  // Handle edit product
  const handleEdit = useCallback(
    (product: Product) => {
      if (product.id) {
        navigate(`/catalog/products/${product.id}/edit`)
      }
    },
    [navigate]
  )

  // Handle view product
  const handleView = useCallback(
    (product: Product) => {
      if (product.id) {
        navigate(`/catalog/products/${product.id}`)
      }
    },
    [navigate]
  )

  // Handle bulk activate using Promise.allSettled for partial success handling
  const handleBulkActivate = useCallback(async () => {
    const results = await Promise.allSettled(selectedRowKeys.map((id) => activateProduct(id, {})))

    const successCount = results.filter((r) => r.status === 'fulfilled').length
    const failureCount = results.filter((r) => r.status === 'rejected').length

    if (failureCount === 0) {
      // All succeeded
      Toast.success(t('products.messages.batchActivateSuccess', { count: successCount }))
    } else if (successCount === 0) {
      // All failed
      Toast.error(t('products.messages.batchActivateError'))
    } else {
      // Partial success - use type assertion for interpolation with dynamic keys
      Toast.warning(
        (t as (key: string, options?: Record<string, unknown>) => string)(
          'products.messages.batchActivatePartial',
          { successCount, failureCount }
        )
      )
    }

    setSelectedRowKeys([])
    fetchProducts()
  }, [selectedRowKeys, fetchProducts, t])

  // Handle bulk deactivate using Promise.allSettled for partial success handling
  const handleBulkDeactivate = useCallback(async () => {
    const results = await Promise.allSettled(selectedRowKeys.map((id) => deactivateProduct(id, {})))

    const successCount = results.filter((r) => r.status === 'fulfilled').length
    const failureCount = results.filter((r) => r.status === 'rejected').length

    if (failureCount === 0) {
      // All succeeded
      Toast.success(t('products.messages.batchDeactivateSuccess', { count: successCount }))
    } else if (successCount === 0) {
      // All failed
      Toast.error(t('products.messages.batchDeactivateError'))
    } else {
      // Partial success - use type assertion for interpolation with dynamic keys
      Toast.warning(
        (t as (key: string, options?: Record<string, unknown>) => string)(
          'products.messages.batchDeactivatePartial',
          { successCount, failureCount }
        )
      )
    }

    setSelectedRowKeys([])
    fetchProducts()
  }, [selectedRowKeys, fetchProducts, t])

  // Status options for filter - memoized with t dependency
  const STATUS_OPTIONS = useMemo(
    () => [
      { label: t('products.allStatus'), value: '' },
      { label: t('products.status.active'), value: 'active' },
      { label: t('products.status.inactive'), value: 'inactive' },
      { label: t('products.status.discontinued'), value: 'discontinued' },
    ],
    [t]
  )

  // Table columns
  const tableColumns: DataTableColumn<Product>[] = useMemo(
    () => [
      {
        title: t('products.columns.code'),
        dataIndex: 'code',
        width: 120,
        sortable: true,
        render: (code: unknown) => <span className="product-code">{(code as string) || '-'}</span>,
      },
      {
        title: t('products.columns.name'),
        dataIndex: 'name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown, record: Product) => (
          <div className="product-name-cell">
            <span
              className="product-name table-cell-link"
              onClick={() => {
                if (record.id) navigate(`/catalog/products/${record.id}`)
              }}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  if (record.id) navigate(`/catalog/products/${record.id}`)
                }
              }}
              role="link"
              tabIndex={0}
            >
              {(name as string) || '-'}
            </span>
            {record.barcode && <span className="product-barcode">{record.barcode}</span>}
          </div>
        ),
      },
      {
        title: t('products.columns.unit'),
        dataIndex: 'unit',
        width: 80,
        align: 'center',
      },
      {
        title: t('products.columns.purchasePrice'),
        dataIndex: 'purchase_price',
        width: 100,
        align: 'right',
        sortable: true,
        render: (price: unknown) => {
          if (price === undefined || price === null) return '-'
          // API returns decimal values as strings, parse them to numbers
          const numValue = typeof price === 'string' ? parseFloat(price) : (price as number)
          if (isNaN(numValue)) return '-'
          return formatCurrency(numValue)
        },
      },
      {
        title: t('products.columns.sellingPrice'),
        dataIndex: 'selling_price',
        width: 100,
        align: 'right',
        sortable: true,
        render: (price: unknown) => {
          if (price === undefined || price === null) return '-'
          // API returns decimal values as strings, parse them to numbers
          const numValue = typeof price === 'string' ? parseFloat(price) : (price as number)
          if (isNaN(numValue)) return '-'
          return formatCurrency(numValue)
        },
      },
      {
        title: t('products.columns.status'),
        dataIndex: 'status',
        width: 90,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as HandlerProductListResponseStatus | undefined
          if (!statusValue) return '-'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>{t(`products.status.${statusValue}`)}</Tag>
          )
        },
      },
      {
        title: t('products.columns.createdAt'),
        dataIndex: 'created_at',
        width: 120,
        sortable: true,
        render: (date: unknown) => {
          const dateStr = date as string | undefined
          if (!dateStr) return '-'
          return formatDate(new Date(dateStr), 'short')
        },
      },
    ],
    [t, formatCurrency, formatDate, navigate]
  )

  // Table row actions
  const tableActions: TableAction<Product>[] = useMemo(
    () => [
      {
        key: 'view',
        label: t('products.actions.view'),
        onClick: handleView,
      },
      {
        key: 'edit',
        label: t('products.actions.edit'),
        onClick: handleEdit,
        hidden: (record) => record.status === 'discontinued',
      },
      {
        key: 'activate',
        label: t('products.actions.activate'),
        type: 'primary',
        onClick: handleActivate,
        hidden: (record) => record.status !== 'inactive',
      },
      {
        key: 'deactivate',
        label: t('products.actions.deactivate'),
        type: 'warning',
        onClick: handleDeactivate,
        hidden: (record) => record.status !== 'active',
      },
      {
        key: 'discontinue',
        label: t('products.actions.discontinue'),
        type: 'danger',
        onClick: handleDiscontinue,
        hidden: (record) => record.status === 'discontinued',
      },
      {
        key: 'delete',
        label: t('products.actions.delete'),
        type: 'danger',
        onClick: handleDelete,
      },
    ],
    [t, handleView, handleEdit, handleActivate, handleDeactivate, handleDiscontinue, handleDelete]
  )

  // Row selection handler
  const onSelectionChange = useCallback((keys: string[], _rows: Product[]) => {
    setSelectedRowKeys(keys)
  }, [])

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchProducts()
  }, [fetchProducts])

  return (
    <Container size="full" className="products-page">
      <Card className="products-card">
        <div className="products-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('products.title')}
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('products.searchPlaceholder')}
          primaryAction={{
            label: t('products.addProduct'),
            icon: <IconPlus />,
            onClick: () => navigate('/catalog/products/new'),
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
            <Space>
              <Select
                placeholder={t('products.statusFilter')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
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
              {t('products.actions.batchActivate')}
            </Tag>
            <Tag color="orange" onClick={handleBulkDeactivate} style={{ cursor: 'pointer' }}>
              {t('products.actions.batchDeactivate')}
            </Tag>
          </BulkActionBar>
        )}

        <Spin spinning={loading}>
          <DataTable<Product>
            data={productList}
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
            resizable
          />
        </Spin>
      </Card>
    </Container>
  )
}
