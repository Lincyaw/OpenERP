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
} from '@douyinfe/semi-ui-19'
import {
  IconArrowLeft,
  IconPlay,
  IconSend,
  IconRefresh,
  IconTickCircle,
  IconClose,
} from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Container } from '@/components/common/layout'
import { useFormatters } from '@/hooks/useFormatters'
import { getStockTaking } from '@/api/stock-taking/stock-taking'
import type {
  HandlerStockTakingResponse,
  HandlerStockTakingItemResponse,
  HandlerRecordCountRequest,
} from '@/api/models'
import './StockTakingExecute.css'

const { Title, Text } = Typography

import type { TagProps } from '@douyinfe/semi-ui-19/lib/es/tag'

type TagColor = TagProps['color']

// Status colors
const STATUS_COLORS: Record<string, TagColor> = {
  DRAFT: 'grey',
  COUNTING: 'blue',
  PENDING_APPROVAL: 'orange',
  APPROVED: 'green',
  REJECTED: 'red',
  CANCELLED: 'grey',
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
  const { t } = useTranslation(['inventory', 'common'])
  const { formatCurrency: formatCurrencyBase, formatDate: formatDateBase } = useFormatters()
  const stockTakingApi = useMemo(() => getStockTaking(), [])

  // Wrapper functions to handle undefined values
  const formatCurrency = useCallback(
    (value?: number): string => (value !== undefined ? formatCurrencyBase(value) : '-'),
    [formatCurrencyBase]
  )
  const formatDate = useCallback(
    (date?: string, style?: 'date' | 'dateTime'): string =>
      date ? formatDateBase(date, style === 'dateTime' ? 'medium' : 'short') : '-',
    [formatDateBase]
  )

  /**
   * Format quantity for display with 2 decimal places
   * Handles both number and string types (API may return decimal as string)
   */
  const formatQuantity = useCallback((quantity?: number | string): string => {
    if (quantity === undefined || quantity === null) return '-'
    const num = typeof quantity === 'string' ? parseFloat(quantity) : quantity
    if (isNaN(num)) return '-'
    return num.toFixed(2)
  }, [])

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
            // Convert string to number (API may return decimal as string)
            const actualQty =
              typeof item.actual_quantity === 'string'
                ? parseFloat(item.actual_quantity)
                : item.actual_quantity
            newLocalItems.set(item.product_id, {
              product_id: item.product_id,
              actual_quantity: item.counted ? (actualQty ?? 0) : null,
              remark: item.remark || '',
              dirty: false,
            })
          }
        })
        setLocalItems(newLocalItems)
      } else {
        Toast.error(t('stockTaking.execute.messages.fetchError'))
        navigate('/inventory/stock-taking')
      }
    } catch {
      Toast.error(t('stockTaking.execute.messages.fetchError'))
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
        Toast.success(t('stockTaking.execute.messages.startSuccess'))
        setStockTaking(response.data)
      } else {
        Toast.error(response.error?.message || t('stockTaking.execute.messages.startError'))
      }
    } catch {
      Toast.error(t('stockTaking.execute.messages.startError'))
    }
  }, [id, stockTakingApi, t])

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
        Toast.warning(t('stockTaking.execute.messages.inputQuantityFirst'))
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
          Toast.success(t('stockTaking.execute.messages.saveSuccess'))
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
          Toast.error(response.error?.message || t('stockTaking.execute.messages.saveError'))
        }
      } catch {
        Toast.error(t('stockTaking.execute.messages.saveError'))
      }
    },
    [id, localItems, stockTakingApi, t]
  )

  // Save all dirty items
  const handleSaveAllCounts = useCallback(async () => {
    if (!id) return

    const dirtyItems = Array.from(localItems.values()).filter(
      (item) => item.dirty && item.actual_quantity !== null
    )

    if (dirtyItems.length === 0) {
      Toast.info(t('stockTaking.execute.messages.noDataToSave'))
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
        Toast.success(
          t('stockTaking.execute.messages.saveAllSuccess', { count: dirtyItems.length })
        )
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
        Toast.error(response.error?.message || t('stockTaking.execute.messages.saveAllError'))
      }
    } catch {
      Toast.error(t('stockTaking.execute.messages.saveAllError'))
    } finally {
      setSubmitting(false)
    }
  }, [id, localItems, stockTakingApi, t])

  // Handle submit for approval
  const handleSubmitForApproval = useCallback(async () => {
    if (!id) return

    setSubmitting(true)
    try {
      const response = await stockTakingApi.postInventoryStockTakingsIdSubmit(id)
      if (response.success && response.data) {
        Toast.success(t('stockTaking.execute.messages.submitSuccess'))
        setStockTaking(response.data)
        setShowSubmitModal(false)
      } else {
        Toast.error(response.error?.message || t('stockTaking.execute.messages.submitError'))
      }
    } catch {
      Toast.error(t('stockTaking.execute.messages.submitError'))
    } finally {
      setSubmitting(false)
    }
  }, [id, stockTakingApi, t])

  // Handle cancel
  const handleCancel = useCallback(async () => {
    if (!id) return

    setSubmitting(true)
    try {
      const response = await stockTakingApi.postInventoryStockTakingsIdCancel(id, {
        reason: cancelReason || undefined,
      })
      if (response.success && response.data) {
        Toast.success(t('stockTaking.execute.messages.cancelSuccess'))
        setStockTaking(response.data)
        setShowCancelModal(false)
        setCancelReason('')
      } else {
        Toast.error(response.error?.message || t('stockTaking.execute.messages.cancelError'))
      }
    } catch {
      Toast.error(t('stockTaking.execute.messages.cancelError'))
    } finally {
      setSubmitting(false)
    }
  }, [id, cancelReason, stockTakingApi, t])

  // Calculate local difference for an item
  const calculateDifference = useCallback(
    (item: HandlerStockTakingItemResponse): { qty: number | null; amount: number | null } => {
      const localItem = localItems.get(item.product_id || '')
      const actualQty = localItem?.actual_quantity
      // Convert string to number (API may return decimal as string)
      const systemQty =
        typeof item.system_quantity === 'string'
          ? parseFloat(item.system_quantity)
          : item.system_quantity || 0
      const unitCost =
        typeof item.unit_cost === 'string' ? parseFloat(item.unit_cost) : item.unit_cost || 0

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
        // Convert string to number (API may return decimal as string)
        const systemQty =
          typeof item.system_quantity === 'string'
            ? parseFloat(item.system_quantity)
            : item.system_quantity || 0
        const unitCost =
          typeof item.unit_cost === 'string' ? parseFloat(item.unit_cost) : item.unit_cost || 0
        const diffQty = localItem.actual_quantity - systemQty
        totalDiff += diffQty * unitCost
        countedItems++
      } else if (item.counted) {
        // Already counted items from server
        const diffAmount =
          typeof item.difference_amount === 'string'
            ? parseFloat(item.difference_amount)
            : item.difference_amount || 0
        totalDiff += diffAmount
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
        title: t('stockTaking.execute.columns.productCode'),
        dataIndex: 'product_code',
        width: 120,
        fixed: 'left' as const,
      },
      {
        title: t('stockTaking.execute.columns.productName'),
        dataIndex: 'product_name',
        width: 180,
      },
      {
        title: t('stockTaking.execute.columns.unit'),
        dataIndex: 'unit',
        width: 60,
        align: 'center' as const,
      },
      {
        title: t('stockTaking.execute.columns.systemQuantity'),
        dataIndex: 'system_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => formatQuantity(qty),
      },
      {
        title: t('stockTaking.execute.columns.actualQuantity'),
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
              placeholder={t('stockTaking.execute.columns.actualQuantityPlaceholder')}
              onChange={(value) => handleQuantityChange(record.product_id || '', value as number)}
              style={{ width: '100%' }}
            />
          )
        },
      },
      {
        title: t('stockTaking.execute.columns.differenceQty'),
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
        title: t('stockTaking.execute.columns.differenceAmount'),
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
        title: t('stockTaking.execute.columns.remark'),
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
              placeholder={t('stockTaking.execute.columns.remarkPlaceholder')}
              onChange={(value) => handleRemarkChange(record.product_id || '', value)}
              style={{ width: '100%' }}
            />
          )
        },
      },
      {
        title: t('stockTaking.execute.columns.status'),
        dataIndex: 'counted',
        width: 80,
        align: 'center' as const,
        render: (counted: boolean, record: HandlerStockTakingItemResponse) => {
          const localItem = localItems.get(record.product_id || '')
          const hasValue =
            localItem?.actual_quantity !== null && localItem?.actual_quantity !== undefined
          if (counted || hasValue) {
            return <Tag color="green">{t('stockTaking.execute.itemStatus.counted')}</Tag>
          }
          return <Tag color="grey">{t('stockTaking.execute.itemStatus.notCounted')}</Tag>
        },
      },
      {
        title: t('stockTaking.execute.columns.operation'),
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
              {t('stockTaking.execute.saveItem')}
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
      t,
      formatQuantity,
      formatCurrency,
    ]
  )

  if (loading) {
    return (
      <Container size="lg" className="stock-taking-execute-page">
        <div className="loading-container">
          <Spin size="large" />
          <Text type="tertiary">{t('stockTaking.execute.loading')}</Text>
        </div>
      </Container>
    )
  }

  if (!stockTaking) {
    return (
      <Container size="lg" className="stock-taking-execute-page">
        <Card>
          <Text type="danger">{t('stockTaking.execute.notExist')}</Text>
          <Button onClick={handleBack} style={{ marginTop: 'var(--spacing-4)' }}>
            {t('stockTaking.execute.backToList')}
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
            {t('stockTaking.execute.back')}
          </Button>
          <Title heading={4} style={{ margin: 0 }}>
            {t('stockTaking.execute.title')} - {stockTaking.taking_number}
          </Title>
          <Tag color={STATUS_COLORS[stockTaking.status || ''] || 'grey'}>
            {String(
              t(`stockTaking.list.status.${stockTaking.status}`, {
                defaultValue: stockTaking.status,
              })
            )}
          </Tag>
        </div>
        <div className="header-right">
          {stockTaking.status === 'DRAFT' && (
            <Button icon={<IconPlay />} type="primary" onClick={handleStartCounting}>
              {t('stockTaking.execute.actions.startCounting')}
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
                {t('stockTaking.execute.actions.saveAll')}
              </Button>
              <Button
                icon={<IconSend />}
                type="primary"
                onClick={() => setShowSubmitModal(true)}
                disabled={!canSubmit}
              >
                {t('stockTaking.execute.actions.submitApproval')}
              </Button>
              <Button
                icon={<IconClose />}
                type="danger"
                theme="light"
                onClick={() => setShowCancelModal(true)}
              >
                {t('stockTaking.execute.actions.cancelTaking')}
              </Button>
            </>
          )}
        </div>
      </div>

      {/* Summary Card */}
      <Card className="stock-taking-summary-card">
        <div className="summary-grid">
          <Descriptions row>
            <Descriptions.Item itemKey={t('stockTaking.execute.summary.warehouse')}>
              {stockTaking.warehouse_name}
            </Descriptions.Item>
            <Descriptions.Item itemKey={t('stockTaking.execute.summary.takingDate')}>
              {formatDate(stockTaking.taking_date, 'date')}
            </Descriptions.Item>
            <Descriptions.Item itemKey={t('stockTaking.execute.summary.createdBy')}>
              {stockTaking.created_by_name}
            </Descriptions.Item>
            <Descriptions.Item itemKey={t('stockTaking.execute.summary.remark')}>
              {stockTaking.remark || '-'}
            </Descriptions.Item>
          </Descriptions>

          <div className="progress-section">
            <div className="progress-info">
              <Text strong>{t('stockTaking.execute.summary.progress')}</Text>
              <Text type="tertiary">
                {t('stockTaking.execute.summary.progressCount', {
                  counted: localTotals.countedItems,
                  total: localTotals.totalItems,
                })}
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
            <Text type="tertiary">{t('stockTaking.execute.summary.totalDifference')}</Text>
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
            {t('stockTaking.execute.detail.title')}
          </Title>
          <Button icon={<IconRefresh />} theme="borderless" onClick={fetchStockTaking}>
            {t('stockTaking.execute.detail.refresh')}
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
        title={t('stockTaking.execute.submitModal.title')}
        visible={showSubmitModal}
        onCancel={() => setShowSubmitModal(false)}
        onOk={handleSubmitForApproval}
        okText={t('stockTaking.execute.submitModal.confirm')}
        cancelText={t('stockTaking.execute.submitModal.cancel')}
        confirmLoading={submitting}
      >
        <div className="submit-modal-content">
          <div className="confirm-item">
            <IconTickCircle style={{ color: 'var(--color-success)', fontSize: 48 }} />
          </div>
          <Text>{t('stockTaking.execute.submitModal.content')}</Text>
          <div className="confirm-details">
            <Descriptions row>
              <Descriptions.Item itemKey={t('stockTaking.execute.submitModal.itemCount')}>
                {localTotals.totalItems} {t('common:unit.items', { defaultValue: '项' })}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('stockTaking.execute.submitModal.differenceAmount')}>
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
        title={t('stockTaking.execute.cancelModal.title')}
        visible={showCancelModal}
        onCancel={() => setShowCancelModal(false)}
        onOk={handleCancel}
        okText={t('stockTaking.execute.cancelModal.confirm')}
        okType="danger"
        cancelText={t('stockTaking.execute.cancelModal.cancel')}
        confirmLoading={submitting}
      >
        <div className="cancel-modal-content">
          <Text>{t('stockTaking.execute.cancelModal.content')}</Text>
          <Input
            placeholder={t('stockTaking.execute.cancelModal.reasonPlaceholder')}
            value={cancelReason}
            onChange={setCancelReason}
            style={{ marginTop: 'var(--spacing-4)' }}
          />
        </div>
      </Modal>
    </Container>
  )
}
