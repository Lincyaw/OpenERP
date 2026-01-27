import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Card,
  Typography,
  Descriptions,
  Table,
  Tag,
  Toast,
  Button,
  Space,
  Spin,
  Modal,
  Empty,
  TextArea,
  Timeline,
  Select,
  DatePicker,
  Input,
} from '@douyinfe/semi-ui-19'
import { IconArrowLeft, IconTick, IconClose, IconSearch, IconRefresh } from '@douyinfe/semi-icons'
import { useSearchParams } from 'react-router-dom'
import { Container } from '@/components/common/layout'
import { getSalesReturns } from '@/api/sales-returns/sales-returns'
import { getCustomers } from '@/api/customers/customers'
import type {
  HandlerSalesReturnResponse,
  HandlerSalesReturnItemResponse,
  HandlerCustomerListResponse,
  GetTradeSalesReturnsParams,
} from '@/api/models'
import './SalesReturnApproval.css'
import { safeToFixed, safeFormatCurrency } from '@/utils'

const { Title, Text } = Typography

// Condition labels
const CONDITION_LABELS: Record<string, string> = {
  intact: '完好',
  damaged: '损坏',
  defective: '有缺陷',
  wrong_item: '错发商品',
  other: '其他',
}

// Status tag color mapping
const STATUS_TAG_COLORS: Record<
  string,
  'blue' | 'cyan' | 'green' | 'grey' | 'red' | 'orange' | 'amber'
> = {
  DRAFT: 'blue',
  PENDING: 'orange',
  APPROVED: 'cyan',
  REJECTED: 'red',
  COMPLETED: 'green',
  CANCELLED: 'grey',
}

// Status labels
const STATUS_LABELS: Record<string, string> = {
  DRAFT: '草稿',
  PENDING: '待审批',
  APPROVED: '已审批',
  REJECTED: '已拒绝',
  COMPLETED: '已完成',
  CANCELLED: '已取消',
}

// Customer option type
interface CustomerOption {
  label: string
  value: string
}

/**
 * Format price for display
 */
function formatPrice(price?: number | string): string {
  return safeFormatCurrency(price, '¥', 2, '-')
}

/**
 * Format datetime for display
 */
