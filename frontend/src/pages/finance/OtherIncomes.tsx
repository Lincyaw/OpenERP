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
  OtherIncomeRecord,
  IncomeCategory,
  IncomeStatus,
  IncomeReceiptStatus,
  IncomeSummary,
  GetOtherIncomeRecordsParams,
} from '@/api/finance'
import type { PaginationMeta } from '@/types/api'
import './OtherIncomes.css'

const { Title, Text } = Typography

// Income type with index signature for DataTable compatibility
type IncomeRow = OtherIncomeRecord & Record<string, unknown>

// Category options for filter
const CATEGORY_OPTIONS = [
  { label: '全部分类', value: '' },
  { label: '投资收益', value: 'INVESTMENT' },
  { label: '补贴收入', value: 'SUBSIDY' },
  { label: '利息收入', value: 'INTEREST' },
  { label: '租金收入', value: 'RENTAL' },
  { label: '退款收入', value: 'REFUND' },
  { label: '赔偿收入', value: 'COMPENSATION' },
  { label: '资产处置', value: 'ASSET_DISPOSAL' },
  { label: '其他收入', value: 'OTHER' },
]

// Status options for filter
const STATUS_OPTIONS = [
  { label: '全部状态', value: '' },
  { label: '草稿', value: 'DRAFT' },
  { label: '已确认', value: 'CONFIRMED' },
  { label: '已取消', value: 'CANCELLED' },
]

// Receipt status options for filter
const RECEIPT_STATUS_OPTIONS = [
  { label: '全部到账状态', value: '' },
  { label: '待到账', value: 'PENDING' },
  { label: '已到账', value: 'RECEIVED' },
]

// Status tag color mapping
const STATUS_TAG_COLORS: Record<IncomeStatus, 'grey' | 'green' | 'red'> = {
  DRAFT: 'grey',
  CONFIRMED: 'green',
  CANCELLED: 'red',
}

// Status labels
const STATUS_LABELS: Record<IncomeStatus, string> = {
  DRAFT: '草稿',
  CONFIRMED: '已确认',
  CANCELLED: '已取消',
}

// Receipt status labels
const RECEIPT_STATUS_LABELS: Record<IncomeReceiptStatus, string> = {
  PENDING: '待到账',
  RECEIVED: '已到账',
}

