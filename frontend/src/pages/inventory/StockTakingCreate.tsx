import { useState, useEffect, useCallback, useMemo } from 'react'
import { z } from 'zod'
import {
  Card,
  Typography,
  Toast,
  Spin,
  Empty,
  Button,
  Table,
  Checkbox,
  Tag,
  Modal,
} from '@douyinfe/semi-ui'
import { IconArrowLeft, IconPlus, IconDelete, IconRefresh } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import {
  Form,
  FormActions,
  FormSection,
  FormRow,
  SelectField,
  DateField,
  TextAreaField,
  useFormWithValidation,
  validationMessages,
} from '@/components/common/form'
import { Container } from '@/components/common/layout'
import { getWarehouses } from '@/api/warehouses/warehouses'
import { getInventory } from '@/api/inventory/inventory'
import { getStockTaking } from '@/api/stock-taking/stock-taking'
import type {
  HandlerWarehouseListResponse,
  HandlerInventoryItemResponse,
  HandlerAddStockTakingItemRequest,
} from '@/api/models'
import { useAuthStore } from '@/store'
import './StockTakingCreate.css'

const { Title, Text } = Typography

// Extended inventory item type that may include product info from joined queries
interface ExtendedInventoryItem extends HandlerInventoryItemResponse {
  product_name?: string
  product_code?: string
  unit?: string
}

// Form validation schema
const stockTakingCreateSchema = z.object({
  warehouse_id: z.string().min(1, validationMessages.required),
  taking_date: z.date().optional(),
  remark: z.string().max(500, validationMessages.maxLength(500)).optional(),
})

type StockTakingCreateFormData = z.infer<typeof stockTakingCreateSchema>

// Selected product for stock taking
interface SelectedProduct {
  product_id: string
  product_name: string
  product_code: string
  unit: string
  system_quantity: number
  unit_cost: number
}

/**
 * Format quantity for display with 2 decimal places
 */
function formatQuantity(quantity?: number): string {
  if (quantity === undefined || quantity === null) return '-'
  return quantity.toFixed(2)
}

/**
 * Format currency value
 */
function formatCurrency(value?: number): string {
  if (value === undefined || value === null) return '-'
  return `¥${value.toFixed(2)}`
}

/**
 * Stock Taking Create Page
 *
 * Features:
 * - Select warehouse for stock taking
 * - Select date and add remark
 * - Import system inventory for selected warehouse
 * - Select products to include in stock taking
 * - Create stock taking document and add items
 *
 * Requirements:
 * - 实现盘点单创建表单
 * - 支持选择仓库和商品范围
 * - 支持导入系统库存
 */
