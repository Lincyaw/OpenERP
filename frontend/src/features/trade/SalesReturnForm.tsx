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
  Empty,
  Descriptions,
  Tag,
} from '@douyinfe/semi-ui-19'
import { IconSearch } from '@douyinfe/semi-icons'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '@/components/common/layout'
import { getSalesReturns } from '@/api/sales-returns/sales-returns'
import { getSalesOrders } from '@/api/sales-orders/sales-orders'
import { listWarehouses } from '@/api/warehouses/warehouses'
import type {
  HandlerSalesOrderResponse,
  HandlerSalesOrderItemResponse,
  HandlerWarehouseListResponse,
  HandlerSalesOrderListResponse,
} from '@/api/models'
import { safeToFixed, safeFormatCurrency, createScopedLogger } from '@/utils'

const log = createScopedLogger('SalesReturnForm')
import './SalesReturnForm.css'

const { Title, Text } = Typography

// Return item form type
interface ReturnItemFormData {
  key: string
  sales_order_item_id: string
  product_id: string
  product_code: string
  product_name: string
  unit: string
  unit_price: number
  original_quantity: number
  return_quantity: number
  refund_amount: number
  reason: string
  condition_on_return: string
  selected: boolean
}

// Return form data type
interface ReturnFormData {
  sales_order_id: string
  warehouse_id?: string
  reason: string
  remark: string
  items: ReturnItemFormData[]
}

// Form validation schema factory
const createReturnFormSchema = (t: (key: string) => string) =>
  z.object({
    sales_order_id: z.string().min(1, t('salesReturnForm.validation.orderRequired')),
    warehouse_id: z.string().optional(),
    reason: z
      .string()
      .min(1, t('salesReturnForm.validation.reasonRequired'))
      .max(500, t('salesReturnForm.validation.reasonTooLong')),
    remark: z.string().max(500, t('salesReturnForm.validation.remarkTooLong')).optional(),
    items: z
      .array(
        z.object({
          sales_order_item_id: z.string().min(1),
          return_quantity: z.number().positive(t('salesReturnForm.validation.quantityPositive')),
        })
      )
      .min(1, t('salesReturnForm.validation.itemsRequired')),
  })

// Order status colors
const ORDER_STATUS_COLORS: Record<
  string,
  'blue' | 'cyan' | 'green' | 'grey' | 'orange' | 'violet'
> = {
  DRAFT: 'blue',
  CONFIRMED: 'cyan',
  SHIPPED: 'orange',
  COMPLETED: 'green',
  CANCELLED: 'grey',
}

/**
 * Format price for display
 */
function formatPrice(price?: number | string): string {
  return safeFormatCurrency(price, '¥', 2, '¥0.00')
}

/**
 * Sales return form component for creating sales returns
 *
 * Features:
 * - Search and select existing sales order
 * - Display order items with selection checkboxes
 * - Set return quantity for each item (up to original quantity)
 * - Set return reason and condition for each item
 * - Real-time refund amount calculation
 * - Form validation with Zod
 */
