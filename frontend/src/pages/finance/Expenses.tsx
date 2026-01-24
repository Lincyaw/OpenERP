import { useState, useEffect, useCallback, useMemo } from 'react'
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

// Category options for filter
const CATEGORY_OPTIONS = [
  { label: '全部分类', value: '' },
  { label: '房租', value: 'RENT' },
  { label: '水电费', value: 'UTILITIES' },
  { label: '工资', value: 'SALARY' },
  { label: '办公费', value: 'OFFICE' },
  { label: '差旅费', value: 'TRAVEL' },
  { label: '市场营销', value: 'MARKETING' },
  { label: '设备费', value: 'EQUIPMENT' },
  { label: '维修费', value: 'MAINTENANCE' },
  { label: '保险费', value: 'INSURANCE' },
  { label: '税费', value: 'TAX' },
  { label: '其他费用', value: 'OTHER' },
]

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '草稿', value: 'DRAFT' },
  { label: '待审批', value: 'PENDING' },
  { label: '已审批', value: 'APPROVED' },
  { label: '已拒绝', value: 'REJECTED' },
  { label: '已取消', value: 'CANCELLED' },
]

// Payment status options for filter
const PAYMENT_STATUS_OPTIONS = [
  { label: '全部付款状态', value: '' },
  { label: '未付款', value: 'UNPAID' },
  { label: '已付款', value: 'PAID' },
]

// Status tag color mapping
const STATUS_TAG_COLORS: Record<ExpenseStatus, 'grey' | 'orange' | 'green' | 'red'> = {
  DRAFT: 'grey',
  PENDING: 'orange',
  APPROVED: 'green',
  REJECTED: 'red',
  CANCELLED: 'grey',
}

// Status labels
const STATUS_LABELS: Record<ExpenseStatus, string> = {
  DRAFT: '草稿',
  PENDING: '待审批',
  APPROVED: '已审批',
  REJECTED: '已拒绝',
  CANCELLED: '已取消',
}