// Receipt status tag colors
const RECEIPT_STATUS_TAG_COLORS: Record<IncomeReceiptStatus, 'orange' | 'green'> = {
  PENDING: 'orange',
  RECEIVED: 'green',
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
 * Other Incomes list page
 *
 * Features:
 * - Other income record listing with pagination
 * - Search by income number, description
 * - Filter by category, status, receipt status, date range
 * - Summary cards showing key metrics
 * - CRUD operations with confirmation workflow
 */
export default function OtherIncomesPage() {
  const navigate = useNavigate()
  const api = useMemo(() => getFinanceApi(), [])

  // State for data
  const [incomeList, setIncomeList] = useState<IncomeRow[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [summary, setSummary] = useState<IncomeSummary | null>(null)
  const [summaryLoading, setSummaryLoading] = useState(false)

  // Filter state
  const [searchKeyword, setSearchKeyword] = useState('')
  const [categoryFilter, setCategoryFilter] = useState<string>('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [receiptStatusFilter, setReceiptStatusFilter] = useState<string>('')
  const [dateRange, setDateRange] = useState<[Date, Date] | null>(null)

  // Modal state for actions
  const [cancelModalVisible, setCancelModalVisible] = useState(false)
  const [actionReason, setActionReason] = useState('')
  const [selectedIncome, setSelectedIncome] = useState<IncomeRow | null>(null)
  const [actionLoading, setActionLoading] = useState(false)

  // Table state hook
  const { state, handleStateChange, setFilter } = useTableState({
    defaultPageSize: 20,
    defaultSortField: 'created_at',
    defaultSortOrder: 'desc',
  })

  // Fetch incomes
  const fetchIncomes = useCallback(async () => {
    setLoading(true)
    try {
      const params: GetOtherIncomeRecordsParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        category: (categoryFilter || undefined) as IncomeCategory | undefined,
        status: (statusFilter || undefined) as IncomeStatus | undefined,
        receipt_status: (receiptStatusFilter || undefined) as IncomeReceiptStatus | undefined,
        from_date: dateRange?.[0]?.toISOString().split('T')[0],
        to_date: dateRange?.[1]?.toISOString().split('T')[0],
      }

      const response = await api.getFinanceIncomes(params)

      if (response.success && response.data) {
        setIncomeList(response.data as IncomeRow[])
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
      Toast.error('获取其他收入列表失败')
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
    receiptStatusFilter,
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
      const response = await api.getFinanceIncomesSummary(params)
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
    fetchIncomes()
  }, [fetchIncomes])

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

  // Handle receipt status filter change
  const handleReceiptStatusChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const receiptStatusValue = typeof value === 'string' ? value : ''
      setReceiptStatusFilter(receiptStatusValue)
      setFilter('receipt_status', receiptStatusValue || null)
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

  // Handle create new income
  const handleCreate = useCallback(() => {
    navigate('/finance/incomes/new')
  }, [navigate])

  // Handle edit income
  const handleEdit = useCallback(
    (income: IncomeRow) => {
      if (income.id) {
        navigate(`/finance/incomes/${income.id}/edit`)
      }
    },
    [navigate]
  )

  // Handle confirm income
  const handleConfirm = useCallback(
    async (income: IncomeRow) => {
      try {
        const response = await api.postFinanceIncomesIdConfirm(income.id)
        if (response.success) {
          Toast.success('收入已确认')
          fetchIncomes()
        } else {
          Toast.error(response.error || '确认失败')
        }
      } catch {
        Toast.error('确认失败')
      }
    },
    [api, fetchIncomes]
  )

  // Open cancel modal
  const openCancelModal = useCallback((income: IncomeRow) => {
    setSelectedIncome(income)
    setActionReason('')
    setCancelModalVisible(true)
  }, [])

  // Handle cancel income
  const handleCancel = useCallback(async () => {
    if (!selectedIncome || !actionReason.trim()) {
      Toast.warning('请输入取消原因')
      return
    }
    setActionLoading(true)
    try {
      const response = await api.postFinanceIncomesIdCancel(selectedIncome.id, {
        reason: actionReason,
      })
      if (response.success) {
        Toast.success('收入已取消')
        setCancelModalVisible(false)
        fetchIncomes()
      } else {
        Toast.error(response.error || '取消失败')
      }
    } catch {
      Toast.error('取消失败')
    } finally {
      setActionLoading(false)
    }
  }, [api, selectedIncome, actionReason, fetchIncomes])

  // Handle delete income
  const handleDelete = useCallback(
    async (income: IncomeRow) => {
      Modal.confirm({
        title: '确认删除',
        content: `确定要删除收入 ${income.income_number} 吗？此操作不可恢复。`,
        okText: '删除',
        cancelText: '取消',
        okType: 'danger',
        onOk: async () => {
          try {
            const response = await api.deleteFinanceIncomesId(income.id)
            if (response.success) {
              Toast.success('收入已删除')
              fetchIncomes()
            } else {
              Toast.error(response.error || '删除失败')
            }
          } catch {
            Toast.error('删除失败')
          }
        },
      })
    },
    [api, fetchIncomes]
  )

  // Refresh handler
  const handleRefresh = useCallback(() => {
    fetchIncomes()
    fetchSummary()
  }, [fetchIncomes, fetchSummary])

  // Table columns
  const tableColumns: DataTableColumn<IncomeRow>[] = useMemo(
    () => [
      {
        title: '收入编号',
        dataIndex: 'income_number',
        width: 140,
        sortable: true,
        render: (number: unknown) => (
          <span className="income-number">{(number as string) || '-'}</span>
        ),
      },
      {
        title: '收入分类',
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
          <span className="amount-cell income">{formatCurrency(amount as number)}</span>
        ),
      },
      {
        title: '收入日期',
        dataIndex: 'received_at',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
      {
        title: '状态',
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as IncomeStatus | undefined
          if (!statusValue) return '-'
          return <Tag color={STATUS_TAG_COLORS[statusValue]}>{STATUS_LABELS[statusValue]}</Tag>
        },
      },
      {
        title: '到账状态',
        dataIndex: 'receipt_status',
        width: 100,
        align: 'center',
        render: (status: unknown, record: IncomeRow) => {
          const statusValue = status as IncomeReceiptStatus | undefined
          if (!statusValue) return '-'
          // Only show receipt status for confirmed incomes
          if (record.status !== 'CONFIRMED') {
            return <span className="text-muted">-</span>
          }
          return (
            <Tag color={RECEIPT_STATUS_TAG_COLORS[statusValue]}>
              {RECEIPT_STATUS_LABELS[statusValue]}
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
  const tableActions: TableAction<IncomeRow>[] = useMemo(
    () => [
      {
        key: 'edit',
        label: '编辑',
        onClick: handleEdit,
        hidden: (record) => record.status !== 'DRAFT',
      },
      {
        key: 'confirm',
        label: '确认',
        type: 'primary',
        onClick: handleConfirm,
        hidden: (record) => record.status !== 'DRAFT',
      },
      {
        key: 'cancel',
        label: '取消',
        onClick: openCancelModal,
        hidden: (record) => record.status === 'CANCELLED' || record.status === 'CONFIRMED',
      },
      {
        key: 'delete',
        label: '删除',
        onClick: handleDelete,
        hidden: (record) => record.status !== 'DRAFT',
      },
    ],
    [handleEdit, handleConfirm, openCancelModal, handleDelete]
  )

  return (
    <Container size="full" className="other-incomes-page">
      {/* Summary Cards */}
      <div className="other-incomes-summary">
        <Spin spinning={summaryLoading}>
          <Descriptions row className="summary-descriptions">
            <Descriptions.Item itemKey="total_confirmed">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  已确认总额
                </Text>
                <Text className="summary-value success">
                  {formatCurrency(summary?.total_confirmed)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="total_draft">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  待确认总额
                </Text>
                <Text className="summary-value warning">
                  {formatCurrency(summary?.total_draft)}
                </Text>
              </div>
            </Descriptions.Item>
          </Descriptions>
        </Spin>
      </div>

      {/* Main Content Card */}
      <Card className="other-incomes-card">
        <div className="other-incomes-header">
          <Title heading={4} style={{ margin: 0 }}>
            其他收入管理
          </Title>
          <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
            新增收入
          </Button>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder="搜索收入编号、描述..."
          secondaryActions={[
            {
              key: 'refresh',
              label: '刷新',
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="other-incomes-filter-container">
              <Select
                placeholder="分类筛选"
                value={categoryFilter}
                onChange={handleCategoryChange}
                optionList={CATEGORY_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="状态"
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={STATUS_OPTIONS}
                style={{ width: 120 }}
              />
              <Select
                placeholder="到账状态"
                value={receiptStatusFilter}
                onChange={handleReceiptStatusChange}
                optionList={RECEIPT_STATUS_OPTIONS}
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
          <DataTable<IncomeRow>
            data={incomeList}
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

      {/* Cancel Modal */}
      <Modal
        title="取消收入"
        visible={cancelModalVisible}
        onOk={handleCancel}
        onCancel={() => setCancelModalVisible(false)}
        okText="取消收入"
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
