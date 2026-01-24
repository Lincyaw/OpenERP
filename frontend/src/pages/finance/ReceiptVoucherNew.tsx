import { useState, useEffect, useCallback, useMemo } from 'react'
import { z } from 'zod'
import { Card, Typography, Toast, Spin, Select, Tag, Banner } from '@douyinfe/semi-ui'
import { useNavigate, useSearchParams } from 'react-router-dom'
import {
  Form,
  FormActions,
  FormSection,
  FormRow,
  TextField,
  NumberField,
  TextAreaField,
  SelectField,
  DateField,
  useFormWithValidation,
  validationMessages,
  createEnumSchema,
} from '@/components/common/form'
import { Container } from '@/components/common/layout'
import { getFinanceApi } from '@/api/finance'
import { getCustomers } from '@/api/customers/customers'
import type { PaymentMethod, CreateReceiptVoucherRequest, AccountReceivable } from '@/api/finance'
import type { HandlerCustomerResponse } from '@/api/models'
import './ReceiptVoucherNew.css'

const { Title, Text } = Typography

// Payment method options
const PAYMENT_METHOD_OPTIONS = [
  { label: '现金', value: 'CASH' },
  { label: '银行转账', value: 'BANK_TRANSFER' },
  { label: '微信支付', value: 'WECHAT' },
  { label: '支付宝', value: 'ALIPAY' },
  { label: '支票', value: 'CHECK' },
  { label: '余额抵扣', value: 'BALANCE' },
  { label: '其他', value: 'OTHER' },
]

// Payment method values
const PAYMENT_METHODS = [
  'CASH',
  'BANK_TRANSFER',
  'WECHAT',
  'ALIPAY',
  'CHECK',
  'BALANCE',
  'OTHER',
] as const

// Form validation schema
const receiptVoucherFormSchema = z.object({
  customer_id: z.string().min(1, validationMessages.required),
  customer_name: z.string().min(1, validationMessages.required),
  amount: z
    .number()
    .positive('收款金额必须大于0')
    .max(999999999.99, '收款金额不能超过999,999,999.99'),
  payment_method: createEnumSchema(PAYMENT_METHODS, true),
  payment_reference: z
    .string()
    .max(100, validationMessages.maxLength(100))
    .optional()
    .transform((val) => val || undefined),
  receipt_date: z.date({ message: validationMessages.required }),
  remark: z
    .string()
    .max(500, validationMessages.maxLength(500))
    .optional()
    .transform((val) => val || undefined),
})

type ReceiptVoucherFormData = z.infer<typeof receiptVoucherFormSchema>

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
 * Receipt Voucher Creation Page
 *
 * Features:
 * - Customer selection with search
 * - Payment method selection
 * - Amount input with validation
 * - Receipt date picker
 * - Optional payment reference (transaction ID, check number, etc.)
 * - Shows customer's outstanding receivables summary
 */
