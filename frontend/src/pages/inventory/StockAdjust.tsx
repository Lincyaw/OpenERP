import { useState, useEffect, useCallback, useMemo } from 'react'
import { z } from 'zod'
import { Card, Typography, Toast, Spin, Empty, Button, Descriptions, Tag } from '@douyinfe/semi-ui'
import { IconArrowLeft } from '@douyinfe/semi-icons'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Form,
  FormActions,
  FormSection,
  FormRow,
  NumberField,
  SelectField,
  TextAreaField,
  useFormWithValidation,
  validationMessages,
} from '@/components/common/form'
import { Container } from '@/components/common/layout'
import { getInventory } from '@/api/inventory/inventory'
import { getWarehouses } from '@/api/warehouses/warehouses'
import { getProducts } from '@/api/products/products'
import type {
  HandlerInventoryItemResponse,
  HandlerWarehouseListResponse,
  HandlerProductListResponse,
} from '@/api/models'
import './StockAdjust.css'

const { Title, Text } = Typography

// Adjustment reason options
const ADJUSTMENT_REASONS = [
  { value: 'STOCK_TAKE', label: '盘点调整' },
  { value: 'DAMAGED', label: '破损报废' },
  { value: 'LOST', label: '丢失' },
  { value: 'CORRECTION', label: '数据校正' },
  { value: 'INITIAL', label: '期初录入' },
  { value: 'OTHER', label: '其他' },
]

// Form validation schema
const stockAdjustSchema = z.object({
  warehouse_id: z.string().min(1, validationMessages.required),
  product_id: z.string().min(1, validationMessages.required),
  actual_quantity: z.number().min(0, validationMessages.nonNegative),
  reason: z
    .string()
    .min(1, validationMessages.required)
    .max(255, validationMessages.maxLength(255)),
  source_type: z.string().optional(),
  source_id: z.string().max(100, validationMessages.maxLength(100)).optional(),
})

type StockAdjustFormData = z.infer<typeof stockAdjustSchema>

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
 * Stock Adjustment Page
 *
 * Features:
 * - Select warehouse and product (or pre-fill from URL params)
 * - Display current stock information
 * - Enter actual quantity and reason
 * - Preview adjustment (difference)
 * - Submit adjustment to API
 */
