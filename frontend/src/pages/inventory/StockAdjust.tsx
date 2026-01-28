import { useState, useEffect, useCallback, useMemo } from 'react'
import { z } from 'zod'
import {
  Card,
  Typography,
  Toast,
  Spin,
  Empty,
  Button,
  Descriptions,
  Tag,
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft } from '@douyinfe/semi-icons'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
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
import { useFormatters } from '@/hooks/useFormatters'
import { getInventory } from '@/api/inventory/inventory'
import { getWarehouses } from '@/api/warehouses/warehouses'
import { listProducts } from '@/api/products/products'
import type {
  HandlerInventoryItemResponse,
  HandlerWarehouseListResponse,
  HandlerProductListResponse,
} from '@/api/models'
import './StockAdjust.css'

const { Title, Text } = Typography

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
 * Handles both number and string values from API
 */
function formatQuantity(quantity?: number | string): string {
  if (quantity === undefined || quantity === null) return '-'
  const num = typeof quantity === 'string' ? parseFloat(quantity) : quantity
  if (isNaN(num)) return '-'
  return num.toFixed(2)
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
  const { t } = useTranslation(['inventory', 'common'])
  const { formatCurrency: formatCurrencyBase } = useFormatters()
  const inventoryApi = useMemo(() => getInventory(), [])
  const warehousesApi = useMemo(() => getWarehouses(), [])

  // Wrapper function to handle undefined values
  const formatCurrency = useCallback(
    (value?: number): string => (value !== undefined ? formatCurrencyBase(value) : '-'),
    [formatCurrencyBase]
  )

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

  // Adjustment reason options
  const ADJUSTMENT_REASONS = useMemo(
    () => [
      { value: 'STOCK_TAKE', label: t('adjust.reasons.STOCK_TAKE') },
      { value: 'DAMAGED', label: t('adjust.reasons.DAMAGED') },
      { value: 'LOST', label: t('adjust.reasons.LOST') },
      { value: 'CORRECTION', label: t('adjust.reasons.CORRECTION') },
      { value: 'INITIAL', label: t('adjust.reasons.INITIAL') },
      { value: 'OTHER', label: t('adjust.reasons.OTHER') },
    ],
    [t]
  )

  const { control, handleFormSubmit, isSubmitting, watch, setValue } =
    useFormWithValidation<StockAdjustFormData>({
      schema: stockAdjustSchema,
      defaultValues,
      successMessage: t('adjust.messages.success'),
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
      const response = await warehousesApi.listWarehouses({
        page: 1,
        page_size: 100,
        status: 'enabled',
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
      Toast.error(t('adjust.messages.fetchWarehouseError'))
    } finally {
      setLoadingWarehouses(false)
    }
  }, [warehousesApi])

  // Fetch products
  const fetchProducts = useCallback(async () => {
    setLoadingProducts(true)
    try {
      const response = await listProducts({
        page: 1,
        page_size: 100,
        status: 'active',
      })
      if (response.status === 200 && response.data.success && response.data.data) {
        const productList = response.data.data as HandlerProductListResponse[]
        setProducts(
          productList.map((p) => ({
            value: p.id || '',
            label: `${p.name || ''} (${p.code || p.id})`,
          }))
        )
      }
    } catch {
      Toast.error(t('adjust.messages.fetchProductError'))
    } finally {
      setLoadingProducts(false)
    }
  }, [t])

  // Fetch inventory item when warehouse and product are selected
  const fetchInventoryItem = useCallback(async () => {
    if (!warehouseId || !productId) {
      setInventoryItem(null)
      return
    }

    setLoadingInventory(true)
    try {
      const response = await inventoryApi.listInventories({
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
    const response = await inventoryApi.adjustStockInventory({
      warehouse_id: data.warehouse_id,
      product_id: data.product_id,
      actual_quantity: data.actual_quantity,
      reason: data.reason,
      source_type: data.source_type || 'MANUAL',
      source_id: data.source_id || undefined,
    })

    if (!response.success) {
      // Check for specific error codes and provide user-friendly messages
      const errorCode = response.error?.code
      if (errorCode === 'HAS_LOCKED_STOCK') {
        throw new Error(t('adjust.messages.hasLockedStock'))
      }
      throw new Error(response.error?.message || t('adjust.messages.error'))
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
            {t('adjust.back')}
          </Button>
          <Title heading={4} style={{ margin: 0 }}>
            {t('adjust.title')}
          </Title>
        </div>
      </div>

      <Card className="stock-adjust-card">
        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          {/* Selection Section */}
          <FormSection
            title={t('adjust.selection.title')}
            description={t('adjust.selection.description')}
          >
            <FormRow cols={2}>
              <SelectField
                name="warehouse_id"
                control={control}
                label={t('adjust.selection.warehouse')}
                placeholder={
                  loadingWarehouses
                    ? t('adjust.selection.loading')
                    : t('adjust.selection.warehousePlaceholder')
                }
                options={warehouses}
                required
                showSearch
                disabled={!!urlWarehouseId || loadingWarehouses}
              />
              <SelectField
                name="product_id"
                control={control}
                label={t('adjust.selection.product')}
                placeholder={
                  loadingProducts
                    ? t('adjust.selection.loading')
                    : t('adjust.selection.productPlaceholder')
                }
                options={products}
                required
                showSearch
                disabled={!!urlProductId || loadingProducts}
              />
            </FormRow>
          </FormSection>

          {/* Current Stock Info */}
          {warehouseId && productId && (
            <FormSection
              title={t('adjust.currentStock.title')}
              description={t('adjust.currentStock.description')}
            >
              {loadingInventory ? (
                <div className="loading-container">
                  <Spin />
                </div>
              ) : inventoryItem ? (
                <div className="current-stock-info">
                  <Descriptions
                    data={[
                      {
                        key: t('adjust.currentStock.totalQuantity'),
                        value: <Text strong>{formatQuantity(inventoryItem.total_quantity)}</Text>,
                      },
                      {
                        key: t('adjust.currentStock.availableQuantity'),
                        value: formatQuantity(inventoryItem.available_quantity),
                      },
                      {
                        key: t('adjust.currentStock.lockedQuantity'),
                        value: formatQuantity(inventoryItem.locked_quantity),
                      },
                      {
                        key: t('adjust.currentStock.unitCost'),
                        value: formatCurrency(inventoryItem.unit_cost),
                      },
                      {
                        key: t('adjust.currentStock.totalValue'),
                        value: formatCurrency(inventoryItem.total_value),
                      },
                    ]}
                  />
                  {inventoryItem.locked_quantity && inventoryItem.locked_quantity > 0 && (
                    <div className="stock-warning">
                      <Tag color="orange">
                        {t('adjust.currentStock.lockedWarning', {
                          quantity: formatQuantity(inventoryItem.locked_quantity),
                        })}
                      </Tag>
                    </div>
                  )}
                </div>
              ) : (
                <Empty
                  title={t('adjust.currentStock.noRecord')}
                  description={t('adjust.currentStock.noRecordDesc')}
                />
              )}
            </FormSection>
          )}

          {/* Adjustment Input */}
          <FormSection title={t('adjust.form.title')} description={t('adjust.form.description')}>
            <FormRow cols={2}>
              <NumberField
                name="actual_quantity"
                control={control}
                label={t('adjust.form.actualQuantity')}
                placeholder={t('adjust.form.actualQuantityPlaceholder')}
                min={0}
                precision={2}
                required
                helperText={t('adjust.form.actualQuantityHelper')}
              />
              <SelectField
                name="reason"
                control={control}
                label={t('adjust.form.reason')}
                placeholder={t('adjust.form.reasonPlaceholder')}
                options={ADJUSTMENT_REASONS}
                required
              />
            </FormRow>
            <TextAreaField
              name="source_id"
              control={control}
              label={t('adjust.form.remark')}
              placeholder={t('adjust.form.remarkPlaceholder')}
              rows={2}
              maxCount={100}
            />
          </FormSection>

          {/* Adjustment Preview */}
          {warehouseId && productId && actualQuantity !== undefined && (
            <FormSection
              title={t('adjust.preview.title')}
              description={t('adjust.preview.description')}
            >
              <div className="adjustment-preview">
                <div className="preview-row">
                  <div className="preview-item">
                    <Text type="tertiary">{t('adjust.preview.currentQuantity')}</Text>
                    <Text className="preview-value">{formatQuantity(currentQuantity)}</Text>
                  </div>
                  <div className="preview-arrow">→</div>
                  <div className="preview-item">
                    <Text type="tertiary">{t('adjust.preview.afterQuantity')}</Text>
                    <Text className="preview-value" strong>
                      {formatQuantity(actualQuantity)}
                    </Text>
                  </div>
                </div>
                <div className="preview-diff">
                  <Text type="tertiary">{t('adjust.preview.changeQuantity')}</Text>
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
                      {adjustmentType === 'increase'
                        ? t('adjust.preview.profit')
                        : t('adjust.preview.loss')}
                    </Tag>
                  )}
                </div>
              </div>
            </FormSection>
          )}

          <FormActions
            submitText={t('adjust.submit')}
            isSubmitting={isSubmitting}
            onCancel={handleCancel}
            showCancel
          />
        </Form>
      </Card>
    </Container>
  )
}
