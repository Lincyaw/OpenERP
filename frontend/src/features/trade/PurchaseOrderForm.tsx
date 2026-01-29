import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { z } from 'zod'
import { Card, Typography, Button, Input, Select, Toast, Space } from '@douyinfe/semi-ui-19'
import { IconPlus, IconSearch, IconEdit, IconSend } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { FormSection } from '@/components/common/form'
import { OrderItemsTable, OrderSummary, type ProductOption } from '@/components/common/order'
import {
  createPurchaseOrder,
  updatePurchaseOrder,
  confirmPurchaseOrder,
} from '@/api/purchase-orders/purchase-orders'
import { listSuppliers } from '@/api/suppliers/suppliers'
import { listProducts } from '@/api/products/products'
import { listWarehouses } from '@/api/warehouses/warehouses'
import type {
  HandlerSupplierListResponse,
  HandlerProductListResponse,
  HandlerWarehouseListResponse,
  HandlerPurchaseOrderResponse,
  HandlerCreatePurchaseOrderItemInput,
} from '@/api/models'
import { useI18n } from '@/hooks/useI18n'
import {
  useOrderCalculations,
  useOrderForm,
  createEmptyPurchaseOrderItem,
  type PurchaseOrderFormData,
  type PurchaseOrderItemFormData,
} from '@/hooks'
import { createScopedLogger } from '@/utils'
import './PurchaseOrderForm.css'

const log = createScopedLogger('PurchaseOrderForm')
const { Title, Text } = Typography

interface PurchaseOrderFormProps {
  /** Order ID for edit mode, undefined for create mode */
  orderId?: string
  /** Initial order data for edit mode */
  initialData?: HandlerPurchaseOrderResponse
}

/**
 * Purchase order form component for creating and editing purchase orders
 *
 * Features:
 * - Supplier search and selection
 * - Warehouse selection (optional, for receiving)
 * - Dynamic product item rows
 * - Real-time amount calculation
 * - Discount support
 * - Form validation with Zod
 */
