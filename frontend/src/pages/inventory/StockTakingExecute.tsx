import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Spin,
  Button,
  Table,
  InputNumber,
  Input,
  Modal,
  Progress,
  Descriptions,
} from '@douyinfe/semi-ui'
import {
  IconArrowLeft,
  IconPlay,
  IconSend,
  IconRefresh,
  IconCheckCircle,
  IconClose,
} from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { getStockTaking } from '@/api/stock-taking/stock-taking'
import type {
  HandlerStockTakingResponse,
  HandlerStockTakingItemResponse,
  HandlerRecordCountRequest,
} from '@/api/models'
import './StockTakingExecute.css'

const { Title, Text } = Typography

// Status colors
const STATUS_COLORS: Record<string, string> = {
  DRAFT: 'grey',
  COUNTING: 'blue',
  PENDING_APPROVAL: 'orange',
  APPROVED: 'green',
  REJECTED: 'red',
  CANCELLED: 'grey',
}

// Status labels
const STATUS_LABELS: Record<string, string> = {
  DRAFT: '草稿',
  COUNTING: '盘点中',
  PENDING_APPROVAL: '待审批',
  APPROVED: '已通过',
  REJECTED: '已拒绝',
  CANCELLED: '已取消',
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
 * Format date for display
 */
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
}

// Local item state for editing
interface LocalItemState {
  product_id: string
  actual_quantity: number | null
  remark: string
  dirty: boolean
}

/**
 * Stock Taking Execute Page
 *
 * Features:
 * - Display stock taking details and items
 * - Allow entering actual counted quantities
 * - Calculate and display differences in real-time
 * - Support submitting for approval when all items are counted
 *
 * Requirements:
 * - 实现盘点录入界面
 * - 实时计算盘点差异
 * - 支持提交审批
 */
