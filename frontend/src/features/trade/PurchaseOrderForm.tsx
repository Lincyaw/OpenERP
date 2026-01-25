import { useState, useEffect, useMemo, useCallback } from 'react'
import { z } from 'zod'
import {
  Card,
  Typography,
  Button,
  Table,
  InputNumber,
  Input,
  Select,
  Toast,
  Space,
  Popconfirm,
  Empty,
} from '@douyinfe/semi-ui-19'
import { IconPlus, IconDelete, IconSearch } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { getPurchaseOrders } from '@/api/purchase-orders/purchase-orders'
import { getSuppliers } from '@/api/suppliers/suppliers'
import { getProducts } from '@/api/products/products'
import { getWarehouses } from '@/api/warehouses/warehouses'
import type {
  HandlerSupplierListResponse,
  HandlerProductListResponse,
  HandlerWarehouseListResponse,
  HandlerPurchaseOrderResponse,
  HandlerCreatePurchaseOrderItemInput,
} from '@/api/models'
import { useI18n } from '@/hooks/useI18n'
import './PurchaseOrderForm.css'

const { Title, Text } = Typography

// Order item form type
interface OrderItemFormData {
  key: string
  product_id: string
  product_code: string
  product_name: string
  unit: string
  unit_cost: number
  quantity: number
  amount: number
  remark?: string
}

// Order form data type
interface OrderFormData {
  supplier_id: string
  supplier_name: string
  warehouse_id?: string
  discount: number
  remark?: string
  items: OrderItemFormData[]
}