export default function StockTakingCreatePage() {
  const navigate = useNavigate()
  const warehousesApi = useMemo(() => getWarehouses(), [])
  const inventoryApi = useMemo(() => getInventory(), [])
  const stockTakingApi = useMemo(() => getStockTaking(), [])
  const { user } = useAuthStore()

  // State for warehouses dropdown
  const [warehouses, setWarehouses] = useState<Array<{ value: string; label: string }>>([])
  const [warehouseMap, setWarehouseMap] = useState<Map<string, string>>(new Map())
  const [loadingWarehouses, setLoadingWarehouses] = useState(false)

  // State for inventory items
  const [inventoryItems, setInventoryItems] = useState<ExtendedInventoryItem[]>([])
  const [loadingInventory, setLoadingInventory] = useState(false)

  // State for selected products
  const [selectedProducts, setSelectedProducts] = useState<SelectedProduct[]>([])
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([])

  // State for product selection modal
  const [showProductModal, setShowProductModal] = useState(false)
  const [modalSelectedKeys, setModalSelectedKeys] = useState<string[]>([])

  // Default form values
  const defaultValues: Partial<StockTakingCreateFormData> = useMemo(
    () => ({
      warehouse_id: '',
      taking_date: new Date(),
      remark: '',
    }),
    []
  )

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const { control, handleFormSubmit, isSubmitting, watch, setValue: _setValue } =
    useFormWithValidation<StockTakingCreateFormData>({
      schema: stockTakingCreateSchema,
      defaultValues,
      successMessage: '盘点单创建成功',
      onSuccess: () => {
        navigate('/inventory/stock-taking')
      },
    })

  // Watch warehouse_id for fetching inventory
  const warehouseId = watch('warehouse_id')

  // Fetch warehouses on mount
  const fetchWarehouses = useCallback(async () => {
    setLoadingWarehouses(true)
    try {
      const response = await warehousesApi.getPartnerWarehouses({
        page_size: 100,
        status: 'active',
      })
      if (response.success && response.data) {
        const warehouseList = response.data as HandlerWarehouseListResponse[]
        setWarehouses(
          warehouseList.map((w) => ({
            value: w.id || '',
            label: w.name || w.code || w.id || '',
          }))
        )
        // Build warehouse map for name lookup
        const map = new Map<string, string>()
        warehouseList.forEach((w) => {
          if (w.id) {
            map.set(w.id, w.name || w.code || w.id)
          }
        })
        setWarehouseMap(map)
      }
    } catch {
      Toast.error('获取仓库列表失败')
    } finally {
      setLoadingWarehouses(false)
    }
  }, [warehousesApi])

  // Fetch inventory for selected warehouse
  const fetchInventory = useCallback(async () => {
    if (!warehouseId) {
      setInventoryItems([])
      return
    }

    setLoadingInventory(true)
    try {
      const response = await inventoryApi.getInventoryItems({
        warehouse_id: warehouseId,
        page_size: 500,
        has_stock: true,
      })
      if (response.success && response.data) {
        setInventoryItems(response.data as ExtendedInventoryItem[])
      }
    } catch {
      Toast.error('获取库存列表失败')
    } finally {
      setLoadingInventory(false)
    }
  }, [warehouseId, inventoryApi])

  useEffect(() => {
    fetchWarehouses()
  }, [fetchWarehouses])

  useEffect(() => {
    if (warehouseId) {
      fetchInventory()
      // Clear selected products when warehouse changes
      setSelectedProducts([])
      setSelectedRowKeys([])
    }
  }, [warehouseId, fetchInventory])

  // Handle importing all inventory items
  const handleImportAll = useCallback(() => {
    const products: SelectedProduct[] = inventoryItems.map((item) => ({
      product_id: item.product_id || '',
      product_name: item.product_name || item.product_id || '',
      product_code: item.product_code || '',
      unit: item.unit || '件',
      system_quantity: item.total_quantity || 0,
      unit_cost: item.unit_cost || 0,
    }))
    setSelectedProducts(products)
    setSelectedRowKeys(products.map((p) => p.product_id))
    Toast.success(`已导入 ${products.length} 个商品`)
  }, [inventoryItems])

  // Handle opening product selection modal
  const handleOpenProductModal = useCallback(() => {
    setModalSelectedKeys(selectedRowKeys)
    setShowProductModal(true)
  }, [selectedRowKeys])

  // Handle confirming product selection from modal
  const handleConfirmProductSelection = useCallback(() => {
    const products: SelectedProduct[] = inventoryItems
      .filter((item) => modalSelectedKeys.includes(item.product_id || ''))
      .map((item) => ({
        product_id: item.product_id || '',
        product_name: item.product_name || item.product_id || '',
        product_code: item.product_code || '',
        unit: item.unit || '件',
        system_quantity: item.total_quantity || 0,
        unit_cost: item.unit_cost || 0,
      }))
    setSelectedProducts(products)
    setSelectedRowKeys(modalSelectedKeys)
    setShowProductModal(false)
    Toast.success(`已选择 ${products.length} 个商品`)
  }, [inventoryItems, modalSelectedKeys])

  // Handle removing a product from selection
  const handleRemoveProduct = useCallback((productId: string) => {
    setSelectedProducts((prev) => prev.filter((p) => p.product_id !== productId))
    setSelectedRowKeys((prev) => prev.filter((key) => key !== productId))
  }, [])

  // Handle form submission
  const onSubmit = async (data: StockTakingCreateFormData) => {
    if (selectedProducts.length === 0) {
      Toast.error('请至少选择一个商品')
      throw new Error('No products selected')
    }

    if (!user?.id) {
      Toast.error('用户未登录')
      throw new Error('User not logged in')
    }

    const warehouseName = warehouseMap.get(data.warehouse_id) || ''

    // Create stock taking
    const createResponse = await stockTakingApi.postInventoryStockTakings({
      warehouse_id: data.warehouse_id,
      warehouse_name: warehouseName,
      taking_date: data.taking_date?.toISOString().split('T')[0],
      remark: data.remark || undefined,
      created_by_id: user.id,
      created_by_name: user.displayName || user.username,
    })

    if (!createResponse.success || !createResponse.data) {
      throw new Error(createResponse.error?.message || '创建盘点单失败')
    }

    const stockTakingId = createResponse.data.id

    // Add items to stock taking
    const items: HandlerAddStockTakingItemRequest[] = selectedProducts.map((p) => ({
      product_id: p.product_id,
      product_name: p.product_name,
      product_code: p.product_code,
      unit: p.unit,
      system_quantity: p.system_quantity,
      unit_cost: p.unit_cost,
    }))

    const addItemsResponse = await stockTakingApi.postInventoryStockTakingsIdItemsBulk(
      stockTakingId || '',
      { items }
    )

    if (!addItemsResponse.success) {
      throw new Error(addItemsResponse.error?.message || '添加盘点商品失败')
    }
  }

  const handleCancel = () => {
    navigate(-1)
  }

  const handleBack = () => {
    navigate(-1)
  }

  // Table columns for selected products
  const selectedProductColumns = [
    {
      title: '商品编码',
      dataIndex: 'product_code',
      width: 120,
    },
    {
      title: '商品名称',
      dataIndex: 'product_name',
      width: 200,
    },
    {
      title: '单位',
      dataIndex: 'unit',
      width: 80,
    },
    {
      title: '系统数量',
      dataIndex: 'system_quantity',
      width: 100,
      align: 'right' as const,
      render: (qty: number) => formatQuantity(qty),
    },
    {
      title: '单位成本',
      dataIndex: 'unit_cost',
      width: 100,
      align: 'right' as const,
      render: (cost: number) => formatCurrency(cost),
    },
    {
      title: '操作',
      width: 80,
      render: (_: unknown, record: SelectedProduct) => (
        <Button
          icon={<IconDelete />}
          type="danger"
          theme="borderless"
          size="small"
          onClick={() => handleRemoveProduct(record.product_id)}
        />
      ),
    },
  ]

  // Table columns for inventory items in modal
  const inventoryColumns = [
    {
      title: '商品编码',
      dataIndex: 'product_code',
      width: 120,
    },
    {
      title: '商品名称',
      dataIndex: 'product_name',
      width: 200,
    },
    {
      title: '单位',
      dataIndex: 'unit',
      width: 80,
    },
    {
      title: '系统数量',
      dataIndex: 'total_quantity',
      width: 100,
      align: 'right' as const,
      render: (qty: number) => formatQuantity(qty),
    },
    {
      title: '单位成本',
      dataIndex: 'unit_cost',
      width: 100,
      align: 'right' as const,
      render: (cost: number) => formatCurrency(cost),
    },
    {
      title: '状态',
      dataIndex: 'is_below_minimum',
      width: 80,
      render: (_: unknown, record: HandlerInventoryItemResponse) => {
        if (record.is_below_minimum) {
          return <Tag color="orange">低库存</Tag>
        }
        return <Tag color="green">正常</Tag>
      },
    },
  ]

  return (
    <Container size="lg" className="stock-taking-create-page">
      {/* Header */}
      <div className="stock-taking-create-header">
        <div className="header-left">
          <Button icon={<IconArrowLeft />} theme="borderless" onClick={handleBack}>
            返回
          </Button>
          <Title heading={4} style={{ margin: 0 }}>
            创建盘点单
          </Title>
        </div>
      </div>

      <Card className="stock-taking-create-card">
        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          {/* Basic Info Section */}
          <FormSection title="基本信息" description="选择要盘点的仓库和盘点日期">
            <FormRow cols={2}>
              <SelectField
                name="warehouse_id"
                control={control}
                label="仓库"
                placeholder={loadingWarehouses ? '加载中...' : '请选择仓库'}
                options={warehouses}
                required
                showSearch
                disabled={loadingWarehouses}
              />
              <DateField
                name="taking_date"
                control={control}
                label="盘点日期"
                placeholder="请选择盘点日期"
              />
            </FormRow>
            <TextAreaField
              name="remark"
              control={control}
              label="备注"
              placeholder="请输入备注信息（可选）"
              rows={2}
              maxCount={500}
            />
          </FormSection>

          {/* Product Selection Section */}
          <FormSection title="盘点商品" description="选择要盘点的商品，将导入系统当前库存数量">
            {!warehouseId ? (
              <Empty
                title="请先选择仓库"
                description="选择仓库后可导入该仓库的库存商品"
              />
            ) : loadingInventory ? (
              <div className="loading-container">
                <Spin />
                <Text type="tertiary">正在加载库存数据...</Text>
              </div>
            ) : inventoryItems.length === 0 ? (
              <Empty
                title="暂无库存数据"
                description="该仓库暂无可盘点的库存商品"
              />
            ) : (
              <>
                {/* Toolbar */}
                <div className="product-toolbar">
                  <div className="toolbar-left">
                    <Text strong>已选择 {selectedProducts.length} 个商品</Text>
                    <Text type="tertiary" style={{ marginLeft: 'var(--spacing-3)' }}>
                      (仓库共有 {inventoryItems.length} 个商品有库存)
                    </Text>
                  </div>
                  <div className="toolbar-right">
                    <Button
                      icon={<IconRefresh />}
                      onClick={fetchInventory}
                      disabled={loadingInventory}
                    >
                      刷新
                    </Button>
                    <Button
                      icon={<IconPlus />}
                      onClick={handleOpenProductModal}
                    >
                      选择商品
                    </Button>
                    <Button
                      type="primary"
                      onClick={handleImportAll}
                    >
                      全部导入
                    </Button>
                  </div>
                </div>

                {/* Selected Products Table */}
                {selectedProducts.length > 0 ? (
                  <Table
                    dataSource={selectedProducts}
                    columns={selectedProductColumns}
                    rowKey="product_id"
                    pagination={false}
                    size="small"
                    className="selected-products-table"
                  />
                ) : (
                  <Empty
                    title="未选择商品"
                    description="点击「选择商品」或「全部导入」添加盘点商品"
                    style={{ padding: 'var(--spacing-8) 0' }}
                  />
                )}
              </>
            )}
          </FormSection>

          <FormActions
            submitText="创建盘点单"
            isSubmitting={isSubmitting}
            onCancel={handleCancel}
            showCancel
          />
        </Form>
      </Card>

      {/* Product Selection Modal */}
      <Modal
        title="选择盘点商品"
        visible={showProductModal}
        onCancel={() => setShowProductModal(false)}
        onOk={handleConfirmProductSelection}
        okText="确认选择"
        cancelText="取消"
        width={800}
        bodyStyle={{ padding: 0 }}
      >
        <div className="product-modal-content">
          <div className="modal-toolbar">
            <Checkbox
              checked={modalSelectedKeys.length === inventoryItems.length && inventoryItems.length > 0}
              indeterminate={
                modalSelectedKeys.length > 0 && modalSelectedKeys.length < inventoryItems.length
              }
              onChange={(e) => {
                if (e.target.checked) {
                  setModalSelectedKeys(inventoryItems.map((item) => item.product_id || ''))
                } else {
                  setModalSelectedKeys([])
                }
              }}
            >
              全选 ({modalSelectedKeys.length}/{inventoryItems.length})
            </Checkbox>
          </div>
          <Table
            dataSource={inventoryItems}
            columns={inventoryColumns}
            rowKey="product_id"
            rowSelection={{
              selectedRowKeys: modalSelectedKeys,
              onChange: (keys) => setModalSelectedKeys(keys as string[]),
            }}
            pagination={{ pageSize: 10 }}
            size="small"
          />
        </div>
      </Modal>
    </Container>
  )
}