export function SalesReturnForm() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const preSelectedOrderId = searchParams.get('order_id')
  const { t } = useTranslation('trade')

  // Memoized translation-dependent constants
  const CONDITION_OPTIONS = useMemo(
    () => [
      { label: t('salesReturnForm.condition.intact'), value: 'intact' },
      { label: t('salesReturnForm.condition.damaged'), value: 'damaged' },
      { label: t('salesReturnForm.condition.defective'), value: 'defective' },
      { label: t('salesReturnForm.condition.wrongItem'), value: 'wrong_item' },
      { label: t('salesReturnForm.condition.other'), value: 'other' },
    ],
    [t]
  )

  const ORDER_STATUS_LABELS: Record<string, string> = useMemo(
    () => ({
      DRAFT: t('salesOrder.status.draft'),
      CONFIRMED: t('salesOrder.status.confirmed'),
      SHIPPED: t('salesOrder.status.shipped'),
      COMPLETED: t('salesOrder.status.completed'),
      CANCELLED: t('salesOrder.status.cancelled'),
    }),
    [t]
  )

  const returnFormSchema = useMemo(() => createReturnFormSchema(t), [t])

  const salesReturnApi = useMemo(() => getSalesReturns(), [])
  const salesOrderApi = useMemo(() => getSalesOrders(), [])

  // Form state
  const [formData, setFormData] = useState<ReturnFormData>({
    sales_order_id: '',
    warehouse_id: undefined,
    reason: '',
    remark: '',
    items: [],
  })

  // UI state
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})

  // Data for dropdowns
  const [orders, setOrders] = useState<HandlerSalesOrderListResponse[]>([])
  const [selectedOrder, setSelectedOrder] = useState<HandlerSalesOrderResponse | null>(null)
  const [warehouses, setWarehouses] = useState<HandlerWarehouseListResponse[]>([])
  const [ordersLoading, setOrdersLoading] = useState(false)
  const [orderLoading, setOrderLoading] = useState(false)
  const [warehousesLoading, setWarehousesLoading] = useState(false)

  // Search state
  const [orderSearch, setOrderSearch] = useState('')

  // Calculate totals
  const calculations = useMemo(() => {
    const selectedItems = formData.items.filter((item) => item.selected && item.return_quantity > 0)
    const totalQuantity = selectedItems.reduce((sum, item) => sum + item.return_quantity, 0)
    const totalRefund = selectedItems.reduce((sum, item) => sum + item.refund_amount, 0)
    return {
      totalQuantity,
      totalRefund,
      itemCount: selectedItems.length,
    }
  }, [formData.items])

  // Fetch orders for selection
  const fetchOrders = useCallback(
    async (search?: string) => {
      setOrdersLoading(true)
      try {
        // Only get shipped or completed orders for return
        const response = await salesOrderApi.listSalesOrders({
          page_size: 50,
          search: search || undefined,
          statuses: ['SHIPPED', 'COMPLETED'],
        })
        if (response.success && response.data) {
          setOrders(response.data)
        } else if (!response.success) {
          log.error('Failed to fetch orders', response.error)
        }
      } catch (error) {
        log.error('Error fetching orders', error)
      } finally {
        setOrdersLoading(false)
      }
    },
    [salesOrderApi]
  )

  // Fetch order detail
  const fetchOrderDetail = useCallback(
    async (orderId: string) => {
      if (!orderId) return
      setOrderLoading(true)
      try {
        const response = await salesOrderApi.getSalesOrderById(orderId)
        if (response.success && response.data) {
          setSelectedOrder(response.data)
          // Convert order items to return items
          const returnItems: ReturnItemFormData[] = (response.data.items || []).map(
            (item: HandlerSalesOrderItemResponse) => ({
              key: item.id || `item-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
              sales_order_item_id: item.id || '',
              product_id: item.product_id || '',
              product_code: item.product_code || '',
              product_name: item.product_name || '',
              unit: item.unit || '',
              unit_price: item.unit_price || 0,
              original_quantity: item.quantity || 0,
              return_quantity: 0,
              refund_amount: 0,
              reason: '',
              condition_on_return: 'intact',
              selected: false,
            })
          )
          setFormData((prev) => ({
            ...prev,
            sales_order_id: orderId,
            items: returnItems,
          }))
        }
      } catch {
        Toast.error(t('salesReturnForm.messages.fetchOrderError'))
      } finally {
        setOrderLoading(false)
      }
    },
    [salesOrderApi, t]
  )

  // Fetch warehouses
  const fetchWarehouses = useCallback(async () => {
    setWarehousesLoading(true)
    try {
      const response = await listWarehouses({
        page_size: 100,
        status: 'enabled',
      })
      if (response.status === 200 && response.data.success && response.data.data) {
        setWarehouses(response.data.data)
        // Set default warehouse if available
        if (!formData.warehouse_id) {
          const defaultWarehouse = response.data.data.find(
            (w: HandlerWarehouseListResponse) => w.is_default
          )
          if (defaultWarehouse?.id) {
            setFormData((prev) => ({ ...prev, warehouse_id: defaultWarehouse.id }))
          }
        }
      } else if (response.status !== 200 || !response.data.success) {
        log.error('Failed to fetch warehouses', response.data.error)
      }
    } catch (error) {
      log.error('Error fetching warehouses', error)
    } finally {
      setWarehousesLoading(false)
    }
  }, [formData.warehouse_id])

  // Initial data loading
  useEffect(() => {
    fetchOrders()
    fetchWarehouses()
  }, [fetchOrders, fetchWarehouses])

  // Handle pre-selected order from URL
  useEffect(() => {
    if (preSelectedOrderId) {
      fetchOrderDetail(preSelectedOrderId)
    }
  }, [preSelectedOrderId, fetchOrderDetail])

  // Debounced order search
  useEffect(() => {
    const timer = setTimeout(() => {
      if (orderSearch) {
        fetchOrders(orderSearch)
      }
    }, 300)
    return () => clearTimeout(timer)
  }, [orderSearch, fetchOrders])

  // Handle order selection
  const handleOrderChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const orderId = typeof value === 'string' ? value : ''
      if (orderId) {
        fetchOrderDetail(orderId)
      } else {
        setSelectedOrder(null)
        setFormData((prev) => ({
          ...prev,
          sales_order_id: '',
          items: [],
        }))
      }
      setErrors((prev) => {
        const newErrors = { ...prev }
        delete newErrors.sales_order_id
        return newErrors
      })
    },
    [fetchOrderDetail]
  )

  // Handle warehouse selection
  const handleWarehouseChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const warehouseId = typeof value === 'string' ? value : undefined
      setFormData((prev) => ({ ...prev, warehouse_id: warehouseId || undefined }))
    },
    []
  )

  // Handle item selection toggle
  const handleItemSelect = useCallback((itemKey: string, selected: boolean) => {
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) => {
        if (item.key !== itemKey) return item
        const newItem = {
          ...item,
          selected,
          return_quantity: selected ? item.original_quantity : 0,
          refund_amount: selected ? item.unit_price * item.original_quantity : 0,
        }
        return newItem
      }),
    }))
  }, [])

  // Handle return quantity change
  const handleQuantityChange = useCallback((itemKey: string, quantity: number | string) => {
    const qty = typeof quantity === 'number' ? quantity : parseFloat(quantity) || 0
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) => {
        if (item.key !== itemKey) return item
        const clampedQty = Math.min(Math.max(0, qty), item.original_quantity)
        return {
          ...item,
          return_quantity: clampedQty,
          refund_amount: item.unit_price * clampedQty,
          selected: clampedQty > 0,
        }
      }),
    }))
  }, [])

  // Handle item reason change
  const handleItemReasonChange = useCallback((itemKey: string, reason: string) => {
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) => {
        if (item.key !== itemKey) return item
        return { ...item, reason }
      }),
    }))
  }, [])

  // Handle item condition change
  const handleItemConditionChange = useCallback((itemKey: string, condition: string) => {
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) => {
        if (item.key !== itemKey) return item
        return { ...item, condition_on_return: condition }
      }),
    }))
  }, [])

  // Handle global reason change
  const handleReasonChange = useCallback((value: string) => {
    setFormData((prev) => ({ ...prev, reason: value }))
    setErrors((prev) => {
      const newErrors = { ...prev }
      delete newErrors.reason
      return newErrors
    })
  }, [])

  // Handle remark change
  const handleRemarkChange = useCallback((value: string) => {
    setFormData((prev) => ({ ...prev, remark: value }))
  }, [])

  // Select all items
  const handleSelectAll = useCallback(() => {
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) => ({
        ...item,
        selected: true,
        return_quantity: item.original_quantity,
        refund_amount: item.unit_price * item.original_quantity,
      })),
    }))
  }, [])

  // Deselect all items
  const handleDeselectAll = useCallback(() => {
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) => ({
        ...item,
        selected: false,
        return_quantity: 0,
        refund_amount: 0,
      })),
    }))
  }, [])

  // Validate form
  const validateForm = useCallback((): boolean => {
    const selectedItems = formData.items
      .filter((item) => item.selected && item.return_quantity > 0)
      .map((item) => ({
        sales_order_item_id: item.sales_order_item_id,
        return_quantity: item.return_quantity,
      }))

    const result = returnFormSchema.safeParse({
      ...formData,
      items: selectedItems,
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
  }, [formData, returnFormSchema])

  // Handle form submission
  const handleSubmit = useCallback(async () => {
    if (!validateForm()) {
      Toast.error(t('salesReturnForm.validation.formError'))
      return
    }

    setIsSubmitting(true)
    try {
      // Filter selected items and prepare for API
      const selectedItems = formData.items.filter(
        (item) => item.selected && item.return_quantity > 0
      )

      const response = await salesReturnApi.createSalesReturn({
        sales_order_id: formData.sales_order_id,
        warehouse_id: formData.warehouse_id,
        reason: formData.reason,
        remark: formData.remark || undefined,
        items: selectedItems.map((item) => ({
          sales_order_item_id: item.sales_order_item_id,
          return_quantity: item.return_quantity,
          reason: item.reason || undefined,
          condition_on_return: item.condition_on_return || undefined,
        })),
      })

      if (!response.success) {
        throw new Error(response.error?.message || t('salesReturnForm.messages.createFailed'))
      }

      Toast.success(t('salesReturnForm.messages.createSuccess'))
      navigate('/trade/sales-returns')
    } catch (error) {
      Toast.error(
        error instanceof Error ? error.message : t('salesReturnForm.messages.createError')
      )
    } finally {
      setIsSubmitting(false)
    }
  }, [formData, salesReturnApi, navigate, validateForm, t])

  // Handle cancel
  const handleCancel = useCallback(() => {
    navigate('/trade/sales-returns')
  }, [navigate])

  // Order options for select
  const orderOptions = useMemo(
    () =>
      orders.map((o) => ({
        value: o.id || '',
        label: `${o.order_number} - ${o.customer_name}`,
        orderNumber: o.order_number,
        customerName: o.customer_name,
        status: o.status,
      })),
    [orders]
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

  // Table columns for return items
  const itemColumns = [
    {
      title: t('salesReturnForm.items.columns.select'),
      dataIndex: 'selected',
      width: 60,
      render: (selected: boolean, record: ReturnItemFormData) => (
        <input
          type="checkbox"
          checked={selected}
          onChange={(e) => handleItemSelect(record.key, e.target.checked)}
          style={{ width: 16, height: 16 }}
        />
      ),
    },
    {
      title: t('salesReturnForm.items.columns.productCode'),
      dataIndex: 'product_code',
      width: 120,
      render: (code: string) => <Text>{code || '-'}</Text>,
    },
    {
      title: t('salesReturnForm.items.columns.productName'),
      dataIndex: 'product_name',
      width: 200,
      ellipsis: true,
      render: (name: string) => <Text>{name || '-'}</Text>,
    },
    {
      title: t('salesReturnForm.items.columns.unit'),
      dataIndex: 'unit',
      width: 60,
      render: (unit: string) => <Text>{unit || '-'}</Text>,
    },
    {
      title: t('salesReturnForm.items.columns.unitPrice'),
      dataIndex: 'unit_price',
      width: 100,
      align: 'right' as const,
      render: (price: number) => <Text>{formatPrice(price)}</Text>,
    },
    {
      title: t('salesReturnForm.items.columns.originalQuantity'),
      dataIndex: 'original_quantity',
      width: 80,
      align: 'center' as const,
      render: (qty: number) => <Text>{qty}</Text>,
    },
    {
      title: t('salesReturnForm.items.columns.returnQuantity'),
      dataIndex: 'return_quantity',
      width: 120,
      render: (qty: number, record: ReturnItemFormData) => (
        <InputNumber
          value={qty}
          onChange={(value) => handleQuantityChange(record.key, value as number)}
          min={0}
          max={record.original_quantity}
          precision={2}
          style={{ width: '100%' }}
        />
      ),
    },
    {
      title: t('salesReturnForm.items.columns.refundAmount'),
      dataIndex: 'refund_amount',
      width: 100,
      align: 'right' as const,
      render: (amount: number) => (
        <Text strong className="refund-amount">
          {formatPrice(amount)}
        </Text>
      ),
    },
    {
      title: t('salesReturnForm.items.columns.condition'),
      dataIndex: 'condition_on_return',
      width: 120,
      render: (condition: string, record: ReturnItemFormData) => (
        <Select
          value={condition}
          onChange={(value) => handleItemConditionChange(record.key, value as string)}
          optionList={CONDITION_OPTIONS}
          style={{ width: '100%' }}
          size="small"
          disabled={!record.selected}
        />
      ),
    },
    {
      title: t('salesReturnForm.items.columns.reason'),
      dataIndex: 'reason',
      width: 150,
      render: (reason: string, record: ReturnItemFormData) => (
        <Input
          value={reason}
          onChange={(value) => handleItemReasonChange(record.key, value)}
          placeholder={t('salesReturnForm.items.columns.reasonPlaceholder')}
          size="small"
          disabled={!record.selected}
        />
      ),
    },
  ]

  return (
    <Container size="lg" className="sales-return-form-page">
      <Card className="sales-return-form-card">
        <div className="sales-return-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('salesReturnForm.title')}
          </Title>
        </div>

        {/* Order Selection Section */}
        <div className="form-section">
          <Title heading={5} className="section-title">
            {t('salesReturnForm.selectOrder.title')}
          </Title>
          <div className="form-row">
            <div className="form-field">
              <label className="form-label required">
                {t('salesReturnForm.selectOrder.label')}
              </label>
              <Select
                value={formData.sales_order_id || undefined}
                placeholder={t('salesReturnForm.selectOrder.placeholder')}
                onChange={handleOrderChange}
                optionList={orderOptions}
                filter
                remote
                onSearch={setOrderSearch}
                loading={ordersLoading}
                style={{ width: '100%' }}
                prefix={<IconSearch />}
                validateStatus={errors.sales_order_id ? 'error' : undefined}
                disabled={!!preSelectedOrderId}
                renderSelectedItem={(option: {
                  label?: string
                  orderNumber?: string
                  customerName?: string
                  status?: string
                }) => (
                  <span>
                    {option.orderNumber} - {option.customerName}
                  </span>
                )}
              />
              {errors.sales_order_id && (
                <Text type="danger" size="small">
                  {errors.sales_order_id}
                </Text>
              )}
            </div>
            <div className="form-field">
              <label className="form-label">{t('salesReturnForm.warehouse.label')}</label>
              <Select
                value={formData.warehouse_id || undefined}
                placeholder={t('salesReturnForm.warehouse.placeholder')}
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

        {/* Order Info Section */}
        {selectedOrder && (
          <div className="form-section order-info-section">
            <Title heading={5} className="section-title">
              {t('salesReturnForm.orderInfo.title')}
            </Title>
            <Descriptions
              data={[
                {
                  key: t('salesReturnForm.orderInfo.orderNumber'),
                  value: selectedOrder.order_number || '-',
                },
                {
                  key: t('salesReturnForm.orderInfo.customer'),
                  value: selectedOrder.customer_name || '-',
                },
                {
                  key: t('salesReturnForm.orderInfo.status'),
                  value: (
                    <Tag color={ORDER_STATUS_COLORS[selectedOrder.status || '']}>
                      {ORDER_STATUS_LABELS[selectedOrder.status || ''] || selectedOrder.status}
                    </Tag>
                  ),
                },
                {
                  key: t('salesReturnForm.orderInfo.orderAmount'),
                  value: formatPrice(selectedOrder.total_amount),
                },
              ]}
              row
            />
          </div>
        )}

        {/* Return Items Section */}
        {formData.items.length > 0 && (
          <div className="form-section">
            <div className="section-header">
              <Title heading={5} className="section-title">
                {t('salesReturnForm.items.title')}
              </Title>
              <Space>
                <Button theme="light" size="small" onClick={handleSelectAll}>
                  {t('salesReturnForm.items.selectAll')}
                </Button>
                <Button theme="light" size="small" onClick={handleDeselectAll}>
                  {t('salesReturnForm.items.deselectAll')}
                </Button>
              </Space>
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
              loading={orderLoading}
              empty={<Empty description={t('salesReturnForm.items.empty')} />}
            />
          </div>
        )}

        {/* Summary Section */}
        {calculations.itemCount > 0 && (
          <div className="form-section summary-section">
            <div className="summary-totals">
              <div className="summary-item">
                <Text type="tertiary">{t('salesReturnForm.summary.itemCount')}</Text>
                <Text>
                  {calculations.itemCount} {t('salesReturnForm.summary.itemCountUnit')}
                </Text>
              </div>
              <div className="summary-item">
                <Text type="tertiary">{t('salesReturnForm.summary.totalQuantity')}</Text>
                <Text>{safeToFixed(calculations.totalQuantity)}</Text>
              </div>
              <div className="summary-item total">
                <Text strong>{t('salesReturnForm.summary.totalRefund')}</Text>
                <Text strong className="total-amount">
                  {formatPrice(calculations.totalRefund)}
                </Text>
              </div>
            </div>
          </div>
        )}

        {/* Reason Section */}
        <div className="form-section">
          <div className="form-field">
            <label className="form-label required">{t('salesReturnForm.reason.label')}</label>
            <Input
              value={formData.reason}
              onChange={handleReasonChange}
              placeholder={t('salesReturnForm.reason.placeholder')}
              maxLength={500}
              showClear
              validateStatus={errors.reason ? 'error' : undefined}
            />
            {errors.reason && (
              <Text type="danger" size="small">
                {errors.reason}
              </Text>
            )}
          </div>
          <div className="form-field" style={{ marginTop: 'var(--spacing-4)' }}>
            <label className="form-label">{t('salesReturnForm.remark.label')}</label>
            <Input
              value={formData.remark}
              onChange={handleRemarkChange}
              placeholder={t('salesReturnForm.remark.placeholder')}
              maxLength={500}
              showClear
            />
          </div>
        </div>

        {/* Form Actions */}
        <div className="form-actions">
          <Space>
            <Button onClick={handleCancel} disabled={isSubmitting}>
              {t('salesReturnForm.actions.cancel')}
            </Button>
            <Button
              type="primary"
              onClick={handleSubmit}
              loading={isSubmitting}
              disabled={isSubmitting || calculations.itemCount === 0}
            >
              {t('salesReturnForm.actions.create')}
            </Button>
          </Space>
        </div>
      </Card>
    </Container>
  )
}

export default SalesReturnForm
