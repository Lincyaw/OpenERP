import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
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
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconRefresh } from '@douyinfe/semi-icons'
import { useNavigate, useParams } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { getFinanceApi } from '@/api/finance'
import type {
  PaymentVoucher,
  AccountPayable,
  ReconcileRequest,
  ManualAllocationInput,
  ReconcilePaymentResult,
} from '@/api/finance'
import './PaymentReconcile.css'

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
 * Get status tag color for payment voucher
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
 * Get payable status tag color
 */
function getPayableStatusColor(status: string): TagColor {
  const statusColors: Record<string, TagColor> = {
    PENDING: 'orange',
    PARTIAL: 'blue',
    PAID: 'green',
    REVERSED: 'red',
    CANCELLED: 'grey',
  }
  return statusColors[status] || 'grey'
}

interface AllocationItem {
  payableId: string
  payableNumber: string
  totalAmount: number
  outstandingAmount: number
  dueDate?: string
  selected: boolean
  allocateAmount: number
}

/**
 * Payment Reconciliation Page
 *
 * Features:
 * - Display payment voucher details
 * - List pending payables for the supplier
 * - Support FIFO (automatic) and manual reconciliation
 * - Allow manual selection and amount input for each payable
 * - Preview and confirm reconciliation
 * - Show reconciliation result
 */