export default function StockAdjustPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const inventoryApi = useMemo(() => getInventory(), [])
  const warehousesApi = useMemo(() => getWarehouses(), [])
  const productsApi = useMemo(() => getProducts(), [])

  // URL params
  const urlWarehouseId = searchParams.get('warehouse_id')
  const urlProductId = searchParams.get('product_id')

  // State for dropdowns
  const [warehouses, setWarehouses] = useState<Array<{ value: string; label: string }>>([])
  const [products, setProducts] = useState<Array<{ value: string; label: string }>>([])
  const [loadingWarehouses, setLoadingWarehouses] = useState(false)
  const [loadingProducts, setLoadingProducts] = useState(false)

  // State for current inventory
  const [inventoryItem, setInventoryItem] = useState<HandlerInventoryItemResponse | null>(null)
  const [loadingInventory, setLoadingInventory] = useState(false)

  // State for preview
  const [actualQuantity, setActualQuantity] = useState<number | undefined>(undefined)

  // Default form values
  const defaultValues: Partial<StockAdjustFormData> = useMemo(
    () => ({
      warehouse_id: urlWarehouseId || '',
      product_id: urlProductId || '',
      actual_quantity: 0,
      reason: '',
      source_type: 'MANUAL',
      source_id: '',
    }),
    [urlWarehouseId, urlProductId]
  )

  const { control, handleFormSubmit, isSubmitting, watch, setValue } =
    useFormWithValidation<StockAdjustFormData>({
      schema: stockAdjustSchema,
      defaultValues,
      successMessage: '库存调整成功',
      onSuccess: () => {
        navigate('/inventory/stock')
      },
    })

  // Watch form values
  const warehouseId = watch('warehouse_id')
  const productId = watch('product_id')
  const watchedActualQuantity = watch('actual_quantity')

  // Update actual quantity for preview
  useEffect(() => {
    setActualQuantity(watchedActualQuantity)
  }, [watchedActualQuantity])

  // Fetch warehouses
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
            label: `${w.name || w.code || w.id}`,
          }))
        )
      }
    } catch {
      Toast.error('获取仓库列表失败')
    } finally {
      setLoadingWarehouses(false)
    }
  }, [warehousesApi])

  // Fetch products
  const fetchProducts = useCallback(async () => {
    setLoadingProducts(true)
    try {
      const response = await productsApi.getCatalogProducts({
        page_size: 500,
        status: 'active',
      })
      if (response.success && response.data) {
        const productList = response.data as HandlerProductListResponse[]
        setProducts(
          productList.map((p) => ({
            value: p.id || '',
            label: `${p.name || ''} (${p.code || p.id})`,
          }))
        )
      }
    } catch {
      Toast.error('获取商品列表失败')
    } finally {
      setLoadingProducts(false)
    }
  }, [productsApi])

  // Fetch inventory item when warehouse and product are selected
  const fetchInventoryItem = useCallback(async () => {
    if (!warehouseId || !productId) {
      setInventoryItem(null)
      return
    }

    setLoadingInventory(true)
    try {
      const response = await inventoryApi.getInventoryItemsLookup({
        warehouse_id: warehouseId,
        product_id: productId,
      })
      if (response.success && response.data) {
        const item = response.data as HandlerInventoryItemResponse
        setInventoryItem(item)
        // Set actual quantity to current total if not already set
        if (actualQuantity === undefined || actualQuantity === 0) {
          setValue('actual_quantity', item.total_quantity || 0)
          setActualQuantity(item.total_quantity || 0)
        }
      } else {
        setInventoryItem(null)
      }
    } catch {
      // No inventory record found - this is OK, it may be initial setup
      setInventoryItem(null)
    } finally {
      setLoadingInventory(false)
    }
  }, [warehouseId, productId, inventoryApi, actualQuantity, setValue])

  // Fetch warehouses and products on mount
  useEffect(() => {
    fetchWarehouses()
    fetchProducts()
  }, [fetchWarehouses, fetchProducts])

  // Fetch inventory when warehouse/product change
  useEffect(() => {
    if (warehouseId && productId) {
      fetchInventoryItem()
    }
  }, [warehouseId, productId, fetchInventoryItem])

  // Handle form submission
  const onSubmit = async (data: StockAdjustFormData) => {
    const response = await inventoryApi.postInventoryStockAdjust({
      warehouse_id: data.warehouse_id,
      product_id: data.product_id,
      actual_quantity: data.actual_quantity,
      reason: data.reason,
      source_type: data.source_type || 'MANUAL',
      source_id: data.source_id || undefined,
    })

    if (!response.success) {
      throw new Error(response.error?.message || '调整失败')
    }
  }

  const handleCancel = () => {
    navigate(-1)
  }

  const handleBack = () => {
    navigate(-1)
  }

  // Calculate adjustment preview
  const currentQuantity = inventoryItem?.total_quantity || 0
  const adjustmentDiff =
    actualQuantity !== undefined && actualQuantity !== null ? actualQuantity - currentQuantity : 0
  const adjustmentType = adjustmentDiff > 0 ? 'increase' : adjustmentDiff < 0 ? 'decrease' : 'none'

  return (
    <Container size="md" className="stock-adjust-page">
      {/* Header */}
      <div className="stock-adjust-header">
        <div className="header-left">
          <Button icon={<IconArrowLeft />} theme="borderless" onClick={handleBack}>
            返回
          </Button>
          <Title heading={4} style={{ margin: 0 }}>
            库存调整
          </Title>
        </div>
      </div>

      <Card className="stock-adjust-card">
        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          {/* Selection Section */}
          <FormSection title="选择库存" description="选择要调整的仓库和商品">
            <FormRow cols={2}>
              <SelectField
                name="warehouse_id"
                control={control}
                label="仓库"
                placeholder={loadingWarehouses ? '加载中...' : '请选择仓库'}
                options={warehouses}
                required
                showSearch
                disabled={!!urlWarehouseId || loadingWarehouses}
              />
              <SelectField
                name="product_id"
                control={control}
                label="商品"
                placeholder={loadingProducts ? '加载中...' : '请选择商品'}
                options={products}
                required
                showSearch
                disabled={!!urlProductId || loadingProducts}
              />
            </FormRow>
          </FormSection>

          {/* Current Stock Info */}
          {warehouseId && productId && (
            <FormSection title="当前库存" description="选定仓库和商品的当前库存信息">
              {loadingInventory ? (
                <div className="loading-container">
                  <Spin />
                </div>
              ) : inventoryItem ? (
                <div className="current-stock-info">
                  <Descriptions
                    data={[
                      {
                        key: '总数量',
                        value: <Text strong>{formatQuantity(inventoryItem.total_quantity)}</Text>,
                      },
                      {
                        key: '可用数量',
                        value: formatQuantity(inventoryItem.available_quantity),
                      },
                      {
                        key: '锁定数量',
                        value: formatQuantity(inventoryItem.locked_quantity),
                      },
                      { key: '单位成本', value: formatCurrency(inventoryItem.unit_cost) },
                      { key: '库存总值', value: formatCurrency(inventoryItem.total_value) },
                    ]}
                  />
                  {inventoryItem.locked_quantity && inventoryItem.locked_quantity > 0 && (
                    <div className="stock-warning">
                      <Tag color="orange">
                        注意：当前有 {formatQuantity(inventoryItem.locked_quantity)} 数量被锁定
                      </Tag>
                    </div>
                  )}
                </div>
              ) : (
                <Empty
                  title="暂无库存记录"
                  description="该仓库和商品组合尚无库存记录，调整后将创建新记录"
                />
              )}
            </FormSection>
          )}

          {/* Adjustment Input */}
          <FormSection title="调整信息" description="输入实际数量和调整原因">
            <FormRow cols={2}>
              <NumberField
                name="actual_quantity"
                control={control}
                label="实际数量"
                placeholder="请输入实际盘点数量"
                min={0}
                precision={2}
                required
                helperText="系统库存将调整为此数量"
              />
              <SelectField
                name="reason"
                control={control}
                label="调整原因"
                placeholder="请选择调整原因"
                options={ADJUSTMENT_REASONS}
                required
              />
            </FormRow>
            <TextAreaField
              name="source_id"
              control={control}
              label="备注"
              placeholder="请输入调整备注信息（可选）"
              rows={2}
              maxCount={100}
            />
          </FormSection>

          {/* Adjustment Preview */}
          {warehouseId && productId && actualQuantity !== undefined && (
            <FormSection title="调整预览" description="确认调整后的库存变化">
              <div className="adjustment-preview">
                <div className="preview-row">
                  <div className="preview-item">
                    <Text type="tertiary">当前数量</Text>
                    <Text className="preview-value">{formatQuantity(currentQuantity)}</Text>
                  </div>
                  <div className="preview-arrow">→</div>
                  <div className="preview-item">
                    <Text type="tertiary">调整后数量</Text>
                    <Text className="preview-value" strong>
                      {formatQuantity(actualQuantity)}
                    </Text>
                  </div>
                </div>
                <div className="preview-diff">
                  <Text type="tertiary">变动数量：</Text>
                  <Text
                    className={`diff-value ${adjustmentType === 'increase' ? 'diff-positive' : adjustmentType === 'decrease' ? 'diff-negative' : ''}`}
                  >
                    {adjustmentDiff > 0 ? '+' : ''}
                    {formatQuantity(adjustmentDiff)}
                  </Text>
                  {adjustmentType !== 'none' && (
                    <Tag
                      color={adjustmentType === 'increase' ? 'green' : 'red'}
                      className="diff-tag"
                    >
                      {adjustmentType === 'increase' ? '盘盈' : '盘亏'}
                    </Tag>
                  )}
                </div>
              </div>
            </FormSection>
          )}

          <FormActions
            submitText="确认调整"
            isSubmitting={isSubmitting}
            onCancel={handleCancel}
            showCancel
          />
        </Form>
      </Card>
    </Container>
  )
}
