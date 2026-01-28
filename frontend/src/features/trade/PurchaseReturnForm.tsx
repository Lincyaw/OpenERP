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
import { Container } from '@/components/common/layout'
import { createPurchaseReturn } from '@/api/purchase-returns/purchase-returns'
import { listPurchaseOrders, getPurchaseOrderById } from '@/api/purchase-orders/purchase-orders'
import { listWarehouses } from '@/api/warehouses/warehouses'
import type {
  HandlerPurchaseOrderResponse,
  HandlerPurchaseOrderItemResponse,
  HandlerWarehouseListResponse,
  HandlerPurchaseOrderListResponse,
} from '@/api/models'
import './PurchaseReturnForm.css'
import { safeToFixed, safeFormatCurrency, createScopedLogger } from '@/utils'

const log = createScopedLogger('PurchaseReturnForm')

const { Title, Text } = Typography

// Return item form type
interface ReturnItemFormData {
  key: string
  purchase_order_item_id: string
  product_id: string
  product_code: string
  product_name: string
  unit: string
  unit_cost: number
  ordered_quantity: number
  received_quantity: number
  return_quantity: number
  refund_amount: number
  reason: string
  condition_on_return: string
  batch_number: string
  selected: boolean
}

// Return form data type
interface ReturnFormData {
  purchase_order_id: string
  warehouse_id?: string
  reason: string
  remark: string
  items: ReturnItemFormData[]
}

// Form validation schema
const returnFormSchema = z.object({
  purchase_order_id: z.string().min(1, '请选择原订单'),
  warehouse_id: z.string().optional(),
  reason: z.string().min(1, '请填写退货原因').max(500, '退货原因不能超过500字'),
  remark: z.string().max(500, '备注不能超过500字').optional(),
  items: z
    .array(
      z.object({
        purchase_order_item_id: z.string().min(1),
        return_quantity: z.number().positive('数量必须大于0'),
      })
    )
    .min(1, '请至少选择一个退货商品'),
})

// Condition options for returned goods
const CONDITION_OPTIONS = [
  { label: '完好', value: 'intact' },
  { label: '损坏', value: 'damaged' },
  { label: '有瑕疵', value: 'defective' },
  { label: '质量问题', value: 'quality_issue' },
  { label: '其他', value: 'other' },
]

// Order status labels
const ORDER_STATUS_LABELS: Record<string, string> = {
  DRAFT: '草稿',
  CONFIRMED: '已确认',
  PARTIAL_RECEIVED: '部分收货',
  COMPLETED: '已完成',
  CANCELLED: '已取消',
}

// Order status colors
const ORDER_STATUS_COLORS: Record<
  string,
  'blue' | 'cyan' | 'green' | 'grey' | 'orange' | 'violet'
