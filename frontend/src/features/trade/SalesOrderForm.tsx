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
} from '@douyinfe/semi-ui'
import { IconPlus, IconDelete, IconSearch } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { getSalesOrders } from '@/api/sales-orders/sales-orders'
import { getCustomers } from '@/api/customers/customers'
import { getProducts } from '@/api/products/products'
import { getWarehouses } from '@/api/warehouses/warehouses'
import type {
  HandlerCustomerListResponse,
  HandlerProductListResponse,
  HandlerWarehouseListResponse,
  HandlerSalesOrderResponse,
  HandlerCreateSalesOrderItemInput,
} from '@/api/models'
import './SalesOrderForm.css'

const { Title, Text } = Typography

// Order item form type
interface OrderItemFormData {
  key: string
  product_id: string
  product_code: string
  product_name: string
  unit: string
  unit_price: number
  quantity: number
  amount: number
  remark?: string
}

// Order form data type
interface OrderFormData {
  customer_id: string
  customer_name: string
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
  unit_price: 0,
  quantity: 1,
  amount: 0,
  remark: '',
})

// Form validation schema
const orderFormSchema = z.object({
  customer_id: z.string().min(1, '请选择客户'),
  customer_name: z.string().min(1, '请选择客户'),
  warehouse_id: z.string().optional(),
  discount: z.number().min(0, '折扣不能为负数').max(100, '折扣不能超过100'),
  remark: z.string().max(500, '备注不能超过500字').optional(),
  items: z
    .array(
      z.object({
        product_id: z.string().min(1, '请选择商品'),
        product_code: z.string().min(1),
        product_name: z.string().min(1),
        unit: z.string().min(1),
        unit_price: z.number().positive('单价必须大于0'),
        quantity: z.number().positive('数量必须大于0'),
      })
    )
    .min(1, '请至少添加一个商品'),
})

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
  const salesOrderApi = useMemo(() => getSalesOrders(), [])
  const customerApi = useMemo(() => getCustomers(), [])
  const productApi = useMemo(() => getProducts(), [])
  const warehouseApi = useMemo(() => getWarehouses(), [])
  const isEditMode = Boolean(orderId)

  // Form state
  const [formData, setFormData] = useState<OrderFormData>({
    customer_id: '',
    customer_name: '',
    warehouse_id: undefined,
    discount: 0,
    remark: '',
    items: [createEmptyItem()],
  })

  // UI state
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})

  // Data for dropdowns
  const [customers, setCustomers] = useState<HandlerCustomerListResponse[]>([])
  const [products, setProducts] = useState<HandlerProductListResponse[]>([])
  const [warehouses, setWarehouses] = useState<HandlerWarehouseListResponse[]>([])
  const [customersLoading, setCustomersLoading] = useState(false)
  const [productsLoading, setProductsLoading] = useState(false)
  const [warehousesLoading, setWarehousesLoading] = useState(false)

  // Search state for customers
  const [customerSearch, setCustomerSearch] = useState('')
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

  // Fetch customers
  const fetchCustomers = useCallback(
    async (search?: string) => {
      setCustomersLoading(true)
      try {
        const response = await customerApi.getPartnerCustomers({
          page_size: 50,
          search: search || undefined,
          status: 'active',
        })
        if (response.success && response.data) {
          setCustomers(response.data)
        }
      } catch {
        // Silently fail
      } finally {
        setCustomersLoading(false)
      }
    },
    [customerApi]
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
    fetchCustomers()
    fetchProducts()
    fetchWarehouses()
  }, [fetchCustomers, fetchProducts, fetchWarehouses])

  // Debounced customer search
  useEffect(() => {
    const timer = setTimeout(() => {
      if (customerSearch) {
        fetchCustomers(customerSearch)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [customerSearch, fetchCustomers])

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
        })) || [createEmptyItem()],
      })
    }
  }, [initialData])

  // Handle customer selection
  const handleCustomerChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const customerId = typeof value === 'string' ? value : ''
      const customer = customers.find((c) => c.id === customerId)
      setFormData((prev) => ({
        ...prev,
        customer_id: customerId,
        customer_name: customer?.name || '',
      }))
      setErrors((prev) => {
        const newErrors = { ...prev }
        delete newErrors.customer_id
        return newErrors
      })
    },
    [customers]
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
          const newItem = {
            ...item,
            product_id: product.id || '',
            product_code: product.code || '',
            product_name: product.name || '',
            unit: product.unit || '',
            unit_price: product.selling_price || 0,
            amount: (product.selling_price || 0) * item.quantity,
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
          amount: item.unit_price * qty,
        }
      }),
    }))
  }, [])

  // Handle unit price change
  const handleUnitPriceChange = useCallback((itemKey: string, price: number | string) => {
    const unitPrice = typeof price === 'number' ? price : parseFloat(price) || 0
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) => {
        if (item.key !== itemKey) return item
        return {
          ...item,
          unit_price: unitPrice,
          amount: unitPrice * item.quantity,
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
  }, [formData])

  // Handle form submission
  const handleSubmit = useCallback(async () => {
    if (!validateForm()) {
      Toast.error('请检查表单填写是否正确')
      return
    }

    setIsSubmitting(true)
    try {
      // Filter out empty items and prepare for API
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
        // Update existing order (customer cannot be changed in edit mode)
        const response = await salesOrderApi.putTradeSalesOrdersId(orderId, {
          warehouse_id: formData.warehouse_id,
          discount: formData.discount,
          remark: formData.remark || undefined,
        })
        if (!response.success) {
          throw new Error(response.error?.message || '更新失败')
        }
        Toast.success('订单更新成功')
      } else {
        // Create new order
        const response = await salesOrderApi.postTradeSalesOrders({
          customer_id: formData.customer_id,
          customer_name: formData.customer_name,
          warehouse_id: formData.warehouse_id,
          discount: formData.discount,
          remark: formData.remark || undefined,
          items: itemsPayload,
        })
        if (!response.success) {
          throw new Error(response.error?.message || '创建失败')
        }
        Toast.success('订单创建成功')
      }
      navigate('/trade/sales')
    } catch (error) {
      Toast.error(error instanceof Error ? error.message : '操作失败')
    } finally {
      setIsSubmitting(false)
    }
  }, [formData, isEditMode, orderId, salesOrderApi, navigate, validateForm])

  // Handle cancel
  const handleCancel = useCallback(() => {
    navigate('/trade/sales')
  }, [navigate])

  // Customer options for select
  const customerOptions = useMemo(
    () =>
      customers.map((c) => ({
        value: c.id || '',
        label: c.name || c.code || '',
        extra: c.code,
      })),
    [customers]
  )

  // Warehouse options for select
  const warehouseOptions = useMemo(
    () =>
      warehouses.map((w) => ({
        value: w.id || '',
        label: w.name || w.code || '',
        extra: w.is_default ? '(默认)' : undefined,
      })),
    [warehouses]
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
        price: p.selling_price,
      })),
    [products]
  )

  // Table columns for order items
  const itemColumns = [
    {
      title: '商品',
      dataIndex: 'product_id',
      width: 280,
      render: (_: unknown, record: OrderItemFormData) => (
        <Select
          value={record.product_id || undefined}
          placeholder="搜索并选择商品"
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
      title: '单位',
      dataIndex: 'unit',
      width: 80,
      render: (unit: string) => <Text>{unit || '-'}</Text>,
    },
    {
      title: '单价',
      dataIndex: 'unit_price',
      width: 120,
      render: (price: number, record: OrderItemFormData) => (
        <InputNumber
          value={price}
          onChange={(value) => handleUnitPriceChange(record.key, value as number)}
          min={0}
          precision={2}
          prefix="¥"
          style={{ width: '100%' }}
          disabled={!record.product_id}
        />
      ),
    },
    {
      title: '数量',
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
      title: '金额',
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
      title: '备注',
      dataIndex: 'remark',
      width: 150,
      render: (remark: string, record: OrderItemFormData) => (
        <Input
          value={remark}
          onChange={(value) => handleItemRemarkChange(record.key, value)}
          placeholder="备注"
          disabled={!record.product_id}
        />
      ),
    },
    {
      title: '操作',
      dataIndex: 'actions',
      width: 60,
      render: (_: unknown, record: OrderItemFormData) => (
        <Popconfirm
          title="确定删除该行吗？"
          onConfirm={() => handleRemoveItem(record.key)}
          position="left"
        >
          <Button icon={<IconDelete />} type="danger" theme="borderless" size="small" />
        </Popconfirm>
      ),
    },
  ]

  return (
    <Container size="lg" className="sales-order-form-page">
      <Card className="sales-order-form-card">
        <div className="sales-order-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {isEditMode ? '编辑销售订单' : '新建销售订单'}
          </Title>
        </div>

        {/* Basic Information Section */}
        <div className="form-section">
          <Title heading={5} className="section-title">
            基本信息
          </Title>
          <div className="form-row">
            <div className="form-field">
              <label className="form-label required">客户</label>
              <Select
                value={formData.customer_id || undefined}
                placeholder="搜索并选择客户"
                onChange={handleCustomerChange}
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
              <label className="form-label">发货仓库</label>
              <Select
                value={formData.warehouse_id || undefined}
                placeholder="选择发货仓库"
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
              商品明细
            </Title>
            <Button icon={<IconPlus />} theme="light" onClick={handleAddItem}>
              添加商品
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
            empty={<Empty description="暂无商品，点击上方按钮添加" />}
          />
        </div>

        {/* Summary Section */}
        <div className="form-section summary-section">
          <div className="summary-row">
            <div className="form-field discount-field">
              <label className="form-label">折扣 (%)</label>
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
                <Text type="tertiary">商品数量：</Text>
                <Text>{calculations.itemCount} 种</Text>
              </div>
              <div className="summary-item">
                <Text type="tertiary">小计：</Text>
                <Text>¥{calculations.subtotal.toFixed(2)}</Text>
              </div>
              {formData.discount > 0 && (
                <div className="summary-item">
                  <Text type="tertiary">折扣 ({formData.discount}%)：</Text>
                  <Text type="danger">-¥{calculations.discountAmount.toFixed(2)}</Text>
                </div>
              )}
              <div className="summary-item total">
                <Text strong>应付金额：</Text>
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
            <label className="form-label">备注</label>
            <Input
              value={formData.remark}
              onChange={handleRemarkChange}
              placeholder="订单备注（可选）"
              maxLength={500}
              showClear
            />
          </div>
        </div>

        {/* Form Actions */}
        <div className="form-actions">
          <Space>
            <Button onClick={handleCancel} disabled={isSubmitting}>
              取消
            </Button>
            <Button
              type="primary"
              onClick={handleSubmit}
              loading={isSubmitting}
              disabled={isSubmitting}
            >
              {isEditMode ? '保存' : '创建订单'}
            </Button>
          </Space>
        </div>
      </Card>
    </Container>
  )
}

export default SalesOrderForm