function formatDateTime(dateStr?: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/**
 * Sales Return Approval Page
 *
 * Features:
 * - Display list of pending returns for approval
 * - View return details
 * - Approve returns with optional note
 * - Reject returns with required reason
 * - Search and filter pending returns
 */
export default function SalesReturnApprovalPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const salesReturnApi = useMemo(() => getSalesReturns(), [])
  const customerApi = useMemo(() => getCustomers(), [])

  // View mode: 'list' or 'detail'
  const [viewMode, setViewMode] = useState<'list' | 'detail'>(
    searchParams.get('id') ? 'detail' : 'list'
  )
  const [selectedReturnId, setSelectedReturnId] = useState<string | null>(searchParams.get('id'))

  // List state
  const [returnList, setReturnList] = useState<HandlerSalesReturnResponse[]>([])
  const [listLoading, setListLoading] = useState(false)
  const [searchKeyword, setSearchKeyword] = useState('')
  const [customerFilter, setCustomerFilter] = useState<string>('')
  const [dateRange, setDateRange] = useState<[Date, Date] | null>(null)

  // Customer options for filter
  const [customerOptions, setCustomerOptions] = useState<CustomerOption[]>([])

  // Detail state
  const [selectedReturn, setSelectedReturn] = useState<HandlerSalesReturnResponse | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [actionLoading, setActionLoading] = useState(false)

  // Modal state
  const [approveModalVisible, setApproveModalVisible] = useState(false)
  const [rejectModalVisible, setRejectModalVisible] = useState(false)
  const [approvalNote, setApprovalNote] = useState('')
  const [rejectionReason, setRejectionReason] = useState('')

  // Fetch customers for filter dropdown
  const fetchCustomers = useCallback(async () => {
    try {
      const response = await customerApi.getPartnerCustomers({ page_size: 100 })
      if (response.success && response.data) {
        const options: CustomerOption[] = response.data.map(
          (customer: HandlerCustomerListResponse) => ({
            label: customer.name || customer.code || '',
            value: customer.id || '',
          })
        )
        setCustomerOptions([{ label: '全部客户', value: '' }, ...options])
      }
    } catch {
      // Silently fail - customer filter just won't be available
    }
  }, [customerApi])

  // Fetch pending returns list
  const fetchPendingReturns = useCallback(async () => {
    setListLoading(true)
    try {
      const params: GetTradeSalesReturnsParams = {
        page: 1,
        page_size: 100,
        status: 'PENDING',
        search: searchKeyword || undefined,
        customer_id: customerFilter || undefined,
        order_by: 'submitted_at',
        order_dir: 'asc', // Oldest first for FIFO approval
      }

      if (dateRange && dateRange[0] && dateRange[1]) {
        params.start_date = dateRange[0].toISOString()
        params.end_date = dateRange[1].toISOString()
      }

      const response = await salesReturnApi.getTradeSalesReturns(params)
      if (response.success && response.data) {
        setReturnList(response.data)
      }
    } catch {
      Toast.error('获取待审批列表失败')
    } finally {
      setListLoading(false)
    }
  }, [salesReturnApi, searchKeyword, customerFilter, dateRange])

  // Fetch return detail
  const fetchReturnDetail = useCallback(
    async (id: string) => {
      setDetailLoading(true)
      try {
        const response = await salesReturnApi.getTradeSalesReturnsId(id)
        if (response.success && response.data) {
          setSelectedReturn(response.data)
        } else {
          Toast.error('退货单不存在')
          setViewMode('list')
          setSelectedReturnId(null)
        }
      } catch {
        Toast.error('获取退货单详情失败')
        setViewMode('list')
        setSelectedReturnId(null)
      } finally {
        setDetailLoading(false)
      }
    },
    [salesReturnApi]
  )

  // Initial fetch
  useEffect(() => {
    fetchCustomers()
    fetchPendingReturns()
  }, [fetchCustomers, fetchPendingReturns])

  // Fetch detail when ID changes
  useEffect(() => {
    if (selectedReturnId && viewMode === 'detail') {
      fetchReturnDetail(selectedReturnId)
    }
  }, [selectedReturnId, viewMode, fetchReturnDetail])

  // Update URL when selection changes
  useEffect(() => {
    if (selectedReturnId) {
      setSearchParams({ id: selectedReturnId })
    } else {
      setSearchParams({})
    }
  }, [selectedReturnId, setSearchParams])

  // Handle view return detail
  const handleViewDetail = useCallback((returnItem: HandlerSalesReturnResponse) => {
    setSelectedReturnId(returnItem.id || null)
    setViewMode('detail')
  }, [])

  // Handle back to list
  const handleBackToList = useCallback(() => {
    setViewMode('list')
    setSelectedReturnId(null)
    setSelectedReturn(null)
    fetchPendingReturns()
  }, [fetchPendingReturns])

  // Handle approve click - open modal
  const handleApproveClick = useCallback(() => {
    setApprovalNote('')
    setApproveModalVisible(true)
  }, [])

  // Handle reject click - open modal
  const handleRejectClick = useCallback(() => {
    setRejectionReason('')
    setRejectModalVisible(true)
  }, [])

  // Handle approve confirm
  const handleApproveConfirm = useCallback(async () => {
    if (!selectedReturn?.id) return

    setActionLoading(true)
    try {
      await salesReturnApi.postTradeSalesReturnsIdApprove(selectedReturn.id, {
        note: approvalNote || undefined,
      })
      Toast.success(`退货单 "${selectedReturn.return_number}" 已审批通过`)
      setApproveModalVisible(false)
      handleBackToList()
    } catch {
      Toast.error('审批失败')
    } finally {
      setActionLoading(false)
    }
  }, [selectedReturn, salesReturnApi, approvalNote, handleBackToList])

  // Handle reject confirm
  const handleRejectConfirm = useCallback(async () => {
    if (!selectedReturn?.id) return

    if (!rejectionReason.trim()) {
      Toast.warning('请填写拒绝原因')
      return
    }

    setActionLoading(true)
    try {
      await salesReturnApi.postTradeSalesReturnsIdReject(selectedReturn.id, {
        reason: rejectionReason,
      })
      Toast.success(`退货单 "${selectedReturn.return_number}" 已拒绝`)
      setRejectModalVisible(false)
      handleBackToList()
    } catch {
      Toast.error('拒绝失败')
    } finally {
      setActionLoading(false)
    }
  }, [selectedReturn, salesReturnApi, rejectionReason, handleBackToList])

  // Handle search
  const handleSearch = useCallback((value: string) => {
    setSearchKeyword(value)
  }, [])

  // Handle customer filter change
  const handleCustomerChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const customerValue = typeof value === 'string' ? value : ''
      setCustomerFilter(customerValue)
    },
    []
  )

  // Handle date range change
  const handleDateRangeChange = useCallback(
    (dates: Date | Date[] | string | string[] | undefined) => {
      if (Array.isArray(dates) && dates.length === 2) {
        const dateValues = dates.map((d) => (typeof d === 'string' ? new Date(d) : d)) as [
          Date,
          Date,
        ]
        setDateRange(dateValues)
      } else {
        setDateRange(null)
      }
    },
    []
  )

  // Return items table columns
  const itemColumns = useMemo(
    () => [
      {
        title: '序号',
        dataIndex: 'index',
        width: 60,
        render: (_: unknown, __: unknown, index: number) => index + 1,
      },
      {
        title: '商品编码',
        dataIndex: 'product_code',
        width: 120,
        render: (code: string) => <Text className="product-code">{code || '-'}</Text>,
      },
      {
        title: '商品名称',
        dataIndex: 'product_name',
        width: 200,
        ellipsis: true,
      },
      {
        title: '单位',
        dataIndex: 'unit',
        width: 80,
        align: 'center' as const,
        render: (unit: string) => unit || '-',
      },
      {
        title: '原数量',
        dataIndex: 'original_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => safeToFixed(qty, 2, '-'),
      },
      {
        title: '退货数量',
        dataIndex: 'return_quantity',
        width: 100,
        align: 'right' as const,
        render: (qty: number) => (
          <Text className="return-quantity">{safeToFixed(qty, 2, '-')}</Text>
        ),
      },
      {
        title: '单价',
        dataIndex: 'unit_price',
        width: 100,
        align: 'right' as const,
        render: (price: number) => formatPrice(price),
      },
      {
        title: '退款金额',
        dataIndex: 'refund_amount',
        width: 120,
        align: 'right' as const,
        render: (amount: number) => <Text className="refund-amount">{formatPrice(amount)}</Text>,
      },
      {
        title: '商品状态',
        dataIndex: 'condition_on_return',
        width: 100,
        align: 'center' as const,
        render: (condition: string) => {
          const label = CONDITION_LABELS[condition] || condition || '-'
          const color =
            condition === 'intact'
              ? 'green'
              : condition === 'damaged'
                ? 'red'
                : condition === 'defective'
                  ? 'orange'
                  : 'grey'
          return <Tag color={color}>{label}</Tag>
        },
      },
      {
        title: '退货原因',
        dataIndex: 'reason',
        ellipsis: true,
        render: (reason: string) => reason || '-',
      },
    ],
    []
  )

  // Pending list columns
  const listColumns = useMemo(
    () => [
      {
        title: '退货单号',
        dataIndex: 'return_number',
        width: 150,
        render: (returnNumber: string, record: HandlerSalesReturnResponse) => (
          <Button theme="borderless" type="primary" onClick={() => handleViewDetail(record)}>
            {returnNumber || '-'}
          </Button>
        ),
      },
      {
        title: '原订单号',
        dataIndex: 'sales_order_number',
        width: 150,
        render: (orderNumber: string) => <span className="order-number">{orderNumber || '-'}</span>,
      },
      {
        title: '客户',
        dataIndex: 'customer_name',
        width: 150,
        ellipsis: true,
      },
      {
        title: '商品数量',
        dataIndex: 'item_count',
        width: 100,
        align: 'center' as const,
        render: (count: number) => `${count || 0} 件`,
      },
      {
        title: '退款金额',
        dataIndex: 'total_refund',
        width: 120,
        align: 'right' as const,
        render: (amount: number) => <span className="refund-amount">{formatPrice(amount)}</span>,
      },
      {
        title: '提交时间',
        dataIndex: 'submitted_at',
        width: 150,
        render: (date: string) => formatDateTime(date),
      },
      {
        title: '退货原因',
        dataIndex: 'reason',
        ellipsis: true,
        render: (reason: string) => reason || '-',
      },
      {
        title: '操作',
        width: 150,
        fixed: 'right' as const,
        render: (_: unknown, record: HandlerSalesReturnResponse) => (
          <Space>
            <Button size="small" theme="light" onClick={() => handleViewDetail(record)}>
              审批
            </Button>
          </Space>
        ),
      },
    ],
    [handleViewDetail]
  )

  // Build timeline items
  const timelineItems = useMemo(() => {
    if (!selectedReturn) return []

    const items = []

    if (selectedReturn.created_at) {
      items.push({
        time: formatDateTime(selectedReturn.created_at),
        content: '退货单创建',
        type: 'default' as const,
      })
    }

    if (selectedReturn.submitted_at) {
      items.push({
        time: formatDateTime(selectedReturn.submitted_at),
        content: '提交审批',
        type: 'ongoing' as const,
      })
    }

    return items
  }, [selectedReturn])

  // Render basic info for detail view
  const renderBasicInfo = () => {
    if (!selectedReturn) return null

    const data = [
      { key: '退货单号', value: selectedReturn.return_number },
      { key: '原订单号', value: selectedReturn.sales_order_number || '-' },
      { key: '客户名称', value: selectedReturn.customer_name || '-' },
      {
        key: '状态',
        value: (
          <Tag color={STATUS_TAG_COLORS[selectedReturn.status || 'PENDING']}>
            {STATUS_LABELS[selectedReturn.status || 'PENDING']}
          </Tag>
        ),
      },
      { key: '商品数量', value: `${selectedReturn.item_count || 0} 件` },
      { key: '总退货数量', value: safeToFixed(selectedReturn.total_quantity, 2, '0.00') },
      { key: '提交时间', value: formatDateTime(selectedReturn.submitted_at) },
      { key: '退货原因', value: selectedReturn.reason || '-' },
      { key: '备注', value: selectedReturn.remark || '-' },
    ]

    return <Descriptions data={data} row className="return-basic-info" />
  }

  // Render amount summary
  const renderAmountSummary = () => {
    if (!selectedReturn) return null

    return (
      <div className="amount-summary">
        <div className="amount-row total-row">
          <Text strong>应退金额</Text>
          <Text className="refund-total" strong>
            {formatPrice(selectedReturn.total_refund)}
          </Text>
        </div>
      </div>
    )
  }

  // Render list view
  const renderListView = () => (
    <Container size="full" className="sales-return-approval-page">
      <Card className="approval-card">
        <div className="page-header">
          <div className="header-left">
            <Title heading={4} style={{ margin: 0 }}>
              销售退货审批
            </Title>
            <Tag color="orange" size="large">
              待审批: {returnList.length}
            </Tag>
          </div>
          <div className="header-right">
            <Button icon={<IconRefresh />} onClick={fetchPendingReturns} loading={listLoading}>
              刷新
            </Button>
          </div>
        </div>

        {/* Filters */}
        <div className="filter-bar">
          <Space wrap>
            <Input
              prefix={<IconSearch />}
              placeholder="搜索退货单号..."
              value={searchKeyword}
              onChange={handleSearch}
              showClear
              style={{ width: 200 }}
            />
            <Select
              placeholder="客户筛选"
              value={customerFilter}
              onChange={handleCustomerChange}
              optionList={customerOptions}
              filter
              style={{ width: 150 }}
            />
            <DatePicker
              type="dateRange"
              placeholder={['开始日期', '结束日期']}
              value={dateRange || undefined}
              onChange={handleDateRangeChange}
              style={{ width: 260 }}
            />
          </Space>
        </div>

        <Spin spinning={listLoading}>
          {returnList.length === 0 ? (
            <Empty
              title="暂无待审批退货单"
              description="所有退货单已审批完成"
              className="empty-state"
            />
          ) : (
            <Table
              columns={listColumns}
              dataSource={returnList as (HandlerSalesReturnResponse & Record<string, unknown>)[]}
              rowKey="id"
              pagination={false}
              scroll={{ x: 1200 }}
            />
          )}
        </Spin>
      </Card>
    </Container>
  )

  // Render detail view
  const renderDetailView = () => {
    if (detailLoading) {
      return (
        <Container size="lg" className="sales-return-approval-page">
          <div className="loading-container">
            <Spin size="large" />
          </div>
        </Container>
      )
    }

    if (!selectedReturn) {
      return (
        <Container size="lg" className="sales-return-approval-page">
          <Empty title="退货单不存在" description="您访问的退货单不存在或已被删除" />
        </Container>
      )
    }

    return (
      <Container size="lg" className="sales-return-approval-page">
        {/* Header */}
        <div className="page-header">
          <div className="header-left">
            <Button icon={<IconArrowLeft />} theme="borderless" onClick={handleBackToList}>
              返回列表
            </Button>
            <Title heading={4} className="page-title">
              审批退货单
            </Title>
            <Tag color={STATUS_TAG_COLORS[selectedReturn.status || 'PENDING']} size="large">
              {STATUS_LABELS[selectedReturn.status || 'PENDING']}
            </Tag>
          </div>
          {selectedReturn.status === 'PENDING' && (
            <div className="header-right">
              <Space>
                <Button
                  type="danger"
                  icon={<IconClose />}
                  onClick={handleRejectClick}
                  loading={actionLoading}
                >
                  拒绝
                </Button>
                <Button
                  type="primary"
                  icon={<IconTick />}
                  onClick={handleApproveClick}
                  loading={actionLoading}
                >
                  通过
                </Button>
              </Space>
            </div>
          )}
        </div>

        {/* Return Info Card */}
        <Card className="info-card" title="基本信息">
          {renderBasicInfo()}
        </Card>

        {/* Return Items Card */}
        <Card className="items-card" title="退货商品明细">
          <Table
            columns={itemColumns}
            dataSource={
              (selectedReturn.items || []) as (HandlerSalesReturnItemResponse &
                Record<string, unknown>)[]
            }
            rowKey="id"
            pagination={false}
            size="small"
            scroll={{ x: 1100 }}
            empty={<Empty description="暂无商品" />}
          />
          {renderAmountSummary()}
        </Card>

        {/* Timeline Card */}
        <Card className="timeline-card" title="状态变更">
          {timelineItems.length > 0 ? (
            <Timeline mode="left" className="status-timeline">
              {timelineItems.map((item, index) => (
                <Timeline.Item
                  key={index}
                  time={item.time}
                  type={item.type as 'default' | 'ongoing' | 'success' | 'warning' | 'error'}
                >
                  {item.content}
                </Timeline.Item>
              ))}
            </Timeline>
          ) : (
            <Empty description="暂无状态记录" />
          )}
        </Card>

        {/* Approve Modal */}
        <Modal
          title="审批通过"
          visible={approveModalVisible}
          onOk={handleApproveConfirm}
          onCancel={() => setApproveModalVisible(false)}
          okText="确认通过"
          cancelText="取消"
          confirmLoading={actionLoading}
        >
          <div className="modal-content">
            <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
              确定要通过退货单 "{selectedReturn.return_number}" 的审批吗？
            </Text>
            <div className="form-field">
              <Text className="field-label">审批意见（可选）</Text>
              <TextArea
                value={approvalNote}
                onChange={setApprovalNote}
                placeholder="请输入审批意见..."
                maxCount={500}
                rows={3}
              />
            </div>
          </div>
        </Modal>

        {/* Reject Modal */}
        <Modal
          title="拒绝退货"
          visible={rejectModalVisible}
          onOk={handleRejectConfirm}
          onCancel={() => setRejectModalVisible(false)}
          okText="确认拒绝"
          okButtonProps={{ type: 'danger' }}
          cancelText="取消"
          confirmLoading={actionLoading}
        >
          <div className="modal-content">
            <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
              确定要拒绝退货单 "{selectedReturn.return_number}" 吗？
            </Text>
            <div className="form-field">
              <Text className="field-label required">拒绝原因</Text>
              <TextArea
                value={rejectionReason}
                onChange={setRejectionReason}
                placeholder="请输入拒绝原因（必填）..."
                maxCount={500}
                rows={3}
              />
            </div>
          </div>
        </Modal>
      </Container>
    )
  }

  return viewMode === 'list' ? renderListView() : renderDetailView()
}
