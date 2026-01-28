import { useState, useEffect, useMemo, useCallback, useRef } from 'react'
import { z } from 'zod'
import { Card, Typography, Button, Input, Select, Toast, Space } from '@douyinfe/semi-ui-19'
import { IconPlus, IconSearch } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { OrderItemsTable, OrderSummary, type ProductOption } from '@/components/common/order'
import { getSalesOrders } from '@/api/sales-orders/sales-orders'
import { listCustomers } from '@/api/customers/customers'
import { getProducts } from '@/api/products/products'
import { getWarehouses } from '@/api/warehouses/warehouses'
import type {
  HandlerCustomerListResponse,
  HandlerProductListResponse,
  HandlerWarehouseListResponse,
  HandlerSalesOrderResponse,
  HandlerCreateSalesOrderItemInput,
} from '@/api/models'
import { useI18n } from '@/hooks/useI18n'
import {
  useOrderCalculations,
  useOrderForm,
  createEmptySalesOrderItem,
  type SalesOrderFormData,
  type SalesOrderItemFormData,
} from '@/hooks'
import { createScopedLogger } from '@/utils'
import './SalesOrderForm.css'

const log = createScopedLogger('SalesOrderForm')
const { Title, Text } = Typography

interface SalesOrderFormProps {
  /** Order ID for edit mode, undefined for create mode */
  orderId?: string
  /** Initial order data for edit mode */
  initialData?: HandlerSalesOrderResponse
}

/**
 * Sales order form component for creating and editing sales orders
 *
 * Features:
 * - Customer search and selection
 * - Warehouse selection (optional)
 * - Dynamic product item rows
 * - Real-time amount calculation
 * - Discount support
 * - Form validation with Zod
 */