export default function StockTakingExecutePage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const stockTakingApi = useMemo(() => getStockTaking(), [])

  // State for stock taking data
  const [stockTaking, setStockTaking] = useState<HandlerStockTakingResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)

  // State for local edits
  const [localItems, setLocalItems] = useState<Map<string, LocalItemState>>(new Map())

  // Modal states
  const [showSubmitModal, setShowSubmitModal] = useState(false)
  const [showCancelModal, setShowCancelModal] = useState(false)
  const [cancelReason, setCancelReason] = useState('')

  // Fetch stock taking data
  const fetchStockTaking = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      const response = await stockTakingApi.getInventoryStockTakingsId(id)
      if (response.success && response.data) {
        setStockTaking(response.data)
        // Initialize local items from fetched data
        const newLocalItems = new Map<string, LocalItemState>()
        response.data.items?.forEach((item) => {
          if (item.product_id) {
            newLocalItems.set(item.product_id, {
              product_id: item.product_id,
              actual_quantity: item.counted ? (item.actual_quantity ?? 0) : null,
              remark: item.remark || '',
              dirty: false,
            })
          }
        })
        setLocalItems(newLocalItems)
      } else {
        Toast.error('获取盘点单失败')
        navigate('/inventory/stock-taking')
      }
    } catch {
      Toast.error('获取盘点单失败')
      navigate('/inventory/stock-taking')
    } finally {
      setLoading(false)
    }
  }, [id, stockTakingApi, navigate])

  useEffect(() => {
    fetchStockTaking()
  }, [fetchStockTaking])

  // Handle back navigation
  const handleBack = useCallback(() => {
    navigate('/inventory/stock-taking')
  }, [navigate])

  // Handle start counting
  const handleStartCounting = useCallback(async () => {
    if (!id) return

    try {
      const response = await stockTakingApi.postInventoryStockTakingsIdStart(id)
      if (response.success && response.data) {
        Toast.success('已开始盘点')
        setStockTaking(response.data)
      } else {
        Toast.error(response.error?.message || '开始盘点失败')
      }
    } catch {
      Toast.error('开始盘点失败')
    }
  }, [id, stockTakingApi])

  // Handle local quantity change
  const handleQuantityChange = useCallback((productId: string, value: number | string) => {
    setLocalItems((prev) => {
      const newMap = new Map(prev)
      const existing = newMap.get(productId)
      if (existing) {
        const numValue = typeof value === 'string' ? parseFloat(value) : value
        newMap.set(productId, {
          ...existing,
          actual_quantity: isNaN(numValue) ? null : numValue,
          dirty: true,
        })
      }
      return newMap
    })
  }, [])

  // Handle local remark change
  const handleRemarkChange = useCallback((productId: string, value: string) => {
    setLocalItems((prev) => {
      const newMap = new Map(prev)
      const existing = newMap.get(productId)
      if (existing) {
        newMap.set(productId, {
          ...existing,
          remark: value,
          dirty: true,
        })
      }
      return newMap
    })
  }, [])

  // Save single item count
  const handleSaveItemCount = useCallback(
    async (productId: string) => {
      if (!id) return

      const localItem = localItems.get(productId)
      if (!localItem || localItem.actual_quantity === null) {
        Toast.warning('请输入实盘数量')
        return
      }

      try {
        const request: HandlerRecordCountRequest = {
          product_id: productId,
          actual_quantity: localItem.actual_quantity,
          remark: localItem.remark || undefined,
        }

        const response = await stockTakingApi.postInventoryStockTakingsIdCount(id, request)
        if (response.success && response.data) {
          Toast.success('保存成功')
          setStockTaking(response.data)
          // Update local item as not dirty
          setLocalItems((prev) => {
            const newMap = new Map(prev)
            const existing = newMap.get(productId)
            if (existing) {
              newMap.set(productId, {
                ...existing,
                dirty: false,
              })
            }
            return newMap
          })
        } else {
          Toast.error(response.error?.message || '保存失败')
        }
      } catch {
        Toast.error('保存失败')
      }
    },
    [id, localItems, stockTakingApi]
  )

  // Save all dirty items
  const handleSaveAllCounts = useCallback(async () => {
    if (!id) return

    const dirtyItems = Array.from(localItems.values()).filter(
      (item) => item.dirty && item.actual_quantity !== null
    )

    if (dirtyItems.length === 0) {
      Toast.info('没有需要保存的数据')
      return
    }

    setSubmitting(true)
    try {
      const counts = dirtyItems.map((item) => ({
        product_id: item.product_id,
        actual_quantity: item.actual_quantity!,
        remark: item.remark || undefined,
      }))

      const response = await stockTakingApi.postInventoryStockTakingsIdCounts(id, { counts })
      if (response.success && response.data) {
        Toast.success(`已保存 ${dirtyItems.length} 条记录`)
        setStockTaking(response.data)
        // Clear dirty flags
        setLocalItems((prev) => {
          const newMap = new Map(prev)
          dirtyItems.forEach((item) => {
            const existing = newMap.get(item.product_id)
            if (existing) {
              newMap.set(item.product_id, { ...existing, dirty: false })
            }
          })
          return newMap
        })
      } else {
        Toast.error(response.error?.message || '保存失败')
      }
    } catch {
      Toast.error('保存失败')
    } finally {
      setSubmitting(false)
    }
  }, [id, localItems, stockTakingApi])

  // Handle submit for approval
  const handleSubmitForApproval = useCallback(async () => {
    if (!id) return

    setSubmitting(true)
    try {
      const response = await stockTakingApi.postInventoryStockTakingsIdSubmit(id)
      if (response.success && response.data) {
        Toast.success('已提交审批')
        setStockTaking(response.data)
        setShowSubmitModal(false)
      } else {
        Toast.error(response.error?.message || '提交审批失败')
      }
    } catch {
      Toast.error('提交审批失败')
    } finally {
      setSubmitting(false)
    }
  }, [id, stockTakingApi])

  // Handle cancel
  const handleCancel = useCallback(async () => {
    if (!id) return

    setSubmitting(true)
    try {
      const response = await stockTakingApi.postInventoryStockTakingsIdCancel(id, {
        reason: cancelReason || undefined,
      })
      if (response.success && response.data) {
        Toast.success('已取消盘点')
        setStockTaking(response.data)
        setShowCancelModal(false)
        setCancelReason('')
      } else {
        Toast.error(response.error?.message || '取消失败')
      }
    } catch {
      Toast.error('取消失败')
    } finally {
      setSubmitting(false)
    }
  }, [id, cancelReason, stockTakingApi])

  // Calculate local difference for an item
  const calculateDifference = useCallback(
    (item: HandlerStockTakingItemResponse): { qty: number | null; amount: number | null } => {
      const localItem = localItems.get(item.product_id || '')
      const actualQty = localItem?.actual_quantity
      const systemQty = item.system_quantity || 0
      const unitCost = item.unit_cost || 0

      if (actualQty === null || actualQty === undefined) {
        return { qty: null, amount: null }
      }

      const diffQty = actualQty - systemQty
      const diffAmount = diffQty * unitCost

      return { qty: diffQty, amount: diffAmount }
    },
    [localItems]
  )

  // Check if item is editable
  const isEditable = useMemo(() => {
    return stockTaking?.status === 'DRAFT' || stockTaking?.status === 'COUNTING'
  }, [stockTaking?.status])

  // Calculate totals from local state
  const localTotals = useMemo(() => {
    let totalDiff = 0
    let countedItems = 0

    stockTaking?.items?.forEach((item) => {
      const localItem = localItems.get(item.product_id || '')
      if (localItem?.actual_quantity !== null && localItem?.actual_quantity !== undefined) {
        const systemQty = item.system_quantity || 0
        const unitCost = item.unit_cost || 0
        const diffQty = localItem.actual_quantity - systemQty
        totalDiff += diffQty * unitCost
        countedItems++
      } else if (item.counted) {
        // Already counted items from server
        totalDiff += item.difference_amount || 0
        countedItems++
      }
    })

    return {
      totalDiff,
      countedItems,
      totalItems: stockTaking?.items?.length || 0,
      progress: stockTaking?.items?.length ? (countedItems / stockTaking.items.length) * 100 : 0,
    }
  }, [stockTaking?.items, localItems])

  // Check if can submit
  const canSubmit = useMemo(() => {
    if (!isEditable) return false
    return localTotals.countedItems === localTotals.totalItems && localTotals.totalItems > 0
  }, [isEditable, localTotals])

  // Check if has dirty items
  const hasDirtyItems = useMemo(() => {
    return Array.from(localItems.values()).some(
      (item) => item.dirty && item.actual_quantity !== null
    )
  }, [localItems])

  // Table columns
  const tableColumns = useMemo(
    () => [
      {
        title: '商品编码',
        dataIndex: 'product_code',
        width: 120,
        fixed: 'left' as const,
      },
      {
        title: '商品名称',
        dataIndex: 'product_name',
        width: 180,
      },
      {
        title: '单位',
        dataIndex: 'unit',
        width: 60,
        align: 'center' as const,
      },
      {
        title: '系统数量',
        dataIndex: 'system_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => formatQuantity(qty),
      },
      {
        title: '实盘数量',
        dataIndex: 'actual_quantity',
        width: 140,
        render: (_: unknown, record: HandlerStockTakingItemResponse) => {
          const localItem = localItems.get(record.product_id || '')
          if (!isEditable) {
            return formatQuantity(record.actual_quantity)
          }
          return (
            <InputNumber
              value={localItem?.actual_quantity ?? undefined}
              min={0}
              precision={2}
              placeholder="输入数量"
              onChange={(value) => handleQuantityChange(record.product_id || '', value as number)}
              style={{ width: '100%' }}
            />
          )
        },
      },
      {
        title: '差异数量',
        dataIndex: 'difference_qty',
        width: 100,
        align: 'right' as const,
        render: (_: unknown, record: HandlerStockTakingItemResponse) => {
          const { qty } = calculateDifference(record)
          if (qty === null) return '-'
          return (
            <span className={qty > 0 ? 'diff-positive' : qty < 0 ? 'diff-negative' : ''}>
              {qty > 0 ? '+' : ''}
              {formatQuantity(qty)}
            </span>
          )
        },
      },
      {
        title: '差异金额',
        dataIndex: 'difference_amount',
        width: 110,
        align: 'right' as const,
        render: (_: unknown, record: HandlerStockTakingItemResponse) => {
          const { amount } = calculateDifference(record)
          if (amount === null) return '-'
          return (
            <span className={amount > 0 ? 'diff-positive' : amount < 0 ? 'diff-negative' : ''}>
              {amount > 0 ? '+' : ''}
              {formatCurrency(amount)}
            </span>
          )
        },
      },
      {
        title: '备注',
        dataIndex: 'remark',
        width: 150,
        render: (_: unknown, record: HandlerStockTakingItemResponse) => {
          const localItem = localItems.get(record.product_id || '')
          if (!isEditable) {
            return record.remark || '-'
          }
          return (
            <Input
              value={localItem?.remark ?? ''}
              placeholder="备注"
              onChange={(value) => handleRemarkChange(record.product_id || '', value)}
              style={{ width: '100%' }}
            />
          )
        },
      },
      {
        title: '状态',
        dataIndex: 'counted',
        width: 80,
        align: 'center' as const,
        render: (counted: boolean, record: HandlerStockTakingItemResponse) => {
          const localItem = localItems.get(record.product_id || '')
          const hasValue =
            localItem?.actual_quantity !== null && localItem?.actual_quantity !== undefined
          if (counted || hasValue) {
            return <Tag color="green">已盘</Tag>
          }
          return <Tag color="grey">未盘</Tag>
        },
      },
      {
        title: '操作',
        width: 80,
        fixed: 'right' as const,
        render: (_: unknown, record: HandlerStockTakingItemResponse) => {
          if (!isEditable) return null
          const localItem = localItems.get(record.product_id || '')
          if (!localItem?.dirty) return null
          return (
            <Button
              size="small"
              type="primary"
              theme="light"
              onClick={() => handleSaveItemCount(record.product_id || '')}
            >
              保存
            </Button>
          )
        },
      },
    ],
    [
      localItems,
      isEditable,
      calculateDifference,
      handleQuantityChange,
      handleRemarkChange,
      handleSaveItemCount,
    ]
  )

  if (loading) {
    return (
      <Container size="lg" className="stock-taking-execute-page">
        <div className="loading-container">
          <Spin size="large" />
          <Text type="tertiary">加载中...</Text>
        </div>
      </Container>
    )
  }

  if (!stockTaking) {
    return (
      <Container size="lg" className="stock-taking-execute-page">
        <Card>
          <Text type="danger">盘点单不存在</Text>
          <Button onClick={handleBack} style={{ marginTop: 'var(--spacing-4)' }}>
            返回列表
          </Button>
        </Card>
      </Container>
    )
  }

  return (
    <Container size="full" className="stock-taking-execute-page">
      {/* Header */}
      <div className="stock-taking-execute-header">
        <div className="header-left">
          <Button icon={<IconArrowLeft />} theme="borderless" onClick={handleBack}>
            返回
          </Button>
          <Title heading={4} style={{ margin: 0 }}>
            盘点执行 - {stockTaking.taking_number}
          </Title>
          <Tag color={STATUS_COLORS[stockTaking.status || ''] || 'grey'}>
            {STATUS_LABELS[stockTaking.status || ''] || stockTaking.status}
          </Tag>
        </div>
        <div className="header-right">
          {stockTaking.status === 'DRAFT' && (
            <Button icon={<IconPlay />} type="primary" onClick={handleStartCounting}>
              开始盘点
            </Button>
          )}
          {isEditable && (
            <>
              <Button
                icon={<IconRefresh />}
                onClick={handleSaveAllCounts}
                loading={submitting}
                disabled={!hasDirtyItems}
              >
                保存全部
              </Button>
              <Button
                icon={<IconSend />}
                type="primary"
                onClick={() => setShowSubmitModal(true)}
                disabled={!canSubmit}
              >
                提交审批
              </Button>
              <Button
                icon={<IconClose />}
                type="danger"
                theme="light"
                onClick={() => setShowCancelModal(true)}
              >
                取消盘点
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Summary Card */}
      <Card className="stock-taking-summary-card">
        <div className="summary-grid">
          <Descriptions row>
            <Descriptions.Item itemKey="仓库">{stockTaking.warehouse_name}</Descriptions.Item>
            <Descriptions.Item itemKey="盘点日期">
              {formatDate(stockTaking.taking_date)}
            </Descriptions.Item>
            <Descriptions.Item itemKey="创建人">{stockTaking.created_by_name}</Descriptions.Item>
            <Descriptions.Item itemKey="备注">{stockTaking.remark || '-'}</Descriptions.Item>
          </Descriptions>

          <div className="progress-section">
            <div className="progress-info">
              <Text strong>盘点进度</Text>
              <Text type="tertiary">
                {localTotals.countedItems}/{localTotals.totalItems} 项已盘点
              </Text>
            </div>
            <Progress
              percent={Math.round(localTotals.progress)}
              showInfo
              style={{ width: 200 }}
              stroke={localTotals.progress === 100 ? 'var(--color-success)' : undefined}
            />
          </div>

          <div className="total-difference">
            <Text type="tertiary">差异金额合计</Text>
            <span
              className={`total-value ${localTotals.totalDiff > 0 ? 'diff-positive' : localTotals.totalDiff < 0 ? 'diff-negative' : ''}`}
            >
              {localTotals.totalDiff > 0 ? '+' : ''}
              {formatCurrency(localTotals.totalDiff)}
            </span>
          </div>
        </div>
      </Card>

      {/* Items Table */}
      <Card className="stock-taking-items-card">
        <div className="items-header">
          <Title heading={5} style={{ margin: 0 }}>
            盘点明细
          </Title>
          <Button icon={<IconRefresh />} theme="borderless" onClick={fetchStockTaking}>
            刷新
          </Button>
        </div>
        <Table
          dataSource={stockTaking.items || []}
          columns={tableColumns}
          rowKey="id"
          pagination={false}
          scroll={{ x: 1200 }}
          size="small"
        />
      </Card>

      {/* Submit Confirmation Modal */}
      <Modal
        title="提交审批确认"
        visible={showSubmitModal}
        onCancel={() => setShowSubmitModal(false)}
        onOk={handleSubmitForApproval}
        okText="确认提交"
        cancelText="取消"
        confirmLoading={submitting}
      >
        <div className="submit-modal-content">
          <div className="confirm-item">
            <IconCheckCircle style={{ color: 'var(--color-success)', fontSize: 48 }} />
          </div>
          <Text>所有商品已完成盘点，确定提交审批吗？</Text>
          <div className="confirm-details">
            <Descriptions row>
              <Descriptions.Item itemKey="盘点商品数">
                {localTotals.totalItems} 项
              </Descriptions.Item>
              <Descriptions.Item itemKey="差异金额">
                <span
                  className={
                    localTotals.totalDiff > 0
                      ? 'diff-positive'
                      : localTotals.totalDiff < 0
                        ? 'diff-negative'
                        : ''
                  }
                >
                  {localTotals.totalDiff > 0 ? '+' : ''}
                  {formatCurrency(localTotals.totalDiff)}
                </span>
              </Descriptions.Item>
            </Descriptions>
          </div>
        </div>
      </Modal>

      {/* Cancel Confirmation Modal */}
      <Modal
        title="取消盘点确认"
        visible={showCancelModal}
        onCancel={() => setShowCancelModal(false)}
        onOk={handleCancel}
        okText="确认取消"
        okType="danger"
        cancelText="返回"
        confirmLoading={submitting}
      >
        <div className="cancel-modal-content">
          <Text>确定要取消此盘点单吗？取消后无法恢复。</Text>
          <Input
            placeholder="请输入取消原因（可选）"
            value={cancelReason}
            onChange={setCancelReason}
            style={{ marginTop: 'var(--spacing-4)' }}
          />
        </div>
      </Modal>
    </Container>
  )
}
