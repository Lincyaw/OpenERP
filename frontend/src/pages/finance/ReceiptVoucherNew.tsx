import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'
import { Card, Typography, Toast, Spin, Select, Tag, Banner } from '@douyinfe/semi-ui-19'
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

// Create form validation schema with translations
function createReceiptVoucherFormSchema(t: (key: string) => string) {
  return z.object({
    customer_id: z.string().min(1, validationMessages.required),
    customer_name: z.string().min(1, validationMessages.required),
    amount: z
      .number()
      .positive(t('receiptVoucher.validation.amountPositive'))
      .max(999999999.99, t('receiptVoucher.validation.amountMax')),
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
}

type ReceiptVoucherFormData = z.infer<ReturnType<typeof createReceiptVoucherFormSchema>>

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
  const { t } = useTranslation('finance')
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const financeApi = useMemo(() => getFinanceApi(), [])
  const customerApi = useMemo(() => getCustomers(), [])

  // Payment method options with translated labels
  const paymentMethodOptions = useMemo(
    () => [
      { label: t('paymentMethod.CASH'), value: 'CASH' },
      { label: t('paymentMethod.BANK_TRANSFER'), value: 'BANK_TRANSFER' },
      { label: t('paymentMethod.WECHAT'), value: 'WECHAT' },
      { label: t('paymentMethod.ALIPAY'), value: 'ALIPAY' },
      { label: t('paymentMethod.CHECK'), value: 'CHECK' },
      { label: t('paymentMethod.BALANCE'), value: 'BALANCE' },
      { label: t('paymentMethod.OTHER'), value: 'OTHER' },
    ],
    [t]
  )

  // Form validation schema with translated messages
  const receiptVoucherFormSchema = useMemo(() => createReceiptVoucherFormSchema(t), [t])

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
      successMessage: t('receiptVoucher.messages.createSuccess'),
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
        const response = await customerApi.listCustomers({
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
        Toast.error(t('receiptVoucher.messages.searchCustomerError'))
      } finally {
        setCustomerLoading(false)
      }
    },
    [customerApi, t]
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
        const receivablesResponse = await financeApi.listFinanceReceivablesReceivables({
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
          const response = await customerApi.getCustomerById(preSelectedCustomerId)
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
          Toast.error(t('receiptVoucher.messages.loadCustomerError'))
        }
      }
      loadCustomer()
    }
  }, [preSelectedCustomerId, customerApi, setValue, fetchCustomerReceivables, t])

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

    const response = await financeApi.createFinanceReceiptReceiptVoucher(request)
    if (!response.success) {
      throw new Error(response.error || t('receiptVoucher.messages.createError'))
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
            {t('receiptVoucher.title')}
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
                        {t('receiptVoucher.customerSummary.hasReceivables', {
                          name: selectedCustomer.name,
                          count: customerReceivables.length,
                        })}{' '}
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
                      {t('receiptVoucher.customerSummary.noReceivables', {
                        name: selectedCustomer.name,
                      })}
                    </Text>
                  }
                />
              )}
            </Spin>

            {/* Pending receivables list */}
            {customerReceivables.length > 0 && (
              <div className="pending-receivables-list">
                <Text type="secondary" className="list-title">
                  {t('receiptVoucher.customerSummary.pendingReceivables')}
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
                    <Tag color="grey">
                      {t('receiptVoucher.customerSummary.more', {
                        count: customerReceivables.length - 5,
                      })}
                    </Tag>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        <Form onSubmit={handleFormSubmit(onSubmit)} isSubmitting={isSubmitting}>
          <FormSection
            title={t('receiptVoucher.customerInfo.title')}
            description={t('receiptVoucher.customerInfo.description')}
          >
            <FormRow cols={1}>
              <div className="customer-select-wrapper">
                <label className="semi-form-field-label">
                  <span className="semi-form-field-label-text">
                    <span className="semi-form-field-label-required">*</span>
                    {t('receiptVoucher.customerInfo.label')}
                  </span>
                </label>
                <Select
                  placeholder={t('receiptVoucher.customerInfo.placeholder')}
                  value={watchedCustomerId || undefined}
                  onChange={handleCustomerChange}
                  onSearch={handleCustomerSearch}
                  optionList={customerOptions}
                  loading={customerLoading}
                  filter
                  remote
                  showClear
                  style={{ width: '100%' }}
                />
              </div>
            </FormRow>
          </FormSection>

          <FormSection
            title={t('receiptVoucher.paymentInfo.title')}
            description={t('receiptVoucher.paymentInfo.description')}
          >
            <FormRow cols={2}>
              <NumberField
                name="amount"
                control={control}
                label={t('receiptVoucher.paymentInfo.amount')}
                placeholder={t('receiptVoucher.paymentInfo.amountPlaceholder')}
                required
                min={0.01}
                max={999999999.99}
                precision={2}
                prefix="¥"
                helperText={
                  totalOutstanding > 0
                    ? t('receiptVoucher.paymentInfo.outstandingHelper', {
                        amount: formatCurrency(totalOutstanding),
                      })
                    : undefined
                }
              />
              <SelectField
                name="payment_method"
                control={control}
                label={t('receiptVoucher.paymentInfo.method')}
                placeholder={t('receiptVoucher.paymentInfo.methodPlaceholder')}
                options={paymentMethodOptions}
                required
              />
            </FormRow>
            <FormRow cols={2}>
              <DateField
                name="receipt_date"
                control={control}
                label={t('receiptVoucher.paymentInfo.date')}
                placeholder={t('receiptVoucher.paymentInfo.datePlaceholder')}
                required
              />
              <TextField
                name="payment_reference"
                control={control}
                label={t('receiptVoucher.paymentInfo.reference')}
                placeholder={t('receiptVoucher.paymentInfo.referencePlaceholder')}
                helperText={t('receiptVoucher.paymentInfo.referenceHelper')}
              />
            </FormRow>
          </FormSection>

          <FormSection
            title={t('receiptVoucher.otherInfo.title')}
            description={t('receiptVoucher.otherInfo.description')}
          >
            <TextAreaField
              name="remark"
              control={control}
              label={t('receiptVoucher.otherInfo.remark')}
              placeholder={t('receiptVoucher.otherInfo.remarkPlaceholder')}
              rows={3}
              maxCount={500}
            />
          </FormSection>

          <FormActions
            submitText={t('receiptVoucher.actions.create')}
            isSubmitting={isSubmitting}
            onCancel={handleCancel}
            showCancel
          />
        </Form>
      </Card>
    </Container>
  )
}