export function SalesOrderForm({ orderId, initialData }: SalesOrderFormProps) {
  const navigate = useNavigate()
  const { t } = useI18n({ ns: 'trade' })
  const salesOrderApi = useMemo(() => getSalesOrders(), [])
  const productApi = useMemo(() => getProducts(), [])
  const warehouseApi = useMemo(() => getWarehouses(), [])
  const isEditMode = Boolean(orderId)

  // Form validation schema
  const orderFormSchema = useMemo(
    () =>
      z.object({
        customer_id: z.string().min(1, t('orderForm.validation.customerRequired')),
        customer_name: z.string().min(1, t('orderForm.validation.customerRequired')),
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
              unit_price: z.number().positive(t('orderForm.validation.priceRequired')),
              quantity: z.number().positive(t('orderForm.validation.quantityRequired')),
            })
          )
          .min(1, t('orderForm.validation.itemsRequired')),
      }),
    [t]
  )

  // Initial form data
  const initialFormData: SalesOrderFormData = useMemo(
    () => ({
      customer_id: '',
      customer_name: '',
      warehouse_id: undefined,
      discount: 0,
      remark: '',
      items: [createEmptySalesOrderItem()],
    }),
    []
  )

  // Use shared order form hook
  const {
    formData,
    setFormData,
    errors,
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
  } = useOrderForm<SalesOrderFormData>({
    initialData: initialFormData,
    schema: orderFormSchema,
    createEmptyItem: createEmptySalesOrderItem,
  })

  // Use shared calculations hook
  const calculations = useOrderCalculations(formData.items, formData.discount)

  // Data for dropdowns
  const [customers, setCustomers] = useState<HandlerCustomerListResponse[]>([])
  const [products, setProducts] = useState<HandlerProductListResponse[]>([])
  const [warehouses, setWarehouses] = useState<HandlerWarehouseListResponse[]>([])
  const [customersLoading, setCustomersLoading] = useState(false)
  const [productsLoading, setProductsLoading] = useState(false)
  const [warehousesLoading, setWarehousesLoading] = useState(false)

  // Search state
  const [customerSearch, setCustomerSearch] = useState('')
  const [productSearch, setProductSearch] = useState('')

  // Track if default warehouse has been set (to avoid re-setting on every render)
  const hasSetDefaultWarehouse = useRef(false)

  // Fetch customers
  const fetchCustomers = useCallback(async (search?: string, signal?: AbortSignal) => {
    setCustomersLoading(true)
    try {
      const response = await listCustomers(
        { page_size: 50, search: search || undefined, status: 'active' },
        { signal }
      )
      if (response.status === 200 && response.data.success && response.data.data) {
        setCustomers(response.data.data)
      } else if (!response.data.success) {
        log.error('Failed to fetch customers', response.data.error)
      }
    } catch (error) {
      if (error instanceof Error && error.name === 'CanceledError') return
      log.error('Error fetching customers', error)
    } finally {
      setCustomersLoading(false)
    }
  }, [])

  // Fetch products
  const fetchProducts = useCallback(
    async (search?: string, signal?: AbortSignal) => {
      setProductsLoading(true)
      try {
        const response = await productApi.listProducts(
          { page_size: 50, search: search || undefined, status: 'active' },
          { signal }
        )
        if (response.success && response.data) {
          setProducts(response.data)
        } else if (!response.success) {
          log.error('Failed to fetch products', response.error)
        }
      } catch (error) {
        if (error instanceof Error && error.name === 'CanceledError') return
        log.error('Error fetching products', error)
      } finally {
        setProductsLoading(false)
      }
    },
    [productApi]
  )

  // Fetch warehouses
  const fetchWarehouses = useCallback(
    async (signal?: AbortSignal) => {
      setWarehousesLoading(true)
      try {
        const response = await warehouseApi.listWarehouses(
          { page_size: 100, status: 'enabled' },
          { signal }
        )
        if (response.success && response.data) {
          setWarehouses(response.data)
          // Set default warehouse only once on initial load (not edit mode)
          if (!isEditMode && !hasSetDefaultWarehouse.current) {
            const defaultWarehouse = response.data.find((w) => w.is_default)
            if (defaultWarehouse?.id) {
              hasSetDefaultWarehouse.current = true
              setFormData((prev) => ({ ...prev, warehouse_id: defaultWarehouse.id }))
            }
          }
        } else if (!response.success) {
          log.error('Failed to fetch warehouses', response.error)
        }
      } catch (error) {
        if (error instanceof Error && error.name === 'CanceledError') return
        log.error('Error fetching warehouses', error)
      } finally {
        setWarehousesLoading(false)
      }
    },
    [warehouseApi, isEditMode, setFormData]
  )

  // Initial data loading
  useEffect(() => {
    const abortController = new AbortController()
    fetchCustomers(undefined, abortController.signal)
    fetchProducts(undefined, abortController.signal)
    fetchWarehouses(abortController.signal)
    return () => abortController.abort()
  }, [fetchCustomers, fetchProducts, fetchWarehouses])

  // Debounced customer search
  useEffect(() => {
    if (!customerSearch) return
    const abortController = new AbortController()
    const timer = setTimeout(() => fetchCustomers(customerSearch, abortController.signal), 300)
    return () => {
      clearTimeout(timer)
      abortController.abort()
    }
  }, [customerSearch, fetchCustomers])

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
        customer_id: initialData.customer_id || '',
        customer_name: initialData.customer_name || '',
        warehouse_id: initialData.warehouse_id || undefined,
        discount: discountPercent,
        remark: initialData.remark || '',
        items: initialData.items?.map((item) => ({
          key: item.id || `item-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
          product_id: item.product_id || '',
          product_code: item.product_code || '',
          product_name: item.product_name || '',
          unit: item.unit || '',
          unit_price: item.unit_price || 0,
          quantity: item.quantity || 1,
          amount: (item.unit_price || 0) * (item.quantity || 1),
          remark: item.remark || '',
        })) || [createEmptySalesOrderItem()],
      })
    }
  }, [initialData, setFormData])

  // Handle customer selection
  const handleCustomerChange = useCallback(
    (selectedOption: string | number | unknown[] | Record<string, unknown> | undefined) => {
      const option = selectedOption as { value?: string; label?: string } | undefined
      setFormData((prev) => ({
        ...prev,
        customer_id: option?.value || '',
        customer_name: option?.label || '',
      }))
      clearError('customer_id')
    },
    [setFormData, clearError]
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
          const unitPrice = product.selling_price || 0
          return {
            ...item,
            product_id: product.id || '',
            product_code: product.code || '',
            product_name: product.name || '',
            unit: product.unit || '',
            unit_price: unitPrice,
            amount: unitPrice * item.quantity,
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
      updateItemWithAmount(itemKey, { quantity: qty }, 'unit_price')
    },
    [updateItemWithAmount]
  )

  // Handle unit price change
  const handleUnitPriceChange = useCallback(
    (itemKey: string, price: number | string | undefined) => {
      const unitPrice = typeof price === 'number' ? price : parseFloat(String(price)) || 0
      updateItemWithAmount(
        itemKey,
        { unit_price: unitPrice } as Partial<SalesOrderItemFormData>,
        'unit_price'
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

  // Handle form submission
  const handleSubmit = useCallback(async () => {
    if (!validateForm()) {
      Toast.error(t('orderForm.validation.itemsRequired'))
      return
    }

    setIsSubmitting(true)
    try {
      const validItems = formData.items.filter((item) => item.product_id)
      const itemsPayload: HandlerCreateSalesOrderItemInput[] = validItems.map((item) => ({
        product_id: item.product_id,
        product_code: item.product_code,
        product_name: item.product_name,
        unit: item.unit,
        unit_price: item.unit_price,
        quantity: item.quantity,
        remark: item.remark || undefined,
      }))

      if (isEditMode && orderId) {
        const response = await salesOrderApi.updateSalesOrder(orderId, {
          warehouse_id: formData.warehouse_id,
          discount: formData.discount,
          remark: formData.remark || undefined,
        })
        if (!response.success) {
          throw new Error(response.error?.message || t('orderForm.messages.updateError'))
        }
        Toast.success(t('orderForm.messages.updateSuccess'))
      } else {
        const response = await salesOrderApi.createSalesOrder({
          customer_id: formData.customer_id,
          customer_name: formData.customer_name,
          warehouse_id: formData.warehouse_id,
          discount: formData.discount,
          remark: formData.remark || undefined,
          items: itemsPayload,
        })
        if (!response.success) {
          throw new Error(response.error?.message || t('orderForm.messages.createError'))
        }
        Toast.success(t('orderForm.messages.createSuccess'))
      }
      // Reset form state before navigation to prevent stale data if navigation fails
      if (!isEditMode) {
        resetForm()
      }
      navigate('/trade/sales')
    } catch (error) {
      Toast.error(error instanceof Error ? error.message : t('orderForm.messages.createError'))
    } finally {
      setIsSubmitting(false)
    }
  }, [
    formData,
    isEditMode,
    orderId,
    salesOrderApi,
    navigate,
    validateForm,
    t,
    setIsSubmitting,
    resetForm,
  ])

  // Handle cancel
  const handleCancel = useCallback(() => navigate('/trade/sales'), [navigate])

  // Select options
  const customerOptions = useMemo(
    () =>
      customers.map((c) => ({ value: c.id || '', label: c.name || c.code || '', extra: c.code })),
    [customers]
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
        price: p.selling_price,
      })),
    [products]
  )

  return (
    <Container size="lg" className="sales-order-form-page">
      <Card className="sales-order-form-card">
        <div className="sales-order-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {isEditMode ? t('orderForm.editTitle') : t('orderForm.createTitle')}
          </Title>
        </div>

        {/* Basic Information Section */}
        <div className="form-section">
          <Title heading={5} className="section-title">
            {t('orderForm.basicInfo.title')}
          </Title>
          <div className="form-row">
            <div className="form-field">
              <label className="form-label required">{t('orderForm.basicInfo.customer')}</label>
              <Select
                value={formData.customer_id || undefined}
                placeholder={t('orderForm.basicInfo.customerPlaceholder')}
                onChange={handleCustomerChange}
                onChangeWithObject
                optionList={customerOptions}
                filter
                remote
                onSearch={setCustomerSearch}
                loading={customersLoading}
                style={{ width: '100%' }}
                prefix={<IconSearch />}
                validateStatus={errors.customer_id ? 'error' : undefined}
                renderSelectedItem={(option: { label?: string; extra?: string }) => (
                  <span>
                    {option.label}
                    {option.extra && <Text type="tertiary"> ({option.extra})</Text>}
                  </span>
                )}
              />
              {errors.customer_id && (
                <Text type="danger" size="small">
                  {errors.customer_id}
                </Text>
              )}
            </div>
            <div className="form-field">
              <label className="form-label">{t('orderForm.basicInfo.warehouse')}</label>
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
        </div>

        {/* Order Items Section */}
        <div className="form-section">
          <div className="section-header">
            <Title heading={5} className="section-title">
              {t('orderForm.items.title')}
            </Title>
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
            onPriceChange={handleUnitPriceChange}
            onItemRemarkChange={handleItemRemarkChange}
            onRemoveItem={removeItem}
            t={t}
            orderType="sales"
            className="items-table"
          />
        </div>

        {/* Summary Section */}
        <div className="form-section summary-section">
          <OrderSummary
            calculations={calculations}
            discount={formData.discount}
            onDiscountChange={handleDiscountChange}
            t={t}
          />
        </div>

        {/* Remark Section */}
        <div className="form-section">
          <div className="form-field">
            <label className="form-label">{t('orderForm.basicInfo.remark')}</label>
            <Input
              value={formData.remark}
              onChange={handleRemarkChange}
              placeholder={t('orderForm.basicInfo.remarkPlaceholder')}
              maxLength={500}
              showClear
            />
          </div>
        </div>

        {/* Form Actions */}
        <div className="form-actions">
          <Space>
            <Button onClick={handleCancel} disabled={isSubmitting}>
              {t('orderForm.actions.cancel')}
            </Button>
            <Button
              type="primary"
              onClick={handleSubmit}
              loading={isSubmitting}
              disabled={isSubmitting}
            >
              {isEditMode ? t('orderForm.actions.save') : t('orderForm.actions.create')}
            </Button>
          </Space>
        </div>
      </Card>
    </Container>
  )
}

export default SalesOrderForm