// Payment status labels
const PAYMENT_STATUS_LABELS: Record<ExpensePaymentStatus, string> = {
  UNPAID: '未付款',
  PAID: '已付款',
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
  const navigate = useNavigate()
  const api = useMemo(() => getFinanceApi(), [])

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
      Toast.error('获取费用列表失败')
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
          Toast.success('费用已提交审批')
          fetchExpenses()
        } else {
          Toast.error(response.error || '提交失败')
        }
      } catch {
        Toast.error('提交失败')
      }
    },
    [api, fetchExpenses]
  )

  // Handle approve expense
  const handleApprove = useCallback(
    async (expense: ExpenseRow) => {
      try {
        const response = await api.postFinanceExpensesIdApprove(expense.id)
        if (response.success) {
          Toast.success('费用已审批通过')
          fetchExpenses()
        } else {
          Toast.error(response.error || '审批失败')
        }
      } catch {
        Toast.error('审批失败')
      }
    },
    [api, fetchExpenses]
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
      Toast.warning('请输入拒绝原因')
      return
    }
    setActionLoading(true)
    try {
      const response = await api.postFinanceExpensesIdReject(selectedExpense.id, {
        reason: actionReason,
      })
      if (response.success) {
        Toast.success('费用已拒绝')
        setRejectModalVisible(false)
        fetchExpenses()
      } else {
        Toast.error(response.error || '拒绝失败')
      }
    } catch {
      Toast.error('拒绝失败')
    } finally {
      setActionLoading(false)
    }
  }, [api, selectedExpense, actionReason, fetchExpenses])

  // Open cancel modal
  const openCancelModal = useCallback((expense: ExpenseRow) => {
    setSelectedExpense(expense)
    setActionReason('')
    setCancelModalVisible(true)
  }, [])

  // Handle cancel expense
  const handleCancel = useCallback(async () => {
    if (!selectedExpense || !actionReason.trim()) {
      Toast.warning('请输入取消原因')
      return
    }
    setActionLoading(true)
    try {
      const response = await api.postFinanceExpensesIdCancel(selectedExpense.id, {
        reason: actionReason,
      })
      if (response.success) {
        Toast.success('费用已取消')
        setCancelModalVisible(false)
        fetchExpenses()
      } else {
        Toast.error(response.error || '取消失败')
      }
    } catch {
      Toast.error('取消失败')
    } finally {
      setActionLoading(false)
    }
  }, [api, selectedExpense, actionReason, fetchExpenses])

  // Handle delete expense
  const handleDelete = useCallback(
    async (expense: ExpenseRow) => {
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除费用 ${expense.expense_number} 吗？此操作不可恢复。`,
        okText: '删除',
        cancelText: '取消',
        okType: 'danger',
        onOk: async () => {
          try {
            const response = await api.deleteFinanceExpensesId(expense.id)
            if (response.success) {
              Toast.success('费用已删除')
              fetchExpenses()
            } else {
              Toast.error(response.error || '删除失败')
            }
          } catch {
            Toast.error('删除失败')
          }
        },
      })
    },
    [api, fetchExpenses]
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
        title: '费用编号',
        dataIndex: 'expense_number',
        width: 140,
        sortable: true,
        render: (number: unknown) => (
          <span className="expense-number">{(number as string) || '-'}</span>
        ),
      },
      {
        title: '费用分类',
        dataIndex: 'category_name',
        width: 100,
        render: (name: unknown) => <span>{(name as string) || '-'}</span>,
      },
      {
        title: '描述',
        dataIndex: 'description',
        ellipsis: true,
        render: (desc: unknown) => <span>{(desc as string) || '-'}</span>,
      },
      {
        title: '金额',
        dataIndex: 'amount',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => (
          <span className="amount-cell">{formatCurrency(amount as number)}</span>
        ),
      },
      {
        title: '发生日期',
        dataIndex: 'incurred_at',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
      {
        title: '审批状态',
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as ExpenseStatus | undefined
          if (!statusValue) return '-'
          return <Tag color={STATUS_TAG_COLORS[statusValue]}>{STATUS_LABELS[statusValue]}</Tag>
        },
      },
      {
        title: '付款状态',
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
              {PAYMENT_STATUS_LABELS[statusValue]}
            </Tag>
          )
        },
      },
      {
        title: '创建时间',
        dataIndex: 'created_at',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
    ],
    []
  )

  // Table row actions
  const tableActions: TableAction<ExpenseRow>[] = useMemo(
    () => [
      {
        key: 'edit',
        label: '编辑',
        onClick: handleEdit,
        hidden: (record) => record.status !== 'DRAFT',
      },
      {
        key: 'submit',
        label: '提交审批',
        type: 'primary',
        onClick: handleSubmit,
        hidden: (record) => record.status !== 'DRAFT',
      },
      {
        key: 'approve',
        label: '审批通过',
        type: 'primary',
        onClick: handleApprove,
        hidden: (record) => record.status !== 'PENDING',
      },
      {
        key: 'reject',
        label: '拒绝',
        onClick: openRejectModal,
        hidden: (record) => record.status !== 'PENDING',
      },
      {
        key: 'cancel',
        label: '取消',
        onClick: openCancelModal,
        hidden: (record) =>
          record.status === 'CANCELLED' ||
          record.status === 'REJECTED' ||
          record.status === 'APPROVED',
      },
      {
        key: 'delete',
        label: '删除',
        onClick: handleDelete,
        hidden: (record) => record.status !== 'DRAFT',
      },
    ],
    [handleEdit, handleSubmit, handleApprove, openRejectModal, openCancelModal, handleDelete]
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
                  已审批总额
                </Text>
                <Text className="summary-value primary">
                  {formatCurrency(summary?.total_approved)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="total_pending">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  待审批数量
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
            费用管理
          </Title>
          <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
            新增费用
          </Button>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索费用编号、描述..."
          secondaryActions={[
            {
              key: 'refresh',
              label: '刷新',
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="expenses-filter-container">
              <Select
                placeholder="分类筛选"
                value={categoryFilter}
                onChange={handleCategoryChange}
                optionList={CATEGORY_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="审批状态"
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="付款状态"
                value={paymentStatusFilter}
                onChange={handlePaymentStatusChange}
                optionList={PAYMENT_STATUS_OPTIONS}
                style={{ width: 130 }}
              />
              <DatePicker
                type="dateRange"
                placeholder={['开始日期', '结束日期']}
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
        title="拒绝费用"
        visible={rejectModalVisible}
        onOk={handleReject}
        onCancel={() => setRejectModalVisible(false)}
        okText="拒绝"
        cancelText="取消"
        confirmLoading={actionLoading}
        okButtonProps={{ type: 'danger' }}
      >
        <div className="modal-content">
          <Text>请输入拒绝原因：</Text>
          <TextArea
            value={actionReason}
            onChange={(v: string) => setActionReason(v)}
            placeholder="请输入拒绝原因"
            rows={3}
            maxCount={500}
            style={{ marginTop: 12 }}
          />
        </div>
      </Modal>

      {/* Cancel Modal */}
      <Modal
        title="取消费用"
        visible={cancelModalVisible}
        onOk={handleCancel}
        onCancel={() => setCancelModalVisible(false)}
        okText="取消费用"
        cancelText="关闭"
        confirmLoading={actionLoading}
      >
        <div className="modal-content">
          <Text>请输入取消原因：</Text>
          <TextArea
            value={actionReason}
            onChange={(v: string) => setActionReason(v)}
            placeholder="请输入取消原因"
            rows={3}
            maxCount={500}
            style={{ marginTop: 12 }}
          />
        </div>
      </Modal>
    </Container>
  )
}
