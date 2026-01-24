import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Toast,
  Spin,
  Button,
  Table,
  Banner,
  Checkbox,
  InputNumber,
  Tag,
  Descriptions,
  Empty,
  Divider,
} from '@douyinfe/semi-ui'
import { IconArrowLeft, IconRefresh } from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { getFinanceApi } from '@/api/finance'
import type {
  ReceiptVoucher,
  AccountReceivable,
  ReconcileRequest,
  ManualAllocationInput,
  ReconcileReceiptResult,
} from '@/api/finance'
import './ReceiptReconcile.css'

const { Title, Text } = Typography

/**
 * Format currency for display
 */
function formatCurrency(amount?: number): string {
  if (amount === undefined || amount === null) return '-'
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: 'CNY',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
}

/**
 * Format date for display
 */
function formatDate(dateString?: string): string {
  if (!dateString) return '-'
  return new Date(dateString).toLocaleDateString('zh-CN')
}

// Tag color type for Semi UI
type TagColor =
  | 'amber'
  | 'blue'
  | 'cyan'
  | 'green'
  | 'grey'
  | 'indigo'
  | 'light-blue'
  | 'light-green'
  | 'lime'
  | 'orange'
  | 'pink'
  | 'purple'
  | 'red'
  | 'teal'
  | 'violet'
  | 'yellow'
  | 'white'

/**
 * Get status tag color
 */
function getVoucherStatusColor(status: string): TagColor {
  const statusColors: Record<string, TagColor> = {
    DRAFT: 'grey',
    CONFIRMED: 'blue',
    ALLOCATED: 'green',
    CANCELLED: 'red',
  }
  return statusColors[status] || 'grey'
}

/**
 * Get status label
 */
function getVoucherStatusLabel(status: string): string {
  const statusLabels: Record<string, string> = {
    DRAFT: '草稿',
    CONFIRMED: '已确认',
    ALLOCATED: '已核销',
    CANCELLED: '已取消',
  }
  return statusLabels[status] || status
}

/**
 * Get receivable status tag color
 */
function getReceivableStatusColor(status: string): TagColor {
  const statusColors: Record<string, TagColor> = {
    PENDING: 'orange',
    PARTIAL: 'blue',
    PAID: 'green',
    REVERSED: 'red',
    CANCELLED: 'grey',
  }
  return statusColors[status] || 'grey'
}

/**
 * Get receivable status label
 */
function getReceivableStatusLabel(status: string): string {
  const statusLabels: Record<string, string> = {
    PENDING: '待收款',
    PARTIAL: '部分收款',
    PAID: '已收款',
    REVERSED: '已红冲',
    CANCELLED: '已取消',
  }
  return statusLabels[status] || status
}

/**
 * Get payment method label
 */
function getPaymentMethodLabel(method: string): string {
  const methodLabels: Record<string, string> = {
    CASH: '现金',
    BANK_TRANSFER: '银行转账',
    WECHAT: '微信支付',
    ALIPAY: '支付宝',
    CHECK: '支票',
    BALANCE: '余额抵扣',
    OTHER: '其他',
  }
  return methodLabels[method] || method
}

interface AllocationItem {
  receivableId: string
  receivableNumber: string
  totalAmount: number
  outstandingAmount: number
  dueDate?: string
  selected: boolean
  allocateAmount: number
}

/**
 * Receipt Reconciliation Page
 *
 * Features:
 * - Display receipt voucher details
 * - List pending receivables for the customer
 * - Support FIFO (automatic) and manual reconciliation
 * - Allow manual selection and amount input for each receivable
 * - Preview and confirm reconciliation
 * - Show reconciliation result
 */