> = {
  DRAFT: 'blue',
  CONFIRMED: 'cyan',
  PARTIAL_RECEIVED: 'orange',
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
 * Purchase return form component for creating purchase returns
 *
 * Features:
 * - Search and select existing purchase order (only received orders)
 * - Display order items with selection checkboxes
 * - Set return quantity for each item (up to received quantity)
 * - Set return reason and condition for each item
 * - Real-time refund amount calculation
 * - Form validation with Zod
 */
export function PurchaseReturnForm() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const preSelectedOrderId = searchParams.get('order_id')

  // Form state
  const [formData, setFormData] = useState<ReturnFormData>({
    purchase_order_id: '',
    warehouse_id: undefined,
    reason: '',
    remark: '',
    items: [],
  })

  // UI state
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})

  // Data for dropdowns
  const [orders, setOrders] = useState<HandlerPurchaseOrderListResponse[]>([])
  const [selectedOrder, setSelectedOrder] = useState<HandlerPurchaseOrderResponse | null>(null)
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
  const fetchOrders = useCallback(async (search?: string) => {
    setOrdersLoading(true)
    try {
      // Only get orders that have received goods (PARTIAL_RECEIVED or COMPLETED)
      const response = await listPurchaseOrders({
        page_size: 50,
        search: search || undefined,
        status: 'partial_received',
      })
      if (response.status === 200 && response.data.success && response.data.data) {
        // Also fetch completed orders
        const completedResponse = await listPurchaseOrders({
          page_size: 50,
          search: search || undefined,
          status: 'completed',
        })
        let allOrders = response.data.data
        if (
          completedResponse.status === 200 &&
          completedResponse.data.success &&
          completedResponse.data.data
        ) {
          allOrders = [...allOrders, ...completedResponse.data.data]
        }
        setOrders(allOrders)
      } else if (response.status !== 200 || !response.data.success) {
        log.error('Failed to fetch orders', response.data.error)
      }
    } catch (error) {
      log.error('Error fetching orders', error)
    } finally {
      setOrdersLoading(false)
    }
  }, [])

  // Fetch order detail
  const fetchOrderDetail = useCallback(async (orderId: string) => {
    if (!orderId) return
    setOrderLoading(true)
    try {
      const response = await getPurchaseOrderById(orderId)
      if (response.status === 200 && response.data.success && response.data.data) {
        setSelectedOrder(response.data.data)
        // Convert order items to return items (only items with received quantity)
        const returnItems: ReturnItemFormData[] = (response.data.data.items || [])
          .filter((item: HandlerPurchaseOrderItemResponse) => (item.received_quantity || 0) > 0)
          .map((item: HandlerPurchaseOrderItemResponse) => ({
            key: item.id || `item-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
            purchase_order_item_id: item.id || '',
            product_id: item.product_id || '',
            product_code: item.product_code || '',
            product_name: item.product_name || '',
            unit: item.unit || '',
            unit_cost: item.unit_cost || 0,
            ordered_quantity: item.ordered_quantity || 0,
            received_quantity: item.received_quantity || 0,
            return_quantity: 0,
            refund_amount: 0,
            reason: '',
            condition_on_return: 'intact',
            batch_number: '',
            selected: false,
          }))
        setFormData((prev) => ({
          ...prev,
          purchase_order_id: orderId,
          items: returnItems,
        }))
      }
    } catch {
      Toast.error('获取订单详情失败')
    } finally {
      setOrderLoading(false)
    }
  }, [])

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
          purchase_order_id: '',
          items: [],
        }))
      }
      setErrors((prev) => {
        const newErrors = { ...prev }
        delete newErrors.purchase_order_id
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
          return_quantity: selected ? item.received_quantity : 0,
          refund_amount: selected ? item.unit_cost * item.received_quantity : 0,
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
        const clampedQty = Math.min(Math.max(0, qty), item.received_quantity)
        return {
          ...item,
          return_quantity: clampedQty,
          refund_amount: item.unit_cost * clampedQty,
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

  // Handle item batch number change
  const handleItemBatchChange = useCallback((itemKey: string, batchNumber: string) => {
    setFormData((prev) => ({
      ...prev,
      items: prev.items.map((item) => {
        if (item.key !== itemKey) return item
        return { ...item, batch_number: batchNumber }
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
        return_quantity: item.received_quantity,
        refund_amount: item.unit_cost * item.received_quantity,
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
        purchase_order_item_id: item.purchase_order_item_id,
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
  }, [formData])

  // Handle form submission
  const handleSubmit = useCallback(async () => {
    if (!validateForm()) {
      Toast.error('请检查表单填写是否正确')
      return
    }

    setIsSubmitting(true)
    try {
      // Filter selected items and prepare for API
      const selectedItems = formData.items.filter(
        (item) => item.selected && item.return_quantity > 0
      )

      const response = await createPurchaseReturn({
        purchase_order_id: formData.purchase_order_id,
        warehouse_id: formData.warehouse_id,
        reason: formData.reason,
        remark: formData.remark || undefined,
        items: selectedItems.map((item) => ({
          purchase_order_item_id: item.purchase_order_item_id,
          return_quantity: item.return_quantity,
          reason: item.reason || undefined,
          condition_on_return: item.condition_on_return || undefined,
          batch_number: item.batch_number || undefined,
        })),
      })

      if (response.status !== 201 || !response.data.success) {
        throw new Error((response.data.error as { message?: string })?.message || '创建失败')
      }

      Toast.success('退货单创建成功')
      navigate('/trade/purchase-returns')
    } catch (error) {
      Toast.error(error instanceof Error ? error.message : '创建退货单失败')
    } finally {
      setIsSubmitting(false)
    }
  }, [formData, navigate, validateForm])

  // Handle cancel
  const handleCancel = useCallback(() => {
    navigate('/trade/purchase-returns')
  }, [navigate])

  // Order options for select
  const orderOptions = useMemo(
    () =>
      orders.map((o) => ({
        value: o.id || '',
        label: `${o.order_number} - ${o.supplier_name}`,
        orderNumber: o.order_number,
        supplierName: o.supplier_name,
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
        extra: w.is_default ? '(默认)' : undefined,
      })),
    [warehouses]
  )

  // Table columns for return items
  const itemColumns = [
    {
      title: '选择',
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
      title: '商品编码',
      dataIndex: 'product_code',
      width: 120,
      render: (code: string) => <Text>{code || '-'}</Text>,
    },
    {
      title: '商品名称',
      dataIndex: 'product_name',
      width: 180,
      ellipsis: true,
      render: (name: string) => <Text>{name || '-'}</Text>,
    },
    {
      title: '单位',
      dataIndex: 'unit',
      width: 60,
      render: (unit: string) => <Text>{unit || '-'}</Text>,
    },
    {
      title: '单价',
      dataIndex: 'unit_cost',
      width: 100,
      align: 'right' as const,
      render: (price: number) => <Text>{formatPrice(price)}</Text>,
    },
    {
      title: '已收数量',
      dataIndex: 'received_quantity',
      width: 80,
      align: 'center' as const,
      render: (qty: number) => <Text>{qty}</Text>,
    },
    {
      title: '退货数量',
      dataIndex: 'return_quantity',
      width: 120,
      render: (qty: number, record: ReturnItemFormData) => (
        <InputNumber
          value={qty}
          onChange={(value) => handleQuantityChange(record.key, value as number)}
          min={0}
          max={record.received_quantity}
          precision={2}
          style={{ width: '100%' }}
        />
      ),
    },
    {
      title: '退款金额',
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
      title: '商品状况',
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
      title: '批次号',
      dataIndex: 'batch_number',
      width: 100,
      render: (batch: string, record: ReturnItemFormData) => (
        <Input
          value={batch}
          onChange={(value) => handleItemBatchChange(record.key, value)}
          placeholder="批次"
          size="small"
          disabled={!record.selected}
        />
      ),
    },
    {
      title: '退货原因',
      dataIndex: 'reason',
      width: 130,
      render: (reason: string, record: ReturnItemFormData) => (
        <Input
          value={reason}
          onChange={(value) => handleItemReasonChange(record.key, value)}
          placeholder="原因"
          size="small"
          disabled={!record.selected}
        />
      ),
    },
  ]

  return (
    <Container size="lg" className="purchase-return-form-page">
      <Card className="purchase-return-form-card">
        <div className="purchase-return-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            新建采购退货
          </Title>
        </div>

        {/* Order Selection Section */}
        <div className="form-section">
          <Title heading={5} className="section-title">
            选择原订单
          </Title>
          <div className="form-row">
            <div className="form-field">
              <label className="form-label required">原采购订单</label>
              <Select
                value={formData.purchase_order_id || undefined}
                placeholder="搜索订单号或供应商名称..."
                onChange={handleOrderChange}
                optionList={orderOptions}
                filter
                remote
                onSearch={setOrderSearch}
                loading={ordersLoading}
                style={{ width: '100%' }}
                prefix={<IconSearch />}
                validateStatus={errors.purchase_order_id ? 'error' : undefined}
                disabled={!!preSelectedOrderId}
                renderSelectedItem={(option: {
                  label?: string
                  orderNumber?: string
                  supplierName?: string
                  status?: string
                }) => (
                  <span>
                    {option.orderNumber} - {option.supplierName}
                  </span>
                )}
              />
              {errors.purchase_order_id && (
                <Text type="danger" size="small">
                  {errors.purchase_order_id}
                </Text>
              )}
            </div>
            <div className="form-field">
              <label className="form-label">退货出库仓库</label>
              <Select
                value={formData.warehouse_id || undefined}
                placeholder="选择退货出库仓库"
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
              订单信息
            </Title>
            <Descriptions
              data={[
                { key: '订单号', value: selectedOrder.order_number || '-' },
                { key: '供应商', value: selectedOrder.supplier_name || '-' },
                {
                  key: '状态',
                  value: (
                    <Tag color={ORDER_STATUS_COLORS[selectedOrder.status || '']}>
                      {ORDER_STATUS_LABELS[selectedOrder.status || ''] || selectedOrder.status}
                    </Tag>
                  ),
                },
                { key: '订单金额', value: formatPrice(selectedOrder.total_amount) },
                {
                  key: '收货进度',
                  value: `${selectedOrder.receive_progress || 0}%`,
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
                选择退货商品
              </Title>
              <Space>
                <Button theme="light" size="small" onClick={handleSelectAll}>
                  全选
                </Button>
                <Button theme="light" size="small" onClick={handleDeselectAll}>
                  取消全选
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
              empty={<Empty description="暂无可退货商品" />}
              scroll={{ x: 1300 }}
            />
          </div>
        )}

        {/* No items message for orders without received goods */}
        {selectedOrder && formData.items.length === 0 && !orderLoading && (
          <div className="form-section">
            <Empty description="该订单暂无可退货商品（只有已收货的商品才能退货）" />
          </div>
        )}

        {/* Summary Section */}
        {calculations.itemCount > 0 && (
          <div className="form-section summary-section">
            <div className="summary-totals">
              <div className="summary-item">
                <Text type="tertiary">退货商品数：</Text>
                <Text>{calculations.itemCount} 种</Text>
              </div>
              <div className="summary-item">
                <Text type="tertiary">退货总数量：</Text>
                <Text>{safeToFixed(calculations.totalQuantity)}</Text>
              </div>
              <div className="summary-item total">
                <Text strong>退款总额：</Text>
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
            <label className="form-label required">退货原因</label>
            <Input
              value={formData.reason}
              onChange={handleReasonChange}
              placeholder="请填写退货原因"
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
            <label className="form-label">备注</label>
            <Input
              value={formData.remark}
              onChange={handleRemarkChange}
              placeholder="备注信息（可选）"
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
              disabled={isSubmitting || calculations.itemCount === 0}
            >
              创建退货单
            </Button>
          </Space>
        </div>
      </Card>
    </Container>
  )
}

export default PurchaseReturnForm