export function PurchaseOrderForm({ orderId, initialData }: PurchaseOrderFormProps) {
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  // Product fetching will use listProducts directly
  const isEditMode = Boolean(orderId)

  // Form validation schema - full validation for confirm submit
  const orderFormSchema = useMemo(
    () =>
      z.object({
        supplier_id: z.string().min(1, t('orderForm.validation.supplierRequired')),
        supplier_name: z.string().min(1, t('orderForm.validation.supplierRequired')),
        warehouse_id: z.string().optional(),
        discount: z.number().min(0).max(100),
        remark: z.string().max(500).optional(),
        items: z
          .array(
            z.object({
              product_id: z.string().min(1),
              product_code: z.string().min(1),
              product_name: z.string().min(1),
              unit: z.string().min(1),
              unit_cost: z.number().positive(t('orderForm.validation.priceRequired')),
              quantity: z.number().positive(t('orderForm.validation.quantityRequired')),
            })
          )
          .min(1, t('orderForm.validation.itemsRequired')),
      }),
    [t]
  )

  // Draft validation schema - relaxed validation for items
  const draftFormSchema = useMemo(
    () =>
      z.object({
        supplier_id: z.string().min(1, t('orderForm.validation.supplierRequired')),
        supplier_name: z.string().min(1, t('orderForm.validation.supplierRequired')),
        warehouse_id: z.string().optional(),
        discount: z.number().min(0).max(100),
        remark: z.string().max(500).optional(),
        items: z
          .array(
            z.object({
              key: z.string(),
              product_id: z.string().optional(),
              product_code: z.string().optional(),
              product_name: z.string().optional(),
              unit: z.string().optional(),
              unit_cost: z.number().optional(),
              quantity: z.number().optional(),
              amount: z.number().optional(),
              remark: z.string().optional(),
            })
          )
          .min(1, t('orderForm.validation.itemsRequired')),
      }),
    [t]
  )

  // Initial form data
  const initialFormData: PurchaseOrderFormData = useMemo(
    () => ({
      supplier_id: '',
      supplier_name: '',
      warehouse_id: undefined,
      discount: 0,
      remark: '',
      items: [createEmptyPurchaseOrderItem()],
    }),
    []
  )

  // Use shared order form hook
  const {
    formData,
    setFormData,
    errors,
    setErrors,
    isSubmitting,
    setIsSubmitting,
    clearError,
    validateForm,
    resetForm,
    addItem,
    removeItem,
    updateItemWithAmount,
    handleDiscountChange,
    handleRemarkChange,
    handleWarehouseChange,
  } = useOrderForm<PurchaseOrderFormData>({
    initialData: initialFormData,
    schema: orderFormSchema,
    createEmptyItem: createEmptyPurchaseOrderItem,
  })

  // Use shared calculations hook
  const calculations = useOrderCalculations(formData.items, formData.discount)

  // Data for dropdowns
  const [suppliers, setSuppliers] = useState<HandlerSupplierListResponse[]>([])
  const [products, setProducts] = useState<HandlerProductListResponse[]>([])
  const [warehouses, setWarehouses] = useState<HandlerWarehouseListResponse[]>([])
  const [suppliersLoading, setSuppliersLoading] = useState(false)
  const [productsLoading, setProductsLoading] = useState(false)
  const [warehousesLoading, setWarehousesLoading] = useState(false)

  // Search state
  const [supplierSearch, setSupplierSearch] = useState('')
  const [productSearch, setProductSearch] = useState('')

  // Track if default warehouse has been set (to avoid re-setting on every render)
  const hasSetDefaultWarehouse = useRef(false)

  // Fetch suppliers
  const fetchSuppliers = useCallback(async (search?: string, signal?: AbortSignal) => {
    setSuppliersLoading(true)
    try {
      const response = await listSuppliers(
        { page_size: 50, search: search || undefined, status: 'active' },
        { signal }
      )
      if (response.status === 200 && response.data.success && response.data.data) {
        setSuppliers(response.data.data)
      } else if (response.status !== 200 || !response.data.success) {
        log.error('Failed to fetch suppliers', response.data.error)
      }
    } catch (error) {
      if (error instanceof Error && error.name === 'CanceledError') return
      log.error('Error fetching suppliers', error)
    } finally {
      setSuppliersLoading(false)
    }
  }, [])

  // Fetch products
  const fetchProducts = useCallback(async (search?: string, signal?: AbortSignal) => {
    setProductsLoading(true)
    try {
      const response = await listProducts(
        { page_size: 50, search: search || undefined, status: 'active' },
        { signal }
      )
      if (response.status === 200 && response.data.success && response.data.data) {
        setProducts(response.data.data)
      } else if (response.status !== 200 || !response.data.success) {
        log.error('Failed to fetch products', response.data.error)
      }
    } catch (error) {
      if (error instanceof Error && error.name === 'CanceledError') return
      log.error('Error fetching products', error)
    } finally {
      setProductsLoading(false)
    }
  }, [])

  // Fetch warehouses
  const fetchWarehouses = useCallback(
    async (signal?: AbortSignal) => {
      setWarehousesLoading(true)
      try {
        const response = await listWarehouses({ page_size: 100, status: 'enabled' }, { signal })
        if (response.status === 200 && response.data.success && response.data.data) {
          setWarehouses(response.data.data)
          // Set default warehouse only once on initial load (not edit mode)
          if (!isEditMode && !hasSetDefaultWarehouse.current) {
            const defaultWarehouse = response.data.data.find(
              (w: HandlerWarehouseListResponse) => w.is_default
            )
            if (defaultWarehouse?.id) {
              hasSetDefaultWarehouse.current = true
              setFormData((prev) => ({ ...prev, warehouse_id: defaultWarehouse.id }))
            }
          }
        } else if (response.status !== 200 || !response.data.success) {
          log.error('Failed to fetch warehouses', response.data.error)
        }
      } catch (error) {
        if (error instanceof Error && error.name === 'CanceledError') return
        log.error('Error fetching warehouses', error)
      } finally {
        setWarehousesLoading(false)
      }
    },
    [isEditMode, setFormData]
  )

  // Initial data loading
  useEffect(() => {
    const abortController = new AbortController()
    fetchSuppliers(undefined, abortController.signal)
    fetchProducts(undefined, abortController.signal)
    fetchWarehouses(abortController.signal)
    return () => abortController.abort()
  }, [fetchSuppliers, fetchProducts, fetchWarehouses])

  // Debounced supplier search
  useEffect(() => {
    if (!supplierSearch) return
    const abortController = new AbortController()
    const timer = setTimeout(() => fetchSuppliers(supplierSearch, abortController.signal), 300)
    return () => {
      clearTimeout(timer)
      abortController.abort()
    }
  }, [supplierSearch, fetchSuppliers])

  // Debounced product search
  useEffect(() => {
    if (!productSearch) return
    const abortController = new AbortController()
    const timer = setTimeout(() => fetchProducts(productSearch, abortController.signal), 300)
    return () => {
      clearTimeout(timer)
      abortController.abort()
    }
  }, [productSearch, fetchProducts])

  // Load initial data for edit mode
  useEffect(() => {
    if (initialData) {
      const totalAmount = initialData.total_amount || 0
      const discountAmount = initialData.discount_amount || 0
      const subtotal = totalAmount + discountAmount
      const discountPercent = subtotal > 0 ? (discountAmount / subtotal) * 100 : 0

      setFormData({
        supplier_id: initialData.supplier_id || '',
        supplier_name: initialData.supplier_name || '',
        warehouse_id: initialData.warehouse_id || undefined,
        discount: discountPercent,
        remark: initialData.remark || '',
        items: initialData.items?.map((item) => ({
          key: item.id || `item-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
          product_id: item.product_id || '',
          product_code: item.product_code || '',
          product_name: item.product_name || '',
          unit: item.unit || '',
          unit_cost: item.unit_cost || 0,
          quantity: item.ordered_quantity || 1,
          amount: (item.unit_cost || 0) * (item.ordered_quantity || 1),
          remark: item.remark || '',
        })) || [createEmptyPurchaseOrderItem()],
      })
    }
  }, [initialData, setFormData])

  // Handle supplier selection
  const handleSupplierChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const supplierId = typeof value === 'string' ? value : ''
      const supplier = suppliers.find((s) => s.id === supplierId)
      setFormData((prev) => ({
        ...prev,
        supplier_id: supplierId,
        supplier_name: supplier?.name || '',
      }))
      clearError('supplier_id')
    },
    [suppliers, setFormData, clearError]
  )

  // Handle product selection for an item
  const handleProductSelect = useCallback(
    (itemKey: string, _productId: string, productOption: ProductOption) => {
      const product = products.find((p) => p.id === productOption.value)
      if (!product) return

      setFormData((prev) => ({
        ...prev,
        items: prev.items.map((item) => {
          if (item.key !== itemKey) return item
          const unitCost = product.purchase_price || product.selling_price || 0
          return {
            ...item,
            product_id: product.id || '',
            product_code: product.code || '',
            product_name: product.name || '',
            unit: product.unit || '',
            unit_cost: unitCost,
            amount: unitCost * item.quantity,
          }
        }),
      }))
      clearError(`items.${itemKey}.product_id`)
    },
    [products, setFormData, clearError]
  )

  // Handle quantity change
  const handleQuantityChange = useCallback(
    (itemKey: string, quantity: number | string | undefined) => {
      const qty = typeof quantity === 'number' ? quantity : parseFloat(String(quantity)) || 0
      updateItemWithAmount(itemKey, { quantity: qty }, 'unit_cost')
    },
    [updateItemWithAmount]
  )

  // Handle unit cost change
  const handleUnitCostChange = useCallback(
    (itemKey: string, cost: number | string | undefined) => {
      const unitCost = typeof cost === 'number' ? cost : parseFloat(String(cost)) || 0
      updateItemWithAmount(
        itemKey,
        { unit_cost: unitCost } as Partial<PurchaseOrderItemFormData>,
        'unit_cost'
      )
    },
    [updateItemWithAmount]
  )

  // Handle item remark change
  const handleItemRemarkChange = useCallback(
    (itemKey: string, remark: string) => {
      setFormData((prev) => ({
        ...prev,
        items: prev.items.map((item) => (item.key !== itemKey ? item : { ...item, remark })),
      }))
    },
    [setFormData]
  )

  // Validate form with draft schema (relaxed validation)
  const validateDraft = useCallback((): boolean => {
    const result = draftFormSchema.safeParse(formData)
    if (!result.success) {
      const newErrors: Record<string, string> = {}
      result.error.issues.forEach((issue) => {
        const path = issue.path.join('.')
        newErrors[path] = issue.message
      })
      setErrors(newErrors)
      return false
    }
    setErrors({})
    return true
  }, [formData, draftFormSchema, setErrors])

  // Handle save as draft (create or update without confirming)
  const handleSaveDraft = useCallback(async () => {
    if (!validateDraft()) {
      Toast.error(t('orderForm.validation.supplierRequired'))
      return
    }

    setIsSubmitting(true)
    try {
      const validItems = formData.items.filter((item) => item.product_id)
      const itemsPayload: HandlerCreatePurchaseOrderItemInput[] = validItems.map((item) => ({
        product_id: item.product_id,
        product_code: item.product_code,
        product_name: item.product_name,
        unit: item.unit,
        unit_cost: item.unit_cost,
        quantity: item.quantity,
        remark: item.remark || undefined,
      }))

      if (isEditMode && orderId) {
        const response = await updatePurchaseOrder(orderId, {
          warehouse_id: formData.warehouse_id,
          discount: formData.discount,
          remark: formData.remark || undefined,
        })
        if (response.status !== 200 || !response.data.success) {
          throw new Error(
            (response.data.error as { message?: string })?.message ||
              t('orderForm.messages.updateError')
          )
        }
        Toast.success(t('orderForm.messages.saveDraftSuccess'))
      } else {
        const response = await createPurchaseOrder({
          supplier_id: formData.supplier_id,
          supplier_name: formData.supplier_name,
          warehouse_id: formData.warehouse_id,
          discount: formData.discount,
          remark: formData.remark || undefined,
          items: itemsPayload,
        })
        if (response.status !== 201 || !response.data.success) {
          throw new Error(
            (response.data.error as { message?: string })?.message ||
              t('orderForm.messages.createError')
          )
        }
        Toast.success(t('orderForm.messages.saveDraftSuccess'))
      }
      if (!isEditMode) {
        resetForm()
      }
      navigate('/trade/purchase')
    } catch (error) {
      Toast.error(error instanceof Error ? error.message : t('orderForm.messages.saveDraftError'))
    } finally {
      setIsSubmitting(false)
    }
  }, [formData, isEditMode, orderId, navigate, validateDraft, t, setIsSubmitting, resetForm])

  // Handle confirm and submit (full validation + create/update + confirm)
  const handleConfirmSubmit = useCallback(async () => {
    if (!validateForm()) {
      Toast.error(t('orderForm.validation.itemsRequired'))
      return
    }

    setIsSubmitting(true)
    try {
      const validItems = formData.items.filter((item) => item.product_id)
      const itemsPayload: HandlerCreatePurchaseOrderItemInput[] = validItems.map((item) => ({
        product_id: item.product_id,
        product_code: item.product_code,
        product_name: item.product_name,
        unit: item.unit,
        unit_cost: item.unit_cost,
        quantity: item.quantity,
        remark: item.remark || undefined,
      }))

      let orderIdToConfirm = orderId
      let orderCreated = false

      if (isEditMode && orderId) {
        // Update existing draft order
        const response = await updatePurchaseOrder(orderId, {
          warehouse_id: formData.warehouse_id,
          discount: formData.discount,
          remark: formData.remark || undefined,
        })
        if (response.status !== 200 || !response.data.success) {
          throw new Error(
            (response.data.error as { message?: string })?.message ||
              t('orderForm.messages.updateError')
          )
        }
      } else {
        // Create new order first
        const response = await createPurchaseOrder({
          supplier_id: formData.supplier_id,
          supplier_name: formData.supplier_name,
          warehouse_id: formData.warehouse_id,
          discount: formData.discount,
          remark: formData.remark || undefined,
          items: itemsPayload,
        })
        if (response.status !== 201 || !response.data.success) {
          throw new Error(
            (response.data.error as { message?: string })?.message ||
              t('orderForm.messages.createError')
          )
        }
        orderIdToConfirm = response.data.data?.id
        orderCreated = true
      }

      // Confirm the order
      if (orderIdToConfirm) {
        try {
          const confirmResponse = await confirmPurchaseOrder(orderIdToConfirm, {
            warehouse_id: formData.warehouse_id,
          })
          if (confirmResponse.status !== 200 || !confirmResponse.data.success) {
            // Order was created but confirmation failed - inform user
            if (orderCreated) {
              Toast.warning(t('orderForm.messages.orderCreatedNotConfirmed'))
              if (!isEditMode) {
                resetForm()
              }
              navigate('/trade/purchase')
              return
            }
            throw new Error(
              (confirmResponse.data.error as { message?: string })?.message ||
                t('orderForm.messages.confirmSubmitError')
            )
          }
        } catch (confirmError) {
          // If order was just created and confirmation fails, inform user
          if (orderCreated) {
            Toast.warning(t('orderForm.messages.orderCreatedNotConfirmed'))
            if (!isEditMode) {
              resetForm()
            }
            navigate('/trade/purchase')
            return
          }
          throw confirmError
        }
      }

      Toast.success(t('orderForm.messages.confirmSubmitSuccess'))
      if (!isEditMode) {
        resetForm()
      }
      navigate('/trade/purchase')
    } catch (error) {
      Toast.error(
        error instanceof Error ? error.message : t('orderForm.messages.confirmSubmitError')
      )
    } finally {
      setIsSubmitting(false)
    }
  }, [formData, isEditMode, orderId, navigate, validateForm, t, setIsSubmitting, resetForm])

  // Handle cancel
  const handleCancel = useCallback(() => navigate('/trade/purchase'), [navigate])

  // Select options
  const supplierOptions = useMemo(
    () =>
      suppliers.map((s) => ({ value: s.id || '', label: s.name || s.code || '', extra: s.code })),
    [suppliers]
  )

  const warehouseOptions = useMemo(
    () =>
      warehouses.map((w) => ({
        value: w.id || '',
        label: w.name || w.code || '',
        extra: w.is_default ? `(${t('common.defaultWarehouse')})` : undefined,
      })),
    [warehouses, t]
  )

  const productOptions: ProductOption[] = useMemo(
    () =>
      products.map((p) => ({
        value: p.id || '',
        label: `${p.code} - ${p.name}`,
        code: p.code,
        name: p.name,
        unit: p.unit,
        price: p.purchase_price || p.selling_price,
      })),
    [products]
  )

  return (
    <Container size="lg" className="purchase-order-form-page">
      <Card className="purchase-order-form-card">
        <div className="purchase-order-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {isEditMode ? t('orderForm.editPurchaseTitle') : t('orderForm.createPurchaseTitle')}
          </Title>
        </div>

        {/* Basic Information Section */}
        <FormSection
          title={t('orderForm.basicInfo.title')}
          subtitle={t('orderForm.basicInfo.subtitle')}
          required
        >
          <div className="form-row">
            <div className="form-field">
              <label className="form-label required">{t('orderForm.basicInfo.supplier')}</label>
              <Select
                value={formData.supplier_id || undefined}
                placeholder={t('orderForm.basicInfo.supplierPlaceholder')}
                onChange={handleSupplierChange}
                optionList={supplierOptions}
                filter
                remote
                onSearch={setSupplierSearch}
                loading={suppliersLoading}
                style={{ width: '100%' }}
                prefix={<IconSearch />}
                validateStatus={errors.supplier_id ? 'error' : undefined}
                disabled={isEditMode}
                renderSelectedItem={(option: { label?: string; extra?: string }) => (
                  <span>
                    {option.label}
                    {option.extra && <Text type="tertiary"> ({option.extra})</Text>}
                  </span>
                )}
              />
              {errors.supplier_id && (
                <Text type="danger" size="small">
                  {errors.supplier_id}
                </Text>
              )}
            </div>
            <div className="form-field">
              <label className="form-label">{t('orderForm.basicInfo.receiveWarehouse')}</label>
              <Select
                value={formData.warehouse_id || undefined}
                placeholder={t('orderForm.basicInfo.warehousePlaceholder')}
                onChange={handleWarehouseChange}
                optionList={warehouseOptions}
                loading={warehousesLoading}
                style={{ width: '100%' }}
                showClear
                renderSelectedItem={(option: { label?: string; extra?: string }) => (
                  <span>
                    {option.label}
                    {option.extra && <Text type="tertiary"> {option.extra}</Text>}
                  </span>
                )}
              />
            </div>
          </div>
        </FormSection>

        {/* Order Items Section */}
        <FormSection
          title={t('orderForm.items.title')}
          subtitle={t('orderForm.items.subtitle')}
          required
        >
          <div className="section-header">
            <div />
            <Button icon={<IconPlus />} theme="light" onClick={addItem}>
              {t('orderForm.items.addProduct')}
            </Button>
          </div>
          {errors.items && (
            <Text type="danger" size="small" className="items-error">
              {errors.items}
            </Text>
          )}
          <OrderItemsTable
            items={formData.items}
            productOptions={productOptions}
            productsLoading={productsLoading}
            onProductSearch={setProductSearch}
            onProductSelect={handleProductSelect}
            onQuantityChange={handleQuantityChange}
            onPriceChange={handleUnitCostChange}
            onItemRemarkChange={handleItemRemarkChange}
            onRemoveItem={removeItem}
            t={t}
            orderType="purchase"
            className="items-table"
          />
        </FormSection>

        {/* Summary Section */}
        <FormSection title={t('orderForm.summary.title')}>
          <OrderSummary
            calculations={calculations}
            discount={formData.discount}
            onDiscountChange={handleDiscountChange}
            t={t}
          />
        </FormSection>

        {/* Remark Section */}
        <FormSection title={t('orderForm.remark.title')} subtitle={t('orderForm.remark.subtitle')}>
          <div className="form-field">
            <Input
              value={formData.remark}
              onChange={handleRemarkChange}
              placeholder={t('orderForm.basicInfo.remarkPlaceholder')}
              maxLength={500}
              showClear
            />
          </div>
        </FormSection>

        {/* Form Actions - Dual Save Buttons */}
        <div className="form-actions">
          <Space>
            <Button onClick={handleCancel} disabled={isSubmitting}>
              {t('orderForm.actions.cancel')}
            </Button>
            <Button
              icon={<IconEdit />}
              onClick={handleSaveDraft}
              loading={isSubmitting}
              disabled={isSubmitting}
            >
              {t('orderForm.actions.saveDraft')}
            </Button>
            <Button
              type="primary"
              theme="solid"
              icon={<IconSend />}
              onClick={handleConfirmSubmit}
              loading={isSubmitting}
              disabled={isSubmitting}
            >
              {t('orderForm.actions.confirmSubmit')}
            </Button>
          </Space>
        </div>
      </Card>
    </Container>
  )
}

export default PurchaseOrderForm