export default function ReceiptReconcilePage() {
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const financeApi = useMemo(() => getFinanceApi(), [])

  // State
  const [voucher, setVoucher] = useState<ReceiptVoucher | null>(null)
  const [receivables, setReceivables] = useState<AccountReceivable[]>([])
  const [loading, setLoading] = useState(true)
  const [reconciling, setReconciling] = useState(false)
  const [reconcileResult, setReconcileResult] = useState<ReconcileReceiptResult | null>(null)

  // Allocation state
  const [allocationItems, setAllocationItems] = useState<AllocationItem[]>([])
  const [reconcileMode, setReconcileMode] = useState<'FIFO' | 'MANUAL'>('FIFO')

  // Load voucher and receivables
  const loadData = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      // Load voucher
      const voucherResponse = await financeApi.getFinanceReceiptsId(id)
      if (!voucherResponse.success || !voucherResponse.data) {
        Toast.error('加载收款单失败')
        navigate('/finance/receivables')
        return
      }

      const loadedVoucher = voucherResponse.data
      setVoucher(loadedVoucher)

      // Load receivables for the customer
      const receivablesResponse = await financeApi.getFinanceReceivables({
        customer_id: loadedVoucher.customer_id,
        page: 1,
        page_size: 100,
      })

      if (receivablesResponse.success && receivablesResponse.data) {
        // Filter to pending/partial receivables only
        const pendingReceivables = receivablesResponse.data.filter(
          (r) => (r.status === 'PENDING' || r.status === 'PARTIAL') && r.outstanding_amount > 0
        )
        setReceivables(pendingReceivables)

        // Initialize allocation items
        const items: AllocationItem[] = pendingReceivables.map((r) => ({
          receivableId: r.id,
          receivableNumber: r.receivable_number,
          totalAmount: r.total_amount,
          outstandingAmount: r.outstanding_amount,
          dueDate: r.due_date,
          selected: false,
          allocateAmount: 0,
        }))
        setAllocationItems(items)
      }
    } catch {
      Toast.error('加载数据失败')
    } finally {
      setLoading(false)
    }
  }, [id, financeApi, navigate])

  useEffect(() => {
    loadData()
  }, [loadData])

  // Calculate total allocation
  const totalAllocation = useMemo(() => {
    if (reconcileMode === 'FIFO') {
      // In FIFO mode, calculate automatic allocation
      if (!voucher) return 0
      let remaining = voucher.unallocated_amount
      let total = 0

      // Sort by due date then created date (FIFO)
      const sortedItems = [...allocationItems].sort((a, b) => {
        if (a.dueDate && b.dueDate) {
          return new Date(a.dueDate).getTime() - new Date(b.dueDate).getTime()
        }
        if (a.dueDate) return -1
        if (b.dueDate) return 1
        return 0
      })

      for (const item of sortedItems) {
        if (remaining <= 0) break
        const allocAmount = Math.min(remaining, item.outstandingAmount)
        total += allocAmount
        remaining -= allocAmount
      }
      return total
    }

    // In manual mode, sum selected items
    return allocationItems
      .filter((item) => item.selected && item.allocateAmount > 0)
      .reduce((sum, item) => sum + item.allocateAmount, 0)
  }, [reconcileMode, allocationItems, voucher])

  // Handle FIFO auto-allocation preview
  const getFIFOAllocations = useCallback((): AllocationItem[] => {
    if (!voucher) return []
    let remaining = voucher.unallocated_amount

    // Sort by due date then receivable number (FIFO)
    const sortedItems = [...allocationItems].sort((a, b) => {
      if (a.dueDate && b.dueDate) {
        return new Date(a.dueDate).getTime() - new Date(b.dueDate).getTime()
      }
      if (a.dueDate) return -1
      if (b.dueDate) return 1
      return a.receivableNumber.localeCompare(b.receivableNumber)
    })

    return sortedItems.map((item) => {
      if (remaining <= 0) {
        return { ...item, selected: false, allocateAmount: 0 }
      }
      const allocAmount = Math.min(remaining, item.outstandingAmount)
      remaining -= allocAmount
      return { ...item, selected: true, allocateAmount: allocAmount }
    })
  }, [voucher, allocationItems])

  // Handle selection toggle
  const handleSelectItem = useCallback(
    (receivableId: string, selected: boolean) => {
      setAllocationItems((items) =>
        items.map((item) => {
          if (item.receivableId === receivableId) {
            const newSelected = selected
            return {
              ...item,
              selected: newSelected,
              // Set default allocation amount when selecting
              allocateAmount: newSelected
                ? Math.min(item.outstandingAmount, voucher?.unallocated_amount || 0)
                : 0,
            }
          }
          return item
        })
      )
    },
    [voucher]
  )

  // Handle allocation amount change
  const handleAmountChange = useCallback(
    (receivableId: string, amount: number | undefined | string) => {
      const numAmount = typeof amount === 'number' ? amount : 0
      setAllocationItems((items) =>
        items.map((item) => {
          if (item.receivableId === receivableId) {
            return {
              ...item,
              allocateAmount: Math.min(numAmount, item.outstandingAmount),
            }
          }
          return item
        })
      )
    },
    []
  )

  // Handle select all
  const handleSelectAll = useCallback(
    (checked: boolean) => {
      if (!voucher) return

      if (checked) {
        // Select all and allocate proportionally
        let remaining = voucher.unallocated_amount
        setAllocationItems((items) =>
          items.map((item) => {
            if (remaining <= 0) {
              return { ...item, selected: false, allocateAmount: 0 }
            }
            const allocAmount = Math.min(remaining, item.outstandingAmount)
            remaining -= allocAmount
            return { ...item, selected: true, allocateAmount: allocAmount }
          })
        )
      } else {
        // Deselect all
        setAllocationItems((items) =>
          items.map((item) => ({ ...item, selected: false, allocateAmount: 0 }))
        )
      }
    },
    [voucher]
  )

  // Handle reconciliation
  const handleReconcile = async () => {
    if (!voucher || !id) return

    // Validate
    if (voucher.status !== 'CONFIRMED') {
      Toast.error('只有已确认的收款单才能进行核销')
      return
    }

    if (voucher.unallocated_amount <= 0) {
      Toast.error('该收款单没有未核销金额')
      return
    }

    let request: ReconcileRequest

    if (reconcileMode === 'FIFO') {
      request = {
        strategy_type: 'FIFO',
      }
    } else {
      // Manual mode
      const manualAllocations: ManualAllocationInput[] = allocationItems
        .filter((item) => item.selected && item.allocateAmount > 0)
        .map((item) => ({
          target_id: item.receivableId,
          amount: item.allocateAmount,
        }))

      if (manualAllocations.length === 0) {
        Toast.error('请选择至少一个应收账款进行核销')
        return
      }

      const totalManual = manualAllocations.reduce((sum, a) => sum + a.amount, 0)
      if (totalManual > voucher.unallocated_amount) {
        Toast.error('核销总额不能超过未核销金额')
        return
      }

      request = {
        strategy_type: 'MANUAL',
        manual_allocations: manualAllocations,
      }
    }

    setReconciling(true)
    try {
      const response = await financeApi.postFinanceReceiptsIdReconcile(id, request)
      if (response.success && response.data) {
        Toast.success('核销成功')
        setReconcileResult(response.data)
        // Reload data to refresh state
        await loadData()
      } else {
        Toast.error(response.error || '核销失败')
      }
    } catch {
      Toast.error('核销请求失败')
    } finally {
      setReconciling(false)
    }
  }

  // Handle back navigation
  const handleBack = () => {
    navigate('/finance/receivables')
  }

  // Determine if can reconcile
  const canReconcile = useMemo(() => {
    if (!voucher) return false
    if (voucher.status !== 'CONFIRMED') return false
    if (voucher.unallocated_amount <= 0) return false
    if (reconcileMode === 'MANUAL') {
      const hasSelection = allocationItems.some((item) => item.selected && item.allocateAmount > 0)
      return hasSelection
    }
    return receivables.length > 0
  }, [voucher, reconcileMode, allocationItems, receivables.length])

  // All selected check
  const allSelected = useMemo(() => {
    return allocationItems.length > 0 && allocationItems.every((item) => item.selected)
  }, [allocationItems])

  // Some selected check
  const someSelected = useMemo(() => {
    return allocationItems.some((item) => item.selected) && !allSelected
  }, [allocationItems, allSelected])

  // Table columns for receivables
  const columns = useMemo(() => {
    const baseColumns = [
      {
        title: '应收账款编号',
        dataIndex: 'receivable_number',
        key: 'receivable_number',
        width: 160,
      },
      {
        title: '来源单据',
        dataIndex: 'source_number',
        key: 'source_number',
        width: 140,
      },
      {
        title: '总金额',
        dataIndex: 'total_amount',
        key: 'total_amount',
        width: 120,
        render: (value: number) => formatCurrency(value),
      },
      {
        title: '待收金额',
        dataIndex: 'outstanding_amount',
        key: 'outstanding_amount',
        width: 120,
        render: (value: number) => (
          <Text strong className="outstanding-amount">
            {formatCurrency(value)}
          </Text>
        ),
      },
      {
        title: '到期日',
        dataIndex: 'due_date',
        key: 'due_date',
        width: 100,
        render: (value: string) => formatDate(value),
      },
      {
        title: '状态',
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (value: string) => (
          <Tag color={getReceivableStatusColor(value)}>{getReceivableStatusLabel(value)}</Tag>
        ),
      },
    ]

    if (reconcileMode === 'MANUAL') {
      return [
        {
          title: (
            <Checkbox
              checked={allSelected}
              indeterminate={someSelected}
              onChange={(e) => handleSelectAll(e.target.checked ?? false)}
            />
          ),
          key: 'select',
          width: 50,
          render: (_: unknown, record: AccountReceivable) => {
            const item = allocationItems.find((i) => i.receivableId === record.id)
            return (
              <Checkbox
                checked={item?.selected || false}
                onChange={(e) => handleSelectItem(record.id, e.target.checked ?? false)}
              />
            )
          },
        },
        ...baseColumns,
        {
          title: '核销金额',
          key: 'allocate_amount',
          width: 150,
          render: (_: unknown, record: AccountReceivable) => {
            const item = allocationItems.find((i) => i.receivableId === record.id)
            if (!item?.selected) return '-'
            return (
              <InputNumber
                value={item.allocateAmount}
                min={0.01}
                max={item.outstandingAmount}
                precision={2}
                prefix="¥"
                style={{ width: 130 }}
                onChange={(value) => handleAmountChange(record.id, value)}
              />
            )
          },
        },
      ]
    }

    // FIFO mode - show preview allocations
    const fifoAllocations = getFIFOAllocations()
    return [
      ...baseColumns,
      {
        title: '预计核销',
        key: 'fifo_allocation',
        width: 120,
        render: (_: unknown, record: AccountReceivable) => {
          const allocation = fifoAllocations.find((a) => a.receivableId === record.id)
          if (!allocation || allocation.allocateAmount <= 0) {
            return <Text type="tertiary">-</Text>
          }
          return (
            <Text strong type="success">
              {formatCurrency(allocation.allocateAmount)}
            </Text>
          )
        },
      },
    ]
  }, [
    reconcileMode,
    allSelected,
    someSelected,
    handleSelectAll,
    allocationItems,
    handleSelectItem,
    handleAmountChange,
    getFIFOAllocations,
  ])

  if (loading) {
    return (
      <Container size="lg" className="receipt-reconcile-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
      </Container>
    )
  }

  if (!voucher) {
    return (
      <Container size="lg" className="receipt-reconcile-page">
        <Empty description="未找到收款单" />
      </Container>
    )
  }

  // Show result if reconciliation was successful
  if (reconcileResult) {
    return (
      <Container size="lg" className="receipt-reconcile-page">
        <Card className="reconcile-result-card">
          <div className="result-header">
            <Title heading={4}>核销完成</Title>
            <Button icon={<IconArrowLeft />} onClick={handleBack}>
              返回列表
            </Button>
          </div>

          <Banner
            type={reconcileResult.fully_reconciled ? 'success' : 'info'}
            description={
              reconcileResult.fully_reconciled
                ? '收款单已完全核销'
                : `部分核销完成，剩余 ${formatCurrency(reconcileResult.remaining_unallocated)} 未核销`
            }
          />

          <div className="result-summary">
            <Descriptions
              data={[
                {
                  key: '收款单编号',
                  value: reconcileResult.voucher.voucher_number,
                },
                {
                  key: '客户名称',
                  value: reconcileResult.voucher.customer_name,
                },
                {
                  key: '收款金额',
                  value: formatCurrency(reconcileResult.voucher.amount),
                },
                {
                  key: '本次核销',
                  value: formatCurrency(reconcileResult.total_reconciled),
                },
                {
                  key: '剩余未核销',
                  value: formatCurrency(reconcileResult.remaining_unallocated),
                },
              ]}
            />
          </div>

          {reconcileResult.updated_receivables.length > 0 && (
            <>
              <Divider />
              <Title heading={5}>已核销应收账款</Title>
              <Table
                dataSource={reconcileResult.updated_receivables}
                columns={[
                  {
                    title: '应收账款编号',
                    dataIndex: 'receivable_number',
                    key: 'receivable_number',
                  },
                  {
                    title: '总金额',
                    dataIndex: 'total_amount',
                    key: 'total_amount',
                    render: (value: number) => formatCurrency(value),
                  },
                  {
                    title: '已收金额',
                    dataIndex: 'paid_amount',
                    key: 'paid_amount',
                    render: (value: number) => formatCurrency(value),
                  },
                  {
                    title: '待收金额',
                    dataIndex: 'outstanding_amount',
                    key: 'outstanding_amount',
                    render: (value: number) => formatCurrency(value),
                  },
                  {
                    title: '状态',
                    dataIndex: 'status',
                    key: 'status',
                    render: (value: string) => (
                      <Tag color={getReceivableStatusColor(value)}>
                        {getReceivableStatusLabel(value)}
                      </Tag>
                    ),
                  },
                ]}
                rowKey="id"
                pagination={false}
              />
            </>
          )}
        </Card>
      </Container>
    )
  }

  return (
    <Container size="lg" className="receipt-reconcile-page">
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button icon={<IconArrowLeft />} theme="borderless" onClick={handleBack}>
            返回
          </Button>
          <Title heading={4} style={{ margin: 0 }}>
            收款核销
          </Title>
        </div>
        <Button icon={<IconRefresh />} onClick={loadData} disabled={loading}>
          刷新
        </Button>
      </div>

      {/* Voucher Details */}
      <Card className="voucher-details-card">
        <Title heading={5}>收款单信息</Title>
        <Descriptions
          row
          data={[
            { key: '收款单编号', value: voucher.voucher_number },
            { key: '客户名称', value: voucher.customer_name },
            {
              key: '状态',
              value: (
                <Tag color={getVoucherStatusColor(voucher.status)}>
                  {getVoucherStatusLabel(voucher.status)}
                </Tag>
              ),
            },
            { key: '收款方式', value: getPaymentMethodLabel(voucher.payment_method) },
            { key: '收款日期', value: formatDate(voucher.receipt_date) },
            { key: '收款金额', value: formatCurrency(voucher.amount) },
            { key: '已核销金额', value: formatCurrency(voucher.allocated_amount) },
            {
              key: '未核销金额',
              value: (
                <Text strong type="warning">
                  {formatCurrency(voucher.unallocated_amount)}
                </Text>
              ),
            },
          ]}
        />

        {voucher.status !== 'CONFIRMED' && (
          <Banner
            type="warning"
            className="status-warning"
            description={
              voucher.status === 'DRAFT'
                ? '该收款单尚未确认，请先确认后再进行核销'
                : voucher.status === 'ALLOCATED'
                  ? '该收款单已完全核销'
                  : '该收款单已取消，无法进行核销'
            }
          />
        )}

        {voucher.status === 'CONFIRMED' && voucher.unallocated_amount <= 0 && (
          <Banner type="success" className="status-warning" description="该收款单已完全核销" />
        )}
      </Card>

      {/* Reconciliation Section */}
      {voucher.status === 'CONFIRMED' && voucher.unallocated_amount > 0 && (
        <Card className="reconcile-section-card">
          <div className="reconcile-header">
            <Title heading={5}>待核销应收账款</Title>
            <div className="mode-switch">
              <Text type="secondary">核销方式：</Text>
              <Button
                theme={reconcileMode === 'FIFO' ? 'solid' : 'borderless'}
                onClick={() => setReconcileMode('FIFO')}
              >
                自动核销 (FIFO)
              </Button>
              <Button
                theme={reconcileMode === 'MANUAL' ? 'solid' : 'borderless'}
                onClick={() => setReconcileMode('MANUAL')}
              >
                手动核销
              </Button>
            </div>
          </div>

          {reconcileMode === 'FIFO' && (
            <Banner
              type="info"
              className="mode-description"
              description="自动核销将按照应收账款到期日期从早到晚的顺序（FIFO）进行核销，直到收款金额用完或所有应收账款核销完毕。"
            />
          )}

          {reconcileMode === 'MANUAL' && (
            <Banner
              type="info"
              className="mode-description"
              description="手动核销允许您选择要核销的应收账款，并指定每笔应收账款的核销金额。"
            />
          )}

          {receivables.length === 0 ? (
            <Empty description="该客户暂无待核销的应收账款" />
          ) : (
            <>
              <Table
                dataSource={receivables}
                columns={columns}
                rowKey="id"
                pagination={false}
                className="receivables-table"
              />

              {/* Reconciliation Summary */}
              <div className="reconcile-summary">
                <div className="summary-item">
                  <Text type="secondary">可核销金额：</Text>
                  <Text strong>{formatCurrency(voucher.unallocated_amount)}</Text>
                </div>
                <div className="summary-item">
                  <Text type="secondary">
                    {reconcileMode === 'FIFO' ? '预计核销金额：' : '已选核销金额：'}
                  </Text>
                  <Text strong type="success">
                    {formatCurrency(totalAllocation)}
                  </Text>
                </div>
                <div className="summary-item">
                  <Text type="secondary">核销后剩余：</Text>
                  <Text strong type={totalAllocation > 0 ? 'warning' : 'tertiary'}>
                    {formatCurrency(voucher.unallocated_amount - totalAllocation)}
                  </Text>
                </div>
              </div>

              {/* Actions */}
              <div className="reconcile-actions">
                <Button onClick={handleBack}>取消</Button>
                <Button
                  type="primary"
                  onClick={handleReconcile}
                  loading={reconciling}
                  disabled={!canReconcile}
                >
                  确认核销
                </Button>
              </div>
            </>
          )}
        </Card>
      )}

      {/* Existing Allocations */}
      {voucher.allocations && voucher.allocations.length > 0 && (
        <Card className="existing-allocations-card">
          <Title heading={5}>已核销记录</Title>
          <Table
            dataSource={voucher.allocations}
            columns={[
              {
                title: '应收账款编号',
                dataIndex: 'receivable_number',
                key: 'receivable_number',
              },
              {
                title: '核销金额',
                dataIndex: 'amount',
                key: 'amount',
                render: (value: number) => formatCurrency(value),
              },
              {
                title: '核销时间',
                dataIndex: 'allocated_at',
                key: 'allocated_at',
                render: (value: string) => (value ? new Date(value).toLocaleString('zh-CN') : '-'),
              },
              {
                title: '备注',
                dataIndex: 'remark',
                key: 'remark',
                render: (value: string) => value || '-',
              },
            ]}
            rowKey="id"
            pagination={false}
          />
        </Card>
      )}
    </Container>
  )
}