export default function PaymentReconcilePage() {
  const { t } = useTranslation('finance')
  const navigate = useNavigate()
  const { id } = useParams<{ id: string }>()
  const financeApi = useMemo(() => getFinanceApi(), [])

  // State
  const [voucher, setVoucher] = useState<PaymentVoucher | null>(null)
  const [payables, setPayables] = useState<AccountPayable[]>([])
  const [loading, setLoading] = useState(true)
  const [reconciling, setReconciling] = useState(false)
  const [reconcileResult, setReconcileResult] = useState<ReconcilePaymentResult | null>(null)

  // Allocation state
  const [allocationItems, setAllocationItems] = useState<AllocationItem[]>([])
  const [reconcileMode, setReconcileMode] = useState<'FIFO' | 'MANUAL'>('FIFO')

  // Load voucher and payables
  const loadData = useCallback(async () => {
    if (!id) return

    setLoading(true)
    try {
      // Load voucher
      const voucherResponse = await financeApi.getFinancePaymentsId(id)
      if (!voucherResponse.success || !voucherResponse.data) {
        Toast.error(t('paymentReconcile.messages.fetchError'))
        navigate('/finance/payables')
        return
      }

      const loadedVoucher = voucherResponse.data
      setVoucher(loadedVoucher)

      // Load payables for the supplier
      const payablesResponse = await financeApi.getFinancePayables({
        supplier_id: loadedVoucher.supplier_id,
        page: 1,
        page_size: 100,
      })

      if (payablesResponse.success && payablesResponse.data) {
        // Filter to pending/partial payables only
        const pendingPayables = payablesResponse.data.filter(
          (p) => (p.status === 'PENDING' || p.status === 'PARTIAL') && p.outstanding_amount > 0
        )
        setPayables(pendingPayables)

        // Initialize allocation items
        const items: AllocationItem[] = pendingPayables.map((p) => ({
          payableId: p.id,
          payableNumber: p.payable_number,
          totalAmount: p.total_amount,
          outstandingAmount: p.outstanding_amount,
          dueDate: p.due_date,
          selected: false,
          allocateAmount: 0,
        }))
        setAllocationItems(items)
      }
    } catch {
      Toast.error(t('paymentReconcile.messages.fetchDataError'))
    } finally {
      setLoading(false)
    }
  }, [id, financeApi, navigate, t])

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

    // Sort by due date then payable number (FIFO)
    const sortedItems = [...allocationItems].sort((a, b) => {
      if (a.dueDate && b.dueDate) {
        return new Date(a.dueDate).getTime() - new Date(b.dueDate).getTime()
      }
      if (a.dueDate) return -1
      if (b.dueDate) return 1
      return a.payableNumber.localeCompare(b.payableNumber)
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
    (payableId: string, selected: boolean) => {
      setAllocationItems((items) =>
        items.map((item) => {
          if (item.payableId === payableId) {
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
    (payableId: string, amount: number | undefined | string) => {
      const numAmount = typeof amount === 'number' ? amount : 0
      setAllocationItems((items) =>
        items.map((item) => {
          if (item.payableId === payableId) {
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
      Toast.error(t('paymentReconcile.messages.onlyConfirmedCanReconcile'))
      return
    }

    if (voucher.unallocated_amount <= 0) {
      Toast.error(t('paymentReconcile.messages.noUnallocatedAmount'))
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
          target_id: item.payableId,
          amount: item.allocateAmount,
        }))

      if (manualAllocations.length === 0) {
        Toast.error(t('paymentReconcile.messages.noPayableSelected'))
        return
      }

      const totalManual = manualAllocations.reduce((sum, a) => sum + a.amount, 0)
      if (totalManual > voucher.unallocated_amount) {
        Toast.error(t('paymentReconcile.messages.exceedUnallocated'))
        return
      }

      request = {
        strategy_type: 'MANUAL',
        manual_allocations: manualAllocations,
      }
    }

    setReconciling(true)
    try {
      const response = await financeApi.postFinancePaymentsIdReconcile(id, request)
      if (response.success && response.data) {
        Toast.success(t('paymentReconcile.messages.allocateSuccess'))
        setReconcileResult(response.data)
        // Reload data to refresh state
        await loadData()
      } else {
        Toast.error(response.error || t('paymentReconcile.messages.allocateError'))
      }
    } catch {
      Toast.error(t('paymentReconcile.messages.requestError'))
    } finally {
      setReconciling(false)
    }
  }

  // Handle back navigation
  const handleBack = () => {
    navigate('/finance/payables')
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
    return payables.length > 0
  }, [voucher, reconcileMode, allocationItems, payables.length])

  // All selected check
  const allSelected = useMemo(() => {
    return allocationItems.length > 0 && allocationItems.every((item) => item.selected)
  }, [allocationItems])

  // Some selected check
  const someSelected = useMemo(() => {
    return allocationItems.some((item) => item.selected) && !allSelected
  }, [allocationItems, allSelected])

  // Table columns for payables
  const columns = useMemo(() => {
    const baseColumns = [
      {
        title: t('paymentReconcile.columns.payableNumber'),
        dataIndex: 'payable_number',
        key: 'payable_number',
        width: 160,
      },
      {
        title: t('paymentReconcile.columns.sourceNumber'),
        dataIndex: 'source_number',
        key: 'source_number',
        width: 140,
      },
      {
        title: t('paymentReconcile.columns.totalAmount'),
        dataIndex: 'total_amount',
        key: 'total_amount',
        width: 120,
        render: (value: number) => formatCurrency(value),
      },
      {
        title: t('paymentReconcile.columns.outstandingAmount'),
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
        title: t('paymentReconcile.columns.dueDate'),
        dataIndex: 'due_date',
        key: 'due_date',
        width: 100,
        render: (value: string) => formatDate(value),
      },
      {
        title: t('paymentReconcile.columns.status'),
        dataIndex: 'status',
        key: 'status',
        width: 100,
        render: (value: string) => (
          <Tag color={getPayableStatusColor(value)}>
            {String(t(`paymentReconcile.payableStatus.${value}` as const)) || value}
          </Tag>
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
          render: (_: unknown, record: AccountPayable) => {
            const item = allocationItems.find((i) => i.payableId === record.id)
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
          title: t('paymentReconcile.columns.allocateAmount'),
          key: 'allocate_amount',
          width: 150,
          render: (_: unknown, record: AccountPayable) => {
            const item = allocationItems.find((i) => i.payableId === record.id)
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
        title: t('paymentReconcile.columns.expectedAllocation'),
        key: 'fifo_allocation',
        width: 120,
        render: (_: unknown, record: AccountPayable) => {
          const allocation = fifoAllocations.find((a) => a.payableId === record.id)
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
    t,
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
      <Container size="lg" className="payment-reconcile-page">
        <div className="loading-container">
          <Spin size="large" />
        </div>
      </Container>
    )
  }

  if (!voucher) {
    return (
      <Container size="lg" className="payment-reconcile-page">
        <Empty description={t('paymentReconcile.voucherNotFound')} />
      </Container>
    )
  }

  // Show result if reconciliation was successful
  if (reconcileResult) {
    return (
      <Container size="lg" className="payment-reconcile-page">
        <Card className="reconcile-result-card">
          <div className="result-header">
            <Title heading={4}>{t('paymentReconcile.result.title')}</Title>
            <Button icon={<IconArrowLeft />} onClick={handleBack}>
              {t('paymentReconcile.result.backToList')}
            </Button>
          </div>

          <Banner
            type={reconcileResult.fully_reconciled ? 'success' : 'info'}
            description={
              reconcileResult.fully_reconciled
                ? t('paymentReconcile.result.fullyReconciled')
                : t('paymentReconcile.result.partiallyReconciled', {
                    amount: formatCurrency(reconcileResult.remaining_unallocated),
                  })
            }
          />

          <div className="result-summary">
            <Descriptions
              data={[
                {
                  key: t('paymentReconcile.result.voucherNumber'),
                  value: reconcileResult.voucher.voucher_number,
                },
                {
                  key: t('paymentReconcile.result.supplierName'),
                  value: reconcileResult.voucher.supplier_name,
                },
                {
                  key: t('paymentReconcile.result.paymentAmount'),
                  value: formatCurrency(reconcileResult.voucher.amount),
                },
                {
                  key: t('paymentReconcile.result.thisReconciled'),
                  value: formatCurrency(reconcileResult.total_reconciled),
                },
                {
                  key: t('paymentReconcile.result.remainingUnallocated'),
                  value: formatCurrency(reconcileResult.remaining_unallocated),
                },
              ]}
            />
          </div>

          {reconcileResult.updated_payables.length > 0 && (
            <>
              <Divider />
              <Title heading={5}>{t('paymentReconcile.result.reconciledPayables')}</Title>
              <Table
                dataSource={reconcileResult.updated_payables}
                columns={[
                  {
                    title: t('paymentReconcile.columns.payableNumber'),
                    dataIndex: 'payable_number',
                    key: 'payable_number',
                  },
                  {
                    title: t('paymentReconcile.columns.totalAmount'),
                    dataIndex: 'total_amount',
                    key: 'total_amount',
                    render: (value: number) => formatCurrency(value),
                  },
                  {
                    title: t('paymentReconcile.result.paidAmount'),
                    dataIndex: 'paid_amount',
                    key: 'paid_amount',
                    render: (value: number) => formatCurrency(value),
                  },
                  {
                    title: t('paymentReconcile.columns.outstandingAmount'),
                    dataIndex: 'outstanding_amount',
                    key: 'outstanding_amount',
                    render: (value: number) => formatCurrency(value),
                  },
                  {
                    title: t('paymentReconcile.columns.status'),
                    dataIndex: 'status',
                    key: 'status',
                    render: (value: string) => (
                      <Tag color={getPayableStatusColor(value)}>
                        {String(t(`paymentReconcile.payableStatus.${value}` as const)) || value}
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
    <Container size="lg" className="payment-reconcile-page">
      {/* Header */}
      <div className="page-header">
        <div className="header-left">
          <Button icon={<IconArrowLeft />} theme="borderless" onClick={handleBack}>
            {t('paymentReconcile.back')}
          </Button>
          <Title heading={4} style={{ margin: 0 }}>
            {t('paymentReconcile.title')}
          </Title>
        </div>
        <Button icon={<IconRefresh />} onClick={loadData} disabled={loading}>
          {t('paymentReconcile.refresh')}
        </Button>
      </div>

      {/* Voucher Details */}
      <Card className="voucher-details-card">
        <Title heading={5}>{t('paymentReconcile.voucherInfo.title')}</Title>
        <Descriptions
          row
          data={[
            { key: t('paymentReconcile.voucherInfo.voucherNumber'), value: voucher.voucher_number },
            { key: t('paymentReconcile.voucherInfo.supplierName'), value: voucher.supplier_name },
            {
              key: t('paymentReconcile.voucherInfo.status'),
              value: (
                <Tag color={getVoucherStatusColor(voucher.status)}>
                  {String(t(`paymentReconcile.voucherStatus.${voucher.status}` as const)) ||
                    voucher.status}
                </Tag>
              ),
            },
            {
              key: t('paymentReconcile.voucherInfo.paymentMethod'),
              value:
                String(t(`paymentMethod.${voucher.payment_method}` as const)) ||
                voucher.payment_method,
            },
            {
              key: t('paymentReconcile.voucherInfo.paymentDate'),
              value: formatDate(voucher.payment_date),
            },
            {
              key: t('paymentReconcile.voucherInfo.totalAmount'),
              value: formatCurrency(voucher.amount),
            },
            {
              key: t('paymentReconcile.voucherInfo.allocatedAmount'),
              value: formatCurrency(voucher.allocated_amount),
            },
            {
              key: t('paymentReconcile.voucherInfo.unallocatedAmount'),
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
                ? t('paymentReconcile.statusWarning.draft')
                : voucher.status === 'ALLOCATED'
                  ? t('paymentReconcile.statusWarning.allocated')
                  : t('paymentReconcile.statusWarning.cancelled')
            }
          />
        )}

        {voucher.status === 'CONFIRMED' && voucher.unallocated_amount <= 0 && (
          <Banner
            type="success"
            className="status-warning"
            description={t('paymentReconcile.statusWarning.fullyAllocated')}
          />
        )}
      </Card>

      {/* Reconciliation Section */}
      {voucher.status === 'CONFIRMED' && voucher.unallocated_amount > 0 && (
        <Card className="reconcile-section-card">
          <div className="reconcile-header">
            <Title heading={5}>{t('paymentReconcile.payables.title')}</Title>
            <div className="mode-switch">
              <Text type="secondary">{t('paymentReconcile.reconcileMode.label')}</Text>
              <Button
                theme={reconcileMode === 'FIFO' ? 'solid' : 'borderless'}
                onClick={() => setReconcileMode('FIFO')}
              >
                {t('paymentReconcile.reconcileMode.fifo')}
              </Button>
              <Button
                theme={reconcileMode === 'MANUAL' ? 'solid' : 'borderless'}
                onClick={() => setReconcileMode('MANUAL')}
              >
                {t('paymentReconcile.reconcileMode.manual')}
              </Button>
            </div>
          </div>

          {reconcileMode === 'FIFO' && (
            <Banner
              type="info"
              className="mode-description"
              description={t('paymentReconcile.reconcileMode.fifoDescription')}
            />
          )}

          {reconcileMode === 'MANUAL' && (
            <Banner
              type="info"
              className="mode-description"
              description={t('paymentReconcile.reconcileMode.manualDescription')}
            />
          )}

          {payables.length === 0 ? (
            <Empty description={t('paymentReconcile.payables.empty')} />
          ) : (
            <>
              <Table
                dataSource={payables}
                columns={columns}
                rowKey="id"
                pagination={false}
                className="payables-table"
              />

              {/* Reconciliation Summary */}
              <div className="reconcile-summary">
                <div className="summary-item">
                  <Text type="secondary">{t('paymentReconcile.summary.availableAmount')}</Text>
                  <Text strong>{formatCurrency(voucher.unallocated_amount)}</Text>
                </div>
                <div className="summary-item">
                  <Text type="secondary">
                    {reconcileMode === 'FIFO'
                      ? t('paymentReconcile.summary.expectedAllocation')
                      : t('paymentReconcile.summary.selectedAllocation')}
                  </Text>
                  <Text strong type="success">
                    {formatCurrency(totalAllocation)}
                  </Text>
                </div>
                <div className="summary-item">
                  <Text type="secondary">{t('paymentReconcile.summary.remainingAfter')}</Text>
                  <Text strong type={totalAllocation > 0 ? 'warning' : 'tertiary'}>
                    {formatCurrency(voucher.unallocated_amount - totalAllocation)}
                  </Text>
                </div>
              </div>

              {/* Actions */}
              <div className="reconcile-actions">
                <Button onClick={handleBack}>{t('paymentReconcile.actions.cancel')}</Button>
                <Button
                  type="primary"
                  onClick={handleReconcile}
                  loading={reconciling}
                  disabled={!canReconcile}
                >
                  {t('paymentReconcile.actions.confirm')}
                </Button>
              </div>
            </>
          )}
        </Card>
      )}

      {/* Existing Allocations */}
      {voucher.allocations && voucher.allocations.length > 0 && (
        <Card className="existing-allocations-card">
          <Title heading={5}>{t('paymentReconcile.existingAllocations.title')}</Title>
          <Table
            dataSource={voucher.allocations}
            columns={[
              {
                title: t('paymentReconcile.existingAllocations.payableNumber'),
                dataIndex: 'payable_number',
                key: 'payable_number',
              },
              {
                title: t('paymentReconcile.existingAllocations.amount'),
                dataIndex: 'amount',
                key: 'amount',
                render: (value: number) => formatCurrency(value),
              },
              {
                title: t('paymentReconcile.existingAllocations.allocatedAt'),
                dataIndex: 'allocated_at',
                key: 'allocated_at',
                render: (value: string) => (value ? new Date(value).toLocaleString() : '-'),
              },
              {
                title: t('paymentReconcile.existingAllocations.remark'),
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
