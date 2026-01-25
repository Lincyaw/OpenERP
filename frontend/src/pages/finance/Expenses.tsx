import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  Card,
  Typography,
  Tag,
  Toast,
  Select,
  Space,
  Spin,
  DatePicker,
  Descriptions,
  Modal,
  TextArea,
  Button,
} from '@douyinfe/semi-ui'
import { IconRefresh, IconPlus } from '@douyinfe/semi-icons'
import { useNavigate } from 'react-router-dom'
import {
  DataTable,
  TableToolbar,
  useTableState,
  type DataTableColumn,
  type TableAction,
} from '@/components/common'
import { Container } from '@/components/common/layout'
import { getFinanceApi } from '@/api/finance'
import type {
  ExpenseRecord,
  ExpenseCategory,
  ExpenseStatus,
  ExpensePaymentStatus,
  ExpenseSummary,
  GetExpenseRecordsParams,
} from '@/api/finance'
import type { PaginationMeta } from '@/types/api'
import './Expenses.css'

const { Title, Text } = Typography

// Expense type with index signature for DataTable compatibility
type ExpenseRow = ExpenseRecord & Record<string, unknown>

// Status tag color mapping
const STATUS_TAG_COLORS: Record<ExpenseStatus, 'grey' | 'orange' | 'green' | 'red'> = {
  DRAFT: 'grey',
  PENDING: 'orange',
  APPROVED: 'green',
  REJECTED: 'red',
  CANCELLED: 'grey',
}

// Payment status tag colors
const PAYMENT_STATUS_TAG_COLORS: Record<ExpensePaymentStatus, 'orange' | 'green'> = {
  UNPAID: 'orange',
  PAID: 'green',
}

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
function formatDate(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
}

/**
 * Expenses list page
 *
 * Features:
 * - Expense record listing with pagination
 * - Search by expense number, description
 * - Filter by category, status, payment status, date range
 * - Summary cards showing key metrics
 * - CRUD operations with approval workflow
 */
