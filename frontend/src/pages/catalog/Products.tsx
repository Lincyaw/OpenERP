import { useState, useEffect, useCallback, useMemo } from 'react'
import { Card, Typography, Tag, Toast, Select, Space, Modal, Spin } from '@douyinfe/semi-ui'
import { IconPlus, IconRefresh } from '@douyinfe/semi-icons'
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
import { getProducts } from '@/api/products/products'
import type {
  HandlerProductListResponse,
  HandlerProductListResponseStatus,
  GetCatalogProductsParams,
  GetCatalogProductsStatus,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './Products.css'

const { Title } = Typography

// Product type with index signature for DataTable compatibility
type Product = HandlerProductListResponse & Record<string, unknown>

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '启用', value: 'active' },
  { label: '禁用', value: 'inactive' },
  { label: '停售', value: 'discontinued' },
]

// Status tag color mapping
const STATUS_TAG_COLORS: Record<HandlerProductListResponseStatus, 'green' | 'grey' | 'red'> = {
  active: 'green',
  inactive: 'grey',
  discontinued: 'red',
}

// Status labels
const STATUS_LABELS: Record<HandlerProductListResponseStatus, string> = {
  active: '启用',
  inactive: '禁用',
  discontinued: '停售',
}

/**
 * Format price for display
 */
function formatPrice(price?: number): string {
  if (price === undefined || price === null) return '-'
  return `¥${price.toFixed(2)}`
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
  const api = useMemo(() => getProducts(), [])

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
      const params: GetCatalogProductsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        status: (statusFilter || undefined) as GetCatalogProductsStatus | undefined,
        order_by: state.sort.field || 'created_at',
        order_dir: state.sort.order === 'asc' ? 'asc' : 'desc',
      }

      const response = await api.getCatalogProducts(params)

      if (response.success && response.data) {
        setProductList(response.data as Product[])
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
      Toast.error('获取商品列表失败')
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
  ])

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
        await api.postCatalogProductsIdActivate(product.id)
        Toast.success(`商品 "${product.name}" 已启用`)
        fetchProducts()
      } catch {
        Toast.error('启用商品失败')
      }
    },
    [api, fetchProducts]
  )

  // Handle deactivate product
  const handleDeactivate = useCallback(
    async (product: Product) => {
      if (!product.id) return
      try {
        await api.postCatalogProductsIdDeactivate(product.id)
        Toast.success(`商品 "${product.name}" 已禁用`)
        fetchProducts()
      } catch {
        Toast.error('禁用商品失败')
      }
    },
    [api, fetchProducts]
  )

  // Handle discontinue product
  const handleDiscontinue = useCallback(
    async (product: Product) => {
      if (!product.id) return
      Modal.confirm({
        title: '确认停售',
        content: `确定要停售商品 "${product.name}" 吗？停售后将无法恢复。`,
        okText: '确认停售',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await api.postCatalogProductsIdDiscontinue(product.id!)
            Toast.success(`商品 "${product.name}" 已停售`)
            fetchProducts()
          } catch {
            Toast.error('停售商品失败')
          }
        },
      })
    },
    [api, fetchProducts]
  )

  // Handle delete product
  const handleDelete = useCallback(
    async (product: Product) => {
      if (!product.id) return
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除商品 "${product.name}" 吗？删除后无法恢复。`,
        okText: '确认删除',
        cancelText: '取消',
        okButtonProps: { type: 'danger' },
        onOk: async () => {
          try {
            await api.deleteCatalogProductsId(product.id!)
            Toast.success(`商品 "${product.name}" 已删除`)
            fetchProducts()
          } catch {
            Toast.error('删除商品失败')
          }
        },
      })
    },
    [api, fetchProducts]
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

  // Handle bulk activate
  const handleBulkActivate = useCallback(async () => {
    try {
      await Promise.all(selectedRowKeys.map((id) => api.postCatalogProductsIdActivate(id)))
      Toast.success(`已启用 ${selectedRowKeys.length} 个商品`)
      setSelectedRowKeys([])
      fetchProducts()
    } catch {
      Toast.error('批量启用失败')
    }
  }, [api, selectedRowKeys, fetchProducts])

  // Handle bulk deactivate
  const handleBulkDeactivate = useCallback(async () => {
    try {
      await Promise.all(selectedRowKeys.map((id) => api.postCatalogProductsIdDeactivate(id)))
      Toast.success(`已禁用 ${selectedRowKeys.length} 个商品`)
      setSelectedRowKeys([])
      fetchProducts()
    } catch {
      Toast.error('批量禁用失败')
    }
  }, [api, selectedRowKeys, fetchProducts])

  // Table columns
  const tableColumns: DataTableColumn<Product>[] = useMemo(
    () => [
      {
        title: '商品编码',
        dataIndex: 'code',
        width: 120,
        sortable: true,
        render: (code: unknown) => (
          <span className="product-code">{(code as string) || '-'}</span>
        ),
      },
      {
        title: '商品名称',
        dataIndex: 'name',
        sortable: true,
        ellipsis: true,
        render: (name: unknown, record: Product) => (
          <div className="product-name-cell">
            <span className="product-name">{(name as string) || '-'}</span>
            {record.barcode && <span className="product-barcode">{record.barcode}</span>}
          </div>
        ),
      },
      {
        title: '单位',
        dataIndex: 'unit',
        width: 80,
        align: 'center',
      },
      {
        title: '进价',
        dataIndex: 'purchase_price',
        width: 100,
        align: 'right',
        sortable: true,
        render: (price: unknown) => formatPrice(price as number | undefined),
      },
      {
        title: '售价',
        dataIndex: 'selling_price',
        width: 100,
        align: 'right',
        sortable: true,
        render: (price: unknown) => formatPrice(price as number | undefined),
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 90,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as HandlerProductListResponseStatus | undefined
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
  const tableActions: TableAction<Product>[] = useMemo(
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
        hidden: (record) => record.status === 'discontinued',
      },
      {
        key: 'activate',
        label: '启用',
        type: 'primary',
        onClick: handleActivate,
        hidden: (record) => record.status !== 'inactive',
      },
      {
        key: 'deactivate',
        label: '禁用',
        type: 'warning',
        onClick: handleDeactivate,
        hidden: (record) => record.status !== 'active',
      },
      {
        key: 'discontinue',
        label: '停售',
        type: 'danger',
        onClick: handleDiscontinue,
        hidden: (record) => record.status === 'discontinued',
      },
      {
        key: 'delete',
        label: '删除',
        type: 'danger',
        onClick: handleDelete,
      },
    ],
    [handleView, handleEdit, handleActivate, handleDeactivate, handleDiscontinue, handleDelete]
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
            商品管理
          </Title>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索商品名称、编码、条码..."
          primaryAction={{
            label: '新增商品',
            icon: <IconPlus />,
            onClick: () => navigate('/catalog/products/new'),
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
            <Space>
              <Select
                placeholder="状态筛选"
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
              批量启用
            </Tag>
            <Tag color="orange" onClick={handleBulkDeactivate} style={{ cursor: 'pointer' }}>
              批量禁用
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
          />
        </Spin>
      </Card>
    </Container>
  )
}
