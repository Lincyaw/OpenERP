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
} from '@douyinfe/semi-ui-19'
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
import {
  listIncomeIncomes,
  getIncomeIncomesSummary,
  confirmIncomeIncome,
  cancelIncomeIncome,
  deleteIncomeIncome,
} from '@/api/incomes/incomes'
import type {
  HandlerOtherIncomeRecordResponse,
  ListIncomeIncomesParams,
  ListIncomeIncomesCategory,
  ListIncomeIncomesStatus,
  ListIncomeIncomesReceiptStatus,
  HandlerIncomeSummaryResponse,
} from '@/api/models'
import type { PaginationMeta } from '@/types/api'
import './OtherIncomes.css'

const { Title, Text } = Typography

// Income status type
type IncomeStatus = 'DRAFT' | 'CONFIRMED' | 'CANCELLED'

// Income receipt status type
type IncomeReceiptStatus = 'PENDING' | 'RECEIVED'

// Income type with index signature for DataTable compatibility
type IncomeRow = HandlerOtherIncomeRecordResponse & Record<string, unknown>

// Status tag color mapping
const STATUS_TAG_COLORS: Record<IncomeStatus, 'grey' | 'green' | 'red'> = {
  DRAFT: 'grey',
  CONFIRMED: 'green',
  CANCELLED: 'red',
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
  const { t } = useTranslation('finance')
  const navigate = useNavigate()

  // State for data
  const [incomeList, setIncomeList] = useState<IncomeRow[]>([])
  const [paginationMeta, setPaginationMeta] = useState<PaginationMeta | undefined>(undefined)
  const [loading, setLoading] = useState(false)
  const [summary, setSummary] = useState<HandlerIncomeSummaryResponse | null>(null)
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

  // Category options for filter (memoized with translations)
  const categoryOptions = useMemo(
    () => [
      { label: t('otherIncomes.filter.allCategory'), value: '' },
      { label: t('otherIncomes.category.INVESTMENT'), value: 'INVESTMENT' },
      { label: t('otherIncomes.category.SUBSIDY'), value: 'SUBSIDY' },
      { label: t('otherIncomes.category.INTEREST'), value: 'INTEREST' },
      { label: t('otherIncomes.category.RENTAL'), value: 'RENTAL' },
      { label: t('otherIncomes.category.REFUND'), value: 'REFUND' },
      { label: t('otherIncomes.category.COMPENSATION'), value: 'COMPENSATION' },
      { label: t('otherIncomes.category.ASSET_DISPOSAL'), value: 'ASSET_DISPOSAL' },
      { label: t('otherIncomes.category.OTHER'), value: 'OTHER' },
    ],
    [t]
  )

  // Status options for filter (memoized with translations)
  const statusOptions = useMemo(
    () => [
      { label: t('otherIncomes.filter.allStatus'), value: '' },
      { label: t('otherIncomes.status.DRAFT'), value: 'DRAFT' },
      { label: t('otherIncomes.status.CONFIRMED'), value: 'CONFIRMED' },
      { label: t('otherIncomes.status.CANCELLED'), value: 'CANCELLED' },
    ],
    [t]
  )

  // Receipt status options for filter (memoized with translations)
  const receiptStatusOptions = useMemo(
    () => [
      { label: t('otherIncomes.filter.allReceiptStatus'), value: '' },
      { label: t('otherIncomes.receiptStatus.PENDING'), value: 'PENDING' },
      { label: t('otherIncomes.receiptStatus.RECEIVED'), value: 'RECEIVED' },
    ],
    [t]
  )

  // Status labels (memoized with translations)
  const statusLabels = useMemo(
    (): Record<IncomeStatus, string> => ({
      DRAFT: t('otherIncomes.status.DRAFT'),
      CONFIRMED: t('otherIncomes.status.CONFIRMED'),
      CANCELLED: t('otherIncomes.status.CANCELLED'),
    }),
    [t]
  )

  // Receipt status labels (memoized with translations)
  const receiptStatusLabels = useMemo(
    (): Record<IncomeReceiptStatus, string> => ({
      PENDING: t('otherIncomes.receiptStatus.PENDING'),
      RECEIVED: t('otherIncomes.receiptStatus.RECEIVED'),
    }),
    [t]
  )

  // Fetch incomes
  const fetchIncomes = useCallback(async () => {
    setLoading(true)
    try {
      const params: ListIncomeIncomesParams = {
        page: state.pagination.page,
        page_size: state.pagination.pageSize,
        search: searchKeyword || undefined,
        category: (categoryFilter || undefined) as ListIncomeIncomesCategory | undefined,
        status: (statusFilter || undefined) as ListIncomeIncomesStatus | undefined,
        receipt_status: (receiptStatusFilter || undefined) as
          | ListIncomeIncomesReceiptStatus
          | undefined,
        from_date: dateRange?.[0]?.toISOString().split('T')[0],
        to_date: dateRange?.[1]?.toISOString().split('T')[0],
      }

      const response = await listIncomeIncomes(params)

      if (response.status === 200 && response.data.success && response.data.data) {
        setIncomeList(response.data.data as IncomeRow[])
        if (response.data.meta) {
          setPaginationMeta({
            page: response.data.meta.page || 1,
            page_size: response.data.meta.page_size || 20,
            total: response.data.meta.total || 0,
            total_pages: response.data.meta.total_pages || 1,
          })
        }
      }
    } catch {
      Toast.error(t('otherIncomes.messages.fetchError'))
    } finally {
      setLoading(false)
    }
  }, [
    state.pagination.page,
    state.pagination.pageSize,
    searchKeyword,
    categoryFilter,
    statusFilter,
    receiptStatusFilter,
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
      const response = await getIncomeIncomesSummary(params)
      if (response.status === 200 && response.data.success && response.data.data) {
        setSummary(response.data.data)
      }
    } catch {
      // Silently fail for summary
    } finally {
      setSummaryLoading(false)
    }
  }, [dateRange])

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
        const response = await confirmIncomeIncome(income.id || '', {})
        if (response.status === 200 && response.data.success) {
          Toast.success(t('otherIncomes.messages.confirmSuccess'))
          fetchIncomes()
        } else {
          Toast.error(response.data.error?.message || t('otherIncomes.messages.confirmError'))
        }
      } catch {
        Toast.error(t('otherIncomes.messages.confirmError'))
      }
    },
    [fetchIncomes, t]
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
      Toast.warning(t('otherIncomes.messages.cancelReasonRequired'))
      return
    }
    setActionLoading(true)
    try {
      const response = await cancelIncomeIncome(selectedIncome.id || '', {
        reason: actionReason,
      })
      if (response.status === 200 && response.data.success) {
        Toast.success(t('otherIncomes.messages.cancelSuccess'))
        setCancelModalVisible(false)
        fetchIncomes()
      } else {
        Toast.error(response.data.error?.message || t('otherIncomes.messages.cancelError'))
      }
    } catch {
      Toast.error(t('otherIncomes.messages.cancelError'))
    } finally {
      setActionLoading(false)
    }
  }, [selectedIncome, actionReason, fetchIncomes, t])

  // Handle delete income
  const handleDelete = useCallback(
    async (income: IncomeRow) => {
      Modal.confirm({
        title: t('otherIncomes.modal.deleteTitle'),
        content: t('otherIncomes.modal.deleteContent', { incomeNumber: income.income_number }),
        okText: t('otherIncomes.modal.deleteOk'),
        cancelText: t('otherIncomes.modal.deleteCancel'),
        okType: 'danger',
        onOk: async () => {
          try {
            const response = await deleteIncomeIncome(income.id || '')
            if (response.status === 200 && response.data.success) {
              Toast.success(t('otherIncomes.messages.deleteSuccess'))
              fetchIncomes()
            } else {
              Toast.error(
                (response.data as { error?: { message?: string } }).error?.message ||
                  t('otherIncomes.messages.deleteError')
              )
            }
          } catch {
            Toast.error(t('otherIncomes.messages.deleteError'))
          }
        },
      })
    },
    [fetchIncomes, t]
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
        title: t('otherIncomes.columns.incomeNumber'),
        dataIndex: 'income_number',
        width: 140,
        sortable: true,
        render: (number: unknown) => (
          <span className="income-number">{(number as string) || '-'}</span>
        ),
      },
      {
        title: t('otherIncomes.columns.category'),
        dataIndex: 'category_name',
        width: 100,
        render: (name: unknown) => <span>{(name as string) || '-'}</span>,
      },
      {
        title: t('otherIncomes.columns.description'),
        dataIndex: 'description',
        ellipsis: true,
        render: (desc: unknown) => <span>{(desc as string) || '-'}</span>,
      },
      {
        title: t('otherIncomes.columns.amount'),
        dataIndex: 'amount',
        width: 120,
        align: 'right',
        sortable: true,
        render: (amount: unknown) => (
          <span className="amount-cell income">{formatCurrency(amount as number)}</span>
        ),
      },
      {
        title: t('otherIncomes.columns.receivedAt'),
        dataIndex: 'received_at',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
      {
        title: t('otherIncomes.columns.status'),
        dataIndex: 'status',
        width: 100,
        align: 'center',
        render: (status: unknown) => {
          const statusValue = status as IncomeStatus | undefined
          if (!statusValue) return '-'
          return <Tag color={STATUS_TAG_COLORS[statusValue]}>{statusLabels[statusValue]}</Tag>
        },
      },
      {
        title: t('otherIncomes.columns.receiptStatus'),
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
              {receiptStatusLabels[statusValue]}
            </Tag>
          )
        },
      },
      {
        title: t('otherIncomes.columns.createdAt'),
        dataIndex: 'created_at',
        width: 110,
        sortable: true,
        render: (date: unknown) => formatDate(date as string | undefined),
      },
    ],
    [t, statusLabels, receiptStatusLabels]
  )

  // Table row actions
  const tableActions: TableAction<IncomeRow>[] = useMemo(
    () => [
      {
        key: 'edit',
        label: t('otherIncomes.actions.edit'),
        onClick: handleEdit,
        hidden: (record) => record.status !== 'DRAFT',
      },
      {
        key: 'confirm',
        label: t('otherIncomes.actions.confirm'),
        type: 'primary',
        onClick: handleConfirm,
        hidden: (record) => record.status !== 'DRAFT',
      },
      {
        key: 'cancel',
        label: t('otherIncomes.actions.cancel'),
        onClick: openCancelModal,
        hidden: (record) => record.status === 'CANCELLED' || record.status === 'CONFIRMED',
      },
      {
        key: 'delete',
        label: t('otherIncomes.actions.delete'),
        onClick: handleDelete,
        hidden: (record) => record.status !== 'DRAFT',
      },
    ],
    [handleEdit, handleConfirm, openCancelModal, handleDelete, t]
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
                  {t('otherIncomes.summary.totalConfirmed')}
                </Text>
                <Text className="summary-value success">
                  {formatCurrency(summary?.total_confirmed)}
                </Text>
              </div>
            </Descriptions.Item>
            <Descriptions.Item itemKey="total_draft">
              <div className="summary-item">
                <Text type="secondary" className="summary-label">
                  {t('otherIncomes.summary.totalDraft')}
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
            {t('otherIncomes.title')}
          </Title>
          <Button type="primary" icon={<IconPlus />} onClick={handleCreate}>
            {t('otherIncomes.newIncome')}
          </Button>
        </div>

        <TableToolbar
          searchValue={searchKeyword}
          onSearchChange={handleSearch}
          searchPlaceholder={t('otherIncomes.searchPlaceholder')}
          secondaryActions={[
            {
              key: 'refresh',
              label: t('otherIncomes.refresh'),
              icon: <IconRefresh />,
              onClick: handleRefresh,
            },
          ]}
          filters={
            <Space className="other-incomes-filter-container">
              <Select
                placeholder={t('otherIncomes.filter.categoryPlaceholder')}
                value={categoryFilter}
                onChange={handleCategoryChange}
                optionList={categoryOptions}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('otherIncomes.filter.statusPlaceholder')}
                value={statusFilter}
                onChange={handleStatusChange}
                optionList={statusOptions}
                style={{ width: 120 }}
              />
              <Select
                placeholder={t('otherIncomes.filter.receiptStatusPlaceholder')}
                value={receiptStatusFilter}
                onChange={handleReceiptStatusChange}
                optionList={receiptStatusOptions}
                style={{ width: 130 }}
              />
              <DatePicker
                type="dateRange"
                placeholder={[t('otherIncomes.filter.startDate'), t('otherIncomes.filter.endDate')]}
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
        title={t('otherIncomes.modal.cancelTitle')}
        visible={cancelModalVisible}
        onOk={handleCancel}
        onCancel={() => setCancelModalVisible(false)}
        okText={t('otherIncomes.modal.cancelIncome')}
        cancelText={t('otherIncomes.modal.close')}
        confirmLoading={actionLoading}
      >
        <div className="modal-content">
          <Text>{t('otherIncomes.modal.cancelLabel')}</Text>
          <TextArea
            value={actionReason}
            onChange={(v: string) => setActionReason(v)}
            placeholder={t('otherIncomes.modal.cancelPlaceholder')}
            rows={3}
            maxCount={500}
            style={{ marginTop: 12 }}
          />
        </div>
      </Modal>
    </Container>
  )
}