export default function ExpensesPage() {
  const { t } = useTranslation('finance')
  const navigate = useNavigate()
  const api = useMemo(() => getFinanceApi(), [])

  // Filter options with i18n
  const categoryOptions = useMemo(
    () => [
      { label: t('expenses.filter.allCategory'), value: '' },
      { label: t('expenses.category.RENT'), value: 'RENT' },
      { label: t('expenses.category.UTILITIES'), value: 'UTILITIES' },
      { label: t('expenses.category.SALARY'), value: 'SALARY' },
      { label: t('expenses.category.OFFICE'), value: 'OFFICE' },
      { label: t('expenses.category.TRAVEL'), value: 'TRAVEL' },
      { label: t('expenses.category.MARKETING'), value: 'MARKETING' },
      { label: t('expenses.category.EQUIPMENT'), value: 'EQUIPMENT' },
      { label: t('expenses.category.MAINTENANCE'), value: 'MAINTENANCE' },
      { label: t('expenses.category.INSURANCE'), value: 'INSURANCE' },
      { label: t('expenses.category.TAX'), value: 'TAX' },
      { label: t('expenses.category.OTHER'), value: 'OTHER' },
    ],
    [t]
  )

  const statusOptions = useMemo(
    () => [
      { label: t('expenses.filter.allStatus'), value: '' },
      { label: t('expenses.status.DRAFT'), value: 'DRAFT' },
      { label: t('expenses.status.PENDING'), value: 'PENDING' },
      { label: t('expenses.status.APPROVED'), value: 'APPROVED' },
      { label: t('expenses.status.REJECTED'), value: 'REJECTED' },
      { label: t('expenses.status.CANCELLED'), value: 'CANCELLED' },
    ],
    [t]
  )

  const paymentStatusOptions = useMemo(
    () => [
      { label: t('expenses.filter.allPaymentStatus'), value: '' },
      { label: t('expenses.paymentStatus.UNPAID'), value: 'UNPAID' },
      { label: t('expenses.paymentStatus.PAID'), value: 'PAID' },
    ],
    [t]
  )

  // State for data
  const [expenseList, setExpenseList] = useState<ExpenseRow[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [summary, setSummary] = useState<ExpenseSummary | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(false)

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<string>('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [paymentStatusFilter, setPaymentStatusFilter] = useState<string>('')
  const [dateRange, setDateRange] = useState<[Date, Date] | null>(null)

  // Modal state for actions
  const [rejectModalVisible, setRejectModalVisible] = useState(false)
  const [cancelModalVisible, setCancelModalVisible] = useState(false)
  const [actionReason, setActionReason] = useState('')
  const [selectedExpense, setSelectedExpense] = useState<ExpenseRow | null>(null)
  const [actionLoading, setActionLoading] = useState(false)

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Fetch expenses
  const fetchExpenses = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetExpenseRecordsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        category: (categoryFilter || undefined) as ExpenseCategory | undefined,
        status: (statusFilter || undefined) as ExpenseStatus | undefined,
        payment_status: (paymentStatusFilter || undefined) as ExpensePaymentStatus | undefined,
        from_date: dateRange?.[0]?.toISOString().split('T')[0],
        to_date: dateRange?.[1]?.toISOString().split('T')[0],
      }

      const response = await api.getFinanceExpenses(params)

      if (response.success && response.data) {
        setExpenseList(response.data as ExpenseRow[])
        if (response.meta) {
          setPaginationMeta({
            page: response.meta.page || 1,
            page_size: response.meta.page_size || 20,
            total: response.meta.total || 0,
            total_pages: response.meta.total_pages || 1,
          })
        }
      }
    } catch {
      Toast.error(t('expenses.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    api,
    state.pagination.page,
    state.pagination.pageSize,
    searchKeyword,
    categoryFilter,
    statusFilter,
    paymentStatusFilter,
    dateRange,
    t,
  ])

  // Fetch summary
  const fetchSummary = useCallback(async () => {
    setSummaryLoading(true)
    try {
      const params: { from_date?: string; to_date?: string } = {}
      if (dateRange?.[0]) {
        params.from_date = dateRange[0].toISOString().split('T')[0]
      }
      if (dateRange?.[1]) {
        params.to_date = dateRange[1].toISOString().split('T')[0]
      }
      const response = await api.getFinanceExpensesSummary(params)
      if (response.success && response.data) {
        setSummary(response.data)
      }
    } catch {
      // Silently fail for summary
    } finally {
      setSummaryLoading(false)
    }
  }, [api, dateRange])

  // Fetch on mount and when state changes
  useEffect(() => {
    fetchExpenses()
  }, [fetchExpenses])

  // Fetch summary on mount
  useEffect(() => {
    fetchSummary()
  }, [fetchSummary])

  // Handle search
  const handleSearch = useCallback(
    (value: string) => {
      setSearchKeyword(value)
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle category filter change
  const handleCategoryChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const categoryValue = typeof value === 'string' ? value : ''
      setCategoryFilter(categoryValue)
      setFilter('category', categoryValue || null)
    },
    [setFilter]
  )

  // Handle status filter change
  const handleStatusChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const statusValue = typeof value === 'string' ? value : ''
      setStatusFilter(statusValue)
      setFilter('status', statusValue || null)
    },
    [setFilter]
  )

  // Handle payment status filter change
  const handlePaymentStatusChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const paymentStatusValue = typeof value === 'string' ? value : ''
      setPaymentStatusFilter(paymentStatusValue)
      setFilter('payment_status', paymentStatusValue || null)
    },
    [setFilter]
  )

  // Handle date range change
  const handleDateRangeChange = useCallback(
    (dates: unknown) => {
      if (Array.isArray(dates) && dates.length === 2) {
        const [start, end] = dates
        if (start instanceof Date && end instanceof Date) {
          setDateRange([start, end])
        } else {
          setDateRange(null)
        }
      } else {
        setDateRange(null)
      }
      handleStateChange({ pagination: { page: 1, pageSize: state.pagination.pageSize } })
    },
    [handleStateChange, state.pagination.pageSize]
  )

  // Handle create new expense
  const handleCreate = useCallback(() => {
    navigate('/finance/expenses/new')
  }, [navigate])

  // Handle edit expense
  const handleEdit = useCallback(
    (expense: ExpenseRow) => {
      if (expense.id) {
        navigate(`/finance/expenses/${expense.id}/edit`)
      }
    },
    [navigate]
  )

  // Handle submit expense for approval
  const handleSubmit = useCallback(
    async (expense: ExpenseRow) => {
      try {
        const response = await api.postFinanceExpensesIdSubmit(expense.id)
        if (response.success) {
          Toast.success(t('expenses.messages.submitSuccess'))
          fetchExpenses()
        } else {
          Toast.error(response.error || t('expenses.messages.submitError'))
        }
      } catch {
        Toast.error(t('expenses.messages.submitError'))
      }
    },
    [api, fetchExpenses, t]
  )

  // Handle approve expense
  const handleApprove = useCallback(
    async (expense: ExpenseRow) => {
      try {
        const response = await api.postFinanceExpensesIdApprove(expense.id)
        if (response.success) {
          Toast.success(t('expenses.messages.approveSuccess'))
          fetchExpenses()
        } else {
          Toast.error(response.error || t('expenses.messages.approveError'))
        }
      } catch {
        Toast.error(t('expenses.messages.approveError'))
      }
    },
    [api, fetchExpenses, t]
  )

  // Open reject modal
  const openRejectModal = useCallback((expense: ExpenseRow) => {
    setSelectedExpense(expense)
    setActionReason('')
    setRejectModalVisible(true)
  }, [])

  // Handle reject expense
  const handleReject = useCallback(async () => {
    if (!selectedExpense || !actionReason.trim()) {
      Toast.warning(t('expenses.messages.reasonRequired'))
      return
    }
    setActionLoading(true)
    try {
      const response = await api.postFinanceExpensesIdReject(selectedExpense.id, {
        reason: actionReason,
      })
      if (response.success) {
        Toast.success(t('expenses.messages.rejectSuccess'))
        setRejectModalVisible(false)
        fetchExpenses()
      } else {
        Toast.error(response.error || t('expenses.messages.rejectError'))
      }
    } catch {
      Toast.error(t('expenses.messages.rejectError'))
    } finally {
      setActionLoading(false)
    }
  }, [api, selectedExpense, actionReason, fetchExpenses, t])

  // Open cancel modal
  const openCancelModal = useCallback((expense: ExpenseRow) => {
    setSelectedExpense(expense)
    setActionReason('')
    setCancelModalVisible(true)
  }, [])

  // Handle cancel expense
  const handleCancel = useCallback(async () => {
    if (!selectedExpense || !actionReason.trim()) {
      Toast.warning(t('expenses.messages.cancelReasonRequired'))
      return
    }
    setActionLoading(true)
    try {
      const response = await api.postFinanceExpensesIdCancel(selectedExpense.id, {
        reason: actionReason,
      })
      if (response.success) {
        Toast.success(t('expenses.messages.cancelSuccess'))
        setCancelModalVisible(false)
        fetchExpenses()
      } else {
        Toast.error(response.error || t('expenses.messages.cancelError'))
      }
    } catch {
      Toast.error(t('expenses.messages.cancelError'))
    } finally {
      setActionLoading(false)
    }
  }, [api, selectedExpense, actionReason, fetchExpenses, t])

  // Handle delete expense
  const handleDelete = useCallback(
    async (expense: ExpenseRow) => {
      Modal.confirm({
        title: t('expenses.modal.deleteTitle'),
        content: t('expenses.modal.deleteContent', { expenseNumber: expense.expense_number }),
        okText: t('expenses.modal.deleteOk'),
        cancelText: t('expenses.modal.deleteCancel'),
        okType: 'danger',
        onOk: async () => {
          try {
            const response = await api.deleteFinanceExpensesId(expense.id)
            if (response.success) {
              Toast.success(t('expenses.messages.deleteSuccess'))
              fetchExpenses()
            } else {
              Toast.error(response.error || t('expenses.messages.deleteError'))
            }
          } catch {
            Toast.error(t('expenses.messages.deleteError'))
          }
        },
      })
    },
    [api, fetchExpenses, t]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchExpenses()
    fetchSummary()
  }, [fetchExpenses, fetchSummary])

  // Table columns
  const tableColumns: DataTableColumn<ExpenseRow>[] = useMemo(
    () => [
      {
        title: t('expenses.columns.expenseNumber'),
        dataIndex: 'expense_number',
        width: 140,
        sortable: true,
        render: (number: unknown) => (
          <span className="expense-number">{(number as string) || '-'}</span>
        ),
      },
      {
        title: t('expenses.columns.category'),
        dataIndex: 'category_name',
        width: 100,
        render: (name: unknown) => <span>{(name as string) || '-'}</span>,
      },
      {
        title: t('expenses.columns.description'),
        dataIndex: 'description',
        ellipsis: true,
        render: (desc: unknown) => <span>{(desc as string) || '-'}</span>,
      },
      {
        title: t('expenses.columns.amount'),
        dataIndex: 'amount',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => (
          <span className="amount-cell">{formatCurrency(amount as number)}</span>
        ),
      },
      {
        title: t('expenses.columns.incurredAt'),
        dataIndex: 'incurred_at',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
      {
        title: t('expenses.columns.status'),
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as ExpenseStatus | undefined
          if (!statusValue) return '-'
          return (
            <Tag color={STATUS_TAG_COLORS[statusValue]}>{t(`expenses.status.${statusValue}`)}</Tag>
          )
        },
      },
      {
        title: t('expenses.columns.paymentStatus'),
        dataIndex: 'payment_status',
        width: 100,
        align: 'center',
        render: (status: unknown, record: ExpenseRow) => {
          const statusValue = status as ExpensePaymentStatus | undefined
          if (!statusValue) return '-'
          // Only show payment status for approved expenses
          if (record.status !== 'APPROVED') {
            return <span className="text-muted">-</span>
          }
          return (
            <Tag color={PAYMENT_STATUS_TAG_COLORS[statusValue]}>
              {t(`expenses.paymentStatus.${statusValue}`)}
            </Tag>
          )
        },
      },
      {
        title: t('expenses.columns.createdAt'),
        dataIndex: 'created_at',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
    ],
    [t]
  )

  // Table row actions
  const tableActions: TableAction<ExpenseRow>[] = useMemo(
    () => [
      {
        key: 'edit',
        label: t('expenses.actions.edit'),
        onClick: handleEdit,
        hidden: (record) => record.status !== 'DRAFT',
      },
      {
        key: 'submit',
        label: t('expenses.actions.submit'),
        type: 'primary',
        onClick: handleSubmit,
        hidden: (record) => record.status !== 'DRAFT',
      },
      {
        key: 'approve',
        label: t('expenses.actions.approve'),
        type: 'primary',
        onClick: handleApprove,
        hidden: (record) => record.status !== 'PENDING',
      },
      {
        key: 'reject',
        label: t('expenses.actions.reject'),
        onClick: openRejectModal,
        hidden: (record) => record.status !== 'PENDING',
      },
      {
        key: 'cancel',
        label: t('expenses.actions.cancel'),
        onClick: openCancelModal,
        hidden: (record) =>
          record.status === 'CANCELLED' ||
          record.status === 'REJECTED' ||
          record.status === 'APPROVED',
      },
      {
        key: 'delete',
        label: t('expenses.actions.delete'),
        onClick: handleDelete,
        hidden: (record) => record.status !== 'DRAFT',
      },
    ],
    [handleEdit, handleSubmit, handleApprove, openRejectModal, openCancelModal, handleDelete, t]
  )

  return (
    <Container size="full" className="expenses-page">
      {/* Summary Cards */}
      <div className="expenses-summary">
        <Spin spinning={summaryLoading}>
          <Descriptions row className="summary-descriptions">
            <Descriptions.Item itemKey="total_approved">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('expenses.summary.totalApproved')}
                </Text>
                <Text className="summary-value primary">
                  {formatCurrency(summary?.total_approved)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="total_pending">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('expenses.summary.totalPending')}
                </Text>
                <Text className="summary-value warning">{summary?.total_pending ?? '-'}</Text>
              </div>
            </Descriptions.Item>
          </Descriptions>
        </Spin>
      </div>

      {/* Main Content Card */}
      <Card className="expenses-card">
        <div className="expenses-header">
          <Title heading={4} style={{ margin: 0 }}>
            {t('expenses.title')}
          </Title>
          <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
            {t('expenses.newExpense')}
          </Button>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('expenses.searchPlaceholder')}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('expenses.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="expenses-filter-container">
              <Select
                placeholder={t('expenses.filter.categoryPlaceholder')}
                value={categoryFilter}
                onChange={handleCategoryChange}
                optionList={categoryOptions}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('expenses.filter.statusPlaceholder')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={statusOptions}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('expenses.filter.paymentStatusPlaceholder')}
                value={paymentStatusFilter}
                onChange={handlePaymentStatusChange}
                optionList={paymentStatusOptions}
                style={{ width: 130 }}
              />
              <DatePicker
                type="dateRange"
                placeholder={[t('expenses.filter.startDate'), t('expenses.filter.endDate')]}
                value={dateRange as [Date, Date] | undefined}
                onChange={handleDateRangeChange}
                style={{ width: 240 }}
              />
            </Space>
          }
        />

        <Spin spinning={loading}>
          <DataTable<ExpenseRow>
            data={expenseList}
            columns={tableColumns}
            rowKey="id"
            loading={loading}
            pagination={paginationMeta}
            actions={tableActions}
            onStateChange={handleStateChange}
            sortState={state.sort}
            scroll={{ x: 1100 }}
          />
        </Spin>
      </Card>

      {/* Reject Modal */}
      <Modal
        title={t('expenses.modal.rejectTitle')}
        visible={rejectModalVisible}
        onOk={handleReject}
        onCancel={() => setRejectModalVisible(false)}
        okText={t('expenses.actions.reject')}
        cancelText={t('expenses.modal.deleteCancel')}
        confirmLoading={actionLoading}
        okButtonProps={{ type: 'danger' }}
      >
        <div className="modal-content">
          <Text>{t('expenses.modal.rejectLabel')}</Text>
          <TextArea
            value={actionReason}
            onChange={(v: string) => setActionReason(v)}
            placeholder={t('expenses.modal.rejectPlaceholder')}
            rows={3}
            maxCount={500}
            style={{ marginTop: 12 }}
          />
        </div>
      </Modal>

      {/* Cancel Modal */}
      <Modal
        title={t('expenses.modal.cancelTitle')}
        visible={cancelModalVisible}
        onOk={handleCancel}
        onCancel={() => setCancelModalVisible(false)}
        okText={t('expenses.modal.cancelExpense')}
        cancelText={t('expenses.modal.close')}
        confirmLoading={actionLoading}
      >
        <div className="modal-content">
          <Text>{t('expenses.modal.cancelLabel')}</Text>
          <TextArea
            value={actionReason}
            onChange={(v: string) => setActionReason(v)}
            placeholder={t('expenses.modal.cancelPlaceholder')}
            rows={3}
            maxCount={500}
            style={{ marginTop: 12 }}
          />
        </div>
      </Modal>
    </Container>
  )
}
