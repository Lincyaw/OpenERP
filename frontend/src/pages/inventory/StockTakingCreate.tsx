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
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconPlus, IconDelete, IconRefresh } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
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
import { useFormatters } from '@/hooks/useFormatters'
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
  const { t } = useTranslation(['inventory', 'common'])
  const { formatCurrency: formatCurrencyBase } = useFormatters()
  const warehousesApi = useMemo(() => getWarehouses(), [])
  const inventoryApi = useMemo(() => getInventory(), [])
  const stockTakingApi = useMemo(() => getStockTaking(), [])
  const { user } = useAuthStore()

  // Wrapper function to handle undefined values
  const formatCurrency = useCallback(
    (value?: number): string => (value !== undefined ? formatCurrencyBase(value) : '-'),
    [formatCurrencyBase]
  )

  /**
   * Format quantity for display with 2 decimal places
   * Note: API returns decimal values as strings to preserve precision
   */
  const formatQuantity = useCallback((quantity?: number | string): string => {
    if (quantity === undefined || quantity === null) return '-'
    const num = typeof quantity === 'string' ? parseFloat(quantity) : quantity
    if (isNaN(num)) return '-'
    return num.toFixed(2)
  }, [])

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

  const {
    control,
    handleFormSubmit,
    isSubmitting,
    watch,
    setValue: _setValue,
  } = useFormWithValidation<StockTakingCreateFormData>({
    schema: stockTakingCreateSchema,
    defaultValues,
    successMessage: t('stockTaking.create.messages.success'),
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
      const response = await warehousesApi.listWarehouses({
        page_size: 100,
        status: 'enabled',
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
      Toast.error(t('stockTaking.create.messages.fetchWarehouseError'))
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
      // Use the warehouse-specific endpoint with path parameter
      const response = await inventoryApi.listInventoryByWarehouse(warehouseId, {
        page_size: 500,
        has_stock: true,
      })
      if (response.success && response.data) {
        setInventoryItems(response.data as ExtendedInventoryItem[])
      }
    } catch {
      Toast.error(t('stockTaking.create.messages.fetchInventoryError'))
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
      system_quantity:
        typeof item.total_quantity === 'string'
          ? parseFloat(item.total_quantity)
          : item.total_quantity || 0,
      unit_cost:
        typeof item.unit_cost === 'string' ? parseFloat(item.unit_cost) : item.unit_cost || 0,
    }))
    setSelectedProducts(products)
    setSelectedRowKeys(products.map((p) => p.product_id))
    Toast.success(t('stockTaking.create.messages.importSuccess', { count: products.length }))
  }, [inventoryItems, t])

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
        system_quantity:
          typeof item.total_quantity === 'string'
            ? parseFloat(item.total_quantity)
            : item.total_quantity || 0,
        unit_cost:
          typeof item.unit_cost === 'string' ? parseFloat(item.unit_cost) : item.unit_cost || 0,
      }))
    setSelectedProducts(products)
    setSelectedRowKeys(modalSelectedKeys)
    setShowProductModal(false)
    Toast.success(t('stockTaking.create.messages.selectSuccess', { count: products.length }))
  }, [inventoryItems, modalSelectedKeys, t])

  // Handle removing a product from selection
  const handleRemoveProduct = useCallback((productId: string) => {
    setSelectedProducts((prev) => prev.filter((p) => p.product_id !== productId))
    setSelectedRowKeys((prev) => prev.filter((key) => key !== productId))
  }, [])

  // Handle form submission
  const onSubmit = async (data: StockTakingCreateFormData) => {
    if (selectedProducts.length === 0) {
      Toast.error(t('stockTaking.create.messages.noProductError'))
      throw new Error('No products selected')
    }

    if (!user?.id) {
      Toast.error(t('stockTaking.create.messages.userNotLoggedIn'))
      throw new Error('User not logged in')
    }

    const warehouseName = warehouseMap.get(data.warehouse_id) || ''

    // Create stock taking
    const createResponse = await stockTakingApi.createStockTaking({
      warehouse_id: data.warehouse_id,
      warehouse_name: warehouseName,
      taking_date: data.taking_date?.toISOString().split('T')[0],
      remark: data.remark || undefined,
      created_by_id: user.id,
      created_by_name: user.displayName || user.username,
    })

    if (!createResponse.success || !createResponse.data) {
      throw new Error(createResponse.error?.message || t('stockTaking.create.messages.createError'))
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

    const addItemsResponse = await stockTakingApi.addItemsStockTaking(
      stockTakingId || '',
      { items }
    )

    if (!addItemsResponse.success) {
      throw new Error(
        addItemsResponse.error?.message || t('stockTaking.create.messages.addItemsError')
      )
    }
  }

  const handleCancel = () => {
    navigate(-1)
  }

  const handleBack = () => {
    navigate(-1)
  }

  // Table columns for selected products
  const selectedProductColumns = useMemo(
    () => [
      {
        title: t('stockTaking.create.products.columns.productCode'),
        dataIndex: 'product_code',
        width: 120,
      },
      {
        title: t('stockTaking.create.products.columns.productName'),
        dataIndex: 'product_name',
        width: 200,
      },
      {
        title: t('stockTaking.create.products.columns.unit'),
        dataIndex: 'unit',
        width: 80,
      },
      {
        title: t('stockTaking.create.products.columns.systemQuantity'),
        dataIndex: 'system_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => formatQuantity(qty),
      },
      {
        title: t('stockTaking.create.products.columns.unitCost'),
        dataIndex: 'unit_cost',
        width: 100,
        align: 'right' as const,
        render: (cost: number) => formatCurrency(cost),
      },
      {
        title: t('stockTaking.create.products.columns.operation'),
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
    ],
    [t, formatQuantity, formatCurrency, handleRemoveProduct]
  )

  // Table columns for inventory items in modal
  const inventoryColumns = useMemo(
    () => [
      {
        title: t('stockTaking.create.products.columns.productCode'),
        dataIndex: 'product_code',
        width: 120,
      },
      {
        title: t('stockTaking.create.products.columns.productName'),
        dataIndex: 'product_name',
        width: 200,
      },
      {
        title: t('stockTaking.create.products.columns.unit'),
        dataIndex: 'unit',
        width: 80,
      },
      {
        title: t('stockTaking.create.products.columns.systemQuantity'),
        dataIndex: 'total_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => formatQuantity(qty),
      },
      {
        title: t('stockTaking.create.products.columns.unitCost'),
        dataIndex: 'unit_cost',
        width: 100,
        align: 'right' as const,
        render: (cost: number) => formatCurrency(cost),
      },
      {
        title: t('stockTaking.create.products.columns.status'),
        dataIndex: 'is_below_minimum',
        width: 80,
        render: (_: unknown, record: HandlerInventoryItemResponse) => {
          if (record.is_below_minimum) {
            return <Tag color="orange">{t('stockTaking.create.products.status.lowStock')}</Tag>
          }
          return <Tag color="green">{t('stockTaking.create.products.status.normal')}</Tag>
        },
      },
    ],
    [t, formatQuantity, formatCurrency]
  )

  return (
    <Container size="lg" className="stock-taking-create-page">
      {/* Header */}
      <div className="stock-taking-create-header">
        <div className="header-left">
          <Button icon={<IconArrowLeft />} theme="borderless" onClick={handleBack}>
            {t('stockTaking.create.back')}
          </Button>
          <Title heading={4} style={{ margin: 0 }}>
            {t('stockTaking.create.title')}
          </Title>
        </div>
      </div>

      <Card className="stock-taking-create-card">
        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          {/* Basic Info Section */}
          <FormSection
            title={t('stockTaking.create.basicInfo.title')}
            description={t('stockTaking.create.basicInfo.description')}
          >
            <FormRow cols={2}>
              <SelectField
                name="warehouse_id"
                control={control}
                label={t('stockTaking.create.basicInfo.warehouse')}
                placeholder={
                  loadingWarehouses
                    ? t('stockTaking.create.products.loading')
                    : t('stockTaking.create.basicInfo.warehousePlaceholder')
                }
                options={warehouses}
                required
                showSearch
                disabled={loadingWarehouses}
              />
              <DateField
                name="taking_date"
                control={control}
                label={t('stockTaking.create.basicInfo.takingDate')}
                placeholder={t('stockTaking.create.basicInfo.takingDatePlaceholder')}
              />
            </FormRow>
            <TextAreaField
              name="remark"
              control={control}
              label={t('stockTaking.create.basicInfo.remark')}
              placeholder={t('stockTaking.create.basicInfo.remarkPlaceholder')}
              rows={2}
              maxCount={500}
            />
          </FormSection>

          {/* Product Selection Section */}
          <FormSection
            title={t('stockTaking.create.products.title')}
            description={t('stockTaking.create.products.description')}
          >
            {!warehouseId ? (
              <Empty
                title={t('stockTaking.create.products.selectWarehouseFirst')}
                description={t('stockTaking.create.products.selectWarehouseFirstDesc')}
              />
            ) : loadingInventory ? (
              <div className="loading-container">
                <Spin />
                <Text type="tertiary">{t('stockTaking.create.products.loadingInventory')}</Text>
              </div>
            ) : inventoryItems.length === 0 ? (
              <Empty
                title={t('stockTaking.create.products.noInventory')}
                description={t('stockTaking.create.products.noInventoryDesc')}
              />
            ) : (
              <>
                {/* Toolbar */}
                <div className="product-toolbar">
                  <div className="toolbar-left">
                    <Text strong>
                      {t('stockTaking.create.products.selectedCount', {
                        count: selectedProducts.length,
                      })}
                    </Text>
                    <Text type="tertiary" style={{ marginLeft: 'var(--spacing-3)' }}>
                      {t('stockTaking.create.products.totalCount', {
                        count: inventoryItems.length,
                      })}
                    </Text>
                  </div>
                  <div className="toolbar-right">
                    <Button
                      icon={<IconRefresh />}
                      onClick={fetchInventory}
                      disabled={loadingInventory}
                    >
                      {t('stockTaking.create.products.refresh')}
                    </Button>
                    <Button icon={<IconPlus />} onClick={handleOpenProductModal}>
                      {t('stockTaking.create.products.selectProducts')}
                    </Button>
                    <Button type="primary" onClick={handleImportAll}>
                      {t('stockTaking.create.products.importAll')}
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
                    title={t('stockTaking.create.products.noProductSelected')}
                    description={t('stockTaking.create.products.noProductSelectedDesc')}
                    style={{ padding: 'var(--spacing-8) 0' }}
                  />
                )}
              </>
            )}
          </FormSection>

          <FormActions
            submitText={t('stockTaking.create.submit')}
            isSubmitting={isSubmitting}
            onCancel={handleCancel}
            showCancel
          />
        </Form>
      </Card>

      {/* Product Selection Modal */}
      <Modal
        title={t('stockTaking.create.modal.title')}
        visible={showProductModal}
        onCancel={() => setShowProductModal(false)}
        onOk={handleConfirmProductSelection}
        okText={t('stockTaking.create.modal.confirm')}
        cancelText={t('stockTaking.create.modal.cancel')}
        width={800}
        bodyStyle={{ padding: 0 }}
      >
        <div className="product-modal-content">
          <div className="modal-toolbar">
            <Checkbox
              checked={
                modalSelectedKeys.length === inventoryItems.length && inventoryItems.length > 0
              }
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
              {t('stockTaking.create.modal.selectAll')} ({modalSelectedKeys.length}/
              {inventoryItems.length})
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