// Initial empty item
const createEmptyItem = (): OrderItemFormData => ({
  key: `item-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
  product_id: '',
  product_code: '',
  product_name: '',
  unit: '',
  unit_cost: 0,
  quantity: 1,
  amount: 0,
  remark: '',
})

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
  const purchaseOrderApi = useMemo(() => getPurchaseOrders(), [])
  const supplierApi = useMemo(() => getSuppliers(), [])
  const productApi = useMemo(() => getProducts(), [])
  const warehouseApi = useMemo(() => getWarehouses(), [])
  const isEditMode = Boolean(orderId)

  // Form validation schema with i18n
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

  // Form state
  const [formData, setFormData] = useState<OrderFormData>({
    supplier_id: '',
    supplier_name: '',
    warehouse_id: undefined,
    discount: 0,
    remark: '',
    items: [createEmptyItem()],
  })

  // UI state
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})

  // Data for dropdowns
  const [suppliers, setSuppliers] = useState<HandlerSupplierListResponse[]>([])
  const [products, setProducts] = useState<HandlerProductListResponse[]>([])
  const [warehouses, setWarehouses] = useState<HandlerWarehouseListResponse[]>([])
  const [suppliersLoading, setSuppliersLoading] = useState(false)
  const [productsLoading, setProductsLoading] = useState(false)
  const [warehousesLoading, setWarehousesLoading] = useState(false)

  // Search state for suppliers
  const [supplierSearch, setSupplierSearch] = useState('')
  const [productSearch, setProductSearch] = useState('')

  // Calculate totals
  const calculations = useMemo(() => {
    const subtotal = formData.items.reduce((sum, item) => sum + item.amount, 0)
    const discountAmount = (subtotal * formData.discount) / 100
    const total = subtotal - discountAmount
    return {
      subtotal,
      discountAmount,
      total,
      itemCount: formData.items.filter((item) => item.product_id).length,
    }
  }, [formData.items, formData.discount])

  // Fetch suppliers
  const fetchSuppliers = useCallback(
    async (search?: string) => {
      setSuppliersLoading(true)
      try {
        const response = await supplierApi.getPartnerSuppliers({
          page_size: 50,
          search: search || undefined,
          status: 'active',
        })
        if (response.success && response.data) {
          setSuppliers(response.data)
        }
      } catch {
        // Silently fail
      } finally {
        setSuppliersLoading(false)
      }
    },
    [supplierApi]
  )

  // Fetch products
  const fetchProducts = useCallback(
    async (search?: string) => {
      setProductsLoading(true)
      try {
        const response = await productApi.getCatalogProducts({
          page_size: 50,
          search: search || undefined,
          status: 'active',
        })
        if (response.success && response.data) {
          setProducts(response.data)
        }
      } catch {
        // Silently fail
      } finally {
        setProductsLoading(false)
      }
    },
    [productApi]
  )

  // Fetch warehouses
  const fetchWarehouses = useCallback(async () => {
    setWarehousesLoading(true)
    try {
      const response = await warehouseApi.getPartnerWarehouses({
        page_size: 100,
        status: 'active',
      })
      if (response.success && response.data) {
        setWarehouses(response.data)
        // Set default warehouse if available and not in edit mode
        if (!isEditMode && !formData.warehouse_id) {
          const defaultWarehouse = response.data.find((w) => w.is_default)
          if (defaultWarehouse?.id) {
            setFormData((prev) => ({ ...prev, warehouse_id: defaultWarehouse.id }))
          }
        }
      }
    } catch {
      // Silently fail
    } finally {
      setWarehousesLoading(false)
    }
  }, [warehouseApi, isEditMode, formData.warehouse_id])

  // Initial data loading
  useEffect(() => {
    fetchSuppliers()
    fetchProducts()
    fetchWarehouses()
  }, [fetchSuppliers, fetchProducts, fetchWarehouses])

  // Debounced supplier search
  useEffect(() => {
    const timer = setTimeout(() => {
      if (supplierSearch) {
        fetchSuppliers(supplierSearch)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [supplierSearch, fetchSuppliers])

  // Debounced product search
  useEffect(() => {
    const timer = setTimeout(() => {
      if (productSearch) {
        fetchProducts(productSearch)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [productSearch, fetchProducts])

  // Load initial data for edit mode
  useEffect(() => {
    if (initialData) {
      // Calculate discount percentage from discount_amount and total_amount
      const totalAmount = initialData.total_amount || 0
      const discountAmount = initialData.discount_amount || 0
      // discount_amount = subtotal * discount_percentage / 100
      // subtotal = total_amount + discount_amount
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
        })) || [createEmptyItem()],
      })
    }
  }, [initialData])

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
      setErrors((prev) => {
        const newErrors = { ...prev }
        delete newErrors.supplier_id
        return newErrors
      })
    },
    [suppliers]
  )

  // Handle warehouse selection
  const handleWarehouseChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const warehouseId = typeof value === 'string' ? value : undefined
      setFormData((prev) => ({ ...prev, warehouse_id: warehouseId || undefined }))
    },
    []
  )

  // Handle product selection for an item
  const handleProductSelect = useCallback(
    (itemKey: string, productId: string) => {
      const product = products.find((p) => p.id === productId)
      if (!product) return

      setFormData((prev) => ({
        ...prev,
        items: prev.items.map((item) => {
          if (item.key !== itemKey) return item
          // Use purchase_price (cost) for purchase orders instead of selling_price
          const unitCost = product.purchase_price || product.selling_price || 0
          const newItem = {
            ...item,
            product_id: product.id || '',
            product_code: product.code || '',
            product_name: product.name || '',
            unit: product.unit || '',
            unit_cost: unitCost,
            amount: unitCost * item.quantity,
          }
          return newItem
        }),
      }))
      setErrors((prev) => {
        const newErrors = { ...prev }
        delete newErrors[`items.${itemKey}.product_id`]
        return newErrors
      })
    },
    [products]
  )

  // Handle quantity change
  const handleQuantityChange = useCallback((itemKey: string, quantity: number | string) => {
    const qty = typeof quantity === 'number' ? quantity : parseFloat(quantity) || 0
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) => {
        if (item.key !== itemKey) return item
        return {
          ...item,
          quantity: qty,
          amount: item.unit_cost * qty,
        }
      }),
    }))
  }, [])

  // Handle unit cost change
  const handleUnitCostChange = useCallback((itemKey: string, cost: number | string) => {
    const unitCost = typeof cost === 'number' ? cost : parseFloat(cost) || 0
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) => {
        if (item.key !== itemKey) return item
        return {
          ...item,
          unit_cost: unitCost,
          amount: unitCost * item.quantity,
        }
      }),
    }))
  }, [])

  // Handle item remark change
  const handleItemRemarkChange = useCallback((itemKey: string, remark: string) => {
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) => {
        if (item.key !== itemKey) return item
        return { ...item, remark }
      }),
    }))
  }, [])

  // Add new item row
  const handleAddItem = useCallback(() => {
    setFormData((prev) => ({
      ...prev,
      items: [...prev.items, createEmptyItem()],
    }))
  }, [])

  // Remove item row
  const handleRemoveItem = useCallback((itemKey: string) => {
    setFormData((prev) => {
      const newItems = prev.items.filter((item) => item.key !== itemKey)
      // Always keep at least one row
      if (newItems.length === 0) {
        return { ...prev, items: [createEmptyItem()] }
      }
      return { ...prev, items: newItems }
    })
  }, [])

  // Handle discount change
  const handleDiscountChange = useCallback((value: number | string) => {
    const discount = typeof value === 'number' ? value : parseFloat(value) || 0
    setFormData((prev) => ({ ...prev, discount }))
  }, [])

  // Handle remark change
  const handleRemarkChange = useCallback((value: string) => {
    setFormData((prev) => ({ ...prev, remark: value }))
  }, [])

  // Validate form
  const validateForm = useCallback((): boolean => {
    const result = orderFormSchema.safeParse({
      ...formData,
      items: formData.items.filter((item) => item.product_id), // Only validate non-empty items
    })

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
  }, [formData, orderFormSchema])

  // Handle form submission
  const handleSubmit = useCallback(async () => {
    if (!validateForm()) {
      Toast.error(t('orderForm.validation.itemsRequired'))
      return
    }

    setIsSubmitting(true)
    try {
      // Filter out empty items and prepare for API
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
        // Update existing order (supplier cannot be changed in edit mode)
        const response = await purchaseOrderApi.putTradePurchaseOrdersId(orderId, {
          warehouse_id: formData.warehouse_id,
          discount: formData.discount,
          remark: formData.remark || undefined,
        })
        if (!response.success) {
          throw new Error(response.error?.message || t('orderForm.messages.updateError'))
        }
        Toast.success(t('orderForm.messages.updateSuccess'))
      } else {
        // Create new order
        const response = await purchaseOrderApi.postTradePurchaseOrders({
          supplier_id: formData.supplier_id,
          supplier_name: formData.supplier_name,
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
      navigate('/trade/purchase')
    } catch (error) {
      Toast.error(error instanceof Error ? error.message : t('orderForm.messages.createError'))
    } finally {
      setIsSubmitting(false)
    }
  }, [formData, isEditMode, orderId, purchaseOrderApi, navigate, validateForm, t])

  // Handle cancel
  const handleCancel = useCallback(() => {
    navigate('/trade/purchase')
  }, [navigate])

  // Supplier options for select
  const supplierOptions = useMemo(
    () =>
      suppliers.map((s) => ({
        value: s.id || '',
        label: s.name || s.code || '',
        extra: s.code,
      })),
    [suppliers]
  )

  // Warehouse options for select
  const warehouseOptions = useMemo(
    () =>
      warehouses.map((w) => ({
        value: w.id || '',
        label: w.name || w.code || '',
        extra: w.is_default ? `(${t('common.defaultWarehouse')})` : undefined,
      })),
    [warehouses, t]
  )

  // Product options for select
  const productOptions = useMemo(
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

  // Table columns for order items
  const itemColumns = useMemo(
    () => [
      {
        title: t('orderForm.items.columns.product'),
        dataIndex: 'product_id',
        width: 280,
        render: (_: unknown, record: OrderItemFormData) => (
          <Select
            value={record.product_id || undefined}
            placeholder={t('orderForm.items.columns.productPlaceholder')}
            onChange={(value) => handleProductSelect(record.key, value as string)}
            optionList={productOptions}
            filter
            remote
            onSearch={setProductSearch}
            loading={productsLoading}
            style={{ width: '100%' }}
            prefix={<IconSearch />}
            renderSelectedItem={(option: { label?: string }) => (
              <span className="selected-product">{option.label}</span>
            )}
          />
        ),
      },
      {
        title: t('orderForm.items.columns.unit'),
        dataIndex: 'unit',
        width: 80,
        render: (unit: string) => <Text>{unit || '-'}</Text>,
      },
      {
        title: t('orderForm.items.columns.purchasePrice'),
        dataIndex: 'unit_cost',
        width: 120,
        render: (cost: number, record: OrderItemFormData) => (
          <InputNumber
            value={cost}
            onChange={(value) => handleUnitCostChange(record.key, value as number)}
            min={0}
            precision={2}
            prefix="¥"
            style={{ width: '100%' }}
            disabled={!record.product_id}
          />
        ),
      },
      {
        title: t('orderForm.items.columns.quantity'),
        dataIndex: 'quantity',
        width: 100,
        render: (qty: number, record: OrderItemFormData) => (
          <InputNumber
            value={qty}
            onChange={(value) => handleQuantityChange(record.key, value as number)}
            min={0.01}
            precision={2}
            style={{ width: '100%' }}
            disabled={!record.product_id}
          />
        ),
      },
      {
        title: t('orderForm.items.columns.amount'),
        dataIndex: 'amount',
        width: 120,
        align: 'right' as const,
        render: (amount: number) => (
          <Text strong className="item-amount">
            ¥{amount.toFixed(2)}
          </Text>
        ),
      },
      {
        title: t('orderForm.items.columns.remark'),
        dataIndex: 'remark',
        width: 150,
        render: (remark: string, record: OrderItemFormData) => (
          <Input
            value={remark}
            onChange={(value) => handleItemRemarkChange(record.key, value)}
            placeholder={t('orderForm.items.columns.remarkPlaceholder')}
            disabled={!record.product_id}
          />
        ),
      },
      {
        title: t('orderForm.items.columns.operation'),
        dataIndex: 'actions',
        width: 60,
        render: (_: unknown, record: OrderItemFormData) => (
          <Popconfirm
            title={t('orderForm.items.remove')}
            onConfirm={() => handleRemoveItem(record.key)}
            position="left"
          >
            <Button icon={<IconDelete />} type="danger" theme="borderless" size="small" />
          </Popconfirm>
        ),
      },
    ],
    [
      t,
      productOptions,
      productsLoading,
      handleProductSelect,
      handleUnitCostChange,
      handleQuantityChange,
      handleItemRemarkChange,
      handleRemoveItem,
    ]
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
        <div className="form-section">
          <Title heading={5} className="section-title">
            {t('orderForm.basicInfo.title')}
          </Title>
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
                disabled={isEditMode} // Cannot change supplier in edit mode
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
        </div>

        {/* Order Items Section */}
        <div className="form-section">
          <div className="section-header">
            <Title heading={5} className="section-title">
              {t('orderForm.items.title')}
            </Title>
            <Button icon={<IconPlus />} theme="light" onClick={handleAddItem}>
              {t('orderForm.items.addProduct')}
            </Button>
          </div>
          {errors.items && (
            <Text type="danger" size="small" className="items-error">
              {errors.items}
            </Text>
          )}
          <Table
            columns={itemColumns}
            dataSource={formData.items}
            rowKey="key"
            pagination={false}
            size="small"
            className="items-table"
            empty={<Empty description={t('orderForm.items.empty')} />}
          />
        </div>

        {/* Summary Section */}
        <div className="form-section summary-section">
          <div className="summary-row">
            <div className="form-field discount-field">
              <label className="form-label">{t('orderForm.summary.discount')} (%)</label>
              <InputNumber
                value={formData.discount}
                onChange={(value) => handleDiscountChange(value as number)}
                min={0}
                max={100}
                precision={2}
                suffix="%"
                style={{ width: 120 }}
              />
            </div>
            <div className="summary-totals">
              <div className="summary-item">
                <Text type="tertiary">{t('orderForm.summary.itemCount')}:</Text>
                <Text>
                  {calculations.itemCount} {t('orderForm.summary.itemCountUnit')}
                </Text>
              </div>
              <div className="summary-item">
                <Text type="tertiary">{t('orderForm.summary.subtotal')}:</Text>
                <Text>¥{calculations.subtotal.toFixed(2)}</Text>
              </div>
              {formData.discount > 0 && (
                <div className="summary-item">
                  <Text type="tertiary">
                    {t('orderForm.summary.discount')} ({formData.discount}%):
                  </Text>
                  <Text type="danger">-¥{calculations.discountAmount.toFixed(2)}</Text>
                </div>
              )}
              <div className="summary-item total">
                <Text strong>{t('orderForm.summary.payableAmount')}:</Text>
                <Text strong className="total-amount">
                  ¥{calculations.total.toFixed(2)}
                </Text>
              </div>
            </div>
          </div>
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

export default PurchaseOrderForm