export default function ReceiptVoucherNewPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const financeApi = useMemo(() => getFinanceApi(), [])
  const customerApi = useMemo(() => getCustomers(), [])

  // URL params for pre-filling
  const preSelectedCustomerId = searchParams.get('customer_id') || ''

  // State
  const [customerOptions, setCustomerOptions] = useState<
    Array<{ label: string; value: string; customer: HandlerCustomerResponse }>
  >([])
  const [customerLoading, setCustomerLoading] = useState(false)
  const [selectedCustomer, setSelectedCustomer] = useState<HandlerCustomerResponse | null>(null)
  const [customerReceivables, setCustomerReceivables] = useState<AccountReceivable[]>([])
  const [receivablesLoading, setReceivablesLoading] = useState(false)

  // Form setup
  const defaultValues: Partial<ReceiptVoucherFormData> = useMemo(
    () => ({
      customer_id: preSelectedCustomerId,
      customer_name: '',
      amount: undefined,
      payment_method: 'CASH' as const,
      payment_reference: '',
      receipt_date: new Date(),
      remark: '',
    }),
    [preSelectedCustomerId]
  )

  const { control, handleFormSubmit, isSubmitting, setValue, watch } =
    useFormWithValidation<ReceiptVoucherFormData>({
      schema: receiptVoucherFormSchema,
      defaultValues,
      successMessage: '收款单创建成功',
      onSuccess: () => {
        navigate('/finance/receivables')
      },
    })

  const watchedCustomerId = watch('customer_id')

  // Search customers
  const searchCustomers = useCallback(
    async (keyword: string) => {
      if (!keyword || keyword.length < 1) {
        setCustomerOptions([])
        return
      }

      setCustomerLoading(true)
      try {
        const response = await customerApi.getPartnerCustomers({
          search: keyword,
          status: 'active',
          page: 1,
          page_size: 20,
        })

        if (response.success && response.data) {
          const options = response.data.map((customer) => ({
            label: `${customer.name} (${customer.code})`,
            value: customer.id || '',
            customer,
          }))
          setCustomerOptions(options)
        }
      } catch {
        Toast.error('搜索客户失败')
      } finally {
        setCustomerLoading(false)
      }
    },
    [customerApi]
  )

  // Fetch customer receivables when customer is selected
  const fetchCustomerReceivables = useCallback(
    async (customerId: string) => {
      if (!customerId) {
        setCustomerReceivables([])
        return
      }

      setReceivablesLoading(true)
      try {
        const receivablesResponse = await financeApi.getFinanceReceivables({
          customer_id: customerId,
          status: 'PENDING',
          page: 1,
          page_size: 100,
        })

        if (receivablesResponse.success && receivablesResponse.data) {
          // Filter receivables for this customer only
          const filteredReceivables = receivablesResponse.data.filter(
            (r) =>
              r.customer_id === customerId && (r.status === 'PENDING' || r.status === 'PARTIAL')
          )
          setCustomerReceivables(filteredReceivables)
        }
      } catch {
        // Silent fail
      } finally {
        setReceivablesLoading(false)
      }
    },
    [financeApi]
  )

  // Load pre-selected customer
  useEffect(() => {
    if (preSelectedCustomerId) {
      const loadCustomer = async () => {
        try {
          const response = await customerApi.getPartnerCustomersId(preSelectedCustomerId)
          if (response.success && response.data) {
            const customer = response.data
            setSelectedCustomer(customer)
            setValue('customer_id', customer.id || '')
            setValue('customer_name', customer.name || '')
            setCustomerOptions([
              {
                label: `${customer.name} (${customer.code})`,
                value: customer.id || '',
                customer,
              },
            ])
            fetchCustomerReceivables(customer.id || '')
          }
        } catch {
          Toast.error('加载客户信息失败')
        }
      }
      loadCustomer()
    }
  }, [preSelectedCustomerId, customerApi, setValue, fetchCustomerReceivables])

  // Watch for customer changes
  useEffect(() => {
    if (watchedCustomerId && watchedCustomerId !== selectedCustomer?.id) {
      fetchCustomerReceivables(watchedCustomerId)
    }
  }, [watchedCustomerId, selectedCustomer?.id, fetchCustomerReceivables])

  // Handle customer selection
  const handleCustomerChange = useCallback(
    (value: string | number | (string | number)[] | Record<string, unknown> | undefined) => {
      const customerId = typeof value === 'string' ? value : ''
      const selectedOption = customerOptions.find((opt) => opt.value === customerId)

      if (selectedOption) {
        setSelectedCustomer(selectedOption.customer)
        setValue('customer_id', customerId)
        setValue('customer_name', selectedOption.customer.name || '')
      } else {
        setSelectedCustomer(null)
        setValue('customer_id', '')
        setValue('customer_name', '')
      }
    },
    [customerOptions, setValue]
  )

  // Handle customer search
  const handleCustomerSearch = useCallback(
    (inputValue: string) => {
      searchCustomers(inputValue)
    },
    [searchCustomers]
  )

  // Handle form submission
  const onSubmit = async (data: ReceiptVoucherFormData) => {
    const request: CreateReceiptVoucherRequest = {
      customer_id: data.customer_id,
      customer_name: data.customer_name,
      amount: data.amount,
      payment_method: data.payment_method as PaymentMethod,
      payment_reference: data.payment_reference,
      receipt_date: data.receipt_date.toISOString().split('T')[0],
      remark: data.remark,
    }

    const response = await financeApi.postFinanceReceipts(request)
    if (!response.success) {
      throw new Error(response.error || '创建收款单失败')
    }
  }

  // Handle cancel
  const handleCancel = () => {
    navigate('/finance/receivables')
  }

  // Calculate total outstanding for selected customer
  const totalOutstanding = useMemo(() => {
    return customerReceivables.reduce((sum, r) => sum + (r.outstanding_amount || 0), 0)
  }, [customerReceivables])

  return (
    <Container size="md" className="receipt-voucher-new-page">
      <Card className="receipt-voucher-form-card">
        <div className="receipt-voucher-form-header">
          <Title heading={4} style={{ margin: 0 }}>
            新增收款单
          </Title>
        </div>

        {/* Customer receivables summary */}
        {selectedCustomer && (
          <div className="customer-receivables-summary">
            <Spin spinning={receivablesLoading}>
              {customerReceivables.length > 0 ? (
                <Banner
                  type="info"
                  description={
                    <div className="receivables-info">
                      <Text>
                        客户 <strong>{selectedCustomer.name}</strong> 共有{' '}
                        <strong>{customerReceivables.length}</strong> 笔待收账款，待收总额:{' '}
                        <strong className="amount-highlight">
                          {formatCurrency(totalOutstanding)}
                        </strong>
                      </Text>
                    </div>
                  }
                />
              ) : (
                <Banner
                  type="success"
                  description={
                    <Text>
                      客户 <strong>{selectedCustomer.name}</strong> 暂无待收账款
                    </Text>
                  }
                />
              )}
            </Spin>

            {/* Pending receivables list */}
            {customerReceivables.length > 0 && (
              <div className="pending-receivables-list">
                <Text type="secondary" className="list-title">
                  待核销应收账款:
                </Text>
                <div className="receivables-tags">
                  {customerReceivables.slice(0, 5).map((receivable) => (
                    <Tag
                      key={receivable.id}
                      color={receivable.status === 'PARTIAL' ? 'blue' : 'orange'}
                      className="receivable-tag"
                    >
                      {receivable.receivable_number}:{' '}
                      {formatCurrency(receivable.outstanding_amount)}
                    </Tag>
                  ))}
                  {customerReceivables.length > 5 && (
                    <Tag color="grey">+{customerReceivables.length - 5} 更多</Tag>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection title="客户信息" description="选择收款的客户">
            <FormRow cols={1}>
              <div className="customer-select-wrapper">
                <label className="semi-form-field-label">
                  <span className="semi-form-field-label-text">
                    <span className="semi-form-field-label-required">*</span>
                    客户
                  </span>
                </label>
                <Select
                  placeholder="请输入客户名称或编码搜索"
                  value={watchedCustomerId || undefined}
                  onChange={handleCustomerChange}
                  onSearch={handleCustomerSearch}
                  optionList={customerOptions}
                  loading={customerLoading}
                  filter={false}
                  remote
                  showClear
                  style={{ width: '100%' }}
                />
              </div>
            </FormRow>
          </FormSection>

          <FormSection title="收款信息" description="填写收款金额和方式">
            <FormRow cols={2}>
              <NumberField
                name="amount"
                control={control}
                label="收款金额"
                placeholder="请输入收款金额"
                required
                min={0.01}
                max={999999999.99}
                precision={2}
                prefix="¥"
                helperText={
                  totalOutstanding > 0 ? `待收总额: ${formatCurrency(totalOutstanding)}` : undefined
                }
              />
              <SelectField
                name="payment_method"
                control={control}
                label="收款方式"
                placeholder="请选择收款方式"
                options={PAYMENT_METHOD_OPTIONS}
                required
              />
            </FormRow>
            <FormRow cols={2}>
              <DateField
                name="receipt_date"
                control={control}
                label="收款日期"
                placeholder="请选择收款日期"
                required
              />
              <TextField
                name="payment_reference"
                control={control}
                label="收款凭证号"
                placeholder="交易流水号、支票号等 (可选)"
                helperText="用于关联银行流水或支付平台交易"
              />
            </FormRow>
          </FormSection>

          <FormSection title="其他信息" description="备注说明">
            <TextAreaField
              name="remark"
              control={control}
              label="备注"
              placeholder="请输入备注信息 (可选)"
              rows={3}
              maxCount={500}
            />
          </FormSection>

          <FormActions
            submitText="创建"
            isSubmitting={isSubmitting}
            onCancel={handleCancel}
            showCancel
          />
        </Form>
      </Card>
    </Container>
  )
}
